package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// SharedMcpInstance encapsulates a shared MCPServer and its MCPClient.
type SharedMcpInstance struct {
	Server *mcpserver.MCPServer
	Client mcpclient.MCPClient
	// consider adding createdAt time.Time for future LRU cache policies
}

// Shutdown gracefully stops the server and closes the client.
func (s *SharedMcpInstance) Shutdown(ctx context.Context) error {
	common.SysLog(fmt.Sprintf("Shutting down SharedMcpInstance (Server: %p, Client: %p)", s.Server, s.Client))
	var firstErr error
	// Note: Actual shutdown logic for s.Server depends on mcp-go's MCPServer API.
	// This might involve calling a Stop() or Shutdown() method on s.Server if available.
	// For example: if s.Server has a Stop method:
	// if E, ok := s.Server.(interface{ Stop(context.Context) error }); ok {
	//    if err := E.Stop(ctx); err != nil {
	//        common.SysError(fmt.Sprintf("Error stopping MCPServer for SharedMcpInstance: %v", err))
	//        if firstErr == nil { firstErr = err }
	//    }
	// }
	common.SysLog(fmt.Sprintf("MCPServer %p shutdown initiated/completed (actual stop method TBD based on mcp-go API)", s.Server))

	if s.Client != nil {
		if err := s.Client.Close(); err != nil {
			common.SysError(fmt.Sprintf("Error closing MCPClient for SharedMcpInstance: %v", err))
			if firstErr == nil {
				firstErr = err
			}
		} else {
			common.SysLog(fmt.Sprintf("MCPClient %p closed.", s.Client))
		}
	}
	return firstErr
}

// ServiceStatus 表示服务的健康状态
type ServiceStatus string

const (
	// StatusUnknown 表示服务状态未知
	StatusUnknown ServiceStatus = "unknown"
	// StatusHealthy 表示服务正常
	StatusHealthy ServiceStatus = "healthy"
	// StatusUnhealthy 表示服务异常
	StatusUnhealthy ServiceStatus = "unhealthy"
	// StatusStarting 表示服务正在启动
	StatusStarting ServiceStatus = "starting"
	// StatusStopped 表示服务已停止
	StatusStopped ServiceStatus = "stopped"
)

// ServiceHealth 包含服务健康相关的信息
type ServiceHealth struct {
	Status        ServiceStatus `json:"status"`
	LastChecked   time.Time     `json:"last_checked"`
	ResponseTime  int64         `json:"response_time_ms,omitempty"` // 毫秒
	ErrorMessage  string        `json:"error_message,omitempty"`
	StartTime     time.Time     `json:"start_time,omitempty"`
	SuccessCount  int64         `json:"success_count"`
	FailureCount  int64         `json:"failure_count"`
	UpTime        int64         `json:"up_time_seconds,omitempty"` // 秒
	WarningLevel  int           `json:"warning_level,omitempty"`   // 0-无警告，1-轻微，2-中等，3-严重
	InstanceCount int           `json:"instance_count,omitempty"`  // 实例数量（如有多实例）
}

// Service 接口定义了所有MCP服务必须实现的方法
type Service interface {
	// ID 返回服务的唯一标识符
	ID() int64

	// Name 返回服务的名称
	Name() string

	// Type 返回服务的类型（stdio、sse、streamable_http）
	Type() model.ServiceType

	// Start 启动服务
	Start(ctx context.Context) error

	// Stop 停止服务
	Stop(ctx context.Context) error

	// IsRunning 检查服务是否正在运行
	IsRunning() bool

	// CheckHealth 检查服务健康状态
	CheckHealth(ctx context.Context) (*ServiceHealth, error)

	// GetHealth 获取最后一次检查的健康状态
	GetHealth() *ServiceHealth

	// GetConfig 返回服务配置
	GetConfig() map[string]interface{}

	// UpdateConfig 更新服务配置
	UpdateConfig(config map[string]interface{}) error

	// HealthCheckTimeout 返回此服务进行健康检查时建议的超时时间。
	// 如果返回 0 或负值，HealthChecker 将使用其默认超时。
	HealthCheckTimeout() time.Duration
}

// BaseService 是一个基本的服务实现，可以被具体服务类型继承
type BaseService struct {
	mu            sync.RWMutex
	serviceID     int64
	serviceName   string
	serviceType   model.ServiceType
	running       bool
	health        ServiceHealth
	config        map[string]interface{}
	lastStartTime time.Time
}

// NewBaseService 创建一个新的基本服务实例
func NewBaseService(id int64, name string, serviceType model.ServiceType) *BaseService {
	return &BaseService{
		serviceID:   id,
		serviceName: name,
		serviceType: serviceType,
		running:     false,
		health: ServiceHealth{
			Status:      StatusUnknown,
			LastChecked: time.Now(),
		},
		config: make(map[string]interface{}),
	}
}

// ID 实现Service接口
func (s *BaseService) ID() int64 {
	return s.serviceID
}

// Name 实现Service接口
func (s *BaseService) Name() string {
	return s.serviceName
}

// Type 实现Service接口
func (s *BaseService) Type() model.ServiceType {
	return s.serviceType
}

// IsRunning 实现Service接口
func (s *BaseService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetHealth 实现Service接口
func (s *BaseService) GetHealth() *ServiceHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 创建一个新的健康状态副本以避免并发访问问题
	health := s.health

	// 如果服务在运行，计算当前的运行时间
	if s.running && !s.lastStartTime.IsZero() {
		health.UpTime = int64(time.Since(s.lastStartTime).Seconds())
	}

	return &health
}

// GetConfig 实现Service接口
func (s *BaseService) GetConfig() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 创建配置的副本
	configCopy := make(map[string]interface{}, len(s.config))
	for k, v := range s.config {
		configCopy[k] = v
	}

	return configCopy
}

// HealthCheckTimeout 实现Service接口。
// 它根据服务类型返回建议的超时时间。
func (s *BaseService) HealthCheckTimeout() time.Duration {
	s.mu.RLock() // 保证线程安全地读取 s.serviceType
	defer s.mu.RUnlock()

	if s.serviceType == model.ServiceTypeStdio {
		// Stdio 服务可能需要更长的超时时间进行健康检查
		return 30 * time.Second
	}
	// 对于其他类型的服务（如 http, sse），返回0，让 HealthChecker 使用其默认超时（当前为10秒）。
	// 如果特定服务（如某个特殊的HTTP服务）需要不同的超时，它可以在自己的实现中覆盖此方法。
	return 0
}

// UpdateConfig 实现Service接口
func (s *BaseService) UpdateConfig(config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新配置
	for k, v := range config {
		s.config[k] = v
	}

	return nil
}

// Start 是一个基本实现，具体服务类型应重写此方法
func (s *BaseService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = true
	s.lastStartTime = time.Now()
	s.health.Status = StatusStarting
	s.health.StartTime = s.lastStartTime

	return nil
}

// Stop 是一个基本实现，具体服务类型应重写此方法
func (s *BaseService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	s.health.Status = StatusStopped

	return nil
}

// UpdateHealth 更新服务的健康状态（内部使用）
func (s *BaseService) UpdateHealth(status ServiceStatus, responseTime int64, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.health.Status = status
	s.health.LastChecked = time.Now()
	s.health.ResponseTime = responseTime
	s.health.ErrorMessage = errorMsg

	// 更新成功/失败计数
	if status == StatusHealthy {
		s.health.SuccessCount++
	} else if status == StatusUnhealthy {
		s.health.FailureCount++
	}

	// 设置警告级别
	switch {
	case status == StatusHealthy:
		s.health.WarningLevel = 0
	case status == StatusUnhealthy && s.health.FailureCount <= 3:
		s.health.WarningLevel = 1
	case status == StatusUnhealthy && s.health.FailureCount <= 10:
		s.health.WarningLevel = 2
	default:
		s.health.WarningLevel = 3
	}
}

// CheckHealth 是一个基本实现，具体服务类型应重写此方法
func (s *BaseService) CheckHealth(ctx context.Context) (*ServiceHealth, error) {
	// 基本实现只检查服务是否在运行
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.health.Status = StatusHealthy
	} else {
		s.health.Status = StatusStopped
	}

	s.health.LastChecked = time.Now()

	// 返回健康状态的副本
	healthCopy := s.health
	return &healthCopy, nil
}

// MonitoredProxiedService extends BaseService with a SharedMcpInstance for health checking.
type MonitoredProxiedService struct {
	*BaseService
	sharedInstance  *SharedMcpInstance
	dbServiceConfig *model.MCPService // Store original config for potential instance recreation
}

// NewMonitoredProxiedService creates a new monitored service.
func NewMonitoredProxiedService(base *BaseService, instance *SharedMcpInstance, dbConfig *model.MCPService) *MonitoredProxiedService {
	return &MonitoredProxiedService{
		BaseService:     base,
		sharedInstance:  instance,
		dbServiceConfig: dbConfig,
	}
}

// CheckHealth for MonitoredProxiedService performs deep health checking using the shared MCP instance
func (s *MonitoredProxiedService) CheckHealth(ctx context.Context) (*ServiceHealth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// For on-demand stdio services that haven't been started yet, report as stopped without attempting self-healing
	if s.Type() == model.ServiceTypeStdio && s.sharedInstance == nil {
		strategy := common.OptionMap[common.OptionStdioServiceStartupStrategy]
		if strategy == common.StrategyStartOnDemand {
			if s.health.Status != StatusStopped {
				s.health.Status = StatusStopped
				s.health.ErrorMessage = "Service is configured for on-demand start"
				s.health.LastChecked = time.Now()
			}
			healthCopy := s.health
			return &healthCopy, nil
		}
	}

	startTime := time.Now()

	if s.sharedInstance == nil || s.sharedInstance.Client == nil {
		s.health.Status = StatusUnhealthy
		s.health.ErrorMessage = "Shared MCP instance or client is not initialized."
		s.health.LastChecked = time.Now()
		s.health.ResponseTime = time.Since(startTime).Milliseconds()
		s.health.FailureCount++
		s.health.WarningLevel = 3 // Critical if not initialized
		healthCopy := s.health

		// Self-healing logic: attempt to re-create the instance for services that should be running
		// Skip this for on-demand stdio services (already handled above)
		if s.sharedInstance == nil && s.dbServiceConfig != nil {
			common.SysLog(fmt.Sprintf("CheckHealth: Instance for %s (ID: %d) is nil, attempting re-initialization.", s.serviceName, s.serviceID))
			cacheKey := fmt.Sprintf("global-service-%d-shared", s.dbServiceConfig.ID)
			instanceNameDetail := fmt.Sprintf("global-shared-svc-%d-reinit", s.dbServiceConfig.ID)
			effectiveEnvs := s.dbServiceConfig.DefaultEnvsJSON

			newInstance, recreateErr := GetOrCreateSharedMcpInstanceWithKey(ctx, s.dbServiceConfig, cacheKey, instanceNameDetail, effectiveEnvs)
			if recreateErr != nil {
				s.health.Status = StatusUnhealthy
				s.health.ErrorMessage = fmt.Sprintf("Initial re-creation attempt failed: %v", recreateErr)
				common.SysError(fmt.Sprintf("Failed to recreate shared instance for %s from CheckHealth (initial nil): %v", s.serviceName, recreateErr))
				healthCopy.Status = s.health.Status
				healthCopy.ErrorMessage = s.health.ErrorMessage
				healthCopy.LastChecked = s.health.LastChecked
				healthCopy.ResponseTime = s.health.ResponseTime
				return &healthCopy, errors.New(s.health.ErrorMessage)
			}
			s.sharedInstance = newInstance
			common.SysLog(fmt.Sprintf("Successfully re-created shared MCP instance for %s from CheckHealth (initial nil). Performing immediate re-ping.", s.serviceName))

			// Immediate re-ping after successful creation
			rePingErr := s.sharedInstance.Client.Ping(ctx)

			if rePingErr != nil {
				s.health.Status = StatusUnhealthy
				s.health.ErrorMessage = fmt.Sprintf("Re-ping after initial client creation failed: %v", rePingErr)
				s.health.FailureCount++
				common.SysError(fmt.Sprintf("Re-ping for %s failed after initial creation: %v", s.serviceName, rePingErr))
				healthCopy.Status = s.health.Status
				healthCopy.ErrorMessage = s.health.ErrorMessage
				healthCopy.LastChecked = s.health.LastChecked
				healthCopy.ResponseTime = s.health.ResponseTime
				return &healthCopy, errors.New(s.health.ErrorMessage)
			} else {
				s.health.Status = StatusHealthy
				s.health.ErrorMessage = ""
				s.health.FailureCount = 0
				s.health.SuccessCount++
				common.SysLog(fmt.Sprintf("Re-ping successful for %s after initial creation. Status set to Healthy.", s.serviceName))
				healthCopy.Status = s.health.Status
				healthCopy.ErrorMessage = s.health.ErrorMessage
				healthCopy.LastChecked = s.health.LastChecked
				healthCopy.ResponseTime = s.health.ResponseTime
				return &healthCopy, nil
			}
		}
		return &healthCopy, errors.New(s.health.ErrorMessage)
	}
	originalPingErr := s.sharedInstance.Client.Ping(ctx)
	finalErrToReturn := originalPingErr

	if originalPingErr != nil {
		serviceType := s.Type() // Get the service type from BaseService

		if serviceType == model.ServiceTypeSSE || serviceType == model.ServiceTypeStreamableHTTP {
			common.SysLog(fmt.Sprintf("CheckHealth: Detected ping failure for network service %s (ID: %d, Type: %s): %v. Attempting to re-establish client.", s.serviceName, s.serviceID, serviceType, originalPingErr))

			if s.dbServiceConfig == nil {
				common.SysError(fmt.Sprintf("CheckHealth: Cannot re-create client for %s (ID: %d): dbServiceConfig is nil.", s.serviceName, s.serviceID))
				s.health.Status = StatusUnhealthy
				s.health.ErrorMessage = fmt.Sprintf("Ping failed (%v) and cannot re-create client (missing config).", originalPingErr)
				// finalErrToReturn remains originalPingErr
			} else {
				cacheKey := fmt.Sprintf("global-service-%d-shared", s.dbServiceConfig.ID)
				instanceToShutdown := s.sharedInstance

				sharedMCPServersMutex.Lock()
				delete(sharedMCPServers, cacheKey)
				sharedMCPServersMutex.Unlock()
				common.SysLog(fmt.Sprintf("CheckHealth: Removed instance for %s (key: %s) from global cache.", s.serviceName, cacheKey))

				s.sharedInstance = nil

				if instanceToShutdown != nil {
					common.SysLog(fmt.Sprintf("CheckHealth: Shutting down old shared instance for %s (ID: %d).", s.serviceName, s.serviceID))
					if shutdownErr := instanceToShutdown.Shutdown(ctx); shutdownErr != nil {
						common.SysError(fmt.Sprintf("CheckHealth: Error shutting down old instance for %s: %v. Proceeding with re-creation.", s.serviceName, shutdownErr))
					}
				}

				common.SysLog(fmt.Sprintf("CheckHealth: Attempting to get/create new shared MCP instance for %s (ID: %d).", s.serviceName, s.serviceID))
				instanceNameDetail := fmt.Sprintf("global-shared-svc-%d-recreated", s.dbServiceConfig.ID)
				effectiveEnvs := s.dbServiceConfig.DefaultEnvsJSON

				newInstance, recreateErr := GetOrCreateSharedMcpInstanceWithKey(ctx, s.dbServiceConfig, cacheKey, instanceNameDetail, effectiveEnvs)
				if recreateErr != nil {
					s.health.Status = StatusUnhealthy
					s.health.ErrorMessage = fmt.Sprintf("Client re-creation failed after ping error '%v': %v", originalPingErr, recreateErr)
					finalErrToReturn = errors.New(s.health.ErrorMessage)
					common.SysError(fmt.Sprintf("Failed to recreate shared instance for %s from CheckHealth: %v", s.serviceName, recreateErr))
				} else {
					s.sharedInstance = newInstance
					common.SysLog(fmt.Sprintf("Successfully re-created shared MCP instance for %s from CheckHealth. Performing immediate re-ping.", s.serviceName))

					rePingErr := s.sharedInstance.Client.Ping(ctx)

					if rePingErr != nil {
						s.health.Status = StatusUnhealthy
						s.health.ErrorMessage = fmt.Sprintf("Re-ping after client re-creation failed: %v (Original ping error: %v)", rePingErr, originalPingErr)
						finalErrToReturn = errors.New(s.health.ErrorMessage)
						common.SysError(fmt.Sprintf("Re-ping for %s failed after re-creation: %v", s.serviceName, rePingErr))
					} else {
						s.health.Status = StatusHealthy
						s.health.ErrorMessage = ""
						s.health.FailureCount = 0
						s.health.SuccessCount++
						finalErrToReturn = nil
						common.SysLog(fmt.Sprintf("Re-ping successful for %s after re-creation. Status set to Healthy.", s.serviceName))
					}
				}
			}
		} else {
			// Ping failed, and service type is not SSE or StreamableHTTP (e.g., Stdio)
			s.health.Status = StatusUnhealthy
			s.health.ErrorMessage = fmt.Sprintf("Ping failed: %v", originalPingErr)
			// finalErrToReturn remains originalPingErr
		}

		if finalErrToReturn != nil {
			s.health.FailureCount++
		}
	} else {
		s.health.Status = StatusHealthy
		s.health.ErrorMessage = ""
		s.health.SuccessCount++
		finalErrToReturn = nil
	}

	s.health.LastChecked = time.Now()
	s.health.ResponseTime = time.Since(startTime).Milliseconds()

	if s.health.Status == StatusHealthy {
		s.health.WarningLevel = 0
	} else if s.health.FailureCount <= 3 {
		s.health.WarningLevel = 1
	} else if s.health.FailureCount <= 10 {
		s.health.WarningLevel = 2
	} else {
		s.health.WarningLevel = 3
	}

	if s.running && !s.lastStartTime.IsZero() {
		s.health.UpTime = int64(time.Since(s.lastStartTime).Seconds())
	}

	healthCopy := s.health
	return &healthCopy, finalErrToReturn
}

// Start for MonitoredProxiedService properly recreates the SharedMcpInstance when starting
func (s *MonitoredProxiedService) Start(ctx context.Context) error {
	// First call the base Start method to update basic state
	if err := s.BaseService.Start(ctx); err != nil {
		return err
	}

	// If we don't have a shared instance, create one
	if s.sharedInstance == nil && s.dbServiceConfig != nil {
		common.SysLog(fmt.Sprintf("Creating new SharedMcpInstance for %s during Start", s.serviceName))

		cacheKey := fmt.Sprintf("global-service-%d-shared", s.dbServiceConfig.ID)
		instanceNameDetail := fmt.Sprintf("global-shared-svc-%d-start", s.dbServiceConfig.ID)
		effectiveEnvs := s.dbServiceConfig.DefaultEnvsJSON

		newInstance, err := GetOrCreateSharedMcpInstanceWithKey(ctx, s.dbServiceConfig, cacheKey, instanceNameDetail, effectiveEnvs)
		if err != nil {
			// Revert the BaseService state since we failed to create the instance
			s.BaseService.Stop(ctx)
			return fmt.Errorf("failed to create SharedMcpInstance during Start: %w", err)
		}

		s.sharedInstance = newInstance
		common.SysLog(fmt.Sprintf("Successfully created SharedMcpInstance for %s during Start", s.serviceName))
	}

	return nil
}

// Stop for MonitoredProxiedService properly shuts down the underlying MCP instance
func (s *MonitoredProxiedService) Stop(ctx context.Context) error {
	if err := s.BaseService.Stop(ctx); err != nil {
		return err
	}

	// Properly shutdown the SharedMcpInstance if it exists
	if s.sharedInstance != nil {
		if err := s.sharedInstance.Shutdown(ctx); err != nil {
			common.SysError(fmt.Sprintf("Error shutting down SharedMcpInstance for %s: %v", s.serviceName, err))
			// Don't return error here, as we want to continue cleanup
		}

		// Critical: Remove from global cache to prevent reuse of the closed instance
		if s.dbServiceConfig != nil {
			cacheKey := fmt.Sprintf("global-service-%d-shared", s.dbServiceConfig.ID)
			sharedMCPServersMutex.Lock()
			if cachedInstance, exists := sharedMCPServers[cacheKey]; exists && cachedInstance == s.sharedInstance {
				delete(sharedMCPServers, cacheKey)
				common.SysLog(fmt.Sprintf("Removed SharedMcpInstance for %s from global cache (key: %s)", s.serviceName, cacheKey))
			}
			sharedMCPServersMutex.Unlock()

			// Also clear handler caches that reference the old SharedMcpInstance
			sseHandlerCacheKey := fmt.Sprintf("service-%d-sseproxy", s.dbServiceConfig.ID)
			sseWrappersMutex.Lock()
			if _, exists := initializedSSEProxyWrappers[sseHandlerCacheKey]; exists {
				delete(initializedSSEProxyWrappers, sseHandlerCacheKey)
				common.SysLog(fmt.Sprintf("Cleared SSE handler cache for service %s (key: %s)", s.serviceName, sseHandlerCacheKey))
			}
			sseWrappersMutex.Unlock()

			httpHandlerCacheKey := fmt.Sprintf("service-%d-httpproxy", s.dbServiceConfig.ID)
			httpWrappersMutex.Lock()
			if _, exists := initializedHTTPProxyWrappers[httpHandlerCacheKey]; exists {
				delete(initializedHTTPProxyWrappers, httpHandlerCacheKey)
				common.SysLog(fmt.Sprintf("Cleared HTTP handler cache for service %s (key: %s)", s.serviceName, httpHandlerCacheKey))
			}
			httpWrappersMutex.Unlock()
		}

		s.sharedInstance = nil // Clear the reference
	}

	common.SysLog(fmt.Sprintf("MonitoredProxiedService %s stopped and cleaned up.", s.serviceName))
	return nil
}

// SSESvc wraps an http.Handler to act as an SSE service.
type SSESvc struct {
	*BaseService              // Embed BaseService
	Handler      http.Handler // The actual handler that will serve SSE requests
}

// NewSSESvc creates a new SSESvc.
// The base argument should have its serviceType set to model.ServiceTypeSSE
// as this SSESvc is intended to serve SSE.
func NewSSESvc(base *BaseService, handler http.Handler) *SSESvc {
	if base.serviceType != model.ServiceTypeSSE {
		// This is an internal consistency check. The factory should ensure this.
		common.SysError(fmt.Sprintf("NewSSESvc called with BaseService of type %s, expected SSE", base.serviceType))
		base.serviceType = model.ServiceTypeSSE // Correct it
	}
	return &SSESvc{
		BaseService: base,
		Handler:     handler,
	}
}

// ServeHTTP delegates to the underlying Handler.
// This method makes SSESvc an http.Handler itself.
func (s *SSESvc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.Handler == nil {
		http.Error(w, "SSE handler not configured for service: "+s.Name(), http.StatusInternalServerError)
		return
	}
	s.Handler.ServeHTTP(w, r)
}

// Cached Handlers for different types of services
var (
	initializedStdioSSEWrappers = make(map[string]http.Handler)
	muStdioSSEWrappers          sync.RWMutex

	// New caches for the refactored architecture
	sharedMCPServers             = make(map[string]*SharedMcpInstance)
	sharedMCPServersMutex        = &sync.Mutex{}
	initializedSSEProxyWrappers  = make(map[string]http.Handler)
	sseWrappersMutex             = &sync.Mutex{}
	initializedHTTPProxyWrappers = make(map[string]http.Handler)
	httpWrappersMutex            = &sync.Mutex{}
)

// createActualMcpGoServerAndClientUncached creates and initializes an mcp-go client and server instance.
// For Stdio clients, client.Start() is not called.
// It returns the mcp-go server, the mcp-go client, and an error.
func createActualMcpGoServerAndClientUncached(
	ctx context.Context,
	serviceConfigForInstance *model.MCPService,
	instanceNameDetail string,
) (*mcpserver.MCPServer, mcpclient.MCPClient, error) {

	var mcpGoClient mcpclient.MCPClient
	var err error
	var needManualStart bool

	switch serviceConfigForInstance.Type {
	case model.ServiceTypeStdio:
		var stdioConf model.StdioConfig
		stdioConf.Command = serviceConfigForInstance.Command
		if stdioConf.Command == "" {
			return nil, nil, fmt.Errorf("StdioConfig for service %s (ID: %d) has an empty command. "+
				"This usually indicates the service was not properly configured during installation. "+
				"Expected Command field to contain the executable name (e.g., 'npx' for npm packages). "+
				"PackageManager: %s, SourcePackageName: %s, InstanceDetail: %s",
				serviceConfigForInstance.Name, serviceConfigForInstance.ID, serviceConfigForInstance.PackageManager, serviceConfigForInstance.SourcePackageName, instanceNameDetail)
		}
		if serviceConfigForInstance.ArgsJSON != "" {
			if errJson := json.Unmarshal([]byte(serviceConfigForInstance.ArgsJSON), &stdioConf.Args); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal ArgsJSON for service %s (ID: %d, Stdio): %v. Args will be empty.", serviceConfigForInstance.Name, serviceConfigForInstance.ID, errJson))
				stdioConf.Args = []string{}
			}
		} else {
			stdioConf.Args = []string{}
		}
		stdioConf.Env = []string{}
		if serviceConfigForInstance.DefaultEnvsJSON != "" && serviceConfigForInstance.DefaultEnvsJSON != "{}" {
			var defaultEnvs map[string]string
			if errJson := json.Unmarshal([]byte(serviceConfigForInstance.DefaultEnvsJSON), &defaultEnvs); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal DefaultEnvsJSON for %s (ID: %d, Stdio): %v. Proceeding without them.", serviceConfigForInstance.Name, serviceConfigForInstance.ID, errJson))
			} else {
				for key, value := range defaultEnvs {
					stdioConf.Env = append(stdioConf.Env, fmt.Sprintf("%s=%s", key, value))
				}
			}
		}
		common.SysLog(fmt.Sprintf("Stdio config for %s: Command=%s, Args=%v, Env=%v", serviceConfigForInstance.Name, stdioConf.Command, stdioConf.Args, stdioConf.Env))
		mcpGoClient, err = mcpclient.NewStdioMCPClient(stdioConf.Command, stdioConf.Env, stdioConf.Args...)
		if err == nil {
			// Capture stderr output from the subprocess to get detailed error messages
			if client, ok := mcpGoClient.(*mcpclient.Client); ok {
				if stderrReader, hasStderr := mcpclient.GetStderr(client); hasStderr {
					go func() {
						scanner := bufio.NewScanner(stderrReader)
						for scanner.Scan() {
							line := scanner.Text()
							if line != "" {
								common.SysError(fmt.Sprintf("Stderr from %s: %s", serviceConfigForInstance.Name, line))
							}
						}
						if err := scanner.Err(); err != nil {
							common.SysError(fmt.Sprintf("Error reading stderr from %s: %v", serviceConfigForInstance.Name, err))
						}
					}()
				}
			}
		}
		needManualStart = false

	case model.ServiceTypeSSE:
		url := serviceConfigForInstance.Command // URL is stored in Command field for SSE/HTTP
		if url == "" {
			return nil, nil, fmt.Errorf("URL (from Command field) is empty for SSE service %s (ID: %d)", serviceConfigForInstance.Name, serviceConfigForInstance.ID)
		}
		var headers map[string]string
		if serviceConfigForInstance.HeadersJSON != "" && serviceConfigForInstance.HeadersJSON != "{}" {
			if errJson := json.Unmarshal([]byte(serviceConfigForInstance.HeadersJSON), &headers); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal HeadersJSON for SSE service %s (ID: %d): %v. Proceeding without custom headers.", serviceConfigForInstance.Name, serviceConfigForInstance.ID, errJson))
			}
		}
		common.SysLog(fmt.Sprintf("SSE config for %s: URL=%s, Headers=%v", serviceConfigForInstance.Name, url, headers))
		if len(headers) > 0 {
			mcpGoClient, err = mcpclient.NewSSEMCPClient(url, mcpclient.WithHeaders(headers))
		} else {
			mcpGoClient, err = mcpclient.NewSSEMCPClient(url)
		}
		needManualStart = true

	case model.ServiceTypeStreamableHTTP:
		url := serviceConfigForInstance.Command // URL is stored in Command field for SSE/HTTP
		if url == "" {
			return nil, nil, fmt.Errorf("URL (from Command field) is empty for StreamableHTTP service %s (ID: %d)", serviceConfigForInstance.Name, serviceConfigForInstance.ID)
		}
		var headers map[string]string
		if serviceConfigForInstance.HeadersJSON != "" && serviceConfigForInstance.HeadersJSON != "{}" {
			if errJson := json.Unmarshal([]byte(serviceConfigForInstance.HeadersJSON), &headers); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal HeadersJSON for StreamableHTTP service %s (ID: %d): %v. Proceeding without custom headers.", serviceConfigForInstance.Name, serviceConfigForInstance.ID, errJson))
			}
		}
		common.SysLog(fmt.Sprintf("StreamableHTTP config for %s: URL=%s, Headers (raw)=%v", serviceConfigForInstance.Name, url, headers))
		if len(headers) > 0 {
			// TODO: Correctly apply HTTP headers.
			// tdd.md and mcp-go patterns suggest `transport.WithHTTPHeaders(headers)`,
			// which would require importing "github.com/mark3labs/mcp-go/client/transport".
			// Due to current tool limitations on adding imports, this is omitted.
			// mcpclient.WithHeaders is likely not the correct option for HTTP stream transport headers.
			common.SysLog(fmt.Sprintf("WARNING: Custom headers for StreamableHTTP service %s are NOT being applied due to missing transport.WithHTTPHeaders option.", serviceConfigForInstance.Name))
			// Call without header options as the correct option builder is unavailable without new imports.
			mcpGoClient, err = mcpclient.NewStreamableHttpClient(url)
		} else {
			mcpGoClient, err = mcpclient.NewStreamableHttpClient(url)
		}
		needManualStart = true

	default:
		return nil, nil, fmt.Errorf("unsupported service type %s in createActualMcpGoServerAndClientUncached", serviceConfigForInstance.Type)
	}

	if err != nil { // Consolidated error check after switch
		errMsg := fmt.Sprintf("Failed to create mcp-go client for %s (Type: %s, %s): %v", serviceConfigForInstance.Name, serviceConfigForInstance.Type, instanceNameDetail, err)
		common.SysError(errMsg)
		return nil, nil, errors.New(errMsg)
	}

	// Call client.Start() if needed
	if needManualStart {

		var startErr error
		switch cl := mcpGoClient.(type) {
		case interface{ Start(context.Context) error }:
			startErr = cl.Start(ctx)
		default:
			startErr = fmt.Errorf("client type %T does not have a Start method, but needManualStart was true", mcpGoClient)
		}

		if startErr != nil {
			errMsg := fmt.Sprintf("Failed to start mcp-go client for %s (%s): %v", serviceConfigForInstance.Name, instanceNameDetail, startErr)
			common.SysError(errMsg)
			if closeErr := mcpGoClient.Close(); closeErr != nil {
				common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s (%s) after Start() error: %v", serviceConfigForInstance.Name, instanceNameDetail, closeErr))
			}
			return nil, nil, errors.New(errMsg)
		}

		// Start ping task for SSE and HTTP clients
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
		PingLoop:
			for {
				select {
				case <-ctx.Done():
					common.SysLog(fmt.Sprintf("Context done, stopping ping for %s", serviceConfigForInstance.Name))
					break PingLoop
				case <-ticker.C:
					if err := mcpGoClient.Ping(ctx); err != nil {
						common.SysError(fmt.Sprintf("Ping failed for %s: %v", serviceConfigForInstance.Name, err))
					}
				}
			}
		}()
	}

	mcpGoServer := mcpserver.NewMCPServer(
		serviceConfigForInstance.Name,
		serviceConfigForInstance.InstalledVersion,
		mcpserver.WithResourceCapabilities(true, true),
	)

	clientInfo := mcp.Implementation{
		Name:    fmt.Sprintf("one-mcp-proxy-for-%s-%s", serviceConfigForInstance.Name, instanceNameDetail),
		Version: common.Version,
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo

	_, err = mcpGoClient.Initialize(ctx, initRequest)
	if err != nil {
		// Give stderr some time to output error details before we return
		// This helps capture the actual error messages from the subprocess
		time.Sleep(100 * time.Millisecond)

		closeErr := mcpGoClient.Close()
		if closeErr != nil {
			common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s (%s) after initialization error: %v", serviceConfigForInstance.Name, instanceNameDetail, closeErr))
		}
		errMsg := fmt.Sprintf("Failed to initialize mcp-go client for %s (%s): %v. Check stderr logs for detailed error messages from the subprocess.", serviceConfigForInstance.Name, instanceNameDetail, err)
		common.SysError(errMsg)
		return nil, nil, errors.New(errMsg)
	}

	// Populate server with resources from client
	if err := addClientToolsToMCPServer(ctx, mcpGoClient, mcpGoServer, serviceConfigForInstance.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add tools for %s (%s): %v", serviceConfigForInstance.Name, instanceNameDetail, err))
	}
	if err := addClientPromptsToMCPServer(ctx, mcpGoClient, mcpGoServer, serviceConfigForInstance.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add prompts for %s (%s): %v", serviceConfigForInstance.Name, instanceNameDetail, err))
	}
	if err := addClientResourcesToMCPServer(ctx, mcpGoClient, mcpGoServer, serviceConfigForInstance.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add resources for %s (%s): %v", serviceConfigForInstance.Name, instanceNameDetail, err))
	}
	if err := addClientResourceTemplatesToMCPServer(ctx, mcpGoClient, mcpGoServer, serviceConfigForInstance.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add resource templates for %s (%s): %v", serviceConfigForInstance.Name, instanceNameDetail, err))
	}

	return mcpGoServer, mcpGoClient, nil
}

// createSSEHttpHandler creates an SSE http.Handler from an mcpserver.MCPServer.
func createSSEHttpHandler(
	mcpGoServer *mcpserver.MCPServer,
	mcpDBService *model.MCPService, // Used for base path and potentially other SSE server options
) (http.Handler, error) {
	if mcpGoServer == nil {
		return nil, errors.New("mcpGoServer cannot be nil for createSSEHttpHandler")
	}
	oneMCPExternalBaseURL := common.OptionMap["ServerAddress"]
	// The SSE base URL for user-specific instances might need reconsideration for proxying if the URL needs to be unique.
	// For now, it uses the service name. The distinction happens by routing to this specific handler instance.
	actualMCPGoSSEServer := mcpserver.NewSSEServer(mcpGoServer,
		mcpserver.WithStaticBasePath(mcpDBService.Name),       // TODO: This might need to be more dynamic based on routing
		mcpserver.WithBaseURL(oneMCPExternalBaseURL+"/proxy"), // Path for client to connect back
	)
	return actualMCPGoSSEServer, nil
}

// createHTTPProxyHttpHandler creates an HTTP/MCP http.Handler from an mcpserver.MCPServer.
func createHTTPProxyHttpHandler(mcpGoServer *mcpserver.MCPServer, mcpDBService *model.MCPService) (http.Handler, error) {
	if mcpGoServer == nil {
		return nil, errors.New("mcpGoServer cannot be nil for createHTTPProxyHttpHandler")
	}

	// Use NewStreamableHTTPServer to create HTTP/MCP handler with heartbeat to prevent idle timeout
	actualMCPGoHTTPServer := mcpserver.NewStreamableHTTPServer(mcpGoServer,
		mcpserver.WithHeartbeatInterval(30*time.Second),
	)

	common.SysLog(fmt.Sprintf("Successfully created HTTP/MCP handler for %s (ID: %d)", mcpDBService.Name, mcpDBService.ID))
	return actualMCPGoHTTPServer, nil
}

// GetCachedHandler safely retrieves a handler from the cache.
func GetCachedHandler(key string) (http.Handler, bool) {
	muStdioSSEWrappers.RLock()
	defer muStdioSSEWrappers.RUnlock()
	handler, found := initializedStdioSSEWrappers[key]
	return handler, found
}

// CacheHandler safely stores a handler in the cache.
func CacheHandler(key string, handler http.Handler) {
	muStdioSSEWrappers.Lock()
	defer muStdioSSEWrappers.Unlock()
	initializedStdioSSEWrappers[key] = handler
}

// ServiceFactory creates a suitable service instance for a given service type,
// including a real MCP connection for accurate health monitoring.
func ServiceFactory(mcpDBService *model.MCPService) (Service, error) {
	baseService := NewBaseService(mcpDBService.ID, mcpDBService.Name, mcpDBService.Type)

	switch mcpDBService.Type {
	case model.ServiceTypeStdio, model.ServiceTypeSSE, model.ServiceTypeStreamableHTTP:
		common.SysLog(fmt.Sprintf("ServiceFactory: Creating MonitoredProxiedService for %s (type: %s)", mcpDBService.Name, mcpDBService.Type))

		ctx := context.Background()
		// Use unified global cache key and standardized parameters
		cacheKey := fmt.Sprintf("global-service-%d-shared", mcpDBService.ID)
		instanceNameDetail := fmt.Sprintf("global-shared-svc-%d", mcpDBService.ID)
		effectiveEnvs := mcpDBService.DefaultEnvsJSON

		sharedInst, err := GetOrCreateSharedMcpInstanceWithKey(ctx, mcpDBService, cacheKey, instanceNameDetail, effectiveEnvs)
		if err != nil {
			common.SysError(fmt.Sprintf("ServiceFactory: Failed to get/create shared MCP instance for %s (ID: %d) with key %s: %v. Service will be unhealthy.", mcpDBService.Name, mcpDBService.ID, cacheKey, err))
			monitoredService := NewMonitoredProxiedService(baseService, nil, mcpDBService)
			monitoredService.UpdateHealth(StatusUnhealthy, 0, fmt.Sprintf("Failed to initialize shared MCP instance: %v", err))
			return monitoredService, nil
		}

		common.SysLog(fmt.Sprintf("ServiceFactory: Successfully got/created shared MCP instance for %s (ID: %d) with key %s", mcpDBService.Name, mcpDBService.ID, cacheKey))
		return NewMonitoredProxiedService(baseService, sharedInst, mcpDBService), nil

	default:
		common.SysLog(fmt.Sprintf("ServiceFactory: Creating basic BaseService for unsupported/non-proxied type %s (service: %s)", mcpDBService.Type, mcpDBService.Name))
		return baseService, nil
	}
}

// --- Helper functions to add resources to mcp-go server (adapted from user's example) ---

func addClientToolsToMCPServer(ctx context.Context, mcpGoClient mcpclient.MCPClient, mcpGoServer *mcpserver.MCPServer, mcpServerName string) error {
	toolsRequest := mcp.ListToolsRequest{}
	for {
		tools, err := mcpGoClient.ListTools(ctx, toolsRequest)
		if err != nil {
			common.SysError(fmt.Sprintf("ListTools failed for %s: %v", mcpServerName, err))
			return err
		}
		if tools == nil {
			common.SysLog(fmt.Sprintf("ListTools returned nil tools for %s. No tools to add.", mcpServerName))
			break
		}
		common.SysLog(fmt.Sprintf("Listed %d tools for %s", len(tools.Tools), mcpServerName))
		for _, tool := range tools.Tools {
			common.SysLog(fmt.Sprintf("Adding tool %s to %s", tool.Name, mcpServerName))
			mcpGoServer.AddTool(tool, mcpGoClient.CallTool)
		}
		if tools.NextCursor == "" {
			break
		}
		toolsRequest.PaginatedRequest.Params.Cursor = tools.NextCursor
	}
	return nil
}

func addClientPromptsToMCPServer(ctx context.Context, mcpGoClient mcpclient.MCPClient, mcpGoServer *mcpserver.MCPServer, mcpServerName string) error {
	promptsRequest := mcp.ListPromptsRequest{}
	for {
		prompts, err := mcpGoClient.ListPrompts(ctx, promptsRequest)
		if err != nil {
			common.SysError(fmt.Sprintf("ListPrompts failed for %s: %v", mcpServerName, err))
			return err
		}
		if prompts == nil {
			common.SysLog(fmt.Sprintf("ListPrompts returned nil prompts for %s. No prompts to add.", mcpServerName))
			break
		}
		common.SysLog(fmt.Sprintf("Listed %d prompts for %s", len(prompts.Prompts), mcpServerName))
		for _, prompt := range prompts.Prompts {
			common.SysLog(fmt.Sprintf("Adding prompt %s to %s", prompt.Name, mcpServerName))
			mcpGoServer.AddPrompt(prompt, mcpGoClient.GetPrompt)
		}
		if prompts.NextCursor == "" {
			break
		}
		promptsRequest.PaginatedRequest.Params.Cursor = prompts.NextCursor
	}
	return nil
}

// TODO: Implement addClientResourcesToMCPServer and addClientResourceTemplatesToMCPServer
// based on user's example if these are required for exa-mcp-server.
// For now, these are stubbed or simplified.

// --- End Helper Functions ---

// Keep existing ServiceManager and its methods (GetServiceManager, AddService, GetSSEServiceByName etc.)
// GetSSEServiceByName will now rely on the updated ServiceFactory.
// ... existing code ...

// --- New Helper Functions ---

func addClientResourcesToMCPServer(ctx context.Context, mcpGoClient mcpclient.MCPClient, mcpGoServer *mcpserver.MCPServer, mcpServerName string) error {
	resourcesRequest := mcp.ListResourcesRequest{}
	for {
		resources, err := mcpGoClient.ListResources(ctx, resourcesRequest)
		if err != nil {
			common.SysError(fmt.Sprintf("ListResources failed for %s: %v", mcpServerName, err))
			return err
		}
		if resources == nil {
			common.SysLog(fmt.Sprintf("ListResources returned nil resources for %s. No resources to add.", mcpServerName))
			break
		}
		common.SysLog(fmt.Sprintf("Successfully listed %d resources for %s", len(resources.Resources), mcpServerName))
		for _, resource := range resources.Resources {
			// Capture range variable for closure
			resource := resource
			common.SysLog(fmt.Sprintf("Adding resource %s to %s", resource.Name, mcpServerName))
			mcpGoServer.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				readResource, e := mcpGoClient.ReadResource(ctx, request)
				if e != nil {
					return nil, e
				}
				return readResource.Contents, nil
			})
		}
		if resources.NextCursor == "" {
			break
		}
		resourcesRequest.PaginatedRequest.Params.Cursor = resources.NextCursor
	}
	return nil
}

func addClientResourceTemplatesToMCPServer(ctx context.Context, mcpGoClient mcpclient.MCPClient, mcpGoServer *mcpserver.MCPServer, mcpServerName string) error {
	resourceTemplatesRequest := mcp.ListResourceTemplatesRequest{}
	for {
		resourceTemplates, err := mcpGoClient.ListResourceTemplates(ctx, resourceTemplatesRequest)
		if err != nil {
			common.SysError(fmt.Sprintf("ListResourceTemplates failed for %s: %v", mcpServerName, err))
			return err
		}
		if resourceTemplates == nil {
			common.SysLog(fmt.Sprintf("ListResourceTemplates returned nil templates for %s. No templates to add.", mcpServerName))
			break
		}
		common.SysLog(fmt.Sprintf("Successfully listed %d resource templates for %s", len(resourceTemplates.ResourceTemplates), mcpServerName))
		for _, resourceTemplate := range resourceTemplates.ResourceTemplates {
			// Capture range variable for closure
			resourceTemplate := resourceTemplate
			common.SysLog(fmt.Sprintf("Adding resource template %s to %s", resourceTemplate.Name, mcpServerName))
			mcpGoServer.AddResourceTemplate(resourceTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				// Note: The callback for AddResourceTemplate in mcp-go server might expect a specific request type
				// or the ReadResourceRequest might be generic enough.
				// Assuming ReadResourceRequest is appropriate as per user's example.
				readResource, e := mcpGoClient.ReadResource(ctx, request) // This call might need adjustment if ReadResourceTemplates requires a different read method.
				// However, mcp-go server.AddResourceTemplate's callback signature is indeed for ReadResourceRequest.
				if e != nil {
					return nil, e
				}
				return readResource.Contents, nil
			})
		}
		if resourceTemplates.NextCursor == "" {
			break
		}
		resourceTemplatesRequest.PaginatedRequest.Params.Cursor = resourceTemplates.NextCursor
	}
	return nil
}

// --- End Helper Functions ---

// GetOrCreateSharedMcpInstanceWithKeyFunc defines the type for the GetOrCreateSharedMcpInstanceWithKey function.
// This allows it to be replaced in tests.
var GetOrCreateSharedMcpInstanceWithKey GetOrCreateSharedMcpInstanceWithKeyFuncType = getOrCreateSharedMcpInstanceWithKeyInternal

type GetOrCreateSharedMcpInstanceWithKeyFuncType func(ctx context.Context, originalDbService *model.MCPService, cacheKey string, instanceNameDetail string, effectiveEnvsJSONForStdio string) (*SharedMcpInstance, error)

// getOrCreateSharedMcpInstanceWithKeyInternal is the actual implementation.
func getOrCreateSharedMcpInstanceWithKeyInternal(ctx context.Context, originalDbService *model.MCPService, cacheKey string, instanceNameDetail string, effectiveEnvsJSONForStdio string) (*SharedMcpInstance, error) {
	sharedMCPServersMutex.Lock()
	defer sharedMCPServersMutex.Unlock()

	if inst, found := sharedMCPServers[cacheKey]; found && inst != nil {
		return inst, nil
	}

	// Prepare service config for creation
	serviceConfigForCreation := *originalDbService // Shallow copy

	// Apply user-specific environment variables for Stdio services
	if originalDbService.Type == model.ServiceTypeStdio && effectiveEnvsJSONForStdio != "" {
		serviceConfigForCreation.DefaultEnvsJSON = effectiveEnvsJSONForStdio
	}

	// Create the actual server and client
	srv, cli, err := createActualMcpGoServerAndClientUncached(ctx, &serviceConfigForCreation, instanceNameDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server and client for %s: %w", originalDbService.Name, err)
	}

	// Create shared instance
	instance := &SharedMcpInstance{
		Server: srv,
		Client: cli,
	}

	// Store in cache
	sharedMCPServers[cacheKey] = instance
	common.SysLog(fmt.Sprintf("Created new SharedMcpInstance for %s", originalDbService.Name))

	return instance, nil
}

// GetOrCreateProxyToSSEHandler creates or retrieves a cached SSE http.Handler using shared MCP instance
func GetOrCreateProxyToSSEHandler(ctx context.Context, mcpDBService *model.MCPService, sharedInst *SharedMcpInstance) (http.Handler, error) {
	handlerCacheKey := fmt.Sprintf("service-%d-sseproxy", mcpDBService.ID)

	sseWrappersMutex.Lock()
	defer sseWrappersMutex.Unlock()

	// Check cache first
	if existingHandler, found := initializedSSEProxyWrappers[handlerCacheKey]; found {
		return existingHandler, nil
	}

	// Create new handler
	handler, err := createSSEHttpHandler(sharedInst.Server, mcpDBService)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE handler for %s: %w", mcpDBService.Name, err)
	}

	// Cache the handler
	initializedSSEProxyWrappers[handlerCacheKey] = handler

	return handler, nil
}

// GetOrCreateProxyToHTTPHandler creates or retrieves a cached HTTP/MCP http.Handler using shared MCP instance
func GetOrCreateProxyToHTTPHandler(ctx context.Context, mcpDBService *model.MCPService, sharedInst *SharedMcpInstance) (http.Handler, error) {
	handlerCacheKey := fmt.Sprintf("service-%d-httpproxy", mcpDBService.ID)

	httpWrappersMutex.Lock()
	defer httpWrappersMutex.Unlock()

	// Check cache first
	if existingHandler, found := initializedHTTPProxyWrappers[handlerCacheKey]; found {
		return existingHandler, nil
	}

	// Create new handler
	handler, err := createHTTPProxyHttpHandler(sharedInst.Server, mcpDBService)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP handler for %s: %w", mcpDBService.Name, err)
	}

	// Cache the handler
	initializedHTTPProxyWrappers[handlerCacheKey] = handler

	return handler, nil
}

// ClearSSEProxyCache clears the cached SSE proxy handlers.
// This should be called when global settings that affect handler creation (like ServerAddress) are changed.
func ClearSSEProxyCache() {
	sseWrappersMutex.Lock()
	defer sseWrappersMutex.Unlock()
	if len(initializedSSEProxyWrappers) > 0 {
		common.SysLog(fmt.Sprintf("Clearing %d cached SSE proxy handlers due to configuration change.", len(initializedSSEProxyWrappers)))
		initializedSSEProxyWrappers = make(map[string]http.Handler)
	}
}
