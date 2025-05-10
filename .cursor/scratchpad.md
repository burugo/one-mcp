# One MCP Service Development Plan

## Background and Motivation

The goal is to develop an One MCP (Multi-Cloud Platform) Service. This service will act as a central hub for managing and interacting with various MCP services. Users will be able to create "MCP Setups" (configurations sets), each containing multiple configured MCP service instances. A key feature is the ability to export these setups into JSON formats انرژی بخش specific to different client applications (e.g., Cursor, Cherry-Studio).

The system needs to support:
- Admins defining abstract MCP Service types (`MCPService`), including:
    - Schema for admin-level configurations (e.g., a shared API key, default root paths for a file service).
    - Default values for these admin-level configurations.
    - Schema for user-level configurations (parameters that users can customize or override).
    - Rules زيت on which admin-defined parameters users are allowed to override.
    - Templates for converting the final effective configuration (admin defaults + user overrides) into various client-specific JSON formats. Configuration parameters can be single-value (string) or multi-value (array of strings, often from multi-line UI input).
- Users creating their own MCP Setups (`UserConfig`), adding service instances (`ConfigService`) to these setups, and providing their own configuration values (`UserOverrideConfigValues`) that override admin defaults where permitted.
- Users exporting an MCP Setup for a chosen client type, triggering a backend process to:
    1. Retrieve all service instances within the setup.
    2. For each instance, calculate the final effective configuration by merging admin defaults with user overrides.
    3. Use the appropriate client template stored in `MCPService` to transform this effective configuration into the client-specific format.
    4. Aggregate all transformed service configurations into a single JSON output matching the target client's expected structure.

This approach aligns with the existing `architecture.md`'s concept of `UserConfig` as a collection and `ConfigService` as a join table, while significantly enhancing the detail and flexibility of service definition and configuration overrides.

## Key Challenges and Analysis

- **Complex Data Modeling**: Defining `MCPService`, `UserConfig`, and `ConfigService` to accurately represent abstract service types, admin-defined defaults/schemas, user-specific overrides, multi-value parameters, and client-specific export templates.
- **Configuration Merging Logic**: Implementing robust logic to merge `MCPService.DefaultAdminConfigValues` with `ConfigService.UserOverrideConfigValues` based on `MCPService.AllowUserOverride` rules.
- **Dynamic Template-Based Transformation**: Designing and implementing a flexible system (likely using Go's `text/template` or a similar templating engine) for `MCPService.ClientConfigTemplates` to convert a set of key-value parameters (which can include arrays for multi-value fields) into arbitrary client JSON structures. This includes handling iterations for array values if the client format requires it.
- **Schema Definition and Validation**: Defining how `AdminConfigSchema` and `UserConfigSchema` (likely JSON Schema) are stored and used, both for UI generation (frontend) and backend validation.
- **API Design**: Crafting APIs for:
    - Admins to manage `MCPService` definitions (schemas, defaults, client templates).
    - Users to manage their `UserConfig` setups and the `ConfigService` instances within them (including providing override values for single/multi-value fields).
    - Users to trigger the export-to-client-format functionality.
- **UI/UX for Configuration**: Frontend needs to dynamically render forms based on JSON schemas for both admin and user configurations, handling single-line and multi-line inputsgracefully (converting multi-line to string arrays).

## High-level Task Breakdown (Feature: MCP Setup Management & Export)

This new feature supersedes previous, simpler "UserConfig" plans.

### 1. Core Model Redesign & Implementation
    - **1.1**: Define/Refactor `MCPService` GORM model: `Task Type: New Feature`
        - Fields: `ID`, `Name`, `DisplayName`, `Type`, `AdminConfigSchema` (JSON/TEXT), `DefaultAdminConfigValues` (JSON/TEXT), `UserConfigSchema` (JSON/TEXT), `AllowUserOverride` (JSON/TEXT or specific structure), `ClientConfigTemplates` (JSON/TEXT).
        - Success Criteria: Model defined, migratable. Schemas and templates will store valid JSON.
    - **1.2**: Define/Refactor `UserConfig` (MCP Setup) GORM model: `Task Type: New Feature`
        - Fields: `ID`, `UserID` (FK), `Name`, `Description`.
        - Success Criteria: Model defined, migratable. Aligns with `architecture.md`.
    - **1.3**: Define/Refactor `ConfigService` (Service Instance in Setup) GORM model: `Task Type: New Feature`
        - Fields: `ID`, `UserConfigID` (FK), `MCPServiceID` (FK), `InstanceName` (optional, for user's reference), `UserOverrideConfigValues` (JSON/TEXT for user's specific overrides), `IsEnabled`.
        - Success Criteria: Model defined, migratable. `UserOverrideConfigValues` stores JSON.
    - **1.4**: Implement GORM `AutoMigrate` for the new/refactored models.
        - Success Criteria: Database schema created/updated successfully.

### 2. Admin: MCPService Definition Management
    - **2.1**: Design & Implement APIs for Admin to CRUD `MCPService` definitions: `Task Type: New Feature`
        - Endpoints: e.g., `POST /api/admin/mcp_services`, `GET /api/admin/mcp_services`, `GET /api/admin/mcp_services/:id`, `PUT /api/admin/mcp_services/:id`, `DELETE /api/admin/mcp_services/:id`.
        - Success Criteria: Admins can fully manage service types, including their schemas, default values, override rules, and client export templates. Secure with Admin role.
    - **2.2**: Implement backend logic for validating `AdminConfigSchema`, `UserConfigSchema` (e.g., ensure they are valid JSON Schema). `Task Type: New Feature`
        - Success Criteria: Invalid schemas are rejected.

### 3. User: MCP Setup & Service Instance Management
    - **3.1**: Design & Implement APIs for User to CRUD `UserConfig` (MCP Setups): `Task Type: New Feature`
        - Endpoints: e.g., `POST /api/mcp_setups`, `GET /api/mcp_setups`, `GET /api/mcp_setups/:id`, `PUT /api/mcp_setups/:id`, `DELETE /api/mcp_setups/:id`.
        - Success Criteria: Users can manage their named configuration sets. Ownership enforced.
    - **3.2**: Design & Implement APIs for User to CRUD `ConfigService` (Service Instances within a Setup): `Task Type: New Feature`
        - Endpoints: e.g., `POST /api/mcp_setups/:setup_id/service_instances` (body includes `MCPServiceID`, `InstanceName`, `UserOverrideConfigValues`), `GET /api/mcp_setups/:setup_id/service_instances/:instance_id`, `PUT /api/mcp_setups/:setup_id/service_instances/:instance_id` (update overrides, name, or is_enabled), `DELETE /api/mcp_setups/:setup_id/service_instances/:instance_id`.
        - Success Criteria: Users can add, configure (with overrides for single/multi-value fields), and remove service instances from their setups. `UserOverrideConfigValues` validated against `MCPService.UserConfigSchema` and `AllowUserOverride` rules.
    - **3.3**: Implement logic for handling multi-value inputs (e.g., ensuring string arrays are correctly stored in `UserOverrideConfigValues` from frontend-converted multi-line inputs). `Task Type: New Feature`
        - Success Criteria: Backend correctly stores and processes array-type config values.

### 4. Core: Configuration Merging & Client Export Logic
    - **4.1**: Implement "Effective Configuration" calculation logic: `Task Type: New Feature`
        - Function: `func calculateEffectiveConfig(mcpService model.MCPService, configService model.ConfigService) (map[string]interface{}, error)`
        - Logic: Merges `mcpService.DefaultAdminConfigValues` with `configService.UserOverrideConfigValues` respecting `mcpService.AllowUserOverride`.
        - Success Criteria: Correctly produces a single map of final key-value parameters for a service instance. Handles single and multi-value fields.
    - **4.2**: Design & Implement API for Exporting `UserConfig` to specific client format: `Task Type: New Feature`
        - Endpoint: `GET /api/mcp_setups/:setup_id/export?client_type=<client_name>` (e.g., "cursor", "cherry-studio").
        - Success Criteria: API authenticates user, verifies ownership of setup.
    - **4.3**: Implement core transformation logic using `MCPService.ClientConfigTemplates`: `Task Type: New Feature`
        - For each enabled `ConfigService` instance in the `UserConfig` setup:
            1. Calculate its effective configuration (using 4.1).
            2. Retrieve the `ClientConfigTemplate` for the target `client_type` from the `MCPService`.
            3. Apply the template (e.g., Go `text/template`) to the effective configuration to generate the client-specific JSON snippet for this service instance.
        - Success Criteria: Transformation logic correctly generates individual service config snippets. Template engine handles single values and iterating over multi-value arrays.
    - **4.4**: Implement aggregation of transformed snippets into final client JSON structure. `Task Type: New Feature`
        - Logic depends on the target client's top-level JSON structure (e.g., Cursor's `mcpServers` object).
        - Success Criteria: A valid, complete `config.json` string for the target client is produced and returned by the API.

### 5. Testing & Documentation
    - **5.1**: Write unit tests for model methods, config merging, and template transformation logic. `Task Type: New Feature`
    - **5.2**: Write integration tests for all new Admin and User APIs, including the export functionality with various client types and multi-value fields. `Task Type: New Feature`
    - **5.3**: Update API documentation. `Task Type: Refactoring (Functional)`
    - **5.4**: Add code comments. `Task Type: Refactoring (Functional)`

## Project Status Board

- **Active Task File**: `.cursor/feature-mcp-setup-management.md` (This file will be created next)
- **Current Focus**: Planning and design for MCP Setup Management & Multi-Client Export feature.

## Executor's Feedback or Assistance Requests

- 已完成 model 层 context 重构与多语言支持的所有任务：包括批量签名修改、handler 调用、Gin 中间件注入、model 层 ctx.Value("lang") 获取、测试用例修正与回归测试。
- 所有相关测试已全部通过。
- 等待 Planner/用户验收或进一步指示。

## Lessons

*   Leveraging `gin-template` provides a useful starting point for the backend but better matches an MVC architecture than DDD.
*   Using GORM `AutoMigrate` with SQLite is suitable for initial development, simplifying schema management early on.
*   Clearly defining the replacement of template features (sessions, embedded UI) is important for planning.
*   Decision: Simplified role management using integer constants (`User=1`, `Admin=10`) in the `User` model instead of a separate `Role`
*   **Internationalization (i18n)**: Implementing a robust i18n framework with error codes and language resource files allows for better error handling and future-proofs the application for international use. This approach separates error codes from their messages, making maintenance easier.
*   Passing `context.Context` is a standard and robust way to handle request-scoped data like language preferences in Go web services.
*   Client-specific configuration formats require a flexible backend that can store generic user inputs and transform them based on templates or rules for different export targets. Admin-defined defaults and user-level overrides add another layer of complexity and utility. Schemas (like JSON Schema) are crucial for managing the structure of these configurations.

## User Specified Lessons

- Include info useful for debugging in the program output.
- Read the file before you try to edit it.
- Always ask before using the -force git command
- For multi-value config items (e.g., root_path), use JSON arrays for storage and ensure UI/template logic can handle them.