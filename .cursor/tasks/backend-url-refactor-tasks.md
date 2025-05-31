# Backend: Adapt Proxy Endpoints to New URL Structure & Fixes

This set of tasks focuses on refactoring the backend proxy endpoints to use service names in the URL, ensuring services have unique names, and fixing related bugs like the "Service is not of type SSE" error.

## Completed Tasks

- [x] Define `SSEProxyHandler` in `backend/api/handler/proxy_handler.go` to handle `GET /api/sse/:serviceName`. `bug-fix`
- [x] Define `HTTPProxyHandler` in `backend/api/handler/proxy_handler.go` to handle `/api/http/:serviceName`. `bug-fix`
- [x] Register new routes in `backend/api/route/api-router.go`. `bug-fix`
- [x] Update `ServiceFactory` in `backend/library/proxy/service.go` to use `getOrCreateActualSSEHandler` for `model.ServiceTypeSSE`. `ref-func`
- [x] Implement `getOrCreateActualSSEHandler` in `backend/library/proxy/service.go` to instantiate and cache actual `mcp-go` SSE handlers (e.g., for "exa-mcp-server"). `new-feat`
- [x] Pass Go unit tests for `proxy_handler_test.go` after changes. `test`
- [x] **Investigate & Fix "Service is not of type SSE" for `exa-mcp-server`** `bug-fix`
    - [x] Determine how "exa-mcp-server" is created/configured (assumed via seed or manual DB entry).
    - [x] Ensure its `Type` in the database is correctly set (e.g. to `model.ServiceTypeStdio` if it's fundamentally stdio).
- [x] **Enable SSE proxying for underlying Stdio services (e.g., `exa-mcp-server`)** `ref-arch`
    - [x] **Database Record for `exa-mcp-server`**:
        - [x] Ensure `SeedDefaultServices()` in `backend/model/mcp_service.go` creates/updates "exa-mcp-server" with `Type: model.ServiceTypeStdio`.
        - [x] The service record must also store its `stdio` configuration (command, args, env). This might involve defining how `DefaultAdminConfigValues` or a similar field stores this for `stdio` types. For `exa-mcp-server`, this could be hardcoded in the seeder for now, e.g., Command: "mcp-hello-world", Args: [], Env: [].
        - [x] Add `StdioConfig` struct to `backend/model/mcp_service.go`.
        - [x] Call `SeedDefaultServices()` from `main.go` after DB init.
    - [x] **Modify `SSEProxyHandler` (`backend/api/handler/proxy_handler.go`)**:
        - [x] Remove the `if service.Type != model.ServiceTypeSSE` check.
        - [x] Modify handler to extract `action` from `GET /api/sse/:serviceName/*action` and set `c.Request.URL.Path = action` for the wrapped handler.
    - [x] **Refactor `ServiceFactory` (`backend/library/proxy/service.go`)**:
        - [x] If `MCPService.Type` is `Stdio`:
            - [x] Instantiate an `mcp-go` `StdioMCPClient` using config from `MCPService.DefaultAdminConfigValues` (parsed into `StdioConfig`).
            - [x] Initialize this client.
            - [x] Create an `mcp-go` `MCPServer` instance.
            - [x] Copy tools, prompts, resources from the `StdioMCPClient` to the `MCPServer` instance (as in user's example code).
            - [x] Wrap this `MCPServer` with `mcpserver.NewSSEServer(...)` to get an `http.Handler`.
            - [x] Cache and reuse this handler for subsequent requests to the same service name.
            - [x] The `Service` returned by `ServiceFactory` should be an `SSESvc` wrapping this handler, and its `BaseService.serviceType` should reflect `model.ServiceTypeSSE` (as it's being served via SSE).
    - [x] **Routing (`backend/api/route/api-router.go`)**: Update SSE route to `/api/sse/:serviceName/*action`.
    - [x] **Configuration for `mcp-go` BaseURL**: Ensure the `BaseURL` for the `mcp-go SSEServer` is correctly constructed (e.g., `http://one-mcp-host:port/api/sse/exa-mcp-server`).
    - [x] Add `github.com/mark3labs/mcp-go` dependency.

## Pending Tasks

- [ ] **HTTP Proxy Implementation (`HTTPProxyHandler` & `ServiceFactory` for `StreamableHTTP`)** `new-feat`
    - [ ] Implement actual HTTP reverse proxy logic in `HTTPProxyHandler`.
    - [ ] `ServiceFactory` needs to create a suitable `Service` for `model.ServiceTypeStreamableHTTP`.
- [ ] **Direct SSE Proxy (`ServiceFactory` for `ServiceTypeSSE`)** `new-feat`
    - [ ] Implement proxying for services that are already native MCP SSE services (using `mcpclient.NewSSEMCPClient` and an `mcpserver.MCPServer` that delegates calls).
    - [ ] Or, for generic non-MCP SSE services, a simpler reverse proxy might be needed.
- [ ] **Robust Lifecycle Management for `mcp-go` clients/servers** `enhancement`
    - [ ] Ensure `mcpGoClient.Close()` is called when a service is stopped/deleted or OneMCP shuts down.
    - [ ] Manage the lifecycle of the `mcpGoServer` instances created in `getOrCreateStdioToSSEHandler`.
- [ ] **Unit/Integration Tests for Stdio-to-SSE proxying** `test`
    - [ ] Test successful SSE connection and message exchange through the proxy.
    - [ ] Test behavior when the underlying stdio command fails.
    - [ ] Test service caching in `getOrCreateStdioToSSEHandler`.
- [ ] Review and refine `oneMCPExternalBaseURL` construction (currently uses env vars, consider central config).

## Future Considerations / Nice-to-haves

- [ ] Client-specific configurations for proxied services (beyond `DefaultAdminConfigValues`).
- [ ] UI to manage/view proxied `stdio` command output/errors for debugging.