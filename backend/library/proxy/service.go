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

	// TODO: Add caches for other types if needed, e.g., direct SSE proxies
)

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

func getOrCreateStdioToSSEHandler(mcpDBService *model.MCPService) (http.Handler, error) {
	globalHandlerKey := fmt.Sprintf("global-service-%d", mcpDBService.ID)

	common.SysLog(fmt.Sprintf("Attempting to get/create stdioToSSEHandler for %s (ID: %d) with key: %s", mcpDBService.Name, mcpDBService.ID, globalHandlerKey))

	muStdioSSEWrappers.RLock()
	existingHandler, found := initializedStdioSSEWrappers[globalHandlerKey]
	muStdioSSEWrappers.RUnlock()
	if found {
		common.SysLog(fmt.Sprintf("Reusing existing stdioToSSEHandler for %s (Key: %s)", mcpDBService.Name, globalHandlerKey))
		return existingHandler, nil
	}

	// If not found, proceed to create (Lock will be acquired before writing)
	common.SysLog(fmt.Sprintf("No existing handler found for key %s. Creating new handler for %s.", globalHandlerKey, mcpDBService.Name))

	// 1. Get StdioConfig from DefaultAdminConfigValues
	var stdioConf model.StdioConfig
	if mcpDBService.DefaultAdminConfigValues == "" {
		return nil, fmt.Errorf("StdioConfig (DefaultAdminConfigValues) is empty for service %s (ID: %d)", mcpDBService.Name, mcpDBService.ID)
	}
	err := json.Unmarshal([]byte(mcpDBService.DefaultAdminConfigValues), &stdioConf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal StdioConfig for service %s (ID: %d): %w", mcpDBService.Name, mcpDBService.ID, err)
	}

	// 2. Load and append default environment variables from DefaultEnvsJSON
	if mcpDBService.DefaultEnvsJSON != "" && mcpDBService.DefaultEnvsJSON != "{}" {
		var defaultEnvs map[string]string
		err = json.Unmarshal([]byte(mcpDBService.DefaultEnvsJSON), &defaultEnvs)
		if err != nil {
			common.SysError(fmt.Sprintf("Failed to unmarshal DefaultEnvsJSON for %s (ID: %d): %v. Proceeding without them.", mcpDBService.Name, mcpDBService.ID, err))
		} else {
			common.SysLog(fmt.Sprintf("Loading DefaultEnvsJSON for %s: %v", mcpDBService.Name, defaultEnvs))
			// Ensure stdioConf.Env is initialized
			if stdioConf.Env == nil {
				stdioConf.Env = []string{}
			}
			for key, value := range defaultEnvs {
				stdioConf.Env = append(stdioConf.Env, fmt.Sprintf("%s=%s", key, value))
			}
			common.SysLog(fmt.Sprintf("Final Env for %s after DefaultEnvsJSON: %v", mcpDBService.Name, stdioConf.Env))
		}
	}

	common.SysLog(fmt.Sprintf("Stdio config for %s: Command=%s, Args=%v, Env=%v", mcpDBService.Name, stdioConf.Command, stdioConf.Args, stdioConf.Env))

	// TODO: This context might need to be the one passed from Start() or a long-lived one for the service
	ctx := context.Background() // Using a background context for now

	// 3. Create Stdio MCP Client
	mcpGoClient, err := mcpclient.NewStdioMCPClient(stdioConf.Command, stdioConf.Env, stdioConf.Args...)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create mcp-go StdioClient for %s: %v", mcpDBService.Name, err)
		common.SysError(errMsg)
		return nil, errors.New(errMsg)
	}
	// TODO: Manage mcpGoClient lifecycle (e.g., Close() on shutdown or when service is disabled/deleted)

	// 4. Initialize Client & Create MCP Server
	mcpGoServer := mcpserver.NewMCPServer(
		mcpDBService.Name,
		mcpDBService.InstalledVersion,
		mcpserver.WithResourceCapabilities(true, true),
	)

	clientInfo := mcp.Implementation{
		Name:    fmt.Sprintf("one-mcp-proxy-for-%s", mcpDBService.Name),
		Version: common.Version,
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo

	common.SysLog(fmt.Sprintf("Initializing mcp-go client for %s...", mcpDBService.Name))

	// Introduce a timeout for Initialize
	initTimeout := 60 * time.Second // Default timeout of 60 seconds
	// TODO: Consider making this timeout configurable, e.g., via MCPService.DefaultAdminConfigValues

	initCtx, cancelInit := context.WithTimeout(ctx, initTimeout)
	defer cancelInit()

	_, err = mcpGoClient.Initialize(initCtx, initRequest)
	if err != nil {
		// Ensure client is closed on initialization failure
		closeErr := mcpGoClient.Close()
		if closeErr != nil {
			common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s after initialization error: %v", mcpDBService.Name, closeErr))
		}
		errMsg := fmt.Sprintf("Failed to initialize mcp-go client for %s (timeout or error): %v", mcpDBService.Name, err)
		common.SysError(errMsg)
		return nil, errors.New(errMsg) // Propagate a new error to avoid extremely long error messages if err contains much detail
	}
	common.SysLog(fmt.Sprintf("Successfully initialized mcp-go client for %s. Adding resources...", mcpDBService.Name))

	if err := addClientToolsToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		mcpGoClient.Close() // Ensure cleanup
		common.SysError(fmt.Sprintf("Failed to add tools for %s: %v", mcpDBService.Name, err))
	}
	if err := addClientPromptsToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		mcpGoClient.Close() // Ensure cleanup
		common.SysError(fmt.Sprintf("Failed to add prompts for %s: %v", mcpDBService.Name, err))
	}
	common.SysLog(fmt.Sprintf("Finished adding resources for %s to mcp-go server.", mcpDBService.Name))

	// 5. Create SSE Server (Wrapper for mcp-go server)
	oneMCPExternalBaseURL := common.OptionMap["ServerAddress"]

	actualMCPGoSSEServer := mcpserver.NewSSEServer(mcpGoServer,
		mcpserver.WithStaticBasePath(""),
		mcpserver.WithBaseURL(oneMCPExternalBaseURL+"/api/sse/"+mcpDBService.Name),
	)

	muStdioSSEWrappers.Lock() // Lock for writing to the map
	// Double check if another goroutine created it while this one was working
	if interimHandler, exists := initializedStdioSSEWrappers[globalHandlerKey]; exists {
		muStdioSSEWrappers.Unlock()
		common.SysLog(fmt.Sprintf("Handler for key %s was created by another goroutine. Closing new one and returning existing.", globalHandlerKey))
		mcpGoClient.Close() // Clean up the newly created, now redundant, client
		return interimHandler, nil
	}
	initializedStdioSSEWrappers[globalHandlerKey] = actualMCPGoSSEServer
	muStdioSSEWrappers.Unlock()

	common.SysLog(fmt.Sprintf("Successfully created and cached Stdio-to-SSE handler for %s (Key: %s)", mcpDBService.Name, globalHandlerKey))
	return actualMCPGoSSEServer, nil
}

// defaultNewStdioSSEHandlerUncached creates a new http.Handler for an Stdio MCP service using the provided configuration.
// This function does NOT handle caching; caching is the responsibility of the caller.
// It takes an effectiveStdioConfig which might be user-specific or a default.
// This is the actual implementation, not typically called directly from outside the package after initialization.
func defaultNewStdioSSEHandlerUncached(ctx context.Context, mcpDBService *model.MCPService, effectiveStdioConfig model.StdioConfig) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("defaultNewStdioSSEHandlerUncached: Creating new handler for %s (ID: %d) with effective StdioConfig: Command=%s, Args=%v, Env=%v",
		mcpDBService.Name, mcpDBService.ID, effectiveStdioConfig.Command, effectiveStdioConfig.Args, effectiveStdioConfig.Env))

	if effectiveStdioConfig.Command == "" {
		return nil, fmt.Errorf("effectiveStdioConfig for service %s (ID: %d) has an empty command", mcpDBService.Name, mcpDBService.ID)
	}

	// 1. Create Stdio MCP Client using effectiveStdioConfig
	// Note: The original ctx passed in is used here. If specific timeouts are needed for sub-operations,
	// they should be derived from this ctx.
	mcpGoClient, err := mcpclient.NewStdioMCPClient(effectiveStdioConfig.Command, effectiveStdioConfig.Env, effectiveStdioConfig.Args...)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create mcp-go StdioClient for %s with effective config: %v", mcpDBService.Name, err)
		common.SysError(errMsg)
		return nil, errors.New(errMsg) // Return a simpler error to avoid exposing too much detail
	}

	// 2. Initialize Client & Create MCP Server
	mcpGoServer := mcpserver.NewMCPServer(
		mcpDBService.Name,
		mcpDBService.InstalledVersion, // Assuming InstalledVersion is relevant; it's from mcpDBService model
		mcpserver.WithResourceCapabilities(true, true),
	)

	clientInfo := mcp.Implementation{
		Name:    fmt.Sprintf("one-mcp-proxy-for-%s-instance", mcpDBService.Name), // Simplified name
		Version: common.Version,
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo

	common.SysLog(fmt.Sprintf("Initializing mcp-go client for %s (instance for effectiveConfig)...", mcpDBService.Name))

	initTimeout := 60 * time.Second                                          // Default timeout of 60 seconds (consistent with getOrCreateStdioToSSEHandler)
	initializeCtx, cancelInitialize := context.WithTimeout(ctx, initTimeout) // Derive from passed-in ctx
	defer cancelInitialize()

	_, err = mcpGoClient.Initialize(initializeCtx, initRequest)
	if err != nil {
		closeErr := mcpGoClient.Close()
		if closeErr != nil {
			common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s after initialization error (effectiveConfig instance): %v", mcpDBService.Name, closeErr))
		}
		errMsg := fmt.Sprintf("Failed to initialize mcp-go client for %s (effectiveConfig instance, timeout or error): %v", mcpDBService.Name, err)
		common.SysError(errMsg)
		return nil, errors.New(errMsg)
	}
	common.SysLog(fmt.Sprintf("Successfully initialized mcp-go client for %s (effectiveConfig instance). Adding resources...", mcpDBService.Name))

	// Pass the original ctx for resource loading, or a derived one if specific timeouts are needed here too.
	if err := addClientToolsToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		mcpGoClient.Close()
		common.SysError(fmt.Sprintf("Failed to add tools for %s (effectiveConfig instance): %v", mcpDBService.Name, err))
		// Depending on policy, might return error or just log and continue
	}
	if err := addClientPromptsToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		mcpGoClient.Close()
		common.SysError(fmt.Sprintf("Failed to add prompts for %s (effectiveConfig instance): %v", mcpDBService.Name, err))
	}
	common.SysLog(fmt.Sprintf("Finished adding resources for %s (effectiveConfig instance) to mcp-go server.", mcpDBService.Name))

	// 3. Create SSE Server (Wrapper for mcp-go server)
	oneMCPExternalBaseURL := common.OptionMap["ServerAddress"]
	// The SSE base URL for user-specific instances might need reconsideration.
	// For now, it uses the service name, which is fine for proxying, but the URL itself won't distinguish user instances.
	// The distinction happens by routing to this specific handler instance.
	actualMCPGoSSEServer := mcpserver.NewSSEServer(mcpGoServer,
		mcpserver.WithStaticBasePath(""),
		mcpserver.WithBaseURL(oneMCPExternalBaseURL+"/api/sse/"+mcpDBService.Name),
	)

	common.SysLog(fmt.Sprintf("Successfully created new (uncached) Stdio-to-SSE handler for %s (effectiveConfig instance)", mcpDBService.Name))
	return actualMCPGoSSEServer, nil
}

// NewStdioSSEHandlerUncached is an exported variable that points to the function for creating a new http.Handler
// for an Stdio MCP service using the provided configuration. This can be replaced in tests to mock behavior.
var NewStdioSSEHandlerUncached = defaultNewStdioSSEHandlerUncached

// ServiceFactory 用于创建适合特定类型的服务实例
func ServiceFactory(mcpDBService *model.MCPService) (Service, error) {
	var effectiveServiceType model.ServiceType
	var httpHandlerForProxy http.Handler
	var err error

	switch mcpDBService.Type {
	case model.ServiceTypeStdio:
		common.SysLog(fmt.Sprintf("ServiceFactory: %s is Stdio type. Attempting to wrap as SSE.", mcpDBService.Name))
		httpHandlerForProxy, err = getOrCreateStdioToSSEHandler(mcpDBService)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio-to-SSE handler for %s: %w", mcpDBService.Name, err)
		}
		effectiveServiceType = model.ServiceTypeSSE

	case model.ServiceTypeSSE:
		common.SysLog(fmt.Sprintf("ServiceFactory: %s is native SSE type. TODO: Implement direct SSE proxy.", mcpDBService.Name))
		return nil, fmt.Errorf("native SSE service proxying not yet implemented for %s", mcpDBService.Name)

	case model.ServiceTypeStreamableHTTP:
		common.SysLog(fmt.Sprintf("ServiceFactory: %s is StreamableHTTP. TODO: Implement.", mcpDBService.Name))
		return nil, fmt.Errorf("streamableHTTP service proxying not yet implemented for %s", mcpDBService.Name)

	default:
		common.SysError(fmt.Sprintf("ServiceFactory: Unsupported service type '%s' for service %s", mcpDBService.Type, mcpDBService.Name))
		return nil, errors.New("unsupported service type: " + string(mcpDBService.Type))
	}

	proxyBaseService := NewBaseService(mcpDBService.ID, mcpDBService.Name, effectiveServiceType)

	if effectiveServiceType == model.ServiceTypeSSE && httpHandlerForProxy != nil {
		return NewSSESvc(proxyBaseService, httpHandlerForProxy), nil
	}

	return nil, fmt.Errorf("could not create a suitable proxy service for %s (type %s, effective %s)", mcpDBService.Name, mcpDBService.Type, effectiveServiceType)
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
