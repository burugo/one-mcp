package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

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
	services      map[int64]Service
	mutex         sync.RWMutex
	healthChecker *HealthChecker
	initialized   bool
}

// globalManager 是全局服务管理器实例
var globalManager *ServiceManager
var managerOnce sync.Once

// GetServiceManager 返回全局服务管理器实例
func GetServiceManager() *ServiceManager {
	managerOnce.Do(func() {
		globalManager = &ServiceManager{
			services:      make(map[int64]Service),
			healthChecker: NewHealthChecker(1 * time.Minute),
			initialized:   false,
		}
	})
	return globalManager
}

// Initialize 初始化服务管理器
func (m *ServiceManager) Initialize(ctx context.Context) error {
	if m.initialized {
		return nil
	}

	// 启动健康检查
	m.healthChecker.Start()

	// 启动自动重启守护线程
	m.StartDaemon()

	// 加载并注册所有启用的服务
	services, err := model.GetEnabledServices()
	if err != nil {
		return fmt.Errorf("failed to load enabled services: %w", err)
	}

	for _, mcpService := range services {
		if err := m.RegisterService(ctx, mcpService); err != nil {
			log.Printf("Failed to register service %s (ID: %d): %v", mcpService.Name, mcpService.ID, err)
			// 继续注册其他服务
			continue
		}
	}

	m.initialized = true
	return nil
}

// Shutdown 关闭服务管理器
func (m *ServiceManager) Shutdown(ctx context.Context) error {
	// 停止健康检查
	m.healthChecker.Stop()

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

	// 注册服务
	m.services[mcpService.ID] = service

	// 注册到健康检查器
	m.healthChecker.RegisterService(service)

	// 如果服务配置为默认启用，则启动服务
	if mcpService.DefaultOn && mcpService.Enabled {
		if err := service.Start(ctx); err != nil {
			// 启动失败，但仍然保留注册
			log.Printf("Failed to start service %s (ID: %d): %v", mcpService.Name, mcpService.ID, err)
		}
	}

	return nil
}

// UnregisterService 从管理器移除一个服务
func (m *ServiceManager) UnregisterService(ctx context.Context, serviceID int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}

	// 如果服务正在运行，先停止它
	if service.IsRunning() {
		if err := service.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}

	// 从健康检查器中移除
	m.healthChecker.UnregisterService(serviceID)

	// 从服务列表中移除
	delete(m.services, serviceID)

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

// UpdateMCPServiceHealth 更新数据库中服务的健康状态
func (m *ServiceManager) UpdateMCPServiceHealth(serviceID int64) error {
	health, err := m.GetServiceHealth(serviceID)
	if err != nil {
		return err
	}

	// 获取服务
	mcpService, err := model.GetServiceByID(serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service from database: %w", err)
	}

	// 序列化健康详情
	healthJSON, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("failed to marshal health details: %w", err)
	}

	// 更新健康状态
	mcpService.HealthStatus = string(health.Status)
	mcpService.LastHealthCheck = health.LastChecked
	mcpService.HealthDetails = string(healthJSON)

	// 保存到数据库
	if err := model.UpdateService(mcpService); err != nil {
		return fmt.Errorf("failed to update service health in database: %w", err)
	}

	return nil
}

// StartDaemon starts the daemon thread for auto-restarting stopped services
func (m *ServiceManager) StartDaemon() {
	go func() {
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.mutex.RLock()
			services := make([]Service, 0, len(m.services))
			for _, service := range m.services {
				services = append(services, service)
			}
			m.mutex.RUnlock()

			for _, service := range services {
				// 检查服务状态
				health, err := m.ForceCheckServiceHealth(service.ID())
				if err != nil {
					continue
				}

				// 如果服务已停止，尝试重启
				if health.Status == StatusStopped {
					ctx := context.Background()
					if err := m.RestartService(ctx, service.ID()); err != nil {
						// 记录错误但继续处理其他服务
						log.Printf("Failed to auto-restart service %d: %v", service.ID(), err)
						continue
					}
					log.Printf("Auto-restarted stopped service: %s (ID: %d)", service.Name(), service.ID())
				}
			}
		}
	}()
}
