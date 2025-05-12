package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Function variables for dependency injection / testing
var (
	getEnabledServicesFunc = model.GetEnabledServices // Default to the real implementation
	newStdioMCPClientFunc  = client.NewStdioMCPClient // Default to the real implementation
)

// MCPClientManager 管理所有 MCP 客户端实例
type MCPClientManager struct {
	clients     map[string]*client.Client // 包名 -> 客户端实例 (使用正确的 *client.Client 类型)
	clientInfo  map[string]*MCPServerInfo // 包名 -> 服务器信息
	clientMutex sync.RWMutex
}

var (
	globalClientManager      *MCPClientManager
	clientManagerInitialized bool
	clientManagerMutex       sync.Mutex
)

// GetMCPClientManager 获取全局客户端管理器
func GetMCPClientManager() *MCPClientManager {
	clientManagerMutex.Lock()
	defer clientManagerMutex.Unlock()

	if !clientManagerInitialized {
		globalClientManager = &MCPClientManager{
			clients:    make(map[string]*client.Client),
			clientInfo: make(map[string]*MCPServerInfo),
		}
		// 初始化时加载已安装的服务
		globalClientManager.loadInstalledServices()
		clientManagerInitialized = true
	}

	return globalClientManager
}

// loadInstalledServices 从数据库加载已安装的服务
func (m *MCPClientManager) loadInstalledServices() {
	services, err := getEnabledServicesFunc() // Use the function variable
	if err != nil {
		log.Printf("Failed to load installed services: %v", err)
		return
	}

	for _, service := range services {
		// 只处理 stdio 类型并且有包名的服务
		if service.Type != model.ServiceTypeStdio || service.SourcePackageName == "" {
			continue
		}

		// 尝试创建客户端并初始化
		if err := m.InitializeClient(service.SourcePackageName, service.ID); err != nil {
			log.Printf("Failed to initialize client for %s: %v", service.SourcePackageName, err)
			continue
		}
	}
}

// InitializeClient 为特定包创建并初始化客户端
func (m *MCPClientManager) InitializeClient(packageName string, serviceID int64) error {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	// 检查是否已存在
	if _, exists := m.clients[packageName]; exists {
		return nil // 已存在，无需重复初始化
	}

	// 创建新客户端
	command := "npx"
	args := []string{"-y", packageName}
	env := os.Environ()

	mcpClient, err := newStdioMCPClientFunc(command, env, args...) // Use the function variable
	if err != nil {
		return fmt.Errorf("failed to create MCP client for %s: %w", packageName, err)
	}

	// 启动客户端 (不需要手动调用 Start)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Use shorter timeout for testing initialization
	defer cancel()

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "one-mcp",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		mcpClient.Close() // 确保关闭
		return fmt.Errorf("failed to initialize MCP client for %s: %w", packageName, err)
	}

	// 保存客户端和服务器信息
	m.clients[packageName] = mcpClient
	m.clientInfo[packageName] = &MCPServerInfo{
		Name:            initResult.ServerInfo.Name,
		Version:         initResult.ServerInfo.Version,
		ProtocolVersion: initResult.ProtocolVersion,
		Capabilities:    initResult.Capabilities,
	}

	// 如果有服务ID，更新服务健康状态
	if serviceID > 0 {
		go updateServiceHealthStatus(serviceID, m.clientInfo[packageName])
	}

	return nil
}

// updateServiceHealthStatus 更新服务的健康状态
func updateServiceHealthStatus(serviceID int64, serverInfo *MCPServerInfo) {
	// 获取服务
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		log.Printf("Failed to get service (ID: %d) for health update: %v", serviceID, err)
		return
	}

	// 更新服务健康状态
	service.HealthStatus = "healthy" // 初始健康状态设为健康，因为我们成功初始化了

	// 创建健康详情
	healthDetails := map[string]interface{}{
		"mcpServer": serverInfo,
		"lastCheck": time.Now().Format(time.RFC3339),
		"status":    "healthy",
		"message":   "MCP server initialized successfully",
	}

	// 将健康详情转换为 JSON
	healthDetailsJSON, err := json.Marshal(healthDetails)
	if err != nil {
		log.Printf("Failed to marshal health details: %v", err)
	} else {
		service.HealthDetails = string(healthDetailsJSON)
	}

	// 更新最后健康检查时间
	service.LastHealthCheck = time.Now()

	// 保存到数据库
	if err := model.UpdateService(service); err != nil {
		log.Printf("Failed to update service health status (ID: %d): %v", serviceID, err)
		return
	}
}

// RemoveClient 移除并关闭客户端
func (m *MCPClientManager) RemoveClient(packageName string) {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	if client, exists := m.clients[packageName]; exists {
		if client != nil {
			client.Close() // 确保关闭连接
		}
		delete(m.clients, packageName)
		delete(m.clientInfo, packageName)
	}
}

// GetClient 获取特定包的客户端
func (m *MCPClientManager) GetClient(packageName string) (*client.Client, bool) { // 返回 *client.Client
	m.clientMutex.RLock()
	defer m.clientMutex.RUnlock()

	client, exists := m.clients[packageName]
	return client, exists
}

// GetServerInfo 获取特定包的服务器信息
func (m *MCPClientManager) GetServerInfo(packageName string) (*MCPServerInfo, bool) {
	m.clientMutex.RLock()
	defer m.clientMutex.RUnlock()

	info, exists := m.clientInfo[packageName]
	return info, exists
}

// ListTools 使用现有客户端查询工具
func (m *MCPClientManager) ListTools(ctx context.Context, packageName string) ([]mcp.Tool, error) {
	// 获取现有客户端
	mcpClient, exists := m.GetClient(packageName)
	if !exists {
		// 客户端不存在，尝试初始化
		if err := m.InitializeClient(packageName, 0); err != nil {
			return nil, fmt.Errorf("failed to initialize client: %w", err)
		}

		// 重新获取初始化后的客户端
		mcpClient, exists = m.GetClient(packageName)
		if !exists {
			return nil, fmt.Errorf("client initialization failed")
		}
	}

	// 使用现有客户端查询工具
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := mcpClient.ListTools(ctx, toolsRequest) // 直接在 *client.Client 上调用
	if err != nil {
		// 如果失败，移除并重新创建客户端
		m.RemoveClient(packageName)
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return toolsResult.Tools, nil
}

// GetAllClientInfo 获取所有已初始化的客户端信息
func (m *MCPClientManager) GetAllClientInfo() map[string]*MCPServerInfo {
	m.clientMutex.RLock()
	defer m.clientMutex.RUnlock()

	// 创建副本以避免并发访问问题
	result := make(map[string]*MCPServerInfo, len(m.clientInfo))
	for k, v := range m.clientInfo {
		result[k] = v
	}

	return result
}

// CloseAll 关闭所有客户端
func (m *MCPClientManager) CloseAll() {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	for name, c := range m.clients {
		c.Close()
		delete(m.clients, name)
		delete(m.clientInfo, name)
	}
}
