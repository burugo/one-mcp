# Feature: Backend Support for MCP Server Installation (npm, PyPI/uvx)

This feature aims to provide robust backend support for installing MCP servers sourced from package managers like npm (already partially supported) and PyPI (via uvx or pip). This involves API endpoints, service logic for command execution, and database updates.

## Completed Tasks

- [x] **Task 1: Enhance `InstallationManager` & Command Execution for PyPI/uvx** `new-feat`
    - **Description**: Modify/extend `backend/library/market/installation.go` and related files to support PyPI package installation using `uvx` (preferred) or `pip`.
    - **Sub-tasks**:
        - [x] Research and decide on the command structure for `uvx/pip install` (e.g., virtual environments, global install if appropriate for `stdio` services).
        - [x] Create a new function similar to `InstallNPMPackage` (e.g., `InstallPyPIPackage`) in a new file like `backend/library/market/pypi.go`.
            - This function should execute the `uvx/pip install` command securely.
            - It should capture output and errors.
            - It should attempt to initialize with the installed Python package using `mcp-go` client if the Python package is an MCP server, similar to how `InstallNPMPackage` does. If not an MCP server, it might just confirm installation.
        - [x] Modify `InstallationManager.runInstallationTask` in `backend/library/market/installation.go` to call `InstallPyPIPackage` based on `task.PackageManager`.
        - [x] Add `CheckUVXAvailable` (or similar) function.
    - **Success Criteria**: `InstallationManager` can successfully trigger and report status for `uvx/pip` installations. `mcp-go` client can interface with Python-based MCP servers if applicable.

- [x] **Task 2: Refine `InstallOrAddService` API Handler** `ref-func`
    - **Description**: Update the existing `InstallOrAddService` API handler in `backend/api/handler/market.go` to robustly handle different `package_manager` types (npm, pypi) passed from the frontend.
    - **Sub-tasks**:
        - [x] Ensure the handler correctly extracts `package_manager` from the request.
        - [x] Ensure it calls `CheckNPXAvailable` or `CheckUVXAvailable` based on the package manager.
        - [x] Ensure it correctly populates the `InstallationTask` with the `PackageManager` field.
    - **Success Criteria**: API can accept requests for both npm and pypi, and trigger the correct installation path in `InstallationManager`.

- [x] **Task 4: Frontend Integration Review & API Contracts** `ref-func`
    - **Description**: Review the frontend installation flow (from `.cursor/feature-mcp-installer-ui.md`) and ensure the backend API (`/install_or_add_service`, `/installation_status`) request/response contracts align with frontend needs.
    - **Sub-tasks**:
        - [x] Document the exact request body for `/install_or_add_service` (e.g., `packageName`, `packageManager`, `version`, `envVars`).
        - [x] Document the response format for `/install_or_add_service` (e.g., task ID, initial status).
        - [x] Document the response format for `/installation_status` (from existing code, ensure it's sufficient).
    - **Success Criteria**: Clear API contracts are documented and confirmed to meet frontend requirements.

- [x] **Task 3: Implement/Complete Post-Installation Database Logic** `new-feat`
    - **Description**: Ensure that after a successful installation (for both npm and PyPI packages), the necessary `MCPService` and `ConfigService` (user-specific instance) records are correctly created or updated in the database.
    - **Sub-tasks**:
        - [x] Review and complete the `addServiceInstanceForUser` function in `backend/api/handler/market.go` or implement similar logic within `InstallationManager.updateServiceStatus`.
        - [x] This logic should:
            - If a global `MCPService` for the installed package doesn't exist, create one. Populate fields like `Name`, `DisplayName` (from package info), `Type` (`stdio`), `PackageManager`, `SourcePackageName`, `InstalledVersion`. `ClientConfigTemplates` might need a default for `stdio`.
            - Create a `ConfigService` record for the current user, linking to the `MCPService` ID. Store any user-provided environment variables in `UserOverrideConfigValues`.
    - **Success Criteria**: Database accurately reflects installed services and user configurations post-installation.

## In Progress Tasks

- [ ] **Task 5: Testing** `new-feat`
    - **Sub-tasks**:
        - [ ] Write unit tests for `InstallPyPIPackage` and any new logic in `InstallationManager`.
        - [ ] Write/update unit tests for the `InstallOrAddService` API handler.
        - [ ] Write unit tests for the database update logic.
        - [ ] Consider integration tests (possibly mocked) for the end-to-end installation flow for both npm and PyPI.
    - **Success Criteria**: Comprehensive tests pass, ensuring reliability.

## Future Tasks

- [ ] Explore providing richer feedback (streaming logs) from the installation process to the frontend, beyond polling status. `new-feat`
- [ ] Investigate workspace/directory strategy for installations (e.g., per-user, shared). `research`

## Implementation Plan

1.  **PyPI/uvx Support**: Start with Task 1 to add the core capability for PyPI installations.
2.  **API Handler Refinement**: Proceed to Task 2 to ensure the API endpoint can correctly route PyPI requests.
3.  **Database Logic**: Implement Task 3 to handle persistence after successful installations.
4.  **API Contracts & Frontend Alignment**: Perform Task 4 to ensure smooth frontend integration.
5.  **Testing**: Conduct thorough testing as per Task 5.

### Relevant Files

- `backend/library/market/installation.go` (Modify)
- `backend/library/market/npm.go` (Reference)
- `backend/library/market/pypi.go` (New, for PyPI/uvx logic)
- `backend/api/handler/market.go` (Modify `InstallOrAddService`)
- `backend/model/mcp_service.go` (Reference)
- `backend/model/config_service.go` (Reference, for creating user instances)
- `backend/api/route/market_routes.go` (Reference, ensure routes are set up)
- `.cursor/feature-mcp-installer-ui.md` (Reference for frontend expectations) 