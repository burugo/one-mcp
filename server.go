package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/TBXark/confstore"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/sync/errgroup"
)

var BuildVersion = "dev"

type StdioMCPClientConfig struct {
	Command string            `json:"command"`
	Env     map[string]string `json:"env"`
	Args    []string          `json:"args"`
}

type SSEMCPClientConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type MCPClientType string

const (
	MCPClientTypeStdio MCPClientType = "stdio"
	MCPClientTypeSSE   MCPClientType = "sse"
)

type MCPClientConfig struct {
	Type           MCPClientType   `json:"type"`
	Config         json.RawMessage `json:"config"`
	PanicIfInvalid bool            `json:"panicIfInvalid"`
}
type SSEServerConfig struct {
	BaseURL string `json:"baseURL"`
	Addr    string `json:"addr"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Config struct {
	Server  SSEServerConfig            `json:"server"`
	Clients map[string]MCPClientConfig `json:"clients"`
}

// LoggingMiddleware wraps an http.Handler and logs request details
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 记录请求开始
		log.Printf("[%s] Request started: %s %s from %s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			r.Header.Get("User-Agent"),
		)

		next.ServeHTTP(w, r)

		// 记录请求结束
		log.Printf("[%s] Request completed: %s took %v",
			r.Method,
			r.URL.Path,
			time.Since(startTime),
		)
	})
}

func cleanPath(path string) string {
	// 移除 "sse/http" 或 "sse/https" 的前缀
	if idx := strings.Index(path, "/sse/http"); idx != -1 {
		if strings.Contains(path, "/message") {
			return path[:idx] + "/message"
		}
		return path[:idx] + "/"
	}
	return path
}

func main() {
	conf := flag.String("config", "config.json", "path to config file or a http(s) url")
	version := flag.Bool("version", false, "print version and exit")
	help := flag.Bool("help", false, "print help and exit")
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}
	if *version {
		fmt.Println(BuildVersion)
		return
	}
	config, err := confstore.Load[Config](*conf)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	start(config)
}

func start(config *Config) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var errorGroup errgroup.Group
	httpMux := http.NewServeMux()

	// 添加路径清理中间件
	cleanPathHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalPath := r.URL.Path
		cleanedPath := cleanPath(originalPath)

		// 记录路径清理的结果
		if originalPath != cleanedPath {
			log.Printf("[DEBUG] Path cleaned: %s -> %s", originalPath, cleanedPath)
		}

		r.URL.Path = cleanedPath
		httpMux.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:    config.Server.Addr,
		Handler: loggingMiddleware(cleanPathHandler),
	}
	info := mcp.Implementation{
		Name:    config.Server.Name,
		Version: config.Server.Version,
	}

	// 创建一个等待组来管理所有客户端的关闭
	var closeGroup sync.WaitGroup

	for name, clientConfig := range config.Clients {
		name := name // 为闭包创建新的变量
		log.Printf("Connecting to %s", name)
		mcpClient, err := newMCPClient(clientConfig)
		if err != nil {
			log.Fatalf("Failed to create MCP client: %v", err)
		}
		mcpServer := server.NewMCPServer(
			config.Server.Name,
			config.Server.Version,
			server.WithResourceCapabilities(true, true),
		)
		sseServer := server.NewSSEServer(mcpServer,
			server.WithBaseURL(config.Server.BaseURL),
			server.WithBasePath(name),
		)
		errorGroup.Go(func() error {
			addErr := addClient(ctx, info, mcpClient, mcpServer)
			if addErr != nil && clientConfig.PanicIfInvalid {
				return addErr
			}
			return nil
		})
		sseBasePath := fmt.Sprintf("/%s/", name)
		log.Printf("[DEBUG] Registering SSE server at path: %s", sseBasePath)
		httpMux.Handle(sseBasePath, sseServer)

		// 打印已注册的路由信息
		log.Printf("[DEBUG] Server routes for %s:", name)
		log.Printf("- SSE endpoint: %s", sseBasePath)
		log.Printf("- Message endpoint: %s", fmt.Sprintf("%smessage", sseBasePath))

		// 注册关闭处理
		closeGroup.Add(1)
		httpServer.RegisterOnShutdown(func() {
			defer closeGroup.Done()
			log.Printf("[DEBUG] Closing client %s", name)
			if err := mcpClient.Close(); err != nil {
				log.Printf("[ERROR] Error closing client %s: %v", name, err)
			}
		})
	}
	err := errorGroup.Wait()
	if err != nil {
		log.Fatalf("Failed to add clients: %v", err)
	}

	go func() {
		log.Printf("Starting SSE server")
		log.Printf("SSE server listening on %s", config.Server.Addr)
		hErr := httpServer.ListenAndServe()
		if hErr != nil && !errors.Is(hErr, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", hErr)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("\n[INFO] Shutdown signal received, stopping server...")

	// 创建一个带超时的上下文用于关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	// 先关闭 HTTP 服务器
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Server shutdown error: %v", err)
	}

	// 等待所有客户端关闭
	closeWaitChan := make(chan struct{})
	go func() {
		closeGroup.Wait()
		close(closeWaitChan)
	}()

	// 等待客户端关闭或超时
	select {
	case <-closeWaitChan:
		log.Println("[INFO] All clients closed successfully")
	case <-shutdownCtx.Done():
		log.Println("[WARN] Shutdown timeout waiting for clients to close")
	}

	log.Println("[INFO] Server shutdown complete")
}

func parseMCPClientConfig(conf MCPClientConfig) (any, error) {
	switch conf.Type {
	case MCPClientTypeStdio:
		var config StdioMCPClientConfig
		err := json.Unmarshal(conf.Config, &config)
		if err != nil {
			return nil, err
		}
		return config, nil
	case MCPClientTypeSSE:
		var config SSEMCPClientConfig
		err := json.Unmarshal(conf.Config, &config)
		if err != nil {
			return nil, err
		}
		return config, nil
	default:
		return nil, errors.New("invalid client type")
	}
}

func newMCPClient(conf MCPClientConfig) (client.MCPClient, error) {
	clientInfo, pErr := parseMCPClientConfig(conf)
	if pErr != nil {
		return nil, pErr
	}
	switch v := clientInfo.(type) {
	case StdioMCPClientConfig:
		envs := make([]string, 0, len(v.Env))
		for kk, vv := range v.Env {
			envs = append(envs, fmt.Sprintf("%s=%s", kk, vv))
		}
		return client.NewStdioMCPClient(v.Command, envs, v.Args...)
	case SSEMCPClientConfig:
		var options []client.ClientOption
		if len(v.Headers) > 0 {
			options = append(options, client.WithHeaders(v.Headers))
		}
		return client.NewSSEMCPClient(v.URL, options...)
	}
	return nil, errors.New("invalid client type")
}

func addClient(ctx context.Context, clientInfo mcp.Implementation, mcpClient client.MCPClient, mcpServer *server.MCPServer) error {
	// 使用带超时的上下文进行初始化
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo
	_, err := mcpClient.Initialize(initCtx, initRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize client: %w", err)
	}
	log.Printf("[INFO] Successfully initialized MCP client")

	// 添加各种资源到服务器
	if err = addClientToolsToServer(ctx, mcpClient, mcpServer); err != nil {
		return fmt.Errorf("failed to add tools: %w", err)
	}

	// 使用 errgroup 并行添加其他资源
	var g errgroup.Group

	g.Go(func() error {
		if err := addClientPromptsToServer(ctx, mcpClient, mcpServer); err != nil {
			return fmt.Errorf("failed to add prompts: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := addClientResourcesToServer(ctx, mcpClient, mcpServer); err != nil {
			return fmt.Errorf("failed to add resources: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := addClientResourceTemplatesToServer(ctx, mcpClient, mcpServer); err != nil {
			return fmt.Errorf("failed to add resource templates: %w", err)
		}
		return nil
	})

	// 等待所有资源添加完成
	if err := g.Wait(); err != nil {
		log.Printf("[WARN] Some resources failed to load: %v", err)
	}

	return nil
}

func addClientToolsToServer(ctx context.Context, mcpClient client.MCPClient, mcpServer *server.MCPServer) error {
	toolsRequest := mcp.ListToolsRequest{}
	for {
		tools, err := mcpClient.ListTools(ctx, toolsRequest)
		if err != nil {
			return err
		}
		log.Printf("Successfully listed %d tools", len(tools.Tools))
		for _, tool := range tools.Tools {
			log.Printf("Adding tool %s", tool.Name)
			mcpServer.AddTool(tool, mcpClient.CallTool)
		}
		if tools.NextCursor == "" {
			break
		}
		toolsRequest.PaginatedRequest.Params.Cursor = tools.NextCursor
	}
	return nil
}

func addClientPromptsToServer(ctx context.Context, mcpClient client.MCPClient, mcpServer *server.MCPServer) error {
	promptsRequest := mcp.ListPromptsRequest{}
	for {
		prompts, err := mcpClient.ListPrompts(ctx, promptsRequest)
		if err != nil {
			return err
		}
		log.Printf("Successfully listed %d prompts", len(prompts.Prompts))
		for _, prompt := range prompts.Prompts {
			log.Printf("Adding prompt %s", prompt.Name)
			mcpServer.AddPrompt(prompt, mcpClient.GetPrompt)
		}
		if prompts.NextCursor == "" {
			break
		}
		promptsRequest.PaginatedRequest.Params.Cursor = prompts.NextCursor
	}
	return nil
}

func addClientResourcesToServer(ctx context.Context, mcpClient client.MCPClient, mcpServer *server.MCPServer) error {
	resourcesRequest := mcp.ListResourcesRequest{}
	for {
		resources, err := mcpClient.ListResources(ctx, resourcesRequest)
		if err != nil {
			return err
		}
		log.Printf("Successfully listed %d resources", len(resources.Resources))
		for _, resource := range resources.Resources {
			log.Printf("Adding resource %s", resource.Name)
			mcpServer.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				readResource, e := mcpClient.ReadResource(ctx, request)
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

func addClientResourceTemplatesToServer(ctx context.Context, mcpClient client.MCPClient, mcpServer *server.MCPServer) error {
	resourceTemplatesRequest := mcp.ListResourceTemplatesRequest{}
	for {
		resourceTemplates, err := mcpClient.ListResourceTemplates(ctx, resourceTemplatesRequest)
		if err != nil {
			return err
		}
		log.Printf("Successfully listed %d resource templates", len(resourceTemplates.ResourceTemplates))
		for _, resourceTemplate := range resourceTemplates.ResourceTemplates {
			log.Printf("Adding resource template %s", resourceTemplate.Name)
			mcpServer.AddResourceTemplate(resourceTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				readResource, e := mcpClient.ReadResource(ctx, request)
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
