# Fix Uninstall Race Condition Leading to Panic

## Background and Motivation
A bug has been identified where uninstalling a service can lead to a panic in the backend. This occurs due to a race condition: if an installation task for a service completes around the same time the service is being uninstalled, the system attempts to initialize an MCP client for the service whose process is being (or has been) terminated. This results in errors within the `mcp-go` client library (e.g., "context deadline exceeded", "file already closed") and can culminate in a "slice bounds out of range" panic.

The goal is to prevent this panic and ensure the system handles the concurrent completion of installation and uninstallation operations gracefully.

## Key Challenges and Analysis
- The core issue is a race condition between `InstallationManager.updateServiceStatus` (triggered by a successful installation) and `handler.UninstallService`.
- `updateServiceStatus` currently may not have the most up-to-date status of the service if an uninstall is happening concurrently.
- An attempt to initialize an MCP client (`MCPClientManager.InitializeClient`) on a terminated or dying process is the direct cause of the panic.
- Database updates in `updateServiceStatus` might also incorrectly overwrite the "deleted" status set by `UninstallService`.

## High-level Task Breakdown
1.  Modify `InstallationManager.updateServiceStatus` in `backend/library/market/installation.go`.
2.  Ensure that before attempting to update the service's status to "installed" in the database and before initializing the MCP client, the function re-checks the service's current status from the database.
3.  If the service is found to be deleted or disabled (indicating an uninstall has occurred), skip the database update (that would mark it as installed) and skip the client initialization.

## Project Status Board
- [ ] **Analysis**: Complete.
- [ ] **Planning**: Complete.
- [ ] **Implementation**: Pending.
- [ ] **Testing**: Pending.
- [ ] **Deployment**: Pending.

## Future Tasks
- **Task 1: Modify `updateServiceStatus` for Robustness** `bug-fix`
    - **Description**: In `backend/library/market/installation.go`, within the `updateServiceStatus` function:
        1. After fetching the initial service record (`service, err := model.GetServiceByID(task.ServiceID)`), and after preparing the installation-related updates to this `service` object (e.g., setting version, health details), but **before** calling `model.UpdateService(service)` and **before** calling `manager.InitializeClient(...)`:
        2. Re-fetch the service directly from the database to get its most current state: `currentDBService, queryErr := model.GetServiceByID(task.ServiceID)`.
        3. Check if `queryErr` is nil and if `currentDBService.Deleted == true` or `currentDBService.Enabled == false`. These conditions indicate that an uninstall has likely occurred or the service has been administratively disabled.
        4. If these conditions are met (service is deleted/disabled):
            - Log a message indicating that client initialization and final DB update are being skipped due to the service's uninstalled/disabled state.
            - Return from `updateServiceStatus` immediately, thereby avoiding the potentially problematic `model.UpdateService` call (which would revert uninstall changes) and the `InitializeClient` call (which would cause the panic).
        5. If the conditions are not met (service is still active):
            - Proceed with the existing logic: call `model.UpdateService(service)` and then, if applicable, `manager.InitializeClient(...)`.
    - **Success Criteria**:
        - The panic during uninstall (when an installation task completes concurrently) is no longer observed.
        - Services that are uninstalled remain in the "deleted" / "disabled" state in the database, even if an installation task for them completes shortly thereafter.
        - MCP client initialization is not attempted for services that have been uninstalled or disabled.
        - Logs clearly indicate when `updateServiceStatus` skips client initialization due to prior uninstallation.

- **Task 2: Enhance `UninstallService` to Explicitly Close Client** `refactor`
    - **Description**: In `backend/api/handler/market.go`, within the `UninstallService` function:
        1. After fetching the service details (`service, err := model.GetServiceByID(serviceID)`).
        2. **Before** attempting physical package uninstallation (e.g., `market.UninstallNPMPackage`) and **before** updating the service record in the database to set `Deleted=true`.
        3. If the service is of a type managed by `MCPClientManager` (e.g., `stdio`), explicitly instruct the `MCPClientManager` to shut down and remove the client associated with `service.ID` (or `service.SourcePackageName`). This might involve calling a method like `MCPClientManager.ShutdownClient(service.ID)` or `GetMCPClientManager().RemoveClient(service.ID)`. The exact method name and availability needs to be verified based on `MCPClientManager`'s implementation.
        4. This step ensures that any active client instance is properly shut down and removed from management before the underlying package is uninstalled or the service is marked as deleted in the DB.
    - **Success Criteria**:
        - `MCPClientManager` no longer holds or attempts to manage an active client instance for a service after `UninstallService` has successfully processed it.
        - Reduces potential for errors if other parts of the system query `MCPClientManager` immediately after an uninstall is initiated.
        - Service processes are more cleanly shut down via the client manager, where applicable, before any forceful termination by package uninstallers might occur.
        - Logs indicate the client shutdown attempt.

## Implementation Plan
The primary change will be in `backend/library/market/installation.go`.

### Relevant Files
- `backend/library/market/installation.go` (Primary file for modification)
- `backend/api/handler/market.go` (For understanding the uninstall logic)
- `backend/model/mcp_service.go` (Or wherever `MCPService` model and `GetServiceByID` are defined)

## Lessons
- Concurrent operations on shared resources (like service state and processes) require careful synchronization or state-checking to prevent race conditions.
- Always re-fetching critical state information before acting can be a good defensive programming practice in distributed or concurrent systems.

## ACT mode Feedback or Assistance Requests
- (None at this time)

## User Specified Lessons
- (None at this time) 