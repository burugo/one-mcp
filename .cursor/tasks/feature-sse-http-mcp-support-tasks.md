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
11. **SSE and HTTP Proxy Initialization Failure**: （已通过重构解决）
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

### 不同协议组合分析 (初始设想)

| 前端协议 | 外部服务协议 | 推荐实现方式 | 原因 |
|---------|-------------|-------------|------|
| **SSE** | **SSE** | 🔄 **代理客户端模式** | 保持一致性，支持资源聚合和权限控制 |
| **SSE** | **HTTP** | ✅ **代理客户端模式** | 协议转换需要，统一前端接口 |
| **HTTP** | **SSE** | ✅ **代理客户端模式** | 协议转换需要，复杂度高 |
| **HTTP** | **HTTP** | 🤔 **可选择简单转发** | 同协议可直接转发，但失去聚合能力 |

### 推荐架构决策 (已演进)
**统一采用代理客户端模式，并重构核心逻辑到 `createMcpGoServer`**，原因：
1. **架构一致性**：与现有 stdio->SSE 模式保持一致
2. **功能完整性**：支持资源聚合、权限控制、缓存等高级功能
3. **统一管理**：所有服务通过统一的 one-mcp 接口访问
4. **协议透明**：前端无需关心外部服务的具体协议
5. **扩展性**：便于未来添加认证、监控、限流等功能
6. **代码复用**: 通过 `createMcpGoServer` 统一处理 Stdio, SSE, 和 HTTP 后端服务的 mcp-go 客户端和服务端创建。

### 实现计划调整 (已演进)
基于架构决策，通过统一的 `createMcpGoServer` 创建 `mcpserver.MCPServer`，然后根据输出需求包装：
- **SSE 输出**:
    - Stdio Backend: `StdioMCPClient` -> `MCPServer` -> `SSEServer`
    - SSE Backend: `SSEMCPClient` -> `MCPServer` -> `SSEServer`
    - HTTP Backend: `StreamableHttpClient` -> `MCPServer` -> `SSEServer`
- **HTTP 输出 (见 @feature-http-output-proxy-tasks.md)**:
    - Stdio/SSE/HTTP Backend -> `MCPServer` -> `StreamableHTTPServer`

## High-level Task Breakdown (已大部分完成)

- **阶段1**：研究mcp-go库的SSE/HTTP客户端能力
- **阶段2**：设计统一的Service接口实现，包括 `MCPService` 模型调整 (增加 `HeadersJSON`)
- **阶段3**：实现SSE和HTTP类型的MCP代理，包括从 `HeadersJSON` 解析和使用 Headers (已重构到 `createMcpGoServer`)
- **阶段4**：更新ServiceFactory和路由处理 (已通过 `ProxyHandler` 和 `HTTPProxyHandler` 调整)
- **阶段5**：更新 `InstallOrAddService` 以支持 `HeadersJSON`
- **阶段6**：前后端联调测试

## Project Status Board

- ✅ **已完成**: SSE代理 (Stdio, SSE, HTTP 后端) 功能已实现并通过初步测试。
- 🔄 **进行中**: 前后端联调测试 (SSE->Stdio, SSE->SSE 已测试, SSE->HTTP 待充分验证)

## Completed Tasks

- [x] 分析mcp-go库传输层支持能力 `ref-struct`
- [x] 重新设计基于mcp-go库的实现方案 `ref-struct`
- [x] 研究mcp-go的SSE/HTTP客户端API `research`
- [x] **代码重构**: 将核心的MCP客户端和服务创建逻辑统一到 `createMcpGoServer` 函数中，支持 Stdio, SSE, 和 HTTP 后端类型。 `refactor` `mcp-go-integration`
- [x] **Ping Task 实现**: 为 SSE 和 HTTP 类型的 mcp-go 客户端添加了 `startPingTask` 以保持连接活跃。 `implementation` `mcp-go-integration`
- [x] **测试修复**: 修复了因函数重命名和逻辑调整导致的单元测试失败问题 (`proxy_handler_test.go`)。 `testing` `bug-fix`
- [x] 实现原生SSE类型MCP服务支持 (sse_native_service.go) `implementation` (注: 后续重构到 `service.go` 的 `createMcpGoServer`)
- [x] 实现HTTP类型MCP服务支持 (http_service.go) `implementation` (注: 后续重构到 `service.go` 的 `createMcpGoServer`)
- [x] 更新ServiceFactory支持新类型 (service.go) `implementation`
- [x] 修复 `NewStreamableHttpClient` 的使用 (http_service.go) (注: 后续重构到 `service.go` 的 `createMcpGoServer`)
- [x] 添加 `HTTPSvc` 和 `NewHTTPSvc` (service.go)
- [x] 更新代理路由，区分SSE和MCP路径 (api-router.go) (已演进为统一的 `/proxy/:serviceName/:action/*` 路由)
- [x] 实现 `HTTPProxyHandler` (proxy_handler.go) (已演进为 `ProxyHandler` 和 `HTTPProxyHandler` 两个，并调整日志)
- [x] 清理 `service.go` 中重复的 handler 创建函数 (已通过 `createMcpGoServer` 重构完成)
- [x] 决策Headers存储方案：新增`HeadersJSON`字段 `design-decision`
- [x] **模型修改**: 在 `MCPService` 模型 (`backend/model/mcp_service.go`) 中添加 `HeadersJSON string \`json:"headers_json,omitempty" db:"headers_json"\`` 字段。 `db-schema`
- [x] **数据库迁移**: 已自动执行数据库迁移添加 `headers_json` 字段。修复现有记录的NULL值问题，将所有NULL值设置为空JSON对象`{}`。 `db-migration`
- [x] **Headers解析**: 更新 `createMcpGoServer` 从 `mcpDBService.HeadersJSON` 读取并填充 `SSEConfig.Headers` 和 `HTTPConfig.Headers`。 `implementation`
- [x] **API更新**: 修改 `InstallOrAddService` (`market.go`) 以接收前端传递的 `headers` 参数 (例如 `map[string]string`)，并将其序列化后存入 `MCPService.HeadersJSON`。新增 `CreateCustomService` 端点支持自定义服务创建。 `api-dev`
- [x] **路由修复**: 解决了proxy路由冲突问题，删除冲突的通配符路由。 `bug-fix`
- [x] **NULL值修复**: 修复了`headers_json`字段的NULL值扫描错误，将所有NULL值更新为空JSON对象`{}`。 `bug-fix`
- [x] **mcp-go Headers传递**: 已在 `createMcpGoServer` 中正确使用 `mcpclient.WithHeaders` 和 `transport.WithHTTPHeaders` 将解析后的Headers传递给mcp-go客户端。 `implementation`
- [x] 解决 `sse_native_service.go` 和 `http_service.go` 中的 linter errors (`undefined: addClientResourcesToMCPServer`, `undefined: addClientResourceTemplatesToMCPServer`)。确保这些辅助函数在 `proxy` 包内可被正确调用。`refactor` (注: 这些辅助函数的功能已部分集成或其需求已改变)
- [x] 验证 `ServiceFactory` 调用 `getOrCreateProxyToSSEHandler` (原 `getOrCreateSSEToSSEHandler` 和 `getOrCreateHTTPToHTTPHandler`) 的正确性。 `testing`
- [x] 基本代码编译测试通过 `testing`
- [x] **修复 SSE 和 HTTP 代理初始化问题**: 通过 `createMcpGoServer` 重构，确保 `mcp-go` SSE 和 HTTP 客户端在代理初始化时正确启动和初始化。 `bug-fix` `mcp-go-integration`
  - **诊断**: (历史记录)
      - `SSEMCPClient` 在 `createSSEToSSEHandlerInstance` 中的 `Initialize()` 调用失败，提示 `transport error: transport not started yet`。
      - 用户提供的参考代码表明 `SSEMCPClient` 和 `StreamableHttpClient` (用于HTTP代理) 在其实现中都需要在 `Initialize` 前手动调用 `Start()`。
  - **解决方案**: (已实现)
      - 在 `createMcpGoServer` 中，对于需要手动启动的客户端 (SSE, HTTP)，在调用 `mcpGoClient.Initialize()` 之前，显式调用 `mcpGoClient.Start(ctx)`。
      - 添加 `startPingTask` 以保持连接。
  - **预期结果**: SSE 和 HTTP 代理应能成功初始化并连接到外部服务。
  - **验证**:
      - SSE: 通过代理访问用户提供的 SSE 测试 URL (`http://home.pika12.com:8880/hello/sse`)，确认能收到数据。(SSE->Stdio, SSE->SSE 已测试)
      - HTTP: 创建一个自定义HTTP服务（例如，指向一个公开的JSON API如 `https://jsonplaceholder.typicode.com/todos/1`），通过代理访问它，确认能收到数据或正确的HTTP响应。(SSE->HTTP 待充分验证)
- [x] 测试SSE->Stdio, SSE->SSE代理功能。 `testing`

## In Progress Tasks

- [ ] 全面测试 SSE->HTTP 代理功能。 `testing`

## Known Issues

- **mcp-go API函数签名**: `addClientResourcesToMCPServer` 和 `addClientResourceTemplatesToMCPServer` 中的函数签名可能与当前 `mcp-go` 版本不匹配或用法有变。当前重构后的代码不直接依赖这些特定函数，但如果未来需要更细致的资源控制，可能需要重新审视。
- **Headers传递 (基本解决)**: `client.WithHeaders` 和 `transport.WithHTTPHeaders` 已用于在 `createMcpGoServer` 中传递headers。特定复杂场景下的headers处理（如动态headers）可能需要进一步考虑。

## Future Tasks

- [ ] 添加对应的健康检查实现 `new-feat`
- [ ] 创建SSE和HTTP类型的测试服务 `test-prep`
- [ ] (承接 In Progress) 前后端联调测试 SSE->HTTP `integration`
- [ ] 完善错误处理和日志 `bug-fix`
- [ ] **UI完善**: 为自定义服务提供更详细的配置界面（例如，更友好的Headers输入方式，环境变量配置等）。 `ui-ux`
- [ ] **错误处理和日志**: 增强代理和MCP服务创建过程中的错误处理和日志记录。 `enhancement`
- [ ] **安全加固**: 审查Headers传递和处理过程中的安全隐患。 `security`
- [ ] **文档更新**: 更新项目文档，说明如何配置和使用HTTP/SSE代理服务。 `documentation`

## Bug Fixes and Refinements (User Request 2024-07-01)

### Issue 1: Custom Stdio Service Creation Endpoint

- **Background**: Custom services of type 'stdio' created via `CustomServiceModal.tsx` are currently sent to the generic `/api/mcp_market/custom_service` endpoint. They should instead be routed to `/api/mcp_market/install_or_add_service` for consistency with how market `stdio` services are handled, allowing for proper package management and installation logic.
- **Task Breakdown**:
    - **[ ] Task B1.1: Modify Frontend Logic for Stdio Custom Service** `refactor` `frontend`
        - **Description**: In `frontend/src/pages/ServicesPage.tsx`, update the `handleCreateCustomService` function.
        - **Details**:
            - If `serviceData.type` is `'stdio'`, call the `/api/mcp_market/install_or_add_service` endpoint.
            - Parse `serviceData.command` (e.g., "npx my-package" or "uvx my-tool") to extract `PackageManager` ("npm" or "uv") and `PackageName` ("my-package" or "my-tool").
            - Construct the request body for `InstallOrAddService` mapping `serviceData.name` to `DisplayName`, `serviceData.environments` to `UserProvidedEnvVars` (parsing if necessary), and extracted `PackageManager` and `PackageName`.
            - `source_type` should be "marketplace" or a similar appropriate value if `install_or_add_service` requires it for this flow.
            - Ensure `serviceData.arguments` are handled appropriately.
        - **Success Criteria**: `stdio` custom services are created by calling the `install_or_add_service` endpoint with the correct payload. Other types (`sse`, `streamableHttp`) continue to use `/api/mcp_market/custom_service`.
    - **[ ] Task B1.2: Verify Backend `InstallOrAddService` for Custom Stdio** `testing` `backend`
        - **Description**: Ensure the existing `InstallOrAddService` handler in `backend/api/handler/market.go` correctly processes requests for custom stdio services as prepared by the updated frontend.
        - **Details**: Pay attention to how `PackageName`, `PackageManager`, `DisplayName`, and `UserProvidedEnvVars` are used. Confirm that `Command`, `ArgsJSON` are correctly set in the `MCPService` record.
        - **Success Criteria**: Backend successfully creates/installs custom `stdio` services sent via this route.

### Issue 2: Service Uninstall Failure (404)

- **Background**: Uninstalling services, particularly custom ones or those where `package_manager` might be "unknown" or `NULL` in DB, fails with a 404 because the backend `UninstallService` handler primarily relies on `package_name` and `package_manager` for lookup. The user reported this for ID=15 with `package_manager: "unknown"`. User suspects that for SSE/HTTP services installed via URL (which don't have an inherent package name), the `source_package_name` field in the database might be empty. If the frontend then sends the service's display name as `package_name` and `package_manager` as "unknown", the backend lookup `GetServicesByPackageDetails` (which queries on `source_package_name` and `package_manager`) will likely fail to find the service if `source_package_name` is indeed empty or different from the display name in the database. The most reliable identifier is the service ID.
- **Task Breakdown**:
    - **[ ] Task B2.1: Modify Frontend Uninstall to Send Only Service ID** `refactor` `frontend`
        - **Description**: In `frontend/src/store/marketStore.ts`, update the `uninstallService` action.
        - **Details**:
            - Modify the payload sent to `/api/mcp_market/uninstall` to include *only* the `service_id`.
            - For example: `{"service_id": serviceId}`.
        - **Success Criteria**: Frontend sends only `service_id` in the uninstall request body.
    - **[ ] Task B2.2: Update Backend Uninstall Logic to Use Only Service ID** `refactor` `backend`
        - **Description**: In `backend/api/handler/market.go`, modify the `UninstallService` handler.
        - **Details**:
            - Expect and use *only* the `service_id` from the request body to identify the service.
            - Fetch the `MCPService` directly using `model.GetServiceByID(serviceIDFromRequest)`.
            - Remove the fallback logic that uses `package_name` and `package_manager`.
            - Remove or update the `StatusNotImplemented` block for `config_service_id` as `service_id` directly addresses this.
            - Proceed with uninstallation steps (e.g., calling package-specific uninstallers if applicable, soft-deleting the record).
        - **Success Criteria**: Services are uninstalled using only their ID. The handler is simplified and more robust.
    - **[ ] Task B2.3: Ensure Uninstall Cleans Up Correctly** `testing` `backend`
        - **Description**: Verify that when a service is uninstalled (identified by ID or package details), all necessary cleanup occurs.
        - **Details**: Test with the new ID-based lookup.
        - **Success Criteria**: Service is correctly uninstalled and marked as deleted in the database.

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
   - 服务创建 (e.g., `createMcpGoServer`)：从 `HeadersJSON` 反序列化，填充到传递给 `mcp-go` 客户端的选项中。
   - `mcp-go` 客户端初始化：使用 `client.WithHeaders(config.Headers)` 等选项。


**3. SSE类型实现** (已整合入 `createMcpGoServer` 和 `createSSEHttpHandler`)
- 使用 `client.NewSSEMCPClient()` 连接外部SSE MCP服务器。
- 仿照现有的stdio->SSE模式，创建中间层服务器。
- 支持持久连接和事件流处理。

**4. HTTP类型实现** (后端连接部分已整合入 `createMcpGoServer`)
- 使用 `client.NewStreamableHttpClient` 连接外部HTTP MCP服务器。
- 实现请求响应模式的代理。
- 支持标准HTTP MCP协议。

**5. 统一接口设计** (Service 接口基本保持，HTTPService 概念通过返回 http.Handler 实现)
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

// 新增HTTP Handler接口 (通过返回标准 http.Handler 实现)
// type HTTPService interface {
// Service
// http.Handler  // 支持直接作为HTTP handler使用
// }
\`\`\`

### 数据流（更新版）

**SSE 输出代理**:
\`\`\`
Frontend -> /proxy/{serviceName}/sse -> ProxyHandler -> [createMcpGoServer -> mcp-go Client (Stdio/SSE/HTTP)] -> MCPServer -> createSSEHttpHandler -> SSEServer -> Client
\`\`\`

**HTTP 输出代理 (见 @feature-http-output-proxy-tasks.md)**:
\`\`\`
Frontend -> /proxy/{serviceName}/mcp -> HTTPProxyHandler -> [createMcpGoServer -> mcp-go Client (Stdio/SSE/HTTP)] -> MCPServer -> createHTTPProxyHttpHandler -> StreamableHTTPServer -> Client
\`\`\`

### 环境配置

SSE和HTTP类型的服务配置：
- **URL/Command**: (必需) 存储在 `mcp_services.command` 字段。
- **Headers**: (可选) 请求头，作为JSON对象字符串存储在新增的 `mcp_services.headers_json` 字段。例如 `{"Authorization": "Bearer token", "X-Custom": "value"}`。
- **其他连接参数 (如 API_KEY, TIMEOUT)**: 仍然可以通过 `default_envs_json` 存储，或者如果适合放入Header，也可以统一放入 `headers_json`。

### ServiceFactory更新 (已按实际实现调整)

\`\`\`go
func ServiceFactory(mcpDBService *model.MCPService) (Service, error) {
    // createMcpGoServer and createSSEHttpHandler (or future createHTTPProxyHttpHandler)
    // are now the primary ways to get handlers. ServiceFactory wraps these.
    // The actual handler (http.Handler) is obtained via functions like getOrCreateProxyToSSEHandler.
    // ServiceFactory then wraps this handler in a struct that implements the Service interface.
    
    switch mcpDBService.Type {
    case model.ServiceTypeStdio, model.ServiceTypeSSE, model.ServiceTypeStreamableHTTP:
        // For SSE output, it always goes through getOrCreateProxyToSSEHandler
        // which uses createMcpGoServer and createSSEHttpHandler internally.
        // It returns an http.Handler. This is then wrapped by NewSSESvc.
        // A similar pattern will apply for HTTP output proxy.
        
        // Simplified view:
        // 1. Get the http.Handler (e.g. from getOrCreateProxyToSSEHandler or getOrCreateProxyToHTTPHandler)
        // 2. Wrap it in a Service implementation (e.g. NewSSESvc, NewHTTPSvc)
        
        // Example for SSE output (current implementation)
        handler, err := getOrCreateProxyToSSEHandler(mcpDBService) // This is a simplified call, actual is in proxy_handler.go
        if err != nil {
            return nil, err
        }
        // The ServiceFactory in service.go correctly creates NewSSESvc or NewHTTPSvc
        // which embed the handler and implement the Service interface.
        // The key is that the underlying handler comes from the generalized `createMcpGoServer`
        // and appropriate output wrapper (e.g. `createSSEHttpHandler`).
        
        // Placeholder for actual logic which is more nuanced and involves caching
        // and specific handler creation functions in service.go
        if mcpDBService.Type == model.ServiceTypeStdio {
             return NewSSESvc(mcpDBService, nil), nil // Simplified, handler would be from getOrCreateProxyToSSEHandler
        } else if mcpDBService.Type == model.ServiceTypeSSE {
             return NewSSESvc(mcpDBService, nil), nil // Simplified
        } else if mcpDBService.Type == model.ServiceTypeStreamableHTTP {
             // If output is SSE:
             return NewSSESvc(mcpDBService, nil), nil // Simplified
             // If output is HTTP (future task):
             // return NewHTTPSvc(mcpDBService, nil), nil // Simplified
        }
        return nil, errors.New("ServiceFactory logic needs to be updated for this type in the documentation")

    default:
        return nil, errors.New("unsupported service type")
    }
}
\`\`\`
Note: The createXxxHandler functions (like `createSSEHttpHandler`) now primarily take an `*mcpserver.MCPServer` that was created by `createMcpGoServer`. The ServiceFactory's role is to orchestrate getting an appropriate `http.Handler` (via cached getters like `getOrCreateProxyToSSEHandler`) and wrapping it in a `Service` struct (e.g., `SSESvc` or `HTTPSvc`).

### mcp-go库集成点

需要研究和使用的mcp-go库功能：
1. **传输层**: Stdio, SSE, 和 HTTP 客户端创建, 关键是 `client.WithHeaders()` 等选项。 (Implemented in `createMcpGoServer`)
2. **服务端封装**: `server.NewMCPServer()`, `server.NewSSEServer()`, `server.NewStreamableHTTPServer()`.
3. **会话管理**: 多客户端连接管理 (handled by mcp-go server components).
4. **错误处理**: 连接重试和恢复 (partially mcp-go, partially our responsibility).
5. **协议支持**: 完整的MCP协议实现 (provided by mcp-go).

### Relevant Files

**核心文件**:
- `backend/library/proxy/service.go` - Service接口, ServiceFactory, `createMcpGoServer`, `createSSEHttpHandler`, handler caching.
- `backend/api/handler/proxy_handler.go` - `ProxyHandler`, `HTTPProxyHandler` (Gin handlers for /proxy/.../sse and /proxy/.../mcp).
- `backend/api/route/api-router.go` - 路由配置.
- `backend/model/mcp_service.go` - 服务类型定义 (contains `HeadersJSON`).
- `backend/api/handler/market.go` - `InstallOrAddService`, `CreateCustomService` (handles `HeadersJSON`).

**主要修改文件 (Refactored)**:
- `backend/library/proxy/sse_native_service.go` - (Obsolete, logic moved to `service.go`)
- `backend/library/proxy/http_service.go` - (Obsolete, logic moved to `service.go`)

**前端相关**:
- `frontend/src/components/market/ServiceConfigModal.tsx` - 前端配置界面 (未来可能需要更新以支持 Headers 输入)

## Lessons

- **库能力调研的重要性**: 深入了解第三方库的完整能力可以显著简化实现
- **mcp-go传输层**: 该库已提供stdio、SSE、HTTP三种传输层，无需重复造轮子
- **配置清晰度**: 为特定用途（如Headers）设置专用字段优于复用通用字段。
- **中心化逻辑**: 将核心的、重复的服务创建逻辑（如 `createMcpGoServer`）中心化，可以极大提高代码的可维护性和一致性。

## ACT mode Feedback or Assistance Requests

暂无

## User Specified Lessons

暂无