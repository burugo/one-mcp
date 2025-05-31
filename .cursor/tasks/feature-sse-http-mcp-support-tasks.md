# SSE和HTTP类型MCP Server支持实现

## Background and Motivation

用户在自定义安装界面中已经可以安装SSE和HTTP类型的MCP server，但现有后端ServiceFactory中这两种类型尚未实现，只有stdio类型有完整的支持。需要在后端增加对这两种类型的完整支持，并与前端ServiceConfigModal.tsx进行联调。

前端已经期望以下端点：
- SSE Endpoint: `${serverAddress}/proxy/${service?.name || ''}/sse`
- HTTP Endpoint: `${serverAddress}/proxy/${service?.name || ''}/mcp`

**重要发现**: 项目使用的 [mcp-go 库](https://github.com/mark3labs/mcp-go/) 已经支持 "stdio, SSE and streamable-HTTP transport layers"，这改变了我们的实现方向。

## Key Challenges and Analysis

1. **ServiceFactory限制**：当前ServiceFactory对SSE和HTTP类型返回未实现错误
2. **mcp-go库能力利用**：需要充分利用库的SSE和HTTP传输层支持
3. **客户端vs代理模式**：决定是使用mcp-go客户端连接还是直接HTTP代理
4. **统一的接口**：保持与现有stdio->SSE转换模式的一致性
5. **连接管理**：不同类型的MCP server需要不同的连接和通信方式
6. **Headers 存储与处理**: 决定采用在 `MCPService` 模型中新增 `HeadersJSON` 字段的方案，用于存储自定义请求头。
7. **HeadersJSON NULL Value**: Newly added `HeadersJSON` field was NULL for existing DB records, causing `sql: Scan error` because Go's `string` type cannot accept NULL. Resolved by an SQL script to update NULLs to `"{}"`.
8. **Routing Conflicts**: Initial proxy routes like `/proxy/:serviceName/*action` conflicted with more specific routes like `/sse` or `/proxy/:serviceName/sse/*action`. Resolved by removing the overly broad legacy wildcard route.
9. **API Endpoint for Custom Services**: Frontend expected a `/mcp_market/custom_service` endpoint which didn't exist. Created this new API endpoint and handler `CreateCustomService` in `market.go`.
10. **JWT Token for API**: `CreateCustomService` API requires JWT authentication. Initial attempts without a valid token or correct port failed.
11. **SSE and HTTP Proxy Initialization Failure**: 
    - The `mcp-go` SSE client fails to initialize with `transport error: transport not started yet` in `createSSEToSSEHandlerInstance` (`sse_native_service.go`).
    - User-provided reference code indicates that both `SSEMCPClient` and `StreamableHttpClient` (used for HTTP proxy) are marked with `needManualStart: true` and require an explicit `client.Start(ctx)` call before `client.Initialize(ctx, ...)`. 
    - This aligns with the SSE error and suggests our HTTP proxy implementation in `http_service.go` might also need a `Start()` call for robustness, even if it hasn't failed with the exact same error message yet.
    - `StdioMCPClient`, in the reference code, does not have `needManualStart: true`, which is consistent with our current Stdio proxy implementation in `service.go` not explicitly calling `Start()` before `Initialize()`.

## 架构决策分析

### 现有 Stdio -> SSE 模式
当前实现采用**代理客户端模式**：
- `mcpclient.NewStdioMCPClient()` 连接外部 stdio 服务
- 创建 `mcpserver.NewMCPServer()` 聚合外部服务能力
- 调用 `addClientToolsToMCPServer()` 等函数进行资源聚合
- 用 `mcpserver.NewSSEServer()` 包装成 SSE 服务

### 不同协议组合分析

| 前端协议 | 外部服务协议 | 推荐实现方式 | 原因 |
|---------|-------------|-------------|------|
| **SSE** | **SSE** | 🔄 **代理客户端模式** | 保持一致性，支持资源聚合和权限控制 |
| **SSE** | **HTTP** | ✅ **代理客户端模式** | 协议转换需要，统一前端接口 |
| **HTTP** | **SSE** | ✅ **代理客户端模式** | 协议转换需要，复杂度高 |
| **HTTP** | **HTTP** | 🤔 **可选择简单转发** | 同协议可直接转发，但失去聚合能力 |

### 推荐架构决策
**统一采用代理客户端模式**，原因：
1. **架构一致性**：与现有 stdio->SSE 模式保持一致
2. **功能完整性**：支持资源聚合、权限控制、缓存等高级功能
3. **统一管理**：所有服务通过统一的 one-mcp 接口访问
4. **协议透明**：前端无需关心外部服务的具体协议
5. **扩展性**：便于未来添加认证、监控、限流等功能

### 实现计划调整
基于架构决策，保持现有的 `addClientXxxToMCPServer` 调用，实现：
- **SSE -> SSE**: `NewSSEMCPClient` + `NewMCPServer` + `NewSSEServer`
- **SSE -> HTTP**: `NewStreamableHttpClient` + `NewMCPServer` + `NewSSEServer`  
- **HTTP -> SSE**: `NewSSEMCPClient` + `NewMCPServer` + 自定义HTTP handler
- **HTTP -> HTTP**: `NewStreamableHttpClient` + `NewMCPServer` + 自定义HTTP handler

## High-level Task Breakdown

- **阶段1**：研究mcp-go库的SSE/HTTP客户端能力
- **阶段2**：设计统一的Service接口实现，包括 `MCPService` 模型调整 (增加 `HeadersJSON`)
- **阶段3**：实现SSE和HTTP类型的MCP代理，包括从 `HeadersJSON` 解析和使用 Headers
- **阶段4**：更新ServiceFactory和路由处理
- **阶段5**：更新 `InstallOrAddService` 以支持 `HeadersJSON`
- **阶段6**：前后端联调测试

## Project Status Board

- 🔄 **进行中**: Headers存储方案已确定，准备实现
- ⏸️ **待开始**: 后端SSE/HTTP类型实现 (包括模型修改、Headers处理)
- ⏸️ **待开始**: 前后端联调测试

## Completed Tasks

- [x] 分析mcp-go库传输层支持能力 `ref-struct`
- [x] 重新设计基于mcp-go库的实现方案 `ref-struct`
- [x] 研究mcp-go的SSE/HTTP客户端API `research`
- [x] 实现原生SSE类型MCP服务支持 (sse\_native\_service.go) `implementation`
- [x] 实现HTTP类型MCP服务支持 (http\_service.go) `implementation`
- [x] 更新ServiceFactory支持新类型 (service.go) `implementation`
- [x] 修复 `NewStreamableHttpClient` 的使用 (http\_service.go)
- [x] 添加 `HTTPSvc` 和 `NewHTTPSvc` (service.go)
- [x] 更新代理路由，区分SSE和MCP路径 (api-router.go)
- [x] 实现 `HTTPProxyHandler` (proxy\_handler.go)
- [x] 清理 `service.go` 中重复的 handler 创建函数
- [x] 决策Headers存储方案：新增`HeadersJSON`字段 `design-decision`

## In Progress Tasks

- [x] **模型修改**: 在 `MCPService` 模型 (`backend/model/mcp_service.go`) 中添加 `HeadersJSON string \`json:"headers_json,omitempty" db:"headers_json"\`` 字段。 `db-schema`
- [x] **数据库迁移**: 已自动执行数据库迁移添加 `headers_json` 字段。修复现有记录的NULL值问题，将所有NULL值设置为空JSON对象`{}`。 `db-migration`
- [x] **Headers解析**: 更新 `createSSEToSSEHandlerInstance` (`sse_native_service.go`) 和 `createHTTPToHTTPHandlerInstance` (`http_service.go`) 从 `mcpDBService.HeadersJSON` 读取并填充 `SSEConfig.Headers` 和 `HTTPConfig.Headers`。 `implementation`
- [x] **API更新**: 修改 `InstallOrAddService` (`market.go`) 以接收前端传递的 `headers` 参数 (例如 `map[string]string`)，并将其序列化后存入 `MCPService.HeadersJSON`。新增 `CreateCustomService` 端点支持自定义服务创建。 `api-dev`
- [x] **路由修复**: 解决了proxy路由冲突问题，删除冲突的通配符路由。 `bug-fix`
- [x] **NULL值修复**: 修复了`headers_json`字段的NULL值扫描错误，将所有NULL值更新为空JSON对象`{}`。 `bug-fix`
- [x] **mcp-go Headers传递**: 已在 `sse_native_service.go` 和 `http_service.go` 中正确使用 `mcpclient.WithHeaders` 和 `transport.WithHTTPHeaders` 将解析后的Headers传递给mcp-go客户端。 `implementation`
- [x] 解决 `sse_native_service.go` 和 `http_service.go` 中的 linter errors (`undefined: addClientResourcesToMCPServer`, `undefined: addClientResourceTemplatesToMCPServer`)。确保这些辅助函数在 `proxy` 包内可被正确调用。`refactor`
- [x] 验证 `ServiceFactory` 调用 `getOrCreateSSEToSSEHandler` 和 `getOrCreateHTTPToHTTPHandler` 的正确性。 `testing`
- [x] 基本代码编译测试通过 `testing`
- [ ] 测试SSE和HTTP类型服务创建和代理功能。 `testing`
- [ ] **修复 SSE 和 HTTP 代理初始化问题**: 调查并解决 `mcp-go` SSE 和 HTTP 客户端在代理初始化时可能存在的启动和初始化顺序问题。 `bug-fix` `mcp-go-integration`
  - **诊断**:
      - `SSEMCPClient` 在 `createSSEToSSEHandlerInstance` 中的 `Initialize()` 调用失败，提示 `transport error: transport not started yet`。
      - 用户提供的参考代码表明 `SSEMCPClient` 和 `StreamableHttpClient` (用于HTTP代理) 在其实现中都需要在 `Initialize` 前手动调用 `Start()`。
  - **尝试的解决方案**:
      - 在 `backend/library/proxy/sse_native_service.go` 的 `createSSEToSSEHandlerInstance` 函数中，在调用 `mcpGoClient.Initialize()` 之前，显式调用 `mcpGoClient.Start(initializeCtx)`。
      - 在 `backend/library/proxy/http_service.go` 的 `createHTTPToHTTPHandlerInstance` 函数中，在调用 `mcpGoClient.Initialize()` 之前，显式调用 `mcpGoClient.Start(initializeCtx)`。
  - **预期结果**: SSE 和 HTTP 代理应能成功初始化并连接到外部服务。
  - **验证**:
      - SSE: 通过代理访问用户提供的 SSE 测试 URL (`http://home.pika12.com:8880/hello/sse`)，确认能收到数据。
      - HTTP: 创建一个自定义HTTP服务（例如，指向一个公开的JSON API如 `https://jsonplaceholder.typicode.com/todos/1`），通过代理访问它，确认能收到数据或正确的HTTP响应。

## Known Issues

- **mcp-go API函数签名**: `addClientResourcesToMCPServer` 和 `addClientResourceTemplatesToMCPServer` 中的函数签名不匹配问题已临时注释，需要进一步研究mcp-go库的正确API使用方式。
- **Headers传递**: 需要研究mcp-go库是否支持客户端自定义Headers，以及如何正确传递认证信息。

## Future Tasks

- [ ] 添加对应的健康检查实现 `new-feat`
- [ ] 创建SSE和HTTP类型的测试服务 `test-prep`
- [ ] 前后端联调测试 `integration`
- [ ] 完善错误处理和日志 `bug-fix`
- [ ] **UI完善**: 为自定义服务提供更详细的配置界面（例如，更友好的Headers输入方式，环境变量配置等）。 `ui-ux`
- [ ] **错误处理和日志**: 增强代理和MCP服务创建过程中的错误处理和日志记录。 `enhancement`
- [ ] **安全加固**: 审查Headers传递和处理过程中的安全隐患。 `security`
- [ ] **文档更新**: 更新项目文档，说明如何配置和使用HTTP/SSE代理服务。 `documentation`

## Implementation Plan

### 技术架构（更新版）

**基于mcp-go库的实现方案**：

**方案A: 客户端连接模式（推荐）**
\`\`\`go
// SSE类型
sseClient := client.NewSSEMCPClient(serverURL, client.WithHeaders(config.Headers), ...) // 使用Headers
sseServer := server.NewSSEServer(mcpServer, options...)

// HTTP类型
httpClient := client.NewHTTPMCPClient(serverURL, client.WithHeaders(config.Headers), ...) // 使用Headers
// 包装成统一的http.Handler接口
\`\`\`

### 实现策略

**1. MCPService 模型**
   - 增加 `HeadersJSON string \`json:"headers_json,omitempty" db:"headers_json"\`` 字段。

**2. Headers 处理**
   - `InstallOrAddService`：接收 `headers: map[string]string`，序列化为JSON字符串存入 `HeadersJSON`。
   - 服务创建 (e.g., `createSSEToSSEHandlerInstance`)：从 `HeadersJSON` 反序列化，填充到 `SSEConfig.Headers` / `HTTPConfig.Headers`。
   - `mcp-go` 客户端初始化：使用 `client.WithHeaders(config.Headers)` 选项。


**3. SSE类型实现**
- 使用 `client.NewSSEMCPClient()` 连接外部SSE MCP服务器。
- 仿照现有的stdio->SSE模式，创建中间层服务器。
- 支持持久连接和事件流处理。

**4. HTTP类型实现**  
- 使用 `client.NewHTTPMCPClient()` 或 `client.NewStreamableHttpClient` 连接外部HTTP MCP服务器。
- 实现请求响应模式的代理。
- 支持标准HTTP MCP协议。

**5. 统一接口设计**
\`\`\`go
type Service interface {
    // 现有接口方法保持不变
    ID() int64
    Name() string
    Type() model.ServiceType
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
    CheckHealth(ctx context.Context) (*ServiceHealth, error)
    GetHealth() *ServiceHealth
    GetConfig() map[string]interface{} // 可能会包含解析后的Headers
    UpdateConfig(config map[string]interface{}) error
}

// 新增HTTP Handler接口
type HTTPService interface {
    Service
    http.Handler  // 支持直接作为HTTP handler使用
}
\`\`\`

### 数据流（更新版）

**SSE类型**:
\`\`\`
Frontend -> /proxy/{serviceName}/sse/* -> SSEProxyService -> mcp-go SSE client (with headers) -> 外部SSE MCP Server
\`\`\`

**HTTP类型**:
\`\`\`
Frontend -> /proxy/{serviceName}/mcp/* -> HTTPProxyService -> mcp-go HTTP client (with headers) -> 外部HTTP MCP Server
\`\`\`

### 环境配置

SSE和HTTP类型的服务配置：
- **URL**: (必需) 存储在 `mcp_services.command` 字段。
- **Headers**: (可选) 请求头，作为JSON对象字符串存储在新增的 `mcp_services.headers_json` 字段。例如 `{"Authorization": "Bearer token", "X-Custom": "value"}`。
- **其他连接参数 (如 API_KEY, TIMEOUT)**: 仍然可以通过 `default_envs_json` 存储，或者如果适合放入Header，也可以统一放入 `headers_json`。

### ServiceFactory更新

\`\`\`go
func ServiceFactory(mcpDBService *model.MCPService) (Service, error) {
    switch mcpDBService.Type {
    case model.ServiceTypeStdio:
        // 现有实现保持不变
        return getOrCreateStdioToSSEHandler(mcpDBService) // 假设此函数返回 Service 类型
        
    case model.ServiceTypeSSE:
        // 新增：使用mcp-go SSE客户端
        return getOrCreateSSEToSSEHandler(mcpDBService) // 假设此函数返回 Service 类型
        
    case model.ServiceTypeStreamableHTTP:
        // 新增：使用mcp-go HTTP客户端
        return getOrCreateHTTPToHTTPHandler(mcpDBService) // 假设此函数返回 Service 类型
        
    default:
        return nil, errors.New("unsupported service type")
    }
}
\`\`\`
Note: The createXxxHandler functions currently return `http.Handler`. The ServiceFactory needs to wrap these into an object that implements the `Service` interface (like `SSESvc` or `HTTPSvc`). The current `ServiceFactory` logic does this correctly by creating `NewSSESvc` or `NewHTTPSvc`.

### mcp-go库集成点

需要研究和使用的mcp-go库功能：
1. **传输层**: SSE和HTTP客户端创建, 关键是 `client.WithHeaders()` 选项。
2. **会话管理**: 多客户端连接管理。
3. **错误处理**: 连接重试和恢复。
4. **协议支持**: 完整的MCP协议实现。

### Relevant Files

**核心文件**:
- `backend/library/proxy/service.go` - Service接口和ServiceFactory
- `backend/api/handler/proxy_handler.go` - 代理请求处理
- `backend/api/route/api-router.go` - 路由配置
- `backend/model/mcp_service.go` - 服务类型定义 (将添加 HeadersJSON)
- `backend/api/handler/market.go` - InstallOrAddService (将处理 HeadersJSON)

**主要修改文件**:
- `backend/library/proxy/sse_native_service.go` - 原生SSE服务实现 (解析 HeadersJSON)
- `backend/library/proxy/http_service.go` - HTTP服务实现 (解析 HeadersJSON)

**前端相关**:
- `frontend/src/components/market/ServiceConfigModal.tsx` - 前端配置界面 (未来可能需要更新以支持 Headers 输入)

## Lessons

- **库能力调研的重要性**: 深入了解第三方库的完整能力可以显著简化实现
- **mcp-go传输层**: 该库已提供stdio、SSE、HTTP三种传输层，无需重复造轮子
- **配置清晰度**: 为特定用途（如Headers）设置专用字段优于复用通用字段。

## ACT mode Feedback or Assistance Requests

暂无

## User Specified Lessons

暂无 