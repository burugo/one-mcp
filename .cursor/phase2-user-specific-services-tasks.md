# Phase 2: User-Specific Service Configurations

**Goal:** Allow users to have their own configurations (primarily environment variables) for services, loaded dynamically based on their authenticated request.

**Core Changes:**
*   The SSE request handling layer will identify users and load their specific configurations.
*   A mechanism to create and cache service handlers (wrapping `mcpclient` instances) that are specific to a `user + service` combination.
*   The existing global/default service handlers will still be used for non-authenticated requests or when a user has no specific configuration.

---

**Detailed Tasks:**

**Task 2.1: Refine Handler Caching and Creation Logic**
   *   **File:** `backend/library/proxy/service.go`
   *   **A. Modify `wrapMap` Keying:**
        *   The global `wrapMap` will store `http.Handler` instances.
        *   Keys will be strings to differentiate global vs. user-specific handlers.
            *   Global: `fmt.Sprintf("global-service-%d", serviceID)`
            *   User-specific: `fmt.Sprintf("user-%d-service-%d", userID, serviceID)`
   *   **B. Adapt `getOrCreateStdioToSSEHandler` for Global Instances:**
        *   This function (as modified in Phase 1) will now *exclusively* create and cache the **global/default** handler for a given `model.MCPService`.
        *   It will use the `global-service-%d` key format for `wrapMap`.
        *   It continues to use `mcpDBService.DefaultEnvsJSON` for environment variables.
        *   The signature remains `getOrCreateStdioToSSEHandler(mcpDBService *model.MCPService) (http.Handler, error)`.
   *   **C. Create New Function: `newStdioSSEHandlerUncached` (or similar)**
        *   Signature: `newStdioSSEHandlerUncached(ctx context.Context, mcpDBService *model.MCPService, effectiveStdioConfig model.StdioConfig) (http.Handler, error)`
        *   This function will contain the core logic for instantiating an `mcpclient.NewStdioMCPClient` using the provided `effectiveStdioConfig`, initializing it, wrapping it in `mcpserver.NewMCPServer`, and then `mcpserver.NewSSEServer`.
        *   It will *not* perform any caching.
        *   It needs a `context.Context`.

**Task 2.2: Implement User-Specific Logic in SSE Request Router**
   *   **File:** `backend/api/sse.go` (or similar SSE request routing file).
   *   **A. Service Lookup:** Resolve `serviceName` from URL to `*model.MCPService`.
   *   **B. User Identification:** Use `getUserIDFromRequest(r *http.Request)` (from Task 2.3).
   *   **C. Handler Dispatch Logic:**
        *   **If `userID > 0` AND `mcpDBService.AllowUserOverride`:**
            *   **1. Generate User-Specific Cache Key.**
            *   **2. Check Cache:** `proxy.GetCachedHandler(userHandlerKey)` (implies helper for `wrapMap`).
            *   **3. If Not in Cache:**
                *   **a. Prepare `effectiveStdioConfig`:** Base from `DefaultAdminConfigValues` & `DefaultEnvsJSON`, then fetch and merge user-specific ENVs from `model.GetUserSpecificEnvs(userID, mcpDBService.ID)` (Task 2.4). User values override.
                *   **b. Create Handler:** Call `proxy.newStdioSSEHandlerUncached(r.Context(), mcpDBService, effectiveStdioConfig)`.
                *   **c. Cache Handler:** `proxy.CacheHandler(userHandlerKey, newHandler)`.
            *   **4. Serve Request:** Use user-specific handler. Handle errors.
        *   **Else (No specific user / no override):**
            *   **1. Get Global Handler:** Call `proxy.ServiceFactory(mcpDBService)` (uses `getOrCreateStdioToSSEHandler`).
            *   **2. Serve Request:** Use global handler. Handle errors.
   *   **D. Error Handling:** Robust error handling throughout.

**Task 2.3: Authentication/User Identification Utility**
   *   **File:** New or existing auth utility (e.g., `backend/middleware/auth.go`).
   *   **A. Implement `getUserIDFromRequest(r *http.Request) (userID int64, err error)`:**
        *   Parse `Authorization: Bearer <token>`. Validate token, extract `userID`.

**Task 2.4: Model Layer for User-Specific Configurations**
   *   **File(s):** `backend/model/user_config.go`, `backend/model/config_service.go`.
   *   **A. Verify/Implement Database Models:** `UserConfig`, `ConfigService`.
   *   **B. Implement `model.GetUserSpecificEnvs(userID int64, serviceID int64) (map[string]string, error)`:**
        *   Query DB for `user_configs` and join with `config_services` to get `key_name:value` map.

**Task 2.5: Lifecycle Management for User-Specific Handlers (Advanced/Optional for First Pass)**
    *   Consider LRU cache or timeout eviction for user-specific handlers in `wrapMap`.
    *   Ensure `mcpGoClient.Close()` is called on eviction.

--- 