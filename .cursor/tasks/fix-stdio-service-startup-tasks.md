# Fix Stdio Service Startup and Deadlock

This plan addresses the deadlock caused by `Stdio` type services blocking during initialization within `ServiceManager.RegisterService`. It involves correcting command configurations and making the startup process in `getOrCreateStdioToSSEHandler` more robust with timeouts and better error handling.

## Future Tasks

- [ ] **1. Correct `exa-mcp-server` Configuration & Command Execution** `bug-fix` `config`
    - **1.1.** Analyze `exa-mcp-server`: Determine if `exa-mcp-server` is an npm package requiring `npx` or a globally installed executable.
        - If it requires `npx`:
            - **1.1.1.** Modify `SeedDefaultServices` in `backend/model/mcp_service.go`:
                - Change `StdioConfig.Command` for `exa-mcp-server` to `"npx"`.
                - Add `"exa-mcp-server"` (and any necessary flags like `-y`) to `StdioConfig.Args`.
                - Consider if `PackageManager` should be updated to `"npm"` for clarity, though the direct `npx` in command might be sufficient.
            - **1.1.2.** Alternatively, introduce a new field in `StdioConfig` or `MCPService` (e.g., `UseNpx bool` or `ExecutionWrapper string`) to make this more generic if other services need similar wrapping (e.g., `python -m`). For now, direct modification for `npx` might be simpler if `exa-mcp-server` is the primary concern.
        - If it's meant to be a globally available command:
            - **1.1.3.** Ensure documentation/setup instructions clearly state this prerequisite. The current code path will remain the same.
    - **1.2.** Verify `StdioConfig.Env` for `exa-mcp-server` in `SeedDefaultServices`. Add any required environment variables as identified by the user.
    - Success Criteria: `exa-mcp-server` can be invoked correctly by the `mcpclient.NewStdioMCPClient` call. The method of invocation (direct or via `npx`) is correctly configured.

- [ ] **2. Implement Robust Startup in `getOrCreateStdioToSSEHandler`** `ref-arch` `bug-fix`
    - **2.1.** Modify `getOrCreateStdioToSSEHandler` in `backend/library/proxy/service.go`.
    - **2.2.** Introduce a timeout for `mcpclient.NewStdioMCPClient()`: This is tricky as `NewStdioMCPClient` itself starts the process. The `mcp-go` library might need modification to accept a context for process startup, or we might need to wrap the command execution. A simpler initial approach might be to focus the timeout on the `Initialize` call.
    - **2.3.** Introduce a configurable timeout for the `mcpGoClient.Initialize(ctx, initRequest)` call.
        - Define a reasonable default timeout (e.g., 30-60 seconds).
        - Consider making this timeout configurable, perhaps via `MCPService.DefaultAdminConfigValues` or a global setting.
        - Use `context.WithTimeout` for this call.
    - **2.4.** Enhance error handling:
        - If `NewStdioMCPClient` returns an error (e.g., command not found from OS), ensure this error is propagated correctly and `getOrCreateStdioToSSEHandler` returns an error.
        - If `Initialize` times out or returns an error, ensure `mcpGoClient.Close()` is called to clean up resources (like the potentially running but unresponsive child process).
        - Propagate these errors so `ServiceFactory` and subsequently `ServiceManager.RegisterService` can log them and potentially mark the service as failed instead of blocking.
    - Success Criteria: `getOrCreateStdioToSSEHandler` does not block indefinitely. Failures in starting or initializing the stdio process result in timely errors and resource cleanup. The `ServiceManager` does not deadlock due to a stuck stdio service.

- [ ] **3. Testing** `test`
    - **3.1.** Test with a correctly configured `exa-mcp-server` (ensure it starts and initializes).
    - **3.2.** Test with a misconfigured `exa-mcp-server` command (e.g., command not found) to ensure it fails fast and doesn't deadlock.
    - **3.3.** Test with a command that starts but hangs during initialization (e.g., a script that starts but doesn't do the MCP handshake) to ensure timeouts trigger and prevent deadlock.
    - **3.4.** Test that other parts of the application remain responsive if one service fails to start.
    - Success Criteria: Deadlock is resolved. Service startup failures are handled gracefully.

## Implementation Plan

### Phase 1: `exa-mcp-server` Command Correction (Tasks 1.1, 1.2)
1.  **Investigation:** Determine the correct way to run `exa-mcp-server` (direct, or `npx exa-mcp-server`, or `npx -y exa-mcp-server`).
2.  **Code Change (if npx needed):** Modify `SeedDefaultServices` in `backend/model/mcp_service.go` to set `StdioConfig.Command = "npx"` and `StdioConfig.Args = ["-y", "exa-mcp-server", <any_other_args_for_exa-mcp-server>]`. Also populate `StdioConfig.Env` as per user's requirement.

### Phase 2: Robust Startup Logic (Task 2)
1.  **Modify `getOrCreateStdioToSSEHandler` in `backend/library/proxy/service.go`:**
    *   Wrap the `mcpGoClient.Initialize` call with a `context.WithTimeout`.
        ```go
        // Example:
        initCtx, cancelInit := context.WithTimeout(context.Background(), 60*time.Second) // 60s timeout
        defer cancelInit()
        _, err = mcpGoClient.Initialize(initCtx, initRequest)
        if err != nil {
            mcpGoClient.Close() // Ensure cleanup
            // Propagate error
            return nil, fmt.Errorf("failed to initialize mcp-go client for %s (timeout or error): %w", mcpDBService.Name, err)
        }
        ```
    *   Review `mcpclient.NewStdioMCPClient`: If this call itself can block indefinitely on command not found (less likely, usually OS returns error fast), this area might need more advanced handling (e.g., running the command in a goroutine with a select and a timeout, which is more complex). For now, assume OS errors for command execution are relatively quick. The primary blocking concern is the `Initialize` step.
    *   Ensure `mcpGoClient.Close()` is also called if `addClientToolsToMCPServer` or `addClientPromptsToMCPServer` fail after a successful Initialize.

### Phase 3: Testing (Task 3)
1.  Set up test cases as described in Task 3.1-3.4.
2.  Run the application and trigger service initialization, observing logs and behavior.
3.  Use debugger if necessary to confirm non-blocking behavior on failure.

## Relevant Files

- `backend/model/mcp_service.go` (for `SeedDefaultServices` and `StdioConfig`)
- `backend/library/proxy/service.go` (for `getOrCreateStdioToSSEHandler`, `ServiceFactory`)
- `backend/library/proxy/manager.go` (to observe deadlock resolution, no direct changes planned here)
- `mcp-go/client/stdio.go` (from `github.com/mark3labs/mcp-go/client`) - for understanding `NewStdioMCPClient` and `Initialize` behavior, though direct modification is out of scope unless strictly necessary and alternatives are exhausted. 