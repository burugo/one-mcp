# MCPService Model and StdioConfig Refactor

Refactor the `MCPService` model to remove deprecated configuration fields and update related logic to use new fields for Stdio service configuration.

## Completed Tasks

## In Progress Tasks

- [ ] Task 1: Remove deprecated fields from `MCPService` model and update seeder. `ref-struct`
- [ ] Task 2: Modify `proxy/service.go` to use new fields for `StdioConfig`. `ref-func`
- [ ] Task 3: Update `handler/mcp_service.go` API for `PackageManager` logic. `ref-func`
- [ ] Task 4: Update tests in `handler/proxy_handler_test.go`. `test`
- [ ] Task 5: Update `handler/proxy_handler.go` to reflect new `StdioConfig` sourcing. `ref-func`


## Future Tasks

## Implementation Plan

Detailed steps for each task:

### Task 1: Remove deprecated fields from `MCPService` model and update seeder
    - **File**: `backend/model/mcp_service.go`
    - **Action**:
        - Remove the fields: `AdminConfigSchema`, `DefaultAdminConfigValues`, `UserConfigSchema`.
        - Review and update the (currently commented out) `SeedDefaultServices` function, particularly for `exa-mcp-server`.
            - Instead of `DefaultAdminConfigValues`, it should populate `Command` (e.g., "npx" if appropriate, or direct command), `ArgsJSON` (e.g., `"[\"-y\", \"exa-mcp-server\"]"`), and `DefaultEnvsJSON`.
            - Remove any logic related to `AdminConfigSchema` or `UserConfigSchema` from the seeder.
    - **Success Criteria**: Fields are removed from the struct, and the seeder logic is updated to use the new fields.

### Task 2: Modify `proxy/service.go` to use new fields for `StdioConfig`
    - **File**: `backend/library/proxy/service.go`
    - **Functions**: `getOrCreateStdioToSSEHandler` and `defaultNewStdioSSEHandlerUncached`
    - **Action**:
        - **Populate `stdioConf.Command`**:
            - Set `stdioConf.Command = mcpDBService.Command`.
        - **Populate `stdioConf.Args`**:
            - If `mcpDBService.ArgsJSON` is not empty, unmarshal it (it's a JSON string array) into `stdioConf.Args`.
            - Handle potential unmarshalling errors. If empty or error, `stdioConf.Args` can be an empty slice.
        - **Populate `stdioConf.Env` (largely existing logic, verify integration):**
            - The existing logic for unmarshalling `mcpDBService.DefaultEnvsJSON` (map[string]string) and appending to `stdioConf.Env` (as `[]string{"KEY=VALUE"}`) should be maintained and correctly integrated with the new Command/Args sourcing.
            - Ensure `stdioConf.Env` is initialized as `[]string{}` if `DefaultEnvsJSON` is empty or missing.
        - Remove all code that reads from `mcpDBService.DefaultAdminConfigValues`.
        - Address the TODO comment about configurable timeout: remove the reference to `DefaultAdminConfigValues`. If a configurable timeout is still desired, it needs a new configuration path (out of scope for this immediate refactor unless specified).
    - **Success Criteria**: `StdioConfig` is correctly populated using `Command`, `ArgsJSON`, and `DefaultEnvsJSON`. Logic relying on `DefaultAdminConfigValues` is removed.

### Task 3: Update `handler/mcp_service.go` API for `PackageManager` logic
    - **File**: `backend/api/handler/mcp_service.go`
    - **Functions**: `CreateMCPService` and `UpdateMCPService`
    - **Action**:
        - **Before saving the `service` entity (i.e., before `model.CreateService(&service)` or `model.UpdateService(service)`):**
            - Add logic:
                - If `service.PackageManager == "npm"`:
                    - `service.Command = "npx"`
                    - If `service.ArgsJSON` is empty and `service.SourcePackageName` is not empty, then set `service.ArgsJSON = fmt.Sprintf("[\"-y\", \"%s\"]", service.SourcePackageName)`.
                - Else if `service.PackageManager == "pypi"`:
                    - `service.Command = "uvx"` // Assuming uvx is the intended command for pypi execution
                    - If `service.ArgsJSON` is empty and `service.SourcePackageName` is not empty, then set `service.ArgsJSON = fmt.Sprintf("[\"-y\", \"%s\"]", service.SourcePackageName)`. // User indicates same arg format for pypi with uvx, though pypi type is not yet fully supported elsewhere.
            - This logic ensures that for specified package managers, the command is set appropriately, and for both npm and pypi (using uvx), ArgsJSON can be defaulted if not provided, using the source package name.
    - **Success Criteria**: `MCPService.Command` is correctly set based on `PackageManager`. For npm and pypi (using uvx), `ArgsJSON` is defaulted from `SourcePackageName` if not provided. Service is saved with these potentially modified fields.

### Task 4: Update tests in `handler/proxy_handler_test.go`
    - **File**: `backend/api/handler/proxy_handler_test.go`
    - **Action**:
        - Review all test cases that set up `model.MCPService` instances.
        - Remove usages of `DefaultAdminConfigValues`.
        - Modify test setups to provide `Command`, `ArgsJSON`, and `DefaultEnvsJSON` directly on the `model.MCPService` instance to achieve the desired `StdioConfig` for test scenarios.
        - For `ArgsJSON`, provide a valid JSON string representing a string array (e.g., `"[\"arg1\", \"arg2\"]"`).
        - For `DefaultEnvsJSON`, provide a valid JSON string representing a map (e.g., `"{\"KEY1\":\"VALUE1\"}"`).
    - **Success Criteria**: All tests pass with the new model structure and configuration sourcing.

### Task 5: Update `handler/proxy_handler.go` to reflect new `StdioConfig` sourcing.
    - **File**: `backend/api/handler/proxy_handler.go`
    - **Function**: `SSEProxyHandler` (and its helper, `getOrCreateEffectiveStdioSSEHandler`)
    - **Action**:
        - The logic for preparing `baseStdioConf` (which was read from `DefaultAdminConfigValues`) needs to be updated.
        - The `baseStdioConf` should now be constructed using `mcpDBService.Command` and `mcpDBService.ArgsJSON`.
        ```go
        // Old way:
        // var baseStdioConf model.StdioConfig
        // if mcpDBService.DefaultAdminConfigValues != "" {
        //     if err := json.Unmarshal([]byte(mcpDBService.DefaultAdminConfigValues), &baseStdioConf); err != nil { ... }
        // }

        // New way (conceptual):
        var baseStdioConf model.StdioConfig
        baseStdioConf.Command = mcpDBService.Command
        if mcpDBService.ArgsJSON != "" {
            var args []string
            if err := json.Unmarshal([]byte(mcpDBService.ArgsJSON), &args); err == nil {
                baseStdioConf.Args = args
            } else {
                // Log error, args will be empty
                common.SysError(fmt.Sprintf("Failed to unmarshal ArgsJSON for service %s: %v", mcpDBService.Name, err))
            }
        }
        // DefaultEnvsJSON is already handled for the effectiveStdioConfig later,
        // but if baseStdioConf.Env was intended to be populated here from DefaultEnvsJSON,
        // that logic would also need to be added/adjusted.
        // However, existing code in proxy_handler.go merges DefaultEnvsJSON *after* this block,
        // so baseStdioConf only needed Command and Args from DefaultAdminConfigValues.
        ```
        - The primary place `DefaultAdminConfigValues` was used was to establish a base `StdioConfig`. The `Command` and `Args` from this base config were then used. The `Env` part from `DefaultEnvsJSON` and user-specific ENVs were merged later.
        - So, the change is to populate `baseStdioConf.Command` from `mcpDBService.Command` and `baseStdioConf.Args` from `mcpDBService.ArgsJSON`.
    - **Success Criteria**: `baseStdioConf` in `SSEProxyHandler` is correctly initialized using the new fields from `mcpDBService`.

### Relevant Files

- `backend/model/mcp_service.go`
- `backend/library/proxy/service.go`
- `backend/api/handler/mcp_service.go`
- `backend/api/handler/proxy_handler_test.go`
- `backend/api/handler/proxy_handler.go` 