package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"
)

var (
	// ErrServiceAlreadyExists 表示服务已经存在
	ErrServiceAlreadyExists = errors.New("service already exists")
	// ErrServiceNotFound 表示服务不存在
	ErrServiceNotFound = errors.New("service not found")
	// ErrServiceStartFailed 表示服务启动失败
	ErrServiceStartFailed = errors.New("service start failed")
	// ErrServiceStopFailed 表示服务停止失败
	ErrServiceStopFailed = errors.New("service stop failed")
)

// ServiceManager 管理所有MCP服务的实例
type ServiceManager struct {
	services                 map[int64]Service
	mutex                    sync.RWMutex
	serviceReadyLocks        map[int64]*sync.Mutex
	healthChecker            *HealthChecker
	initialized              bool
	lastAccessed             map[int64]time.Time
	stdioOnDemandIdleTimeout time.Duration
}

// globalManager 是全局服务管理器实例
var globalManager *ServiceManager
var managerOnce sync.Once

// GetServiceManager 返回全局服务管理器实例
func GetServiceManager() *ServiceManager {
	managerOnce.Do(func() {
		globalManager = &ServiceManager{
			services:                 make(map[int64]Service),
			serviceReadyLocks:        make(map[int64]*sync.Mutex),
			healthChecker:            NewHealthChecker(10 * time.Minute),
			initialized:              false,
			lastAccessed:             make(map[int64]time.Time),
			stdioOnDemandIdleTimeout: 10 * time.Minute, // Default 10 minutes for idle timeout
		}
	})
	return globalManager
}

// Initialize 初始化服务管理器
func (m *ServiceManager) Initialize(ctx context.Context) error {
	if m.initialized {
		return nil
	}

	// Note: HealthChecker is used for registration and health caching, but we don't start
	// its separate checking routine since StartDaemon() already performs comprehensive
	// health checking with additional service management features.

	// 加载并注册所有启用的服务
	services, err := model.GetEnabledServices()
	if err != nil {
		return fmt.Errorf("failed to load enabled services: %w", err)
	}

	// 并发注册服务，避免一个服务失败阻塞其他服务
	var wg sync.WaitGroup
	for _, mcpService := range services {
		wg.Add(1)
		go func(service *model.MCPService) {
			defer wg.Done()
			if err := m.RegisterService(ctx, service); err != nil {
				log.Printf("Failed to register service %s (ID: %d): %v. Please check system logs for details.", service.Name, service.ID, err)
			} else {
				log.Printf("Successfully registered service %s (ID: %d)", service.Name, service.ID)
			}
		}(mcpService)
	}

	// 等待所有服务注册完成（但不阻塞太久）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("All services registration completed")
	case <-time.After(60 * time.Second):
		log.Printf("Service registration timeout after 60 seconds, but continuing...")
	}

	m.initialized = true

	// Start management only after the initial registration pass, so its first
	// health check cannot race an empty or partially populated service map.
	m.StartDaemon()
	return nil
}

// Shutdown 关闭服务管理器
func (m *ServiceManager) Shutdown(ctx context.Context) error {
	// Note: HealthChecker doesn't run a separate daemon anymore, so no need to stop it.
	// The StartDaemon goroutine will naturally terminate when the program exits.

	// 停止所有服务
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, service := range m.services {
		if service.IsRunning() {
			if err := service.Stop(ctx); err != nil {
				log.Printf("Error stopping service %s (ID: %d): %v", service.Name(), service.ID(), err)
				// 继续停止其他服务
			}
		}
	}

	// 清空服务列表
	m.services = make(map[int64]Service)
	m.initialized = false

	return nil
}

// RegisterService 注册一个服务到管理器
func (m *ServiceManager) RegisterService(ctx context.Context, mcpService *model.MCPService) error {
	if !mcpService.Enabled || mcpService.Deleted {
		return fmt.Errorf("cannot register disabled or uninstalled service %d", mcpService.ID)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查服务是否已经存在
	if _, exists := m.services[mcpService.ID]; exists {
		return ErrServiceAlreadyExists
	}

	// 创建服务实例
	service, err := ServiceFactory(mcpService)
	if err != nil {
		return fmt.Errorf("failed to create service instance: %w", err)
	}
	if freshService, refreshErr := model.GetServiceByID(mcpService.ID); refreshErr == nil &&
		(!freshService.Enabled || freshService.Deleted) {
		_ = service.Stop(ctx)
		return fmt.Errorf("cannot register disabled or uninstalled service %d", mcpService.ID)
	}

	// 注册服务
	m.services[mcpService.ID] = service

	// Register to health checker
	m.healthChecker.RegisterService(service)

	// Start service if it's enabled and default on (always start stdio services regardless of strategy)
	if mcpService.DefaultOn && mcpService.Enabled {
		if err := service.Start(ctx); err != nil {
			// Failed to start, but keep the registration
			log.Printf("Failed to start service %s (ID: %d): %v", mcpService.Name, mcpService.ID, err)
		} else {
			log.Printf("Service %s (ID: %d) started at registration", mcpService.Name, mcpService.ID)
		}
	}

	// Prewarm stdio services configured for on-demand startup to avoid first-request installation delays.
	if mcpService.Type == model.ServiceTypeStdio && mcpService.Enabled {
		strategy := common.OptionMap[common.OptionStdioServiceStartupStrategy]
		if strategy == common.StrategyStartOnDemand {
			serviceCopy := *mcpService
			go func(svc model.MCPService) {
				if err := prewarmStdioService(context.Background(), &svc); err != nil {
					common.SysError(fmt.Sprintf("Prewarm failed for stdio service %s (ID: %d): %v", svc.Name, svc.ID, err))
				}
			}(serviceCopy)
		}
	}

	return nil
}

// UnregisterService 从管理器移除一个服务
func (m *ServiceManager) UnregisterService(ctx context.Context, serviceID int64) error {
	unlock := m.lockServiceReady(serviceID)
	defer unlock()
	return m.unregisterService(ctx, serviceID)
}

// UnregisterServiceAndFinalize keeps first-request recovery blocked until the
// caller has persisted the lifecycle state that prevents re-registration.
func (m *ServiceManager) UnregisterServiceAndFinalize(ctx context.Context, serviceID int64, finalize func() error) error {
	if finalize == nil {
		return errors.New("service lifecycle finalizer is nil")
	}

	unlock := m.lockServiceReady(serviceID)
	defer unlock()

	if err := m.unregisterService(ctx, serviceID); err != nil && !errors.Is(err, ErrServiceNotFound) {
		return err
	}
	return finalize()
}

func (m *ServiceManager) unregisterService(ctx context.Context, serviceID int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		m.healthChecker.UnregisterService(serviceID)
		GetHealthCacheManager().DeleteServiceHealth(serviceID)
		GetToolsCacheManager().DeleteServiceTools(serviceID)
		delete(m.lastAccessed, serviceID)
		return ErrServiceNotFound
	}

	// Always stop the service to ensure thorough cleanup (don't rely on IsRunning() check)
	// Add timeout control for stop operation to prevent hanging in container environments
	stopCtx, stopCancel := context.WithTimeout(ctx, 15*time.Second)
	defer stopCancel()

	if err := service.Stop(stopCtx); err != nil {
		// Log the error but continue with cleanup
		log.Printf("Warning: failed to stop service %d during unregister: %v", serviceID, err)
	}

	// 从健康检查器中移除
	m.healthChecker.UnregisterService(serviceID)

	// 从健康状态缓存中移除
	cacheManager := GetHealthCacheManager()
	cacheManager.DeleteServiceHealth(serviceID)
	GetToolsCacheManager().DeleteServiceTools(serviceID)

	// 从服务列表中移除
	delete(m.services, serviceID)
	delete(m.lastAccessed, serviceID)

	return nil
}

// GetService 获取一个服务实例
func (m *ServiceManager) GetService(serviceID int64) (Service, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	service, exists := m.services[serviceID]
	if !exists {
		return nil, ErrServiceNotFound
	}

	return service, nil
}

// EnsureServiceReady registers a missing enabled service and starts it if needed.
// RegisterService serializes creation, so concurrent first requests converge on
// the same manager entry; ErrServiceAlreadyExists is an expected race outcome.
func (m *ServiceManager) EnsureServiceReady(ctx context.Context, mcpService *model.MCPService) (Service, error) {
	if mcpService == nil {
		return nil, errors.New("service config is nil")
	}
	unlock := m.lockServiceReady(mcpService.ID)
	defer unlock()

	// The caller may have loaded this config before waiting for an uninstall or
	// disable operation. Reload under the lifecycle lock so stale enabled state
	// cannot recreate a service after that operation completes.
	freshService, err := model.GetServiceByID(mcpService.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload service %d: %w", mcpService.ID, err)
	}
	if !freshService.Enabled || freshService.Deleted {
		return nil, fmt.Errorf("cannot prepare disabled or uninstalled service %d", freshService.ID)
	}
	mcpService = freshService

	service, err := m.GetService(mcpService.ID)
	if errors.Is(err, ErrServiceNotFound) {
		if registerErr := m.RegisterService(ctx, mcpService); registerErr != nil && !errors.Is(registerErr, ErrServiceAlreadyExists) {
			return nil, fmt.Errorf("failed to register service: %w", registerErr)
		}
		service, err = m.GetService(mcpService.ID)
	}
	if err != nil {
		return nil, err
	}

	if !service.IsRunning() {
		if err := m.StartService(ctx, mcpService.ID); err != nil {
			return nil, err
		}
	}
	m.UpdateServiceAccessTime(mcpService.ID)
	return service, nil
}

func (m *ServiceManager) lockServiceReady(serviceID int64) func() {
	m.mutex.Lock()
	if m.serviceReadyLocks == nil {
		m.serviceReadyLocks = make(map[int64]*sync.Mutex)
	}
	serviceLock, ok := m.serviceReadyLocks[serviceID]
	if !ok {
		serviceLock = &sync.Mutex{}
		m.serviceReadyLocks[serviceID] = serviceLock
	}
	m.mutex.Unlock()

	serviceLock.Lock()
	return serviceLock.Unlock
}

// UpdateServiceAccessTime 更新服务的最后访问时间
func (m *ServiceManager) UpdateServiceAccessTime(serviceID int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.lastAccessed[serviceID] = time.Now()
}

// StartService 启动一个服务
func (m *ServiceManager) StartService(ctx context.Context, serviceID int64) error {
	service, err := m.GetService(serviceID)
	if err != nil {
		return err
	}

	if service.IsRunning() {
		// 服务已经在运行，不需要再次启动
		return nil
	}

	if err := service.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// StopService 停止一个服务
func (m *ServiceManager) StopService(ctx context.Context, serviceID int64) error {
	service, err := m.GetService(serviceID)
	if err != nil {
		return err
	}

	if !service.IsRunning() {
		// 服务已经停止，不需要再次停止
		return nil
	}

	if err := service.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

// RestartService 重启一个服务
func (m *ServiceManager) RestartService(ctx context.Context, serviceID int64) error {
	service, err := m.GetService(serviceID)
	if err != nil {
		return err
	}

	// 如果服务正在运行，先停止它
	if service.IsRunning() {
		if err := service.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop service during restart: %w", err)
		}
	}

	// 启动服务
	if err := service.Start(ctx); err != nil {
		return fmt.Errorf("failed to start service during restart: %w", err)
	}

	return nil
}

// GetServiceHealth 获取服务的健康状态
func (m *ServiceManager) GetServiceHealth(serviceID int64) (*ServiceHealth, error) {
	return m.healthChecker.GetServiceHealth(serviceID)
}

// ForceCheckServiceHealth 强制检查服务的健康状态
func (m *ServiceManager) ForceCheckServiceHealth(serviceID int64) (*ServiceHealth, error) {
	return m.healthChecker.ForceCheckService(serviceID)
}

// UpdateServiceConfig 更新服务配置
func (m *ServiceManager) UpdateServiceConfig(serviceID int64, config map[string]interface{}) error {
	service, err := m.GetService(serviceID)
	if err != nil {
		return err
	}

	return service.UpdateConfig(config)
}

// GetAllServices 获取所有服务
func (m *ServiceManager) GetAllServices() []Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	services := make([]Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}

	return services
}

// GetServiceHealthJSON 获取服务健康状态的JSON字符串
func (m *ServiceManager) GetServiceHealthJSON(serviceID int64) (string, error) {
	health, err := m.GetServiceHealth(serviceID)
	if err != nil {
		return "", err
	}

	healthJSON, err := json.Marshal(health)
	if err != nil {
		return "", fmt.Errorf("failed to marshal health data: %w", err)
	}

	return string(healthJSON), nil
}

// UpdateMCPServiceHealth 更新缓存中服务的健康状态
func (m *ServiceManager) UpdateMCPServiceHealth(serviceID int64) error {
	health, err := m.GetServiceHealth(serviceID)
	if err != nil {
		return err
	}

	// Attach tool summary from tools cache
	if entry, found := GetToolsCacheManager().GetServiceTools(serviceID); found {
		health.ToolCount = len(entry.Tools)
		health.ToolsFetched = true
	} else {
		health.ToolCount = 0
		health.ToolsFetched = false
	}

	// 获取全局健康状态缓存管理器
	cacheManager := GetHealthCacheManager()

	// 将健康状态存储到缓存中
	cacheManager.SetServiceHealth(serviceID, health)

	return nil
}

// StartDaemon starts the primary service management daemon that handles:
// 1. Health checking for all services
// 2. Auto-restart of stopped services (except on-demand stdio services)
// 3. Idle shutdown for on-demand stdio services
// This replaces the need for a separate HealthChecker daemon.
func (m *ServiceManager) StartDaemon() {
	go func() {
		// Wait a short time for services to stabilize after registration
		time.Sleep(5 * time.Second)

		// Perform initial health check
		m.performHealthCheckAndManagement()

		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.performHealthCheckAndManagement()
		}
	}()
}

// performHealthCheckAndManagement performs health checking and service management operations
func (m *ServiceManager) performHealthCheckAndManagement() {
	m.mutex.RLock()
	services := make([]Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}
	lastAccessedCopy := make(map[int64]time.Time)
	for k, v := range m.lastAccessed {
		lastAccessedCopy[k] = v
	}
	m.mutex.RUnlock()

	for _, service := range services {
		// Check if this is a stdio service with on-demand strategy
		if service.Type() == model.ServiceTypeStdio {
			strategy := common.OptionMap[common.OptionStdioServiceStartupStrategy]
			if strategy == common.StrategyStartOnDemand && service.IsRunning() {
				// Check for idle timeout
				if lastAccess, exists := lastAccessedCopy[service.ID()]; exists {
					if time.Since(lastAccess) > m.stdioOnDemandIdleTimeout {
						ctx := context.Background()
						if err := m.StopService(ctx, service.ID()); err != nil {
							log.Printf("Failed to stop idle stdio service %s (ID: %d): %v", service.Name(), service.ID(), err)
						} else {
							log.Printf("Stopped idle stdio service: %s (ID: %d) after %v of inactivity",
								service.Name(), service.ID(), time.Since(lastAccess))
							if _, err := m.healthChecker.ForceCheckService(service.ID()); err != nil {
								log.Printf("Failed to refresh health after stopping idle stdio service %s (ID: %d): %v", service.Name(), service.ID(), err)
							}
						}
						continue // Skip auto-restart logic for this service
					}
				}
			}
		}

		// Standard auto-restart logic for services that should not be idle-stopped
		health, err := m.ForceCheckServiceHealth(service.ID())
		if err != nil {
			continue
		}

		// Only auto-restart services that are not stdio services with on-demand strategy
		shouldAutoRestart := true
		if service.Type() == model.ServiceTypeStdio {
			strategy := common.OptionMap[common.OptionStdioServiceStartupStrategy]
			if strategy == common.StrategyStartOnDemand {
				shouldAutoRestart = false
			}
		}

		// Get current service config from database to check if it's still enabled
		currentService, err := model.GetServiceByID(service.ID())
		if err != nil {
			log.Printf("Failed to get current service config for %d: %v", service.ID(), err)
			continue
		}

		// Don't auto-restart disabled services
		if !currentService.Enabled {
			shouldAutoRestart = false
			log.Printf("Skipping auto-restart for disabled service: %s (ID: %d)", service.Name(), service.ID())
		}

		if shouldAutoRestart && health.Status == StatusStopped {
			ctx := context.Background()
			if err := m.RestartService(ctx, service.ID()); err != nil {
				// Log error but continue processing other services
				log.Printf("Failed to auto-restart service %d: %v", service.ID(), err)
				continue
			}
			log.Printf("Auto-restarted stopped service: %s (ID: %d)", service.Name(), service.ID())
		}
	}
}

// GetSSEServiceByName 根据服务名查找 SSESvc 实例
func (m *ServiceManager) GetSSEServiceByName(serviceName string) (*SSESvc, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, svc := range m.services {
		if svc.Name() == serviceName && svc.Type() == model.ServiceTypeSSE {
			if sseSvc, ok := svc.(*SSESvc); ok {
				return sseSvc, nil
			}
		}
	}
	return nil, ErrServiceNotFound
}

// SetService 允许注入 mock Service（测试专用）
func (m *ServiceManager) SetService(serviceID int64, svc Service) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.services[serviceID] = svc
}
