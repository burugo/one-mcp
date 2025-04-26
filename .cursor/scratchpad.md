# One MCP Service Development Plan

## Background and Motivation

The goal is to develop an One MCP (Multi-Cloud Platform) Service. This service will act as a central hub for managing and interacting with various MCP services, providing a unified interface and configuration management capabilities. The system utilizes a modern frontend (React) and backend (Go) stack, as detailed in `doc/technical_architecture.md`. Development will leverage the existing `gin-template` as a starting point for the backend, adapting it to the project's specific needs. The development will follow the phased approach outlined in `doc/roadmap.md`, starting with core user management and web interface features. **Note:** This involves replacing the template's session-based authentication with JWT and replacing its embedded web UI with a separate React Single Page Application (SPA). **Roles will be simplified using integer constants within the User model (e.g., Admin=10, User=1) instead of a separate Role table and RBAC middleware.**

## Key Challenges and Analysis

- **Backend Development**: Adapting the `gin-template`, implementing robust JWT authentication, **simplified integer-based role checks**, and secure interactions with underlying MCP services. Refactoring existing template code (e.g., user model, session handling) into the target architecture (`internal/` structure, repositories).
- **Frontend Development**: Building a responsive React SPA with Tailwind CSS, separate from the backend Go code. Replacing the template's embedded UI approach. Managing state and API integration. **Displaying UI elements conditionally based on user role retrieved from JWT.**
- **Authentication Overhaul**: Replacing the template's `gin-contrib/sessions` with a custom JWT implementation requires careful handling of token generation, validation, **including the user's integer role in the claims**, and middleware.
- **MCP Service Core**: The core logic for handling services (stdio/sse), managing connections, and handling errors remains complex.
- **Database Design & Migration**: Ensuring the GORM models (`User` with integer `Role`) are correct and using `AutoMigrate` for initial development with SQLite. Planning for potential future migrations to PostgreSQL. **No separate `Role` table needed.**
- **Integration**: Seamless integration between the new React frontend, the adapted Go backend, SQLite/Redis, and external MCP services.

## High-level Task Breakdown (Phase 1: v1.1 - Q3 2024)

This plan focuses on Phase 1 as defined in the roadmap, adapting the `gin-template` and using simplified role management.

### 1. Backend: Basic Setup & User/Role Management (Adapting gin-template, Simplified Roles)
    - **1.1**: Adapt `gin-template` structure to match `backend/` layout defined in `technical_architecture.md`.
        - Success Criteria: Project files reorganized into `backend/cmd`, `backend/internal`, etc. `go.mod` path updated (e.g., `module one-mcp/backend`). `go mod tidy` runs successfully. Basic Gin server starts via `go run backend/cmd/server/main.go`.
    - **1.2**: Define/Refactor GORM model for `User` in `backend/internal/model/`. Adapt the existing `User` model, **ensure it includes an integer `Role` field**. Define role constants (e.g., `RoleUser = 1`, `RoleAdmin = 10`) in `model` package or a shared `common` package. **Remove any `Role` model.**
        - Success Criteria: GORM model `user.go` defined with correct fields (including `Role int`) and tags. Role constants defined. No `role.go` model file exists.
    - **1.3**: Implement initial DB schema setup using GORM `AutoMigrate` **only for the `User` table**, configured for **SQLite**.
        - Success Criteria: `model.InitDB()` (or refactored equivalent) uses SQLite driver and `DB.AutoMigrate(&model.User{})`. On first run, `data/one-mcp.db` is created with the `users` table (including the `role` column). **No `roles` table is created.**
    - **1.4**: Implement logic to seed a default Admin user (e.g., in `main.go` or a seeding function) with the `RoleAdmin` constant value.
        - Success Criteria: A default admin user is created in the `users` table on initialization if no users exist, with the correct integer value in the `role` column.
    - **1.5**: Implement `UserRepository` in `backend/internal/repository/` adapting existing user logic from template's `model/user.go`. Implement basic User CRUD operations.
        - Success Criteria: `UserRepository` interface defined in `repository`. Concrete implementation using GORM exists. Functions for CreateUser, GetUserByUsername, GetUserByID, UpdateUser, DeleteUser exist. Basic unit tests pass (mocking GORM or using test DB).
    - **1.6**: Verify/Implement password hashing (bcrypt) within the `UserRepository` for user creation/update.
        - Success Criteria: `CreateUser` and relevant `UpdateUser` methods use `bcrypt.GenerateFromPassword`. Password verification logic (e.g., `CheckPassword`) uses `bcrypt.CompareHashAndPassword`. Unit tests verify hashing and comparison.

### 2. Backend: Authentication (JWT with Simplified Roles)
    - **2.1**: Implement JWT generation logic (e.g., in `internal/auth/`) and `/api/auth/login` endpoint. Ensure the generated JWT includes the user's integer `Role` in its claims. Remove template's session login logic.
        - Success Criteria: Login endpoint uses `UserRepository`, verifies password hash, generates JWT (containing user ID, **integer role**, expiry) on success, returns token and refresh token. Test with seeded admin user.
    - **2.2**: Implement JWT validation middleware (e.g., in `internal/api/middlewares/`). Ensure middleware extracts the integer `Role` from claims and adds it to the Gin context. Remove template's session middleware.
        - Success Criteria: Middleware parses token, validates signature/expiry, extracts claims (user ID, **integer role**), adds claims to Gin context. Unit tests pass.
    - **2.3**: Implement token refresh endpoint (`/api/auth/refresh`). Remove template session logout.
        - Success Criteria: Endpoint validates refresh token, generates new access token and refresh token pair. Unit tests pass.
    - **2.4**: Implement logout mechanism (JWT blacklisting in Redis). 
        - Success Criteria: `/api/auth/logout` endpoint adds token to blacklist in Redis with appropriate expiry. JWT validation middleware checks blacklist. Unit tests pass.
    - **2.5**: Preserve and adapt captcha functionality for registration and login.
        - Success Criteria: `/api/auth/captcha` endpoint generates captcha images and stores codes in Redis with short expiry. Registration and login validate captcha. Unit tests pass.

### 3. Backend: MCP Service Management (Core)
    - **3.1**: Define GORM model for `MCPService` in `backend/internal/model/`.
        - Success Criteria: `mcpservice.go` defined with fields as per architecture doc.
    - **3.2**: Add `MCPService` to GORM `AutoMigrate` in DB initialization.
        - Success Criteria: `mcp_services` table created successfully in SQLite DB on startup.
    - **3.3**: Implement `MCPServiceRepository` and basic CRUD API endpoints (`/api/services`) protected by JWT auth middleware. **Perform role checks directly in handlers for Admin-only operations (POST, PUT, DELETE) by checking the integer role from Gin context.**
        - Admin: POST (Create), PUT (Update), DELETE (Delete).
        - All authenticated: GET (List/Single).
        - Success Criteria: API endpoints (`internal/api/handlers/mcpservice_handler.go`) function correctly. Admin endpoints explicitly check `claims.Role >= model.RoleAdmin`. Integration tests pass.
    - **3.4**: Implement `toggle` endpoint (`/api/services/:id/toggle`). **Ensure handler checks for Admin role.**
        - Success Criteria: Endpoint updates `is_active` field via repository. Handler verifies user role is Admin. Integration test passes.
    - **3.5**: Implement stub config copy endpoint (`/api/services/:id/config/:client`). (Requires auth, no specific role check needed per current spec).
        - Success Criteria: Endpoint exists, requires authentication, and returns placeholder JSON `{"message": "config copy not implemented"}`.

### 4. Backend: User Configuration Management (Core)
    - **4.1**: Define GORM models for `UserConfig`, `ConfigService` in `backend/internal/model/`.
        - Success Criteria: `userconfig.go`, `configservice.go` defined with correct fields and relationships.
    - **4.2**: Add `UserConfig`, `ConfigService` to GORM `AutoMigrate`.
        - Success Criteria: `user_configs`, `config_services` tables created successfully in SQLite DB.
    - **4.3**: Implement `UserConfigRepository` and basic CRUD API endpoints (`/api/configs`) protected by JWT auth. Check ownership within handlers/service layer.
        - Self: GET (List/Single), POST (Create), PUT (Update), DELETE (Delete).
        - Success Criteria: API endpoints (`internal/api/handlers/userconfig_handler.go`) function correctly, ensuring users can only access/modify their own configs (check `user_id` from JWT against config's `user_id`). Integration tests pass.
    - **4.4**: Implement stub config export endpoint (`/api/configs/:id/:client`).
        - Success Criteria: Endpoint exists, requires authentication, checks ownership, and returns placeholder JSON.

### 5. Backend: MCP Service Core (Placeholder) & Health Checks
    - **5.1**: Define basic directory structure for MCP service (`backend/internal/infrastructure/proxy/`). (No actual service handling yet).
        - Success Criteria: Directory and basic placeholder files created.
    - **5.2**: Implement basic health check logic (placeholder, e.g., a function in `internal/service/health_service.go` called periodically).
        - Success Criteria: A background goroutine (started in `main.go`) periodically calls a health check function (which currently does nothing or simulates checks).
    - **5.3**: Store/update health status (placeholder - maybe add `status` string field to `MCPService` model/table for simplicity now). Add field to `AutoMigrate`.
        - Success Criteria: `status` field exists in `mcp_services` table. Placeholder health check updates this field (e.g., to "UNKNOWN" or "SIMULATED_OK").

### 6. Frontend: Setup & Basic Layout (Replacing Embedded UI)
    - **6.1**: Create a new React project structure in `frontend/` using Vite. Remove the old `web/` directory and `embed` directives from the Go backend (`main.go`, etc.).
        - Success Criteria: `frontend/` directory exists with Vite/React setup. `go build` for backend works without the `web/` dir. `npm run dev` in `frontend/` starts the React dev server.
    - **6.2**: Implement basic routing (React Router) and layout components (e.g., `Navbar`, `MainContent`) in `frontend/src/`.
        - Success Criteria: Basic app shell with placeholders for navigation and content area exists. Different routes show different placeholder components.
    - **6.3**: Integrate Tailwind CSS into the React project.
        - Success Criteria: Tailwind classes can be used for styling components in `frontend/`. Basic styled elements appear correctly.

### 7. Frontend: Authentication Pages
    - **7.1**: Create Login page component (`frontend/src/pages/auth/LoginPage.jsx`).
        - Success Criteria: Page renders with styled username/password fields and login button.
    - **7.2**: Implement login API call (using Axios) to backend `/api/auth/login`. Handle JWT response: store token, **decode JWT payload to get user role**, update global auth state (e.g., using React Context with user ID and role).
        - Success Criteria: User can log in. Token stored. App state reflects logged-in status and **user role**.
    - **7.3**: Implement protected routes logic (e.g., a `ProtectedRoute` component) using React Router and auth state.
        - Success Criteria: Unauthenticated users accessing protected pages are redirected to the login page.
    - **7.4**: Implement Logout functionality (button in Navbar).
        - Success Criteria: Logout button clears stored token, updates auth state (clears user ID and role), redirects to login page.
    - **7.5**: Implement Registration with captcha validation
    - **7.6**: Implement token refresh mechanism

### 8. Frontend: Home (MCP Service List) Page
    - **8.1**: Create Home page component (`frontend/src/pages/dashboard/HomePage.jsx` or similar) accessible only when logged in.
        - Success Criteria: Page component created and renders basic structure.
    - **8.2**: Implement API call (authenticated Axios instance) to fetch MCP services (`/api/services`).
        - Success Criteria: Service list is fetched on page load using the stored JWT.
    - **8.3**: Display service list (e.g., using a styled table or cards).
        - Success Criteria: Services (if any created via API testing/seeding) are displayed with name, type, status.
    - **8.4**: Implement enable/disable button (calls `/api/services/:id/toggle`). **Conditionally render/enable button based on user role (>= Admin) from auth context.**
        - Success Criteria: Button updates service status via API call. UI reflects the change. Button is hidden/disabled for non-admins based on role check.
    - **8.5**: Implement Add/Edit/Delete buttons (placeholders). **Conditionally render based on user role (>= Admin) from auth context.**
        - Success Criteria: Buttons are visible only to Admins based on role check. Clicking does nothing yet or opens placeholder modals.
    - **8.6**: Implement Copy Config button (placeholder). (Visible to all authenticated users).
        - Success Criteria: Button exists, triggers placeholder action (e.g., console log).

### 9. Frontend: My Configurations Page
    - **9.1**: Create My Configurations page component (`frontend/src/pages/configs/ConfigsPage.jsx`) accessible only when logged in.
        - Success Criteria: Page component created and renders basic structure.
    - **9.2**: Implement API call (authenticated Axios instance) to fetch user's configurations (`/api/configs`).
        - Success Criteria: User's configurations (if any) are fetched and displayed.
    - **9.3**: Implement Add/Edit/Delete functionality (placeholder buttons/forms).
        - Success Criteria: Buttons/forms exist. Clicking does nothing yet or opens placeholder modals.
    - **9.4**: Implement Export button (placeholder).
        - Success Criteria: Button exists, triggers placeholder action (e.g., console log).

## Project Status Board (Phase 1)

- [ ] **Backend: Basic Setup & User Management (Adapting gin-template, Simplified Roles)**
    - [ ] 1.1: Adapt `gin-template` structure to `backend/` layout
    - [X] 1.2: Define/Refactor User model (with `Role int`), define Role constants, remove Role model
    - [ ] 1.3: Implement initial DB schema (User only) with GORM AutoMigrate (SQLite)
    - [ ] 1.4: Implement Admin user seeding logic
    - [ ] 1.5: Implement UserRepository & CRUD (adapting template logic)
    - [ ] 1.6: Verify/Implement password hashing (bcrypt)
- [ ] **Backend: Authentication (JWT with Simplified Roles)**
    - [ ] 2.1: Implement JWT generation (with integer role claim) & /api/auth/login endpoint
    - [ ] 2.2: Implement JWT validation middleware (extracting integer role)
    - [ ] 2.3: Implement token refresh endpoint
    - [ ] 2.4: Implement logout mechanism (JWT blacklisting in Redis)
    - [ ] 2.5: Preserve and adapt captcha functionality for registration and login
- [ ] **Backend: MCP Service Management (Core)**
    - [X] 3.1: Define MCPService model
    - [ ] 3.2: Add MCPService to AutoMigrate
    - [ ] 3.3: Implement MCPServiceRepository & CRUD APIs (JWT protected, **inline Admin role checks**)
    - [ ] 3.4: Implement toggle endpoint (**with inline Admin role check**)
    - [ ] 3.5: Implement stub config copy endpoint (Auth required)
- [ ] **Backend: User Configuration Management (Core)**
    - [X] 4.1: Define UserConfig, ConfigService models
    - [ ] 4.2: Add UserConfig, ConfigService to AutoMigrate
    - [ ] 4.3: Implement UserConfigRepository & CRUD APIs (JWT protected, check ownership)
    - [ ] 4.4: Implement stub config export endpoint (Auth required, check ownership)
- [ ] **Backend: MCP Service Core (Placeholder) & Health Checks**
    - [ ] 5.1: Define basic service structure in `infrastructure/proxy/`
    - [ ] 5.2: Implement basic health check logic (placeholder)
    - [ ] 5.3: Store/update health status (placeholder, e.g., new field in MCPService)
- [ ] **Frontend: Setup & Basic Layout (Replacing Embedded UI)**
    - [ ] 6.2: Implement basic routing & layout components
    - [ ] 6.3: Integrate Tailwind CSS
- [ ] **Frontend: Authentication Pages**
    - [ ] 7.1: Create Login page component
    - [ ] 7.2: Implement login API call & JWT/state handling (**including role**)
    - [ ] 7.3: Implement protected routes
    - [ ] 7.4: Implement Logout functionality (**clearing role**)
    - [ ] 7.5: Implement Registration with captcha validation
    - [ ] 7.6: Implement token refresh mechanism
- [ ] **Frontend: Home (MCP Service List) Page**
    - [ ] 8.1: Create Home page component
    - [ ] 8.2: Implement API call to fetch services (authenticated)
    - [ ] 8.3: Display service list
    - [ ] 8.4: Implement enable/disable button (**conditional on Admin role**)
    - [ ] 8.5: Implement Add/Edit/Delete buttons (**conditional on Admin role**, placeholders)
    - [ ] 8.6: Implement Copy Config button (placeholder)
- [ ] **Frontend: My Configurations Page**
    - [ ] 9.1: Create My Configurations page component
    - [ ] 9.2: Implement API call to fetch user configs (authenticated)
    - [ ] 9.3: Implement Add/Edit/Delete functionality (placeholders)
    - [ ] 9.4: Implement Export button (placeholder)

## Current Status / Progress Tracking

*Plan refined to use simplified integer-based role management. Ready to start Phase 1, focusing on adapting the backend structure.*
*   [DONE] 1.2: Defined User model in `backend/internal/model/user.go` with integer Role field and constants.
*   **[DONE] 3.1**: Defined MCPService model in `backend/internal/model/mcpservice.go`.
*   **[DONE] 4.1**: Defined UserConfig and ConfigService models in `backend/internal/model/`.

## Executor's Feedback or Assistance Requests

*No requests at this time.*

## Lessons

*   Leveraging `gin-template` provides a useful starting point for the backend but requires significant refactoring for the target architecture (internal packages, repositories) and feature changes (JWT auth, separate React frontend).
*   Using GORM `AutoMigrate` with SQLite is suitable for initial development, simplifying schema management early on.
*   Clearly defining the replacement of template features (sessions, embedded UI) is important for planning.
*   Decision: Simplified role management using integer constants (`User=1`, `Admin=10`) in the `User` model instead of a separate `Role` table and RBAC middleware. Authorization checks for admin actions will be performed directly in API handlers.
*   While converting from session-based to JWT authentication, keep useful gin-template features like captcha generation for registration security.