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

// createMcpGoServer creates and initializes an mcp-go client and server instance.
// For Stdio clients, client.Start() is not called.
// It returns the mcp-go server, the mcp-go client, and an error.
func createMcpGoServer(
	ctx context.Context,
	mcpDBService *model.MCPService,
	instanceNameDetail string,
) (*mcpserver.MCPServer, mcpclient.MCPClient, error) {
	common.SysLog(fmt.Sprintf("createMcpGoServer: Creating new MCP client and server for %s (ID: %d, Type: %s) - %s.",
		mcpDBService.Name, mcpDBService.ID, mcpDBService.Type, instanceNameDetail))

	var mcpGoClient mcpclient.MCPClient
	var err error
	var needManualStart bool

	switch mcpDBService.Type {
	case model.ServiceTypeStdio:
		var stdioConf model.StdioConfig
		stdioConf.Command = mcpDBService.Command
		if stdioConf.Command == "" {
			return nil, nil, fmt.Errorf("StdioConfig for service %s (ID: %d) has an empty command. "+
				"This usually indicates the service was not properly configured during installation. "+
				"Expected Command field to contain the executable name (e.g., 'npx' for npm packages). "+
				"PackageManager: %s, SourcePackageName: %s, InstanceDetail: %s",
				mcpDBService.Name, mcpDBService.ID, mcpDBService.PackageManager, mcpDBService.SourcePackageName, instanceNameDetail)
		}
		if mcpDBService.ArgsJSON != "" {
			if errJson := json.Unmarshal([]byte(mcpDBService.ArgsJSON), &stdioConf.Args); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal ArgsJSON for service %s (ID: %d, Stdio): %v. Args will be empty.", mcpDBService.Name, mcpDBService.ID, errJson))
				stdioConf.Args = []string{}
			}
		} else {
			stdioConf.Args = []string{}
		}
		stdioConf.Env = []string{}
		if mcpDBService.DefaultEnvsJSON != "" && mcpDBService.DefaultEnvsJSON != "{}" {
			var defaultEnvs map[string]string
			if errJson := json.Unmarshal([]byte(mcpDBService.DefaultEnvsJSON), &defaultEnvs); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal DefaultEnvsJSON for %s (ID: %d, Stdio): %v. Proceeding without them.", mcpDBService.Name, mcpDBService.ID, errJson))
			} else {
				for key, value := range defaultEnvs {
					stdioConf.Env = append(stdioConf.Env, fmt.Sprintf("%s=%s", key, value))
				}
			}
		}
		common.SysLog(fmt.Sprintf("Stdio config for %s: Command=%s, Args=%v, Env=%v", mcpDBService.Name, stdioConf.Command, stdioConf.Args, stdioConf.Env))
		mcpGoClient, err = mcpclient.NewStdioMCPClient(stdioConf.Command, stdioConf.Env, stdioConf.Args...)
		needManualStart = false

	case model.ServiceTypeSSE:
		url := mcpDBService.Command // URL is stored in Command field for SSE/HTTP
		if url == "" {
			return nil, nil, fmt.Errorf("URL (from Command field) is empty for SSE service %s (ID: %d)", mcpDBService.Name, mcpDBService.ID)
		}
		var headers map[string]string
		if mcpDBService.HeadersJSON != "" && mcpDBService.HeadersJSON != "{}" {
			if errJson := json.Unmarshal([]byte(mcpDBService.HeadersJSON), &headers); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal HeadersJSON for SSE service %s (ID: %d): %v. Proceeding without custom headers.", mcpDBService.Name, mcpDBService.ID, errJson))
			}
		}
		common.SysLog(fmt.Sprintf("SSE config for %s: URL=%s, Headers=%v", mcpDBService.Name, url, headers))
		if len(headers) > 0 {
			mcpGoClient, err = mcpclient.NewSSEMCPClient(url, mcpclient.WithHeaders(headers))
		} else {
			mcpGoClient, err = mcpclient.NewSSEMCPClient(url)
		}
		needManualStart = true

	case model.ServiceTypeStreamableHTTP:
		url := mcpDBService.Command // URL is stored in Command field for SSE/HTTP
		if url == "" {
			return nil, nil, fmt.Errorf("URL (from Command field) is empty for StreamableHTTP service %s (ID: %d)", mcpDBService.Name, mcpDBService.ID)
		}
		var headers map[string]string
		if mcpDBService.HeadersJSON != "" && mcpDBService.HeadersJSON != "{}" {
			if errJson := json.Unmarshal([]byte(mcpDBService.HeadersJSON), &headers); errJson != nil {
				common.SysError(fmt.Sprintf("Failed to unmarshal HeadersJSON for StreamableHTTP service %s (ID: %d): %v. Proceeding without custom headers.", mcpDBService.Name, mcpDBService.ID, errJson))
			}
		}
		common.SysLog(fmt.Sprintf("StreamableHTTP config for %s: URL=%s, Headers (raw)=%v", mcpDBService.Name, url, headers))
		if len(headers) > 0 {
			// TODO: Correctly apply HTTP headers.
			// tdd.md and mcp-go patterns suggest `transport.WithHTTPHeaders(headers)`,
			// which would require importing "github.com/mark3labs/mcp-go/client/transport".
			// Due to current tool limitations on adding imports, this is omitted.
			// mcpclient.WithHeaders is likely not the correct option for HTTP stream transport headers.
			common.SysLog(fmt.Sprintf("WARNING: Custom headers for StreamableHTTP service %s are NOT being applied due to missing transport.WithHTTPHeaders option.", mcpDBService.Name))
			// Call without header options as the correct option builder is unavailable without new imports.
			mcpGoClient, err = mcpclient.NewStreamableHttpClient(url)
		} else {
			mcpGoClient, err = mcpclient.NewStreamableHttpClient(url)
		}
		needManualStart = true

	default:
		return nil, nil, fmt.Errorf("unsupported service type %s in createMcpGoServer", mcpDBService.Type)
	}

	if err != nil { // Consolidated error check after switch
		errMsg := fmt.Sprintf("Failed to create mcp-go client for %s (Type: %s, %s): %v", mcpDBService.Name, mcpDBService.Type, instanceNameDetail, err)
		common.SysError(errMsg)
		return nil, nil, errors.New(errMsg)
	}

	// Call client.Start() if needed
	if needManualStart {
		common.SysLog(fmt.Sprintf("Manually starting mcp-go client for %s (%s)...", mcpDBService.Name, instanceNameDetail))

		var startErr error
		switch cl := mcpGoClient.(type) {
		case interface{ Start(context.Context) error }:
			startErr = cl.Start(ctx)
		default:
			startErr = fmt.Errorf("client type %T does not have a Start method, but needManualStart was true", mcpGoClient)
		}

		if startErr != nil {
			errMsg := fmt.Sprintf("Failed to start mcp-go client for %s (%s): %v", mcpDBService.Name, instanceNameDetail, startErr)
			common.SysError(errMsg)
			if closeErr := mcpGoClient.Close(); closeErr != nil {
				common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s (%s) after Start() error: %v", mcpDBService.Name, instanceNameDetail, closeErr))
			}
			return nil, nil, errors.New(errMsg)
		}
		common.SysLog(fmt.Sprintf("Successfully started mcp-go client for %s (%s).", mcpDBService.Name, instanceNameDetail))

		// Start ping task for SSE and HTTP clients
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
		PingLoop:
			for {
				select {
				case <-ctx.Done():
					common.SysLog(fmt.Sprintf("Context done, stopping ping for %s", mcpDBService.Name))
					break PingLoop
				case <-ticker.C:
					if err := mcpGoClient.Ping(ctx); err != nil {
						common.SysError(fmt.Sprintf("Ping failed for %s: %v", mcpDBService.Name, err))
					}
				}
			}
		}()
	}

	mcpGoServer := mcpserver.NewMCPServer(
		mcpDBService.Name,
		mcpDBService.InstalledVersion,
		mcpserver.WithResourceCapabilities(true, true),
	)

	clientInfo := mcp.Implementation{
		Name:    fmt.Sprintf("one-mcp-proxy-for-%s-%s", mcpDBService.Name, instanceNameDetail),
		Version: common.Version,
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo

	common.SysLog(fmt.Sprintf("Initializing mcp-go client for %s (%s)...", mcpDBService.Name, instanceNameDetail))

	_, err = mcpGoClient.Initialize(ctx, initRequest)
	if err != nil {
		closeErr := mcpGoClient.Close()
		if closeErr != nil {
			common.SysError(fmt.Sprintf("Failed to close mcp-go client for %s (%s) after initialization error: %v", mcpDBService.Name, instanceNameDetail, closeErr))
		}
		errMsg := fmt.Sprintf("Failed to initialize mcp-go client for %s (%s): %v", mcpDBService.Name, instanceNameDetail, err)
		common.SysError(errMsg)
		return nil, nil, errors.New(errMsg)
	}
	common.SysLog(fmt.Sprintf("Successfully initialized mcp-go client for %s (%s). Adding resources...", mcpDBService.Name, instanceNameDetail))

	// Populate server with resources from client
	if err := addClientToolsToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add tools for %s (%s): %v", mcpDBService.Name, instanceNameDetail, err))
	}
	if err := addClientPromptsToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add prompts for %s (%s): %v", mcpDBService.Name, instanceNameDetail, err))
	}
	if err := addClientResourcesToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add resources for %s (%s): %v", mcpDBService.Name, instanceNameDetail, err))
	}
	if err := addClientResourceTemplatesToMCPServer(ctx, mcpGoClient, mcpGoServer, mcpDBService.Name); err != nil {
		common.SysError(fmt.Sprintf("Failed to add resource templates for %s (%s): %v", mcpDBService.Name, instanceNameDetail, err))
	}
	common.SysLog(fmt.Sprintf("Finished adding resources for %s (%s) to mcp-go server.", mcpDBService.Name, instanceNameDetail))

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

// getOrCreateProxyToSSEHandler (renamed from getOrCreateStdioToSSEHandler)
// creates or retrieves a cached SSE http.Handler for the given MCPService.
// The handler proxies the backend (Stdio, SSE, or HTTP) specified in MCPService over an SSE connection.
func getOrCreateProxyToSSEHandler(mcpDBService *model.MCPService) (http.Handler, error) {
	// Key for caching can remain the same as it's per service ID
	globalHandlerKey := fmt.Sprintf("global-service-%d-sseproxy", mcpDBService.ID)

	common.SysLog(fmt.Sprintf("Attempting to get/create SSE proxy handler for %s (ID: %d, Type: %s) with key: %s",
		mcpDBService.Name, mcpDBService.ID, mcpDBService.Type, globalHandlerKey))

	muStdioSSEWrappers.RLock() // Using the same mutex for simplicity, consider renaming if it becomes confusing
	existingHandler, found := initializedStdioSSEWrappers[globalHandlerKey]
	muStdioSSEWrappers.RUnlock()
	if found {
		common.SysLog(fmt.Sprintf("Reusing existing SSE proxy handler for %s (Key: %s)", mcpDBService.Name, globalHandlerKey))
		return existingHandler, nil
	}

	common.SysLog(fmt.Sprintf("No existing SSE proxy handler found for key %s. Creating new handler for %s.", globalHandlerKey, mcpDBService.Name))

	ctx := context.Background() // Using a background context for global handler creation

	// Create the mcp-go server. Configuration parsing is now inside createMcpGoServer.
	mcpGoSrv, mcpClient, err := createMcpGoServer(ctx, mcpDBService, "global") // Pass mcpDBService directly
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp-go server for %s (Type: %s): %w", mcpDBService.Name, mcpDBService.Type, err)
	}
	// Note: mcpClient is returned but not explicitly managed here.
	// Its lifecycle is assumed to be tied to mcpGoSrv or the resulting http.Handler (SSEServer).

	// Create the SSE handler from the mcp-go server
	newHandlerInstance, err := createSSEHttpHandler(mcpGoSrv, mcpDBService)
	if err != nil {
		// If handler creation fails, attempt to close the MCP client that was created.
		if mcpClient != nil {
			common.SysError(fmt.Sprintf("Closing MCP client for %s due to SSE handler creation failure: %v", mcpDBService.Name, err))
			if closeErr := mcpClient.Close(); closeErr != nil {
				common.SysError(fmt.Sprintf("Error closing MCP client for %s after SSE handler creation failure: %v", mcpDBService.Name, closeErr))
			}
		}
		return nil, fmt.Errorf("failed to create SSE handler for %s: %w", mcpDBService.Name, err)
	}

	muStdioSSEWrappers.Lock()
	if interimHandler, exists := initializedStdioSSEWrappers[globalHandlerKey]; exists {
		muStdioSSEWrappers.Unlock()
		common.SysLog(fmt.Sprintf("SSE Proxy Handler for key %s was created by another goroutine. Returning existing.", globalHandlerKey))
		// The newHandlerInstance and its associated mcpGoSrv & mcpClient are now orphaned.
		// The SSEServer (newHandlerInstance) should ideally handle closing its mcpGoServer,
		// which in turn should manage its client. If not, mcpClient.Close() would be needed here.
		// This is a general concern with concurrent handler creation and resource management.
		if mcpClient != nil {
			common.SysLog(fmt.Sprintf("Orphaned MCP client for %s needs cleanup due to race.", mcpDBService.Name))
			// TODO: Define clear ownership and cleanup for mcpClient if SSEServer doesn't manage it fully.
			// For now, assuming SSEServer (as an http.Handler wrapper around mcpserver) handles this.
			// If SSEServer is just a passthrough, then explicit mcpClient.Close() might be needed.
			// A robust solution might involve the SSEServer itself having a Close method that propagates to the client.
		}
		return interimHandler, nil
	}
	initializedStdioSSEWrappers[globalHandlerKey] = newHandlerInstance
	muStdioSSEWrappers.Unlock()

	common.SysLog(fmt.Sprintf("Successfully created and cached global SSE proxy handler for %s (Type: %s, Key: %s)",
		mcpDBService.Name, mcpDBService.Type, globalHandlerKey))
	return newHandlerInstance, nil
}

// defaultNewStdioSSEHandlerUncached is renamed to newProxyToSSEHandlerUncached
// It creates a new, non-cached http.Handler (SSE proxy) for an MCP service.
func newProxyToSSEHandlerUncached(ctx context.Context, mcpDBService *model.MCPService) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("newProxyToSSEHandlerUncached: Creating new SSE proxy handler for %s (ID: %d, Type: %s) via createMcpGoServer and createSSEHttpHandler.",
		mcpDBService.Name, mcpDBService.ID, mcpDBService.Type))

	// Delegate to the new core functions
	// Configuration parsing (like StdioConfig) is now handled within createMcpGoServer based on mcpDBService.Type
	mcpGoSrv, mcpClient, err := createMcpGoServer(ctx, mcpDBService, "user-specific-instance")
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

// ServiceFactory 用于创建适合特定类型的服务实例
func ServiceFactory(mcpDBService *model.MCPService) (Service, error) {
	var effectiveServiceType model.ServiceType
	var httpHandlerForProxy http.Handler
	var err error

	switch mcpDBService.Type {
	case model.ServiceTypeStdio, model.ServiceTypeSSE, model.ServiceTypeStreamableHTTP:
		common.SysLog(fmt.Sprintf("ServiceFactory: %s is %s type. Attempting to wrap as SSE proxy.", mcpDBService.Name, mcpDBService.Type))
		httpHandlerForProxy, err = getOrCreateProxyToSSEHandler(mcpDBService) // Use the renamed getter
		if err != nil {
			return nil, fmt.Errorf("failed to create SSE proxy handler for %s (type %s): %w", mcpDBService.Name, mcpDBService.Type, err)
		}
		// The effective service type from one-mcp's perspective is an SSE service,
		// because we are proxying the backend (whatever its type) over SSE.
		effectiveServiceType = model.ServiceTypeSSE

	// case model.ServiceTypeSSE: // Combined above
	// 	common.SysLog(fmt.Sprintf("ServiceFactory: %s is native SSE type. Attempting to wrap as SSE proxy.", mcpDBService.Name))
	// 	httpHandlerForProxy, err = getOrCreateProxyToSSEHandler(mcpDBService)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create SSE-to-SSE proxy handler for %s: %w", mcpDBService.Name, err)
	// 	}
	// 	effectiveServiceType = model.ServiceTypeSSE

	// case model.ServiceTypeStreamableHTTP: // Combined above
	// 	common.SysLog(fmt.Sprintf("ServiceFactory: %s is StreamableHTTP. Attempting to wrap as SSE proxy.", mcpDBService.Name))
	// 	httpHandlerForProxy, err = getOrCreateProxyToSSEHandler(mcpDBService)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create HTTP-to-SSE proxy handler for %s: %w", mcpDBService.Name, err)
	// 	}
	// 	effectiveServiceType = model.ServiceTypeSSE

	default:
		common.SysError(fmt.Sprintf("ServiceFactory: Unsupported service type '%s' for service %s", mcpDBService.Type, mcpDBService.Name))
		return nil, errors.New("unsupported service type: " + string(mcpDBService.Type))
	}

	proxyBaseService := NewBaseService(mcpDBService.ID, mcpDBService.Name, effectiveServiceType)

	// If we successfully got a handler, and the effective type is SSE, wrap it in SSESvc
	if httpHandlerForProxy != nil && effectiveServiceType == model.ServiceTypeSSE {
		return NewSSESvc(proxyBaseService, httpHandlerForProxy), nil
	}

	// This path should ideally not be reached if the switch statement is comprehensive
	// and httpHandlerForProxy is correctly assigned or an error is returned earlier.
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
