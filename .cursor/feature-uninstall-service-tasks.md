# Feature: Service Uninstallation

Implement the ability for users to uninstall services from the marketplace.

## Completed Tasks

### Phase 1: Backend Implementation

- [x] **Task 1.1: Design Uninstall API Endpoint** `new-feat`
    - **Objective**: Define the API endpoint, method (e.g., DELETE or POST), and request/response contract for uninstalling a service.
    - **Considerations**: Authentication, service identification (e.g., by ID or package name).
    - **Output**: API specification (e.g., `DELETE /api/mcp_market/services/{service_id}/uninstall` or `POST /api/mcp_market/uninstall_service` with body `{ "packageName": "..." }`).
    - **Success Criteria**: API contract is clearly defined and documented.

- [x] **Task 1.2: Implement Backend Uninstallation Logic** `new-feat` (Assumed complete for frontend planning)
    - **Objective**: Write the backend service logic to handle the uninstall request.
    - **Steps**:
        1.  Identify the service to be uninstalled.
        2.  Determine the package manager (npm, PyPI) and construct the correct uninstall command.
        3.  Execute the uninstall command securely.
        4.  Handle output and errors from the command.
        5.  If successful, update the database (e.g., remove service record or mark as uninstalled).
    - **Success Criteria**: Backend can successfully execute uninstall commands and update database status. Robust error handling is in place.

- [x] **Task 1.3: Expose Uninstallation API** `new-feat` (Assumed complete for frontend planning)
    - **Objective**: Implement the HTTP handler for the defined API endpoint, linking it to the uninstallation logic.
    - **Success Criteria**: The uninstall API is accessible and functional.

### Phase 2: Frontend State Management (Zustand Store)

- [x] **Task 2.1: Extend `marketStore` for Uninstallation** `ref-func`
    - **Objective**: Add state properties to `marketStore.ts` to track uninstallation status (e.g., `uninstallTasks: { [serviceId: string]: { status: 'idle' | 'uninstalling' | 'error', error?: string } }`).
    - **Success Criteria**: Store can hold per-service uninstallation status.

- [x] **Task 2.2: Implement `uninstallService` Action** `new-feat`
    - **Objective**: Create a new async action in `marketStore.ts` (e.g., `uninstallService(serviceId: string)`).
    - **Steps**:
        1.  Set `uninstallTasks[serviceId].status` to `'uninstalling'`.
        2.  Call the new backend uninstall API.
        3.  On success:
            - Update `uninstallTasks[serviceId].status` to `'idle'`.
            - Refresh the list of installed services or update the specific service's status to not installed.
            - Show success toast.
        4.  On failure:
            - Set `uninstallTasks[serviceId].status` to `'error'` with the error message.
            - Show error toast.
    - **Success Criteria**: Store action correctly handles the uninstallation lifecycle and API communication.

## Future Tasks

### Phase 3: Frontend UI & Integration

- [x] **Task 3.1: Add "Uninstall" Button to `ServiceCard.tsx`** `ref-struct`
    - **Objective**: For services that are installed, display an "Uninstall" button instead of or in addition to "Installed" / "Details".
    - **Logic**: Button should be visible if `service.isInstalled` (or similar flag) is true and `uninstallTasks[service.id]?.status` is not `'uninstalling'`.
    - **Interaction**: Clicking "Uninstall" should trigger the display of a confirmation dialog.
    - **Success Criteria**: "Uninstall" button appears correctly on installed service cards.

- [x] **Task 3.2: Implement Uninstall Confirmation Dialog** `new-feat`
    - **Objective**: Create or reuse a generic confirmation dialog component (`ConfirmDialog.tsx`).
    - **Content**: "Are you sure you want to uninstall [Service Name]? This action cannot be undone." Buttons: "Cancel", "Uninstall".
    - **Invocation**: `ServiceCard` will show this dialog before calling the `uninstallService` store action.
    - **Success Criteria**: Confirmation dialog functions correctly and is displayed before uninstallation. (Component created, integration pending in Task 3.3)

- [ ] **Task 3.3: Integrate Uninstallation Flow in `ServiceCard.tsx`** `new-feat`
    - **Objective**: Wire up the "Uninstall" button, confirmation dialog, and the `uninstallService` store action.
    - **State Feedback**:
        - When `uninstallTasks[service.id]?.status` is `'uninstalling'`, the "Uninstall" button should show a loading spinner and be disabled.
        - After successful uninstallation, the card should revert to its "not installed" state (e.g., showing an "Install" button).
    - **Success Criteria**: Full uninstallation flow is working from the `ServiceCard`, including confirmation and visual feedback.

- [ ] **Task 3.4: Visual Feedback for Installation/Uninstallation** `new-feat`
    - **Objective**: Implement the "installing" toast, and the "installed" large semi-transparent checkmark animation (fading out over the card center) as previously discussed. Apply similar feedback for "uninstalling" (toast) and "uninstalled" (card reverts to install state). (Installation part DONE, loading spinner + success checkmark implemented)
    - **Success Criteria**: Enhanced visual feedback improves user experience for both install and uninstall operations.

- [ ] **Task 3.5 (BUG): Incorrect `isInstalled` status in marketplace search results** `bug-fix`
    - **Objective**: Ensure services shown in search results accurately reflect their installation status.
    - **Investigation**: Check `marketStore.ts` `searchServices` logic and backend API `/mcp_market/services` response.
    - **Success Criteria**: Search results always show correct installation status.

- [ ] **Task 3.6 (BUG): Uninstall API (`/api/mcp_market/services/{serviceId}/uninstall`) fails** `bug-fix`
    - **Objective**: Ensure successful service uninstallation via the API.
    - **Investigation**: Check frontend request in `marketStore.ts` `uninstallService`, check backend API logs/implementation for the uninstall endpoint. Request error details from user.
    - **Success Criteria**: Uninstall API call succeeds, and service is uninstalled.

- [ ] **Task 3.7: Fix uninstall API route mismatch (前后端卸载接口路由不一致)** `bug-fix`
    - **Objective**: Align frontend uninstallService API call with backend handler.
    - **Steps**:
        1. Update `marketStore.ts` uninstallService to use `POST /api/mcp_market/uninstall` with body `{ package_name, package_manager }`.
        2. Ensure correct data is passed from service object.
        3. Test uninstall flow, UI, and feedback.
    - **Success Criteria**: Uninstall works, returns JSON, UI/UX correct.

### Phase 4: Testing

- [ ] **Task 4.1: End-to-End Testing** `bug-fix`
    - **Objective**: Manually test the entire uninstallation flow for various scenarios.
    - **Scenarios**:
        1.  Successful uninstallation of an npm package.
        2.  Successful uninstallation of a PyPI package (if applicable).
        3.  Attempt to uninstall while offline / backend error.
        4.  Cancelling the uninstall confirmation.
        5.  UI state updates correctly throughout the process.
    - **Success Criteria**: Uninstallation feature is robust and works as expected across different scenarios.

## Implementation Plan

1.  **Backend First**: Implement API and core logic for uninstallation.
2.  **Store Logic**: Extend frontend store to support uninstallation states and actions.
3.  **UI Integration**: Update `ServiceCard` with Uninstall button, confirmation dialog, and connect to store actions.
4.  **Feedback Polish**: Implement visual feedback enhancements (toasts, animations).
5.  **Thorough Testing**.

## Relevant Files (Anticipated)

- `frontend/src/components/market/ServiceCard.tsx`
- `frontend/src/stores/marketStore.ts`
- `frontend/src/components/ui/ConfirmDialog.tsx` (or similar)
- `frontend/src/utils/api.ts`
- Backend files related to service management and package uninstallation. 