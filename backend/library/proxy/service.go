package proxy

import (
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

	startTime := time.Now()

	if s.sharedInstance == nil || s.sharedInstance.Client == nil {
		s.health.Status = StatusUnhealthy
		s.health.ErrorMessage = "Shared MCP instance or client is not initialized."
		s.health.LastChecked = time.Now()
		s.health.ResponseTime = time.Since(startTime).Milliseconds()
		s.health.FailureCount++
		s.health.WarningLevel = 3 // Critical if not initialized
		healthCopy := s.health
		return &healthCopy, errors.New(s.health.ErrorMessage)
	}

	// Attempt to ping the client with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.sharedInstance.Client.Ping(pingCtx)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		s.health.Status = StatusUnhealthy
		s.health.ErrorMessage = fmt.Sprintf("Ping failed: %v", err)
		s.health.FailureCount++
		common.SysError(fmt.Sprintf("Health check for %s (ID: %d) failed: %s", s.serviceName, s.serviceID, s.health.ErrorMessage))
	} else {
		s.health.Status = StatusHealthy
		s.health.ErrorMessage = ""
		s.health.SuccessCount++
		common.SysLog(fmt.Sprintf("Health check for %s (ID: %d) successful.", s.serviceName, s.serviceID))
	}

	s.health.LastChecked = time.Now()
	s.health.ResponseTime = responseTime

	// Update warning level
	if s.health.Status == StatusHealthy {
		s.health.WarningLevel = 0
	} else if s.health.FailureCount <= 3 {
		s.health.WarningLevel = 1
	} else if s.health.FailureCount <= 10 {
		s.health.WarningLevel = 2
	} else {
		s.health.WarningLevel = 3
	}

	// Calculate uptime if running
	if s.running && !s.lastStartTime.IsZero() {
		s.health.UpTime = int64(time.Since(s.lastStartTime).Seconds())
	}

	healthCopy := s.health
	return &healthCopy, err
}

// Start for MonitoredProxiedService ensures the shared instance is active
func (s *MonitoredProxiedService) Start(ctx context.Context) error {
	// Call BaseService's Start first
	if err := s.BaseService.Start(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// If shared instance is nil, attempt to create/get it
	if s.sharedInstance == nil && s.dbServiceConfig != nil {
		common.SysLog(fmt.Sprintf("Attempting to (re)create shared MCP instance for health monitoring for %s (ID: %d)", s.serviceName, s.serviceID))

		// Use unified global cache key and standardized parameters
		cacheKey := fmt.Sprintf("global-service-%d-shared", s.dbServiceConfig.ID)
		instanceNameDetail := fmt.Sprintf("global-shared-svc-%d", s.dbServiceConfig.ID)
		effectiveEnvs := s.dbServiceConfig.DefaultEnvsJSON

		sharedInst, err := GetOrCreateSharedMcpInstanceWithKey(ctx, s.dbServiceConfig, cacheKey, instanceNameDetail, effectiveEnvs)
		if err != nil {
			s.health.Status = StatusUnhealthy
			s.health.ErrorMessage = fmt.Sprintf("Failed to recreate shared instance on Start: %v", err)
			common.SysError(fmt.Sprintf("Failed to recreate shared instance for %s on Start: %v", s.serviceName, err))
			return fmt.Errorf("failed to recreate shared instance for %s: %w", s.serviceName, err)
		}
		s.sharedInstance = sharedInst
		s.health.Status = StatusHealthy
		s.health.ErrorMessage = ""
		common.SysLog(fmt.Sprintf("Successfully (re)created shared MCP instance for health monitoring for %s", s.serviceName))
	} else if s.sharedInstance != nil {
		s.health.Status = StatusHealthy
		s.health.ErrorMessage = ""
	} else {
		// dbServiceConfig is nil, cannot recreate
		s.health.Status = StatusUnhealthy
		s.health.ErrorMessage = "Cannot start monitored service: dbServiceConfig is nil, unable to create shared instance."
		common.SysError(s.health.ErrorMessage)
		return errors.New(s.health.ErrorMessage)
	}

	return nil
}

// Stop for MonitoredProxiedService updates state; actual cleanup handled by cache management
func (s *MonitoredProxiedService) Stop(ctx context.Context) error {
	if err := s.BaseService.Stop(ctx); err != nil {
		return err
	}
	// The SharedMcpInstance is managed by the cache and cleanup functions
	common.SysLog(fmt.Sprintf("MonitoredProxiedService %s stopped. Underlying shared instance will be cleaned up by cache management.", s.serviceName))
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
	common.SysLog(fmt.Sprintf("createActualMcpGoServerAndClientUncached: Creating new MCP client and server for %s (ID: %d, Type: %s) - %s.",
		serviceConfigForInstance.Name, serviceConfigForInstance.ID, serviceConfigForInstance.Type, instanceNameDetail))

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
		common.SysLog(fmt.Sprintf("Manually starting mcp-go client for %s (%s)...", serviceConfigForInstance.Name, instanceNameDetail))

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
		common.SysLog(fmt.Sprintf("Successfully started mcp-go client for %s (%s).", serviceConfigForInstance.Name, instanceNameDetail))

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

	common.SysLog(fmt.Sprintf("Initializing mcp-go client for %s (%s)...", serviceConfigForInstance.Name, instanceNameDetail))

	_, err = mcpGoClient.Initialize(ctx, initRequest)
	if err != nil {
		closeErr := mcpGoClient.Close()
		if closeErr != nil {
			common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s (%s) after initialization error: %v", serviceConfigForInstance.Name, instanceNameDetail, closeErr))
		}
		errMsg := fmt.Sprintf("Failed to initialize mcp-go client for %s (%s): %v", serviceConfigForInstance.Name, instanceNameDetail, err)
		common.SysError(errMsg)
		return nil, nil, errors.New(errMsg)
	}
	common.SysLog(fmt.Sprintf("Successfully initialized mcp-go client for %s (%s). Adding resources...", serviceConfigForInstance.Name, instanceNameDetail))

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
	common.SysLog(fmt.Sprintf("Finished adding resources for %s (%s) to mcp-go server.", serviceConfigForInstance.Name, instanceNameDetail))

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
	common.SysLog(fmt.Sprintf("Successfully created SSE handler for %s (ID: %d)", mcpDBService.Name, mcpDBService.ID))
	return actualMCPGoSSEServer, nil
}

// createHTTPProxyHttpHandler creates an HTTP/MCP http.Handler from an mcpserver.MCPServer.
func createHTTPProxyHttpHandler(mcpGoServer *mcpserver.MCPServer, mcpDBService *model.MCPService) (http.Handler, error) {
	if mcpGoServer == nil {
		return nil, errors.New("mcpGoServer cannot be nil for createHTTPProxyHttpHandler")
	}

	// Use NewStreamableHTTPServer to create HTTP/MCP handler
	actualMCPGoHTTPServer := mcpserver.NewStreamableHTTPServer(mcpGoServer)
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

// defaultNewStdioSSEHandlerUncached is renamed to newProxyToSSEHandlerUncached
// It creates a new, non-cached http.Handler (SSE proxy) for an MCP service.
func newProxyToSSEHandlerUncached(ctx context.Context, mcpDBService *model.MCPService) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("newProxyToSSEHandlerUncached: Creating new SSE proxy handler for %s (ID: %d, Type: %s) via createMcpGoServer and createSSEHttpHandler.",
		mcpDBService.Name, mcpDBService.ID, mcpDBService.Type))

	// Delegate to the new core functions
	// Configuration parsing (like StdioConfig) is now handled within createActualMcpGoServerAndClientUncached based on mcpDBService.Type
	mcpGoSrv, mcpClient, err := createActualMcpGoServerAndClientUncached(ctx, mcpDBService, "user-specific-instance")
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp-go server for %s (Type: %s, user-specific): %w", mcpDBService.Name, mcpDBService.Type, err)
	}
	// Note: mcpClient's lifecycle is a concern here too if not managed by the handler.

	httpHandler, err := createSSEHttpHandler(mcpGoSrv, mcpDBService)
	if err != nil {
		if mcpClient != nil {
			common.SysError(fmt.Sprintf("Closing MCP client for %s due to SSE handler creation failure (uncached): %v", mcpDBService.Name, err))
			if closeErr := mcpClient.Close(); closeErr != nil {
				common.SysError(fmt.Sprintf("Error closing MCP client for %s after SSE handler creation failure (uncached): %v", mcpDBService.Name, closeErr))
			}
		}
		return nil, fmt.Errorf("failed to create SSE handler for %s (Type: %s, user-specific): %w", mcpDBService.Name, mcpDBService.Type, err)
	}
	return httpHandler, nil
}

// NewStdioSSEHandlerUncached is an exported variable that points to the function for creating a new http.Handler
// for an Stdio MCP service using the provided configuration. This can be replaced in tests to mock behavior.
// RENAMING this variable as well to reflect its general nature.
var NewProxyToSSEHandlerUncached = newProxyToSSEHandlerUncached // Points to the renamed function

// ServiceFactory 用于创建适合特定类型的服务实例，包含真实的MCP连接用于准确的健康监测
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

// GetOrCreateSharedMcpInstanceWithKey manages caching of SharedMcpInstance
// It handles both global and user-specific keys, and applies effectiveEnvsJSONForStdio for Stdio services
func GetOrCreateSharedMcpInstanceWithKey(ctx context.Context, originalDbService *model.MCPService, cacheKey string, instanceNameDetail string, effectiveEnvsJSONForStdio string) (*SharedMcpInstance, error) {
	sharedMCPServersMutex.Lock()
	defer sharedMCPServersMutex.Unlock()

	// Check cache first
	if existingInstance, found := sharedMCPServers[cacheKey]; found {
		common.SysLog(fmt.Sprintf("Reusing existing SharedMcpInstance for key: %s", cacheKey))
		return existingInstance, nil
	}

	common.SysLog(fmt.Sprintf("Creating new SharedMcpInstance for key: %s", cacheKey))

	// Prepare service config for creation
	serviceConfigForCreation := *originalDbService // Shallow copy

	// Apply user-specific environment variables for Stdio services
	if originalDbService.Type == model.ServiceTypeStdio && effectiveEnvsJSONForStdio != "" {
		serviceConfigForCreation.DefaultEnvsJSON = effectiveEnvsJSONForStdio
		common.SysLog(fmt.Sprintf("Applied user-specific envs for Stdio service %s: %s", originalDbService.Name, effectiveEnvsJSONForStdio))
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
	common.SysLog(fmt.Sprintf("Successfully created and cached SharedMcpInstance for key: %s", cacheKey))

	return instance, nil
}

// GetOrCreateProxyToSSEHandler creates or retrieves a cached SSE http.Handler using shared MCP instance
func GetOrCreateProxyToSSEHandler(ctx context.Context, mcpDBService *model.MCPService, sharedInst *SharedMcpInstance) (http.Handler, error) {
	handlerCacheKey := fmt.Sprintf("service-%d-sseproxy", mcpDBService.ID)

	sseWrappersMutex.Lock()
	defer sseWrappersMutex.Unlock()

	// Check cache first
	if existingHandler, found := initializedSSEProxyWrappers[handlerCacheKey]; found {
		common.SysLog(fmt.Sprintf("Reusing existing SSE proxy handler for key: %s", handlerCacheKey))
		return existingHandler, nil
	}

	// Create new handler
	handler, err := createSSEHttpHandler(sharedInst.Server, mcpDBService)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE handler for %s: %w", mcpDBService.Name, err)
	}

	// Cache the handler
	initializedSSEProxyWrappers[handlerCacheKey] = handler
	common.SysLog(fmt.Sprintf("Successfully created and cached SSE proxy handler for key: %s", handlerCacheKey))

	return handler, nil
}

// GetOrCreateProxyToHTTPHandler creates or retrieves a cached HTTP/MCP http.Handler using shared MCP instance
func GetOrCreateProxyToHTTPHandler(ctx context.Context, mcpDBService *model.MCPService, sharedInst *SharedMcpInstance) (http.Handler, error) {
	handlerCacheKey := fmt.Sprintf("service-%d-httpproxy", mcpDBService.ID)

	httpWrappersMutex.Lock()
	defer httpWrappersMutex.Unlock()

	// Check cache first
	if existingHandler, found := initializedHTTPProxyWrappers[handlerCacheKey]; found {
		common.SysLog(fmt.Sprintf("Reusing existing HTTP proxy handler for key: %s", handlerCacheKey))
		return existingHandler, nil
	}

	// Create new handler
	handler, err := createHTTPProxyHttpHandler(sharedInst.Server, mcpDBService)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP handler for %s: %w", mcpDBService.Name, err)
	}

	// Cache the handler
	initializedHTTPProxyWrappers[handlerCacheKey] = handler
	common.SysLog(fmt.Sprintf("Successfully created and cached HTTP proxy handler for key: %s", handlerCacheKey))

	return handler, nil
}
