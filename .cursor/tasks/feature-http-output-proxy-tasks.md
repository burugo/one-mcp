# HTTP Output Proxy for MCP Services

## Background and Motivation

Currently, `one-mcp` can proxy various backend MCP service types (Stdio, SSE, HTTP) and expose them over an SSE/MCP connection. This task outlines the plan to also allow exposing these backend services over a standard HTTP/MCP connection. This would provide clients with an alternative way to interact with proxied services, catering to scenarios where a direct HTTP/MCP communication is preferred over SSE.

The frontend already anticipates an HTTP/MCP endpoint at `/proxy/${service?.name || ''}/mcp`. This plan aims to make that endpoint functional.

## Key Challenges and Analysis

1.  **`mcpserver.NewStreamableHTTPServer`**: Understanding and correctly using this function from the `mcp-go` library is crucial. This includes its signature, required parameters, and any specific options for configuration (e.g., base path, context handling).
2.  **Handler Caching**: Similar to SSE proxy handlers, a caching mechanism will be needed for HTTP proxy handlers to avoid re-creating them on every request for the same service.
3.  **Routing Integration**: The new HTTP proxy handlers need to be correctly integrated into the existing routing mechanism, likely via `api-router.go` and the `proxy_handler.go`.
4.  **Client Lifecycle**: Ensuring that the underlying `mcpclient.MCPClient` (for the backend service) is properly managed (started, initialized, and closed) when proxied via HTTP. The `createMcpGoServer` function already handles much of this, but its integration with the new HTTP handler path needs to be seamless.

## High-level Task Breakdown

1.  **Research**: Thoroughly examine `mcpserver.NewStreamableHTTPServer` in the `mcp-go` library.
2.  **Implementation**:
    *   Create a new function `createHTTPProxyHttpHandler` to wrap an `mcpserver.MCPServer` with `mcpserver.NewStreamableHTTPServer`.
    *   Create a new getter function `getOrCreateProxyToHTTPHandler` that uses `createMcpGoServer` (already generalized) and `createHTTPProxyHttpHandler`, including caching.
3.  **Integration**: Update `proxy_handler.go` (specifically the `HTTPProxyHandler` function or similar that serves `/proxy/:serviceName/mcp/*`) to use `getOrCreateProxyToHTTPHandler` to serve requests.
4.  **Testing**: Thoroughly test the HTTP proxy with Stdio, SSE, and HTTP backend services.

## Implementation Plan

### 1. Define `createHTTPProxyHttpHandler`

*   **Signature**: `func createHTTPProxyHttpHandler(mcpGoServer *mcpserver.MCPServer, mcpDBService *model.MCPService) (http.Handler, error)`
*   **Logic**:
    *   Takes a fully initialized `mcpserver.MCPServer` (obtained from `createMcpGoServer`).
    *   Uses `mcpserver.NewStreamableHTTPServer(mcpGoServer, ...options)` to create an `http.Handler`.
    *   Determine necessary options for `NewStreamableHTTPServer` (e.g., endpoint paths, context functions, logger) by consulting `mcp-go` documentation or source.
    *   Return the created `http.Handler`.

### 2. Define `getOrCreateProxyToHTTPHandler`

*   **Signature**: `func getOrCreateProxyToHTTPHandler(mcpDBService *model.MCPService) (http.Handler, error)`
*   **Logic**:
    *   Manages a cache for HTTP proxy handlers (e.g., `initializedHTTPProxyWrappers map[string]http.Handler` with a corresponding mutex).
    *   **Cache Lookup**: Checks if a handler for the given `mcpDBService.ID` already exists. If so, returns it.
    *   **Handler Creation (if not cached)**:
        1.  Calls `createMcpGoServer(ctx, mcpDBService, "global_http_proxy")` to obtain the `*mcpserver.MCPServer` and `mcpclient.MCPClient`.
        2.  Calls `createHTTPProxyHttpHandler(mcpGoSrv, mcpDBService)` to get the HTTP `http.Handler`.
        3.  Stores the new handler in the cache.
        4.  Handles potential race conditions during cache write, similar to `getOrCreateProxyToSSEHandler`.
        5.  Manages cleanup of the `mcpClient` if handler creation fails or if an instance is orphaned due to a race.
    *   Returns the `http.Handler`.

### 3. Integrate with Routing (e.g., in `backend/api/handler/proxy_handler.go`)

*   The existing router in `api-router.go` likely routes `/proxy/:serviceName/mcp/*` to a function in `proxy_handler.go`, let's assume it's `ServeMCPOverHTTP`.
*   **`ServeMCPOverHTTP(c *gin.Context)` function**:
    1.  Extract `serviceName` from `c.Param("serviceName")`.
    2.  Retrieve the `*model.MCPService` from the database using the `serviceName`. Handle "not found" errors.
    3.  Call `getOrCreateProxyToHTTPHandler(mcpDBService)` to obtain the `http.Handler` specific to this service. Handle errors.
    4.  Use the obtained `http.Handler` to serve the current request: `specificHttpHandler.ServeHTTP(c.Writer, c.Request)`.

### Relevant Files (Potential)

*   `backend/library/proxy/service.go` (for new handler creation and getter functions)
*   `backend/api/handler/proxy_handler.go` (for the main Gin handler that uses the getter)
*   `backend/api/route/api-router.go` (to ensure routing to the Gin handler)
*   `backend/model/mcp_service.go` (if any changes are needed, e.g., a field to specify preferred proxy output, though current plan avoids this).

## Future Tasks (Post-Implementation)

*   Refine error handling and logging specific to the HTTP proxy.
*   Consider if any specific metrics or health checks are needed for HTTP proxied services beyond what `createMcpGoServer` already provides.
*   Update any relevant documentation.

## Notes on `mcp-go` API usage

*   The exact options for `mcpserver.NewStreamableHTTPServer` need to be verified from `mcp-go` documentation/source code. This might include:
    *   `mcpserver.WithHTTPContextFunc`
    *   `mcpserver.WithEndpointPath` (though this might be implicitly handled if the server is mounted at `/proxy/:serviceName/mcp/`)
    *   `mcpserver.WithLogger`
*   The `StreamableHTTPServer` from `mcp-go` likely handles the MCP framing over standard HTTP POST requests.

This plan defers the actual coding but captures the necessary steps for when this feature is prioritized. 