## Test-Driven Development (TDD)
**Final Verification:**
Always use: `go test -v ./backend/api/handler/... | grep FAIL`

**Debugging & Handling Verbose Output:**
**Avoid** `go test -v ./...` directly in the terminal due to excessive output.
**Recommended Alternatives:**
*   **Specific Tests:** `go test -v ./... -run ^TestSpecificFunction$` (Fastest for pinpointing).

## Project Startup and Logging Rules

**Service Startup:**
- Use `bash ./run.sh` to start the one-mcp service
- This script automatically loads .env, ensures PATH, kills port 3000 processes, and starts the Go backend in background
- Logs are output to `backend.log`

**Log Monitoring:**
- Use `tail -f backend.log` to monitor real-time logs
- Use `tail -n 50 backend.log` to view recent log entries
- Use `grep "ERROR\|WARN\|Failed" backend.log` to filter error messages

**Service Management:**
- Use `pkill -f one-mcp` to stop the service
- Check service status with `ps aux | grep one-mcp | grep -v grep`
- API status endpoint: `curl "http://localhost:3003/api/status"`

## Plan Mode Tasks

### Task: Fix Service Detail Display Issues (Frontend)

**Background:** The backend API `/api/mcp_market/package_details` now returns rich information including `is_installed` status and saved environment variables within `mcp_config`. However, the frontend `ServiceDetails.tsx` page is not correctly reflecting this information.

**Analysis (from previous PLAN session):**
- The issue lies in `frontend/src/store/marketStore.ts` within the `fetchServiceDetails` action.
- `is_installed` from the API response is not mapped to `selectedService.isInstalled`.
- Saved environment variable values from `response.data.mcp_config.mcpServers[serverKey].env` are not mapped to the `value` field of `EnvVarType` objects in `selectedService.envVars`.

**Task Breakdown & Plan:**

1.  **Modify `frontend/src/store/marketStore.ts` - `fetchServiceDetails` function:**
    *   **Task Type:** `ref-func`
    *   **Objective:** Ensure `is_installed` and saved environment variable values are correctly mapped from the API response to the `selectedService` state.
    *   **Details:**
        *   In the section where `selectedService` is being constructed using `set({...})`:
            *   Add the mapping for `isInstalled`:
                ```typescript
                // selectedService: {
                //   ...
                isInstalled: details.is_installed || false, 
                //   ...
                // }
                ```
            *   Before mapping `details.env_vars` to the store's `envVars` array:
                *   Safely extract the `savedValues` map from `details.mcp_config.mcpServers`. Handle potential null/undefined cases for `details.mcp_config` or `details.mcp_config.mcpServers`. Assume the first server key in `mcpServers` is the primary one if multiple exist.
                    ```typescript
                    // const details = response.data;
                    // let savedValues: Record<string, string> = {};
                    // if (details.mcp_config && details.mcp_config.mcpServers) {
                    //     const serverKeys = Object.keys(details.mcp_config.mcpServers);
                    //     if (serverKeys.length > 0) {
                    //         savedValues = details.mcp_config.mcpServers[serverKeys[0]]?.env || {};
                    //     }
                    // }
                    ```
            *   Inside the `details.env_vars.map((envDef: any) => ({ ... }))` call, when creating each `EnvVarType` object:
                *   Populate the `value` field:
                    ```typescript
                    // return {
                    //    name: envDef.name,
                    //    description: envDef.description,
                    //    isSecret: envDef.is_secret,
                    //    isRequired: !envDef.optional,
                    //    defaultValue: envDef.default_value,
                    value: savedValues[envDef.name] !== undefined ? savedValues[envDef.name] : (envDef.default_value || '')
                    // };
                    ```
    *   **Success Criteria:**
        *   When `fetchServiceDetails` is called, the `selectedService.isInstalled` in the store accurately reflects the boolean value from `response.data.is_installed`.
        *   Each object in `selectedService.envVars` has its `value` property correctly set based on `response.data.mcp_config` if a saved value exists, otherwise falling back to `defaultValue` or an empty string.

**Relevant Files:**
- `frontend/src/store/marketStore.ts`
- `frontend/src/components/market/ServiceDetails.tsx` (consumer of the store state)

**Next Steps:**
- Await user approval of this plan.
- If approved, proceed to ACT mode to apply changes to `frontend/src/store/marketStore.ts`.

### Task: Address Service Detail Page Issues (Input Type & Save Configuration)

**Background:** User reports two issues on the Service Details page:
1. Environment variable input fields for secrets are masked (type=password), but should be plain text.
2. Saving environment variables fails because a string service ID (e.g., "pkg-name-npm") is sent to the `PATCH /api/mcp_market/env_var` endpoint, which expects a numeric ID.

**Analysis (from PLAN session):**
- **Input Type:** `ServiceDetails.tsx` conditionally sets `type="password"` for secret env vars. This needs to be changed to `type="text"`.
- **Save Configuration `service_id`:**
    - `ServiceDetails.tsx` uses `selectedService.id` (string) in the `handleSaveConfiguration` function for the API call.
    - The backend (`market.go -> GetPackageDetails`) determines the numeric `installedServiceID` but does not currently include it in the main response payload for the frontend to consume directly for this purpose.
    - The frontend store (`marketStore.ts`) needs to be updated to store this numeric ID, and `ServiceDetails.tsx` needs to use it.

**Task Breakdown & Plan:**

1.  **Backend: Modify `GetPackageDetails` in `backend/api/handler/market.go`**
    *   **Task Type:** `ref-func`
    *   **Objective:** Include the numeric `installedServiceID` in the API response if the service is installed, so the frontend can use it for subsequent API calls like saving env vars.
    *   **Details:**
        *   In the `GetPackageDetails` function, after determining `isInstalled` and `installedServiceID`:
        *   If `isInstalled` is true, add `"installed_service_id": installedServiceID` to the main `response` map that is sent to `common.RespSuccess(c, response)`.
            ```go
            // response := map[string]interface{...}
            // if isInstalled {
            //     response["installed_service_id"] = installedServiceID
            // }
            ```
    *   **Success Criteria:** The `/api/mcp_market/package_details` API response includes an `installed_service_id` field (numeric) at the top level of the `data` object when the service is installed.

2.  **Frontend Store: Modify `frontend/src/store/marketStore.ts`**
    *   **Task Type:** `ref-struct`, `ref-func`
    *   **Objective:** Add a field to store the numeric `installed_service_id` for the selected service and populate it from the API response.
    *   **Details:**
        *   Add `installed_service_id?: number;` to the `ServiceType` interface (and by extension, `ServiceDetailType`).
        *   In the `fetchServiceDetails` action, when processing the successful API response (`details = response.data`):
            *   Map the new `details.installed_service_id` from the API response to `selectedService.installed_service_id`.
                ```typescript
                // set({
                //     selectedService: {
                //         ...
                //         isInstalled: details.is_installed || false,
                installed_service_id: details.installed_service_id, // Can be undefined if not installed
                //         ...
                //     }
                // });
                ```
    *   **Success Criteria:** `selectedService.installed_service_id` in the store holds the numeric ID if the service is installed and the API provides it.

3.  **Frontend Component: Modify `frontend/src/components/market/ServiceDetails.tsx`**
    *   **Task Type:** `ref-comp`, `bug-fix`
    *   **Objective:** Change environment variable input fields to always be `type="text"` and use the correct numeric `installed_service_id` when saving configurations.
    *   **Details:**
        *   **Input Field Type:**
            *   Locate the `Input` component used for rendering environment variables (within the `.map` loop for `selectedService.envVars`).
            *   Change the `type` prop from `type={envVar.isSecret ? "password" : "text"}` to `type="text"`.
        *   **Save Configuration (`handleSaveConfiguration` function):**
            *   Modify the `api.patch` call.
            *   Change `service_id: selectedService.id,` to `service_id: selectedService.installed_service_id,`.
            *   Ensure the function only attempts to save if `selectedService.isInstalled` is true and `selectedService.installed_service_id` is a valid number.
                ```typescript
                // if (!selectedService || !selectedService.isInstalled || !selectedService.installed_service_id) return;
                // ...
                // await api.patch('/mcp_market/env_var', {
                //     service_id: selectedService.installed_service_id, 
                //     var_name: envVar.name,
                //     var_value: envVar.value || ''
                // });
                ```
    *   **Success Criteria:**
        *   All environment variable input fields in the "Configuration" tab are plain text fields.
        *   Clicking "Save Configuration" sends the numeric `installed_service_id` to the backend API, and the save operation succeeds (assuming correct API key and backend logic for saving).

**Relevant Files:**
- `backend/api/handler/market.go`
- `frontend/src/store/marketStore.ts`
- `frontend/src/components/market/ServiceDetails.tsx`

**Next Steps:**
- Await user approval of this multi-part plan.
- If approved, proceed to ACT mode to apply changes sequentially.

### Task: Fix Service Detail Page Uninstall and Env Var Placeholder Issues

**Background:** User reports two new issues on the Service Details page:
1. The "Uninstall Service" button is not working correctly.
2. When installing from the "Configuration" tab, placeholder values like "your-api-key-here" for environment variables are treated as actual values instead of prompting for user input if the variable is required.

**Analysis (from PLAN session):**

**1. Uninstall Button Not Working:**
    - `ServiceDetails.tsx` (`handleUninstall`): Calls `uninstallService(selectedService.id)`. `selectedService.id` is a string (e.g., "package-name-npm").
    - `marketStore.ts` (`uninstallService` action): Expects a numeric `serviceId` because it calls `parseInt(serviceId, 10)` to use with the backend API `/api/mcp_market/uninstall` which takes a numeric `service_id`.
    - **Mismatch:** The string ID from the component needs to be the numeric `selectedService.installed_service_id` when calling the store's `uninstallService` action.

**2. Environment Variable Placeholder Issue During Install:**
    - `ServiceDetails.tsx` (`startInstallation` function): When collecting environment variables from the "Configuration" tab, it currently includes any non-empty `env.value`. Values like "your-api-key-here" are considered valid.
    - **Desired Behavior:** If a *required* environment variable has only a placeholder value (e.g., "your-api-key-here") or is empty, it should be treated as missing. The installation process should then prompt the user for input via the `EnvVarInputModal` if the backend identifies it as a required missing variable.
    - **Solution Approach:** Modify `startInstallation` in `ServiceDetails.tsx` to exclude placeholder values (and empty strings) from `envVarsToSubmit` sent to the `installService` store action. This will allow the backend's existing missing variable check to function correctly.

**Task Breakdown & Plan:**

**Part 1: Fix Uninstall Button**

1.  **Frontend Component: Modify `frontend/src/components/market/ServiceDetails.tsx`**
    *   **Task Type:** `bug-fix`
    *   **Objective:** Ensure the correct numeric service ID is used when calling the uninstall action.
    *   **Details:**
        *   In the `handleUninstall` function:
            *   Change the call from `uninstallService(selectedService.id)` to `uninstallService(selectedService.installed_service_id)`. 
            *   Add a check to ensure `selectedService.installed_service_id` is a valid number before calling.
                ```typescript
                // if (!selectedService || typeof selectedService.installed_service_id !== 'number') return;
                // if (window.confirm(...)) {
                //     uninstallService(selectedService.installed_service_id);
                //     // ...
                // }
                ```
    *   **Success Criteria:** Clicking "Uninstall Service" correctly triggers the uninstallation process, using the numeric service ID, and the UI updates appropriately.

2.  **Frontend Store: Review `frontend/src/store/marketStore.ts` - `uninstallService` action (Verification)**
    *   **Task Type:** `verify`
    *   **Objective:** Confirm `uninstallService` correctly handles the numeric ID and updates state.
    *   **Details:** The current implementation `const serviceIdNum = parseInt(serviceId, 10);` is intended for a numeric string. If `installed_service_id` (which is already a number) is passed, `parseInt` might not be ideal. It should directly use the number or ensure type consistency.
        *   Change `uninstallService: async (serviceId: string)` to `uninstallService: async (serviceId: number)`. Remove `parseInt`.
    *   **Success Criteria:** Store logic is robust in handling the numeric ID for uninstallation.

**Part 2: Fix Environment Variable Placeholder Issue During Install**

1.  **Frontend Component: Modify `frontend/src/components/market/ServiceDetails.tsx` - `startInstallation` function**
    *   **Task Type:** `ref-func`, `bug-fix`
    *   **Objective:** Prevent common placeholder values from being submitted as actual environment variable values, allowing the backend to prompt for required missing variables.
    *   **Details:**
        *   When `initialEnvVars` is NOT provided (i.e., collecting from the Configuration tab inputs):
            *   Modify the logic for populating `envVarsToSubmit`:
                ```typescript
                // const placeholderValue = "your-api-key-here"; // Define common placeholder(s)
                // selectedService.envVars.forEach(env => {
                //     if (env.value && env.value.trim() !== placeholderValue && env.value.trim() !== "") {
                //         envVarsToSubmit[env.name] = env.value;
                //     }
                // });
                ```
    *   **Success Criteria:** If a required environment variable in the "Configuration" tab has an empty value or "your-api-key-here", and the user clicks "Install Service" (or "Install with Configuration"), the `EnvVarInputModal` should appear, prompting for that variable.

**Relevant Files:**
- `frontend/src/components/market/ServiceDetails.tsx`
- `frontend/src/store/marketStore.ts`

**Next Steps:**
- Await user approval of this multi-part plan.
- If approved, proceed to ACT mode to apply changes sequentially, starting with Part 1 (Uninstall Button).

### Task: Consistent Installation UI/UX from Marketplace List

**Background:** User wants the installation process initiated from the main marketplace service cards (`ServiceCard.tsx`) to mirror the UI/UX of installing from the `ServiceDetails.tsx` page.

**Current Behavior (Marketplace Card Install):**
- Clicking "Install" shows a simple loading state on the card.
- Does not explicitly handle environment variable input via a modal before installation.
- Does not show the detailed terminal-like installation log dialog.

**Desired Behavior (Marketplace Card Install):**
1.  **Environment Variable Check:** Before installation, check for required environment variables. If any are missing (or have placeholder values like "your-api-key-here"), an `EnvVarInputModal` should appear to collect them.
2.  **Terminal-like Log Dialog:** After environment variables are satisfied, a dialog showing installation logs (similar to the one in `ServiceDetails.tsx`) should appear and display progress.

**Analysis & Plan:**

This involves refactoring UI components and logic, primarily impacting `ServiceMarketplace.tsx` (as the parent of `ServiceCard.tsx`) and reusing/adapting elements from `ServiceDetails.tsx` and `marketStore.ts`.

**Task Breakdown & Plan:**

1.  **Modify `frontend/src/components/market/ServiceMarketplace.tsx`:**
    *   **Task Type:** `ref-comp`, `feat`
    *   **Objective:** Implement the new installation flow (env var modal + log dialog) triggered from service cards.
    *   **Details:**
        *   **Add State Variables:**
            *   `envModalVisibleForCard: boolean` (for `EnvVarInputModal`)
            *   `missingVarsForCard: string[]`
            *   `pendingInstallServiceCard: ServiceType | null` (the service being installed from a card)
            *   `currentEnvVarsForCard: Record<string, string>` (env vars collected, potentially for re-submission)
            *   `showInstallDialogForCard: boolean` (for the installation log dialog)
        *   **Adapt/Implement Handler Functions (similar to `ServiceDetails.tsx` but scoped for card installs):**
            *   `handleInstallFromCard(service: ServiceType)`: 
                *   This will be the new primary function called when a card's "Install" button is clicked.
                *   Prepare `initialEnvVars`: Iterate through `service.envVars`. Exclude empty values and common placeholders (e.g., "your-api-key-here") from `initialEnvVars` passed to the store's `installService`.
                    ```typescript
                    // const initialEnvVars: Record<string, string> = {};
                    // const placeholderValue = "your-api-key-here"; 
                    // service.envVars?.forEach(env => { // Ensure service.envVars exists
                    //    if (env.value && env.value.trim() !== placeholderValue && env.value.trim() !== "") {
                    //        initialEnvVars[env.name] = env.value;
                    //    }
                    // });
                    ```
                *   Call `store.installService(service.id, initialEnvVars)`.
                *   Handle response: If `response.data.required_env_vars` is present, set `missingVarsForCard` and `envModalVisibleForCard = true`.
                *   If no missing vars (or after modal submission), set `showInstallDialogForCard = true`.
            *   `handleEnvModalSubmitForCard(userInputVars: Record<string, string>)`: Merges `userInputVars` with `currentEnvVarsForCard` and re-triggers `handleInstallFromCard` (or a part of it) with the combined env vars.
            *   `handleEnvModalCancelForCard()`: Resets modal-related state.
            *   `closeInstallDialogForCard()`: Resets log dialog state and handles post-installation actions (e.g., toast, refetching installed services).
        *   **Integrate `EnvVarInputModal`:**
            *   Render `<EnvVarInputModal open={envModalVisibleForCard} missingVars={missingVarsForCard} onSubmit={handleEnvModalSubmitForCard} onCancel={handleEnvModalCancelForCard} />`.
        *   **Integrate Installation Log Dialog:**
            *   Copy or refactor the installation log dialog JSX from `ServiceDetails.tsx`.
            *   Connect its visibility to `showInstallDialogForCard`.
            *   Display logs from `store.installTasks[pendingInstallServiceCard?.id]?.logs`.
            *   Show status (installing, success, error) based on `store.installTasks[pendingInstallServiceCard?.id]?.status`.

2.  **Modify `frontend/src/components/market/ServiceCard.tsx`:**
    *   **Task Type:** `ref-comp`
    *   **Objective:** Update the "Install" button to trigger the new flow in `ServiceMarketplace.tsx`.
    *   **Details:**
        *   The `onInstall` prop (or direct click handler) should now call `handleInstallFromCard(service)` passed down from `ServiceMarketplace.tsx`.
        *   The card's local loading state (e.g., button text changing to "Installing...") might be simplified or removed, as the primary feedback will come from the modal and the shared log dialog.

3.  **Verify `frontend/src/store/marketStore.ts` (No changes anticipated initially):**
    *   **Task Type:** `verify`
    *   **Objective:** Ensure existing store actions (`installService`, `pollInstallationStatus`, etc.) support this new UI flow without modification.
    *   **Details:** The `installService` action already supports returning `required_env_vars`, and `pollInstallationStatus` updates `installTasks` which the new log dialog will consume. This should be largely compatible.

**Success Criteria:**
- Clicking "Install" on a service card in the marketplace list initiates a flow that first checks for (and prompts for if necessary) required environment variables using `EnvVarInputModal`.
- Placeholder values like "your-api-key-here" in `service.envVars` are not treated as valid inputs for required variables during this check.
- After environment variables are satisfied, a terminal-like dialog appears, showing the installation logs and status, identical in behavior to the one on the `ServiceDetails` page.

**Relevant Files:**
- `frontend/src/components/market/ServiceMarketplace.tsx`
- `frontend/src/components/market/ServiceCard.tsx`
- `frontend/src/components/market/ServiceDetails.tsx` (for reference/copying UI elements)
- `frontend/src/components/market/EnvVarInputModal.tsx` (reuse)
- `frontend/src/store/marketStore.ts` (reuse actions)

**Next Steps:**
- This plan is added after the plan for "Fix Service Detail Page Uninstall and Env Var Placeholder Issues".
- Await user decision on whether to proceed with the previously approved plan first, or to refine/prioritize this new requirement. If proceeding with prior plan, this task will be addressed later.

### Task: Address Post-Uninstall UI Issues and Card Uninstall ID

**Background:** User reports two issues after recent fixes:
1. Uninstalling from a marketplace service card (`ServiceCard.tsx`) still uses a string ID, while the store now expects a numeric ID.
2. After uninstalling from the `ServiceDetails.tsx` page and navigating back to the marketplace, then re-entering the details page for the same service, the button incorrectly still shows "Uninstall" instead of reverting to "Install".

**Analysis (from PLAN session):**

**Issue 1: Uninstall from Marketplace Card Uses String ID**
- **Root Cause:** `ServiceCard.tsx` uses service data from `searchResults` (via `marketStore.ts`). The `searchServices` action might not populate `installed_service_id` (numeric) from the `/api/mcp_market/search` endpoint, or the endpoint itself doesn't provide it for already-installed search results.
- **Solution Plan:**
    1.  **Backend (`/api/mcp_market/search` - `market.go` & `npm.go`):** Modify `ConvertNPMToSearchResult` (called by `SearchMCPMarket`) to include the numeric `InstalledServiceID` in `SearchPackageResult` for packages found to be already installed.
    2.  **Frontend Store (`marketStore.ts` - `searchServices`):** Ensure this `installed_service_id` from the search API response is correctly mapped to `ServiceType.installed_service_id` in the `searchResults` array.
    3.  **Frontend Component (`ServiceCard.tsx`):** Update its "Uninstall" button handler to use `props.service.installed_service_id` (numeric, with a check) when calling `store.uninstallService()`.

**Issue 2: Details Page "Uninstall" Button State Not Updating After Re-navigation**
- **Root Cause:** After uninstall, `ServiceDetails.tsx` navigates back. If the user re-selects the same service, `fetchServiceDetails` calls `/api/mcp_market/package_details`. This endpoint might not be correctly reporting the service as uninstalled (i.e., `is_installed: false`) because the `UninstallService` backend function performs a soft delete (`Deleted = true`). The `GetPackageDetails` handler needs to respect this soft delete status.
- **Solution Plan:**
    1.  **Backend (`/api/mcp_market/package_details` - `market.go` and `model` layer):** Ensure `GetPackageDetails` (and its underlying data retrieval, e.g., `model.GetServicesByPackageDetails`) correctly considers soft-deleted services (where `Deleted = true`) as *not* currently installed when setting the `isInstalled` flag in the API response.
    2.  **Frontend Store (`marketStore.ts` - `uninstallService`) (Review):** Confirm that after successful uninstallation, `selectedService` is either cleared or subsequent fetches get the updated `isInstalled: false` status. The `onBack()` from `ServiceDetails.tsx` typically calls `clearSelectedService()`, which should handle this if the backend provides correct fresh data.

**Task Breakdown & Plan:**

**Part 1: Fix Uninstall from Marketplace Card**

1.  **Backend: Modify `backend/library/market/npm.go` - `ConvertNPMToSearchResult`**
    *   **Task Type:** `ref-func`
    *   **Objective:** Include `InstalledServiceID` in search results for installed packages.
    *   **Details:** The `installedMap` passed to `ConvertNPMToSearchResult` currently is `map[string]bool`. This needs to change to `map[string]int64` (packageName -> installedServiceID) or `ConvertNPMToSearchResult` needs a way to look up the ID if `isInstalled` is true.
        *   Modify `market.GetInstalledMCPServersFromDB()` to return `map[string]int64`.
        *   Update `SearchMCPMarket` in `market.go` to create and pass this map.
        *   In `ConvertNPMToSearchResult`, if a package `isInstalled`, retrieve and set `InstalledServiceID` on the `SearchPackageResult` struct (which needs this field added).

2.  **Backend: Modify `backend/library/market/types.go` (or where `SearchPackageResult` is defined)**
    *   **Task Type:** `ref-struct`
    *   **Objective:** Add `InstalledServiceID *int64 `json:"installed_service_id,omitempty"` field to `SearchPackageResult` struct.

3.  **Frontend Store: Modify `frontend/src/store/marketStore.ts` - `searchServices` action**
    *   **Task Type:** `ref-func`
    *   **Objective:** Map `installed_service_id` from search API to frontend `ServiceType`.
    *   **Details:** When mapping API results to `ServiceType` objects, ensure `item.installed_service_id` is assigned to `installed_service_id`.

4.  **Frontend Component: Modify `frontend/src/components/market/ServiceCard.tsx`**
    *   **Task Type:** `ref-comp`, `bug-fix`
    *   **Objective:** Use the numeric `installed_service_id` for uninstall action.
    *   **Details:** Change the "Uninstall" button's `onClick` handler to call `props.onUninstall(service.installed_service_id)` (assuming `onUninstall` prop is updated or directly call store: `uninstallService(service.installed_service_id)`), after checking `service.installed_service_id` is a valid number.

**Part 2: Fix Details Page Button State After Uninstall & Re-navigation**

1.  **Backend: Modify `backend/model/service.go` - `GetServicesByPackageDetails` (and similar lookups)**
    *   **Task Type:** `ref-func`
    *   **Objective:** Ensure service lookups for checking installation status exclude soft-deleted services.
    *   **Details:** Modify the database query in `GetServicesByPackageDetails` (and any other functions used by `GetPackageDetails` to determine `isInstalled`) to include a condition like `AND deleted_at IS NULL` or `AND deleted = false`.

2.  **Frontend Store: Review `frontend/src/store/marketStore.ts` - `clearSelectedService` (Verification)**
    *   **Task Type:** `verify`
    *   **Objective:** Confirm `clearSelectedService()` is effectively called when navigating back from details page after an uninstall, ensuring no stale `selectedService` data.
    *   **Details:** The `onBack()` prop in `ServiceDetails.tsx` is typically connected to `clearSelectedService()` in `MarketPage.tsx`. This should be sufficient if the backend correctly reports the uninstalled status upon next `fetchServiceDetails`.

**Relevant Files:**
- `backend/library/market/npm.go`
- `backend/library/market/types.go` (or equivalent for `SearchPackageResult`)
- `backend/api/handler/market.go`
- `backend/model/service.go`
- `frontend/src/store/marketStore.ts`
- `frontend/src/components/market/ServiceCard.tsx`
- `frontend/src/components/market/ServiceDetails.tsx`

**Next Steps:**
- This plan consolidates the two new issues. Await user confirmation to proceed with ACT mode for these tasks, starting with Part 1 (Marketplace Card Uninstall).

