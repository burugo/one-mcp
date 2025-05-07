# One MCP Service Development Plan

## Background and Motivation

The goal is to develop an One MCP (Multi-Cloud Platform) Service. This service will act as a central hub for managing and interacting with various MCP services, providing a unified interface and configuration management capabilities. The system utilizes a modern frontend (React) and backend (Go) stack, as detailed in `doc/architecture.md`. Development will leverage the existing `gin-template` as a starting point for the backend, adapting it to the project's specific needs. The development will follow the phased approach outlined in `doc/roadmap.md`, starting with core user management and web interface features. **Note:** This involves replacing the template's session-based authentication with JWT and replacing its embedded web UI with a separate React Single Page Application (SPA). **Roles will be simplified using integer constants within the User model (e.g., Admin=10, User=1) instead of a separate Role table and RBAC middleware.**

**UPDATE:** The project will now follow a traditional MVC (Model-View-Controller) architecture instead of the originally planned DDD (Domain-Driven Design) approach. This change allows for a simpler, more familiar code structure that better matches the existing gin-template codebase. The structure will consist of Models (data structures and database operations), Views (frontend React components), and Controllers (API handlers in the backend).

## Key Challenges and Analysis

- **Backend Development (MVC Pattern)**: Adapting the `gin-template` to follow a clean MVC pattern, implementing robust JWT authentication, **simplified integer-based role checks**, and secure interactions with underlying MCP services. Models will handle data structures and database operations, Controllers will handle API requests, and Views will be handled by the React frontend.
- **Frontend Development**: Building a responsive React SPA with Tailwind CSS, separate from the backend Go code. Replacing the template's embedded UI approach. Managing state and API integration. **Displaying UI elements conditionally based on user role retrieved from JWT.**
- **Authentication Overhaul**: Replacing the template's `gin-contrib/sessions` with a custom JWT implementation requires careful handling of token generation, validation, **including the user's integer role in the claims**, and middleware.
- **MVC Implementation**: Organizing the codebase following MVC principles: Models in `model/` directory, Controllers in `api/handler/` directory, and Views represented by the React frontend. This provides a cleaner separation of concerns compared to DDD.
- **Database Design & Migration**: Ensuring the GORM models (`User` with integer `Role`) are correct and using `AutoMigrate` for initial development with SQLite. Planning for potential future migrations to PostgreSQL. **No separate `Role` table needed.**
- **Integration**: Seamless integration between the React frontend (View), Go backend controllers (Controller), and data models (Model), along with SQLite/Redis and external MCP services.

## High-level Task Breakdown (Phase 1: v1.1 - Q3 2024)

This plan focuses on Phase 1 as defined in the roadmap, adapting the `gin-template` and using simplified role management.

### 1. Backend: Basic Setup & User/Role Management (MVC Architecture)
    - **1.1**: Adapt `gin-template` structure to match `backend/` layout defined in `architecture.md`.
        - Success Criteria: Project files reorganized into standard MVC structure: `model/`, `api/handler/`, `api/route/`, etc. `go.mod` path updated (e.g., `module one-mcp/backend`). `go mod tidy` runs successfully. Basic Gin server starts via `go run main.go`.
    - **1.2**: Define/Refactor GORM model for `User` in `backend/model/`. Adapt the existing `User` model, **ensure it includes an integer `Role` field**. Define role constants (e.g., `RoleUser = 1`, `RoleAdmin = 10`) in `model` package or a shared `common` package. **Remove any `Role` model.**
        - Success Criteria: GORM model `user.go` defined with correct fields (including `Role int`) and tags. Role constants defined. No `role.go` model file exists.
    - **1.3**: Implement initial DB schema setup using GORM `AutoMigrate` **only for the `User` table**, configured for **SQLite**.
        - Success Criteria: `model.InitDB()` (or refactored equivalent) uses SQLite driver and `DB.AutoMigrate(&model.User{})`. On first run, `data/one-mcp.db` is created with the `users` table (including the `role` column). **No `roles` table is created.**
    - **1.4**: Implement logic to seed a default Admin user (e.g., in `main.go` or a seeding function) with the `RoleAdmin` constant value.
        - Success Criteria: A default admin user is created in the `users` table on initialization if no users exist, with the correct integer value in the `role` column.
    - **1.5**: Enhance the User model with CRUD methods following MVC pattern. Adapt existing user logic from template's `model/user.go`.
        - Success Criteria: User model with methods for Create, Read, Update, Delete operations. Basic unit tests pass (testing GORM database operations).
    - **1.6**: Verify/Implement password hashing (bcrypt) within the User model methods for user creation/update.
        - Success Criteria: CreateUser and UpdateUser methods use `bcrypt.GenerateFromPassword`. Password verification logic (e.g., `CheckPassword`) uses `bcrypt.CompareHashAndPassword`. Unit tests verify hashing and comparison.

### 2. Backend: Authentication (JWT with Simplified Roles)
    - **2.1**: Implement JWT generation logic (e.g., in `service/auth_service.go`) and `/api/auth/login` handler in `api/handler/auth.go`. Ensure the generated JWT includes the user's integer `Role` in its claims. Remove template's session login logic.
        - Success Criteria: Login handler uses User model methods, verifies password hash, generates JWT (containing user ID, **integer role**, expiry) on success, returns token and refresh token. Test with seeded admin user.
    - **2.2**: Implement JWT validation middleware (e.g., in `api/middleware/auth.go`). Ensure middleware extracts the integer `Role` from claims and adds it to the Gin context. Remove template's session middleware.
        - Success Criteria: Middleware parses token, validates signature/expiry, extracts claims (user ID, **integer role**), adds claims to Gin context. Unit tests pass.
    - **2.3**: Implement token refresh endpoint (`/api/auth/refresh`). Remove template session logout.
        - Success Criteria: Endpoint validates refresh token, generates new access token and refresh token pair. Unit tests pass.
    - **2.4**: Implement logout mechanism (JWT blacklisting in Redis). 
        - Success Criteria: `/api/auth/logout` endpoint adds token to blacklist in Redis with appropriate expiry. JWT validation middleware checks blacklist. Unit tests pass.
    - **2.5**: Preserve and adapt captcha functionality for registration and login.
        - Success Criteria: `/api/auth/captcha` endpoint generates captcha images and stores codes in Redis with short expiry. Registration and login validate captcha. Unit tests pass.

### 3. Backend: MCP Service Management (Core)
    - **Task Type: New Feature**
    - **3.1**: Define GORM model for `MCPService` in `backend/model/`.
        - Success Criteria: `mcpservice.go` defined with fields as per architecture doc.
    - **3.2**: Add `MCPService` to GORM `AutoMigrate` in DB initialization.
        - Success Criteria: `mcp_services` table created successfully in SQLite DB on startup.
    - **3.3**: Implement CRUD methods for MCPService model and API handlers (`/api/services`) in `backend/api/handler/service.go`, protected by JWT auth middleware. **Perform role checks directly in handlers for Admin-only operations (POST, PUT, DELETE) by checking the integer role from Gin context.**
        - Admin: POST (Create), PUT (Update), DELETE (Delete).
        - All authenticated: GET (List/Single).
        - Success Criteria: API endpoints function correctly. `backend/api/handler/service.go` created and handlers implemented. Admin endpoints explicitly check `claims.Role >= model.RoleAdmin`. Integration tests pass. Routes in `api-router.go` uncommented.
    - **3.4**: Implement `toggle` endpoint (`/api/services/:id/toggle`) in `backend/api/handler/service.go`. **Ensure handler checks for Admin role.**
        - Success Criteria: Endpoint updates `enabled` field (not `is_active`) via model method. Handler verifies user role is Admin. Integration test passes.
    - **3.5**: Implement stub config copy endpoint (`/api/services/:id/config/:client`) in `backend/api/handler/service.go`. (Requires auth, no specific role check needed per current spec).
        - Success Criteria: Endpoint exists, requires authentication, and returns placeholder JSON `{"message": "config copy not implemented for this service/client"}`.

### 4. Backend: User Configuration Management (Core)
    - **Task Type: New Feature**
    - **4.1**: Define GORM models for `UserConfig`, `ConfigService` in `backend/model/`.
        - Success Criteria: `userconfig.go`, `configservice.go` defined with correct fields and relationships.
    - **4.2**: Add `UserConfig`, `ConfigService` to GORM `AutoMigrate`.
        - Success Criteria: `user_configs`, `config_services` tables created successfully in SQLite DB.
    - **4.3**: Implement CRUD methods for UserConfig model and API handlers (`/api/configs`) in `backend/api/handler/config.go`, protected by JWT auth. Check ownership within handlers.
        - Self: GET (List/Single), POST (Create), PUT (Update), DELETE (Delete).
        - Success Criteria: API endpoints function correctly. `backend/api/handler/config.go` created and handlers implemented. Users can only access/modify their own configs (check `user_id` from JWT against config's `user_id`). Integration tests pass. Routes in `api-router.go` uncommented.
    - **4.4**: Implement stub config export endpoint (`/api/configs/:id/:client`) in `backend/api/handler/config.go`.
        - Success Criteria: Endpoint exists, requires authentication, checks ownership, and returns placeholder JSON `{"message": "config export not implemented for this config/client"}`.

### 5. Backend: MCP Service Core (Placeholder) & Health Checks
    - **5.1**: Define basic directory structure for MCP service (`backend/infrastructure/proxy/`). (No actual service handling yet).
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

### 10. Backend: Thing ORM Model Tag & Index Feature Update
    - **10.1**: 升级 @thing.mdc 依赖，确保 model 标签支持省略 db 字段（自动 snake_case），并支持 index/unique 复合索引声明。
        - Success Criteria: 依赖已升级，model 结构体可省略 db 标签，index/unique 支持复合索引，相关单元测试通过。
    - **10.2**: 更新所有模型定义，去除冗余 db 标签，复合索引用法符合最新 Thing 规范。
        - Success Criteria: 所有模型文件已按新规范调整，代码可编译通过。
    - **10.3**: 验证 AutoMigrate 及索引创建逻辑，确保表结构和索引与模型定义一致。
        - Success Criteria: SQLite 数据库表结构和索引与模型定义完全一致，复合索引生效。

## Project Status Board (Phase 1)

- [ ] **Backend: Basic Setup & User Management (MVC Architecture)**
    - [X] 1.1: Adapt `gin-template` structure to MVC layout
    - [X] 1.2: Define/Refactor User model (with `Role int`), define Role constants, remove Role model
    - [X] 1.3: Implement initial DB schema (User only) with GORM AutoMigrate (SQLite)
    - [X] 1.4: Implement Admin user seeding logic
    - [ ] 1.5: Enhance User model with CRUD methods following MVC pattern
    - [ ] 1.6: Verify/Implement password hashing (bcrypt)
- [ ] **Backend: Authentication (JWT with Simplified Roles)**
    - [ ] 2.1: Implement JWT generation & /api/auth/login handler
    - [ ] 2.2: Implement JWT validation middleware
    - [ ] 2.3: Implement token refresh endpoint
    - [ ] 2.4: Implement logout mechanism (JWT blacklisting in Redis)
    - [ ] 2.5: Preserve and adapt captcha functionality for registration and login
- [ ] **Backend: MCP Service Management (Core)**
    - [X] 3.1: Define MCPService model
    - [X] 3.2: Add MCPService to AutoMigrate
    - [ ] 3.3: Implement CRUD methods for MCPService model and API handlers (Task Type: New Feature)
    - [ ] 3.4: Implement toggle endpoint (Task Type: New Feature)
    - [ ] 3.5: Implement stub config copy endpoint (Task Type: New Feature)
- [ ] **Backend: User Configuration Management (Core)**
    - [X] 4.1: Define UserConfig, ConfigService models
    - [X] 4.2: Add UserConfig, ConfigService to AutoMigrate
    - [ ] 4.3: Implement CRUD methods for UserConfig model and API handlers (Task Type: New Feature)
    - [ ] 4.4: Implement stub config export endpoint (Task Type: New Feature)
- [ ] **Backend: MCP Service Core (Placeholder) & Health Checks**
    - [ ] 5.1: Define basic service structure in `library/proxy/`
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
- [ ] **Backend: Thing ORM Model Tag & Index Feature Update**
    - [ ] 10.1: Upgrade @thing.mdc dependency to ensure model tags support omitting db field (auto snake_case) and support index/unique compound index declarations.
    - [ ] 10.2: Update all model definitions to remove redundant db tags and compound index usage to comply with latest Thing specification.
    - [ ] 10.3: Verify AutoMigrate and index creation logic to ensure table structure and index match model definitions.

## Current Status / Progress Tracking

*Plan refined to use MVC architecture instead of DDD approach. This better matches the existing gin-template structure and simplifies development.*
*   [DONE] 1.2: Defined User model in `backend/model/user.go` with integer Role field and constants.
*   **[DONE] 3.1**: Defined MCPService model in `backend/model/mcpservice.go`.
*   **[DONE] 4.1**: Defined UserConfig and ConfigService models in `backend/model/`.
*   **[FIXED]**: Login error resolved by updating SQLitePath in `backend/common/constants.go` to use `data/one-mcp.db` instead of `one-mcp.db` and creating the data directory at project root. This ensures the database file is created in a valid directory path.
*   **[DONE] 1.3**: Basic DB initialization is now working properly. The `one-mcp.db` SQLite database is created successfully in the data directory and can be accessed by the application.
*   **[DONE] 1.4**: Admin user seeding logic verified to be working correctly - creates a root user with username "root" and password "123456" with RoleRootUser.
*   **[DONE] 3.2**: Added MCPService model to GORM AutoMigrate in DB initialization, creating the mcp_services table.
*   **[DONE] 4.2**: Added UserConfig and ConfigService models to GORM AutoMigrate in DB initialization, creating the user_configs and config_services tables.
*   **[UPDATE]**: Modified technical architecture to use MVC pattern instead of the original DDD approach. Updated directory structure and module descriptions to reflect this change.

## Executor's Feedback or Assistance Requests

**Planner Update (Current Focus):**
The immediate next steps are to implement the API handlers and uncomment routes for:
1.  **MCP Service Management (Tasks 3.3, 3.4, 3.5):** Create `backend/api/handler/service.go` and implement the defined handlers for MCP services. This includes CRUD operations, toggling service status, and a stub for config copying. Ensure appropriate authentication (JWT) and authorization (Admin role checks for write operations).
2.  **User Configuration Management (Tasks 4.3, 4.4):** Create `backend/api/handler/config.go` and implement handlers for user configurations. This includes CRUD operations and a stub for config export. Ensure JWT authentication and ownership checks (users can only manage their own configs).

The `Task Type` for all these implementations is `New Feature`. Refer to the "High-level Task Breakdown" and "Project Status Board" for detailed success criteria.

*No prior requests at this time.*

## Lessons

*   Leveraging `gin-template` provides a useful starting point for the backend but better matches an MVC architecture than DDD.
*   Using GORM `AutoMigrate` with SQLite is suitable for initial development, simplifying schema management early on.
*   Clearly defining the replacement of template features (sessions, embedded UI) is important for planning.
*   Decision: Simplified role management using integer constants (`User=1`, `Admin=10`) in the `User` model instead of a separate `Role`