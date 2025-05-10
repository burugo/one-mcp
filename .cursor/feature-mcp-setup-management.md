# MCP Setup Management & Multi-Client Export

This document outlines the tasks for implementing the MCP Setup Management feature, allowing users to create sets of configured MCP service instances and export them into client-specific JSON formats. This includes support for admin-defined service defaults, user overrides, and handling of multi-value configuration parameters.

## Completed Tasks

- [ ] Initial planning and design based on user feedback and architecture review.

## In Progress Tasks

### 1. Core Model Redesign & Implementation
- [ ] **1.1**: Define/Refactor `MCPService` GORM model `Task Type: New Feature`
    - Fields: `ID`, `Name`, `DisplayName`, `Type`, `AdminConfigSchema` (JSON/TEXT), `DefaultAdminConfigValues` (JSON/TEXT), `UserConfigSchema` (JSON/TEXT), `AllowUserOverride` (JSON/TEXT or specific structure), `ClientConfigTemplates` (JSON/TEXT).
    - Success Criteria: Model defined, migratable. Schemas and templates will store valid JSON.
- [ ] **1.2**: Define/Refactor `UserConfig` (MCP Setup) GORM model `Task Type: New Feature`
    - Fields: `ID`, `UserID` (FK), `Name`, `Description`.
    - Success Criteria: Model defined, migratable. Aligns with `architecture.md`.
- [ ] **1.3**: Define/Refactor `ConfigService` (Service Instance in Setup) GORM model `Task Type: New Feature`
    - Fields: `ID`, `UserConfigID` (FK), `MCPServiceID` (FK), `InstanceName` (optional, for user's reference), `UserOverrideConfigValues` (JSON/TEXT for user's specific overrides), `IsEnabled`.
    - Success Criteria: Model defined, migratable. `UserOverrideConfigValues` stores JSON.
- [ ] **1.4**: Implement GORM `AutoMigrate` for the new/refactored models. `Task Type: New Feature`
    - Success Criteria: Database schema created/updated successfully.

### 2. Admin: MCPService Definition Management
- [ ] **2.1**: Design & Implement APIs for Admin to CRUD `MCPService` definitions `Task Type: New Feature`
    - Endpoints: e.g., `POST /api/admin/mcp_services`, `GET /api/admin/mcp_services`, `GET /api/admin/mcp_services/:id`, `PUT /api/admin/mcp_services/:id`, `DELETE /api/admin/mcp_services/:id`.
    - Success Criteria: Admins can fully manage service types, including their schemas, default values, override rules, and client export templates. Secure with Admin role.
- [ ] **2.2**: Implement backend logic for validating `AdminConfigSchema`, `UserConfigSchema` (e.g., ensure they are valid JSON Schema). `Task Type: New Feature`
    - Success Criteria: Invalid schemas are rejected.

### 3. User: MCP Setup & Service Instance Management
- [ ] **3.1**: Design & Implement APIs for User to CRUD `UserConfig` (MCP Setups) `Task Type: New Feature`
    - Endpoints: e.g., `POST /api/mcp_setups`, `GET /api/mcp_setups`, `GET /api/mcp_setups/:id`, `PUT /api/mcp_setups/:id`, `DELETE /api/mcp_setups/:id`.
    - Success Criteria: Users can manage their named configuration sets. Ownership enforced.
- [ ] **3.2**: Design & Implement APIs for User to CRUD `ConfigService` (Service Instances within a Setup) `Task Type: New Feature`
    - Endpoints: e.g., `POST /api/mcp_setups/:setup_id/service_instances` (body includes `MCPServiceID`, `InstanceName`, `UserOverrideConfigValues`), `GET /api/mcp_setups/:setup_id/service_instances/:instance_id`, `PUT /api/mcp_setups/:setup_id/service_instances/:instance_id` (update overrides, name, or is_enabled), `DELETE /api/mcp_setups/:setup_id/service_instances/:instance_id`.
    - Success Criteria: Users can add, configure (with overrides for single/multi-value fields), and remove service instances from their setups. `UserOverrideConfigValues` validated against `MCPService.UserConfigSchema` and `AllowUserOverride` rules.
- [ ] **3.3**: Implement logic for handling multi-value inputs (e.g., ensuring string arrays are correctly stored in `UserOverrideConfigValues` from frontend-converted multi-line inputs). `Task Type: New Feature`
    - Success Criteria: Backend correctly stores and processes array-type config values.

### 4. Core: Configuration Merging & Client Export Logic
- [ ] **4.1**: Implement "Effective Configuration" calculation logic `Task Type: New Feature`
    - Function: `func calculateEffectiveConfig(mcpService model.MCPService, configService model.ConfigService) (map[string]interface{}, error)`
    - Logic: Merges `mcpService.DefaultAdminConfigValues` with `configService.UserOverrideConfigValues` respecting `mcpService.AllowUserOverride`.
    - Success Criteria: Correctly produces a single map of final key-value parameters for a service instance. Handles single and multi-value fields.
- [ ] **4.2**: Design & Implement API for Exporting `UserConfig` to specific client format `Task Type: New Feature`
    - Endpoint: `GET /api/mcp_setups/:setup_id/export?client_type=<client_name>` (e.g., "cursor", "cherry-studio").
    - Success Criteria: API authenticates user, verifies ownership of setup.
- [ ] **4.3**: Implement core transformation logic using `MCPService.ClientConfigTemplates` `Task Type: New Feature`
    - For each enabled `ConfigService` instance in the `UserConfig` setup:
        1. Calculate its effective configuration (using 4.1).
        2. Retrieve the `ClientConfigTemplate` for the target `client_type` from the `MCPService`.
        3. Apply the template (e.g., Go `text/template`) to the effective configuration to generate the client-specific JSON snippet for this service instance.
    - Success Criteria: Transformation logic correctly generates individual service config snippets. Template engine handles single values and iterating over multi-value arrays.
- [ ] **4.4**: Implement aggregation of transformed snippets into final client JSON structure. `Task Type: New Feature`
    - Logic depends on the target client's top-level JSON structure (e.g., Cursor's `mcpServers` object).
    - Success Criteria: A valid, complete `config.json` string for the target client is produced and returned by the API.

### 5. Testing & Documentation
- [ ] **5.1**: Write unit tests for model methods, config merging, and template transformation logic. `Task Type: New Feature`
- [ ] **5.2**: Write integration tests for all new Admin and User APIs, including the export functionality with various client types and multi-value fields. `Task Type: New Feature`
- [ ] **5.3**: Update API documentation. `Task Type: Refactoring (Functional)`
- [ ] **5.4**: Add code comments. `Task Type: Refactoring (Functional)`

## Future Tasks

- [ ] UI/UX implementation for Admin `MCPService` management.
- [ ] UI/UX implementation for User MCP Setup and Service Instance management, including dynamic form generation from schemas and handling of multi-line inputs for array values.

## Implementation Plan

- Prioritize backend model and core logic (merging, transformation) implementation.
- Develop Admin APIs for `MCPService` definition as a foundational step.
- Implement User APIs for managing setups and instances.
- Finally, implement the export API and its underlying transformation and aggregation logic.
- Testing will be conducted incrementally alongside development.

### Relevant Files

- `backend/model/mcp_service.go` (to be created/refactored)
- `backend/model/user_config.go` (to be refactored as MCPSetup)
- `backend/model/config_service.go` (to be refactored as ServiceInstanceInSetup)
- `backend/api/handler/admin_mcp_service.go` (to be created)
- `backend/api/handler/user_mcp_setup.go` (to be created)
- `backend/api/handler/export.go` (or similar, for export logic)
- `backend/api/route/api-router.go` (to be updated with new routes)
- `tests/` (new test files for models and APIs) 