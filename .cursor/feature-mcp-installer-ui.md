# MCP Server Installation UI

This document outlines the design and tasks for implementing the MCP Server Installation User Interface, allowing users to discover, install, and manage MCP servers from various marketplaces.

## Conceptual Design

### 1. Overall Layout & Navigation (OpenRouter Style)

*   **Theme**: Consistent with OpenRouter (dark theme primary, optional light theme), emphasizing simplicity, professionalism, and ease of use.
*   **Top Navigation Bar (Header)**:
    *   Left: App Logo / "Back" button.
    *   Center: Page Title (e.g., "MCP Service Marketplace" or "Install MCP Services").
    *   Right: User avatar/settings (if applicable).
*   **Main Content Area**:
    *   **Search & Filter Area (Top of content area)**:
        *   Prominent Search Box: Placeholder `Search service name, keywords, package name...`
        *   View Toggle Tabs/Pills:
            *   `[ Marketplace ]` (Default view, shows all installable services)
            *   `[ Installed ]` (Shows locally installed services)
            *   `(Optional)` `[ Recommended ]` (Shows curated built-in services)
    *   **Service List Area (Below search area)**:
        *   Card-based grid or list display for services.
        *   Supports pagination or infinite scroll.

### 2. Service Card Design

*   **Card Style**: Clean, clear information, ample spacing between cards.
*   **Card Elements**:
    *   **Top-left/Top**: Service Icon/Logo (placeholder or initial if none).
    *   **Main Info Area (Upper-middle of card)**:
        *   **Service Name**: (e.g., "Airtable Data Connector") - Larger font, prominent.
        *   **Version**: (e.g., `v0.6.2`) - Secondary text.
        *   **Source/Package Name**: (e.g., `npm: @modelcontextprotocol/server-airtable`, `pypi: mcp-airtable-connector`) - Link to original package page if possible.
    *   **Description Area (Middle of card)**: Short service description (2-3 lines).
    *   **Action Area (Bottom-right/Bottom of card)**:
        *   **Install/Add Button ("+")**:
            *   **For `stdio` type services (from npm/PyPI)**: Tooltip shows install command (e.g., "Install via npx: `npx ...`" or "Install via uvx: `uvx ...`").
            *   **For `remote` type services**: Tooltip shows "Add this service configuration".
            *   **Button States**:
                *   Default: "+" or "Add" / "Install"
                *   Processing: Spinner or "Installing..." / "Adding...", button disabled.
                *   Completed: "Configured" (green check icon) or "Manage". Implies a user-specific instance is ready or created.
                *   Error: "Failed" (red warning icon), option to retry.
        *   `(Optional)` Link to service documentation or GitHub repository.

### 3. User Interaction Flow

1.  **Entry**: User navigates to MCP service installation page. Default to "Marketplace" view.
2.  **Browse/Search**: User finds a service. Service card indicates if it's `stdio` (from npm/PyPI) or `remote`.
3.  **Initiate Add/Install**: User clicks "+" button on a service card.
    *   **Pre-check for Env Vars**:
        *   **For `stdio` from npm/PyPI**: UI may trigger a backend call to attempt discovery of required environment variables from the package's homepage/readme (best-effort).
        *   **For `remote` or admin-defined `stdio`**: The `MCPService` definition (from backend) already lists required env vars.
    *   **Environment Variable Dialog (If Needed)**:
        *   If any env vars are identified as required or suggested, a dialog pops up.
        *   Dialog lists variable names (e.g., `API_KEY`, `ENDPOINT_URL`), descriptions, and input fields.
        *   User can fill in the variables or choose to "Skip & Configure Later".
4.  **Execute Action (Post Env Var Dialog or if no Env Vars needed)**:
    *   **If `stdio` type (npm/PyPI)**:
        *   UI calls backend to install using `npx` or `uvx`. User-provided env vars (if any) are passed.
        *   Backend executes command. On success, it auto-creates a basic `MCPService` definition (if not existing for this package) and a `ConfigService` instance for the user, storing provided env vars in `UserOverrideConfigValues`.
        *   UI shows progress/status. On success, card updates to "Configured" or "Manage".
    *   **If `remote` type**:
        *   UI calls backend to create a user-specific configuration (a `ConfigService` instance) linked to the predefined `MCPService`. User-provided env vars (if any) are passed and stored.
        *   No command execution.
        *   UI shows status. On success, card updates to "Configured" or "Manage".
5.  **Manage Configured/Installed Services**:
    *   In "Installed" (or a similar "My Services") view, users see their `ConfigService` instances.
    *   Actions: "Configure" (to update env vars/overrides), "Uninstall" (for `stdio` types, removes package and `ConfigService`), "Remove Configuration" (for `remote` types, removes `ConfigService`).

### 4. Style Guide (Inspired by OpenRouter)

*   **Minimalism**: Avoid unnecessary decoration, clear information hierarchy.
*   **Whitespace**: Ample spacing and margins for readability and visual comfort.
*   **Borders & Shadows**: Subtle borders or soft shadows for cards.
*   **Icons**: Clean, modern SVG icons (e.g., Feather Icons, Material Icons).
*   **Responsive Design**: Adaptable to different window sizes (if applicable).
*   **Typography**: Clear sans-serif fonts, using weight and size for hierarchy.

## Future Tasks

- [ ] **UI Design Mockups**: Create detailed visual mockups or wireframes based on this concept. `Task Type: New Feature`
- [ ] **Frontend Component Development**: Implement reusable UI components (search bar, service card, tabs, buttons). `Task Type: New Feature`
- [ ] **Marketplace Search Integration (Frontend)**: Develop frontend logic to call backend APIs for searching npm & PyPI. `Task Type: New Feature`
- [ ] **Installation Flow (Frontend)**: Implement frontend logic to trigger installation, display progress, and handle success/failure states. `Task Type: New Feature`
- [ ] **Backend API for Search**: Design and implement backend API(s) to search across npm and PyPI. `Task Type: New Feature`
- [ ] **Backend API for Installation/Configuration**: Design and implement backend API(s) to:
    *   Trigger `npx` and `uvx` installations securely for `stdio` services.
    *   Automatically create a basic `MCPService` definition post-install for new `stdio` packages.
    *   Create `ConfigService` instances for users (for both `stdio` and `remote` types), storing any provided env vars.
    *   **(New/Research)** Attempt to discover required env vars from npm/PyPI package pages/readmes for `stdio` services. `Task Type: Research/New Feature`
- [ ] **State Management (Backend)**: Store information about discovered and installed services. `Task Type: New Feature`
- [ ] **Testing**: Unit and integration tests for UI and backend components. `Task Type: New Feature`

## Implementation Plan

1.  Finalize UI mockups.
2.  Develop core frontend components.
3.  Implement backend APIs for search and installation.
4.  Integrate frontend with backend APIs.
5.  Implement the full installation and state management flow.
6.  Thorough testing.

### Relevant Files

- (To be created, e.g., `frontend/components/mcp_server_installer/`, `backend/api/handler/mcp_market.go`) 