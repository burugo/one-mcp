# MCP Setup Management & Multi-Client Export with Proxying

This document outlines tasks for implementing MCP Setup Management. Users configure underlying MCP services (`stdio`, `sse`, `streamable_http`) and access them via this One MCP service's own SSE or Streamable HTTP proxy endpoints. Setups are exported to client-specific JSON. The proxy protocol (SSE/Streamable HTTP) used in the export is fixed per client type.

## Completed Tasks

- [x] Initial planning, design, and refinements based on user feedback.
- [x] **1.1**: Define/Refactor `MCPService` GORM model `Task Type: New Feature`
    - Fields: `ID`, `Name`, `DisplayName`, `Type` (enum: `stdio`, `sse`, `streamable_http`), `AdminConfigSchema`, `DefaultAdminConfigValues`, `UserConfigSchema`, `AllowUserOverride`, `ClientConfigTemplates` (JSON/TEXT - Map of `client_type` to `{ "template_string": string, "client_expected_protocol": string, "our_proxy_protocol_for_this_client": string }`).
    - Success Criteria: Model defined using Thing ORM. `ClientConfigTemplates` stores new simplified structure determining client-fixed proxy protocol and URL format (e.g. with `/sse` or `/mcp` path suffix).
- [x] **1.2**: Define/Refactor `UserConfig` GORM model. `Task Type: New Feature`
- [x] **1.3**: Define/Refactor `ConfigService` GORM model. `Task Type: New Feature`
- [x] **1.4**: Implement GORM `AutoMigrate`. `Task Type: New Feature`

## In Progress Tasks

### 2. Admin: MCPService Definition Management
- [ ] **2.1**: APIs for Admin to CRUD `MCPService` (including simplified `ClientConfigTemplates` structure). `Task Type: New Feature`
    - Success Criteria: Admins can fully manage service types, explicitly defining `our_proxy_protocol_for_this_client` for each client template.

### 3. User: MCP Setup & Service Instance Management
- [ ] **3.1**: Design & Implement APIs for User to CRUD `UserConfig` (MCP Setups). `Task Type: New Feature`
- [ ] **3.2**: Design & Implement APIs for User to CRUD `ConfigService` (Service Instances). `Task Type: New Feature`
- [ ] **3.3**: Implement logic for handling multi-value inputs. `Task Type: New Feature`

### 4. Core: Configuration Merging & Client Export Logic
- [ ] **4.1**: Implement "Effective Configuration" calculation for an underlying service instance. `Task Type: New Feature`
- [ ] **4.2**: Design & Implement API for Exporting `UserConfig`: `Task Type: New Feature`
    - Endpoint: `GET /api/mcp_setups/:setup_id/export?client_type=<client_name>` (No `protocol` query param is needed as it's fixed by `client_type`).
    - Logic: Retrieves `ClientTemplateDetail` for `client_name`. The `effective_our_proxy_protocol` is directly set from `templateDetail.our_proxy_protocol_for_this_client`.
    - Success Criteria: API authenticates user, verifies ownership, and correctly determines `effective_our_proxy_protocol` solely from the `client_type`'s stored template configuration.
- [ ] **4.3**: Implement core transformation logic: `Task Type: New Feature`
    - For each enabled `ConfigService` instance:
        1. Calculate its effective configuration for the underlying service.
        2. Get `ClientTemplateDetail` for the target `client_type`.
        3. Set `effective_our_proxy_protocol = templateDetail.our_proxy_protocol_for_this_client`.
        4. Prepare data for the template: effective config of the underlying service, and `EffectiveOurProxyProtocol: effective_our_proxy_protocol`. The template will use `EffectiveOurProxyProtocol` to construct the correct proxy URL, appending `/sse` or `/mcp` as appropriate (e.g., `https://our.service.com/api/proxy/:instance_id/{{ if eq .EffectiveOurProxyProtocol "sse" }}sse{{ else }}mcp{{ end }}`).
        5. Render `templateDetail.template_string`. The template generates a service entry structure consistent with `templateDetail.client_expected_protocol`.
    - Success Criteria: Generates correct client-specific snippets with proxy URLs correctly reflecting the client-fixed protocol (e.g., containing `/sse` or `/mcp` path segments).
- [ ] **4.4**: Implement aggregation of transformed snippets into final client JSON. `Task Type: New Feature`

### 5. Core: Service Proxying Runtime
- [ ] **5.1**: Design proxy endpoints with protocol-specific paths: e.g., `/api/proxy/:instance_id/sse` and `/api/proxy/:instance_id/mcp`. `Task Type: New Feature`
    - Success Criteria: Proxy URLs are distinct for SSE and Streamable HTTP, matching what the export template logic generates.
- [ ] **5.2**: Implement request handling for these distinct proxy endpoints (e.g., one handler for `/api/proxy/:instance_id/sse`, another for `/api/proxy/:instance_id/mcp`). `Task Type: New Feature`
- [ ] **5.3**: Implement dispatch logic based on `MCPService.Type` of the *underlying* service, invoked by the specific protocol handler from 5.2. `Task Type: New Feature`
- [ ] **5.4**: Handle parameter mapping from proxy to underlying service. `Task Type: New Feature`
    - Success Criteria: One MCP service successfully proxies requests to underlying stdio, sse, and streamable_http services, exposing them via SSE or Streamable HTTP based on the specific proxy path suffix (`/sse` or `/mcp`) invoked by the client.

### 6. Testing & Documentation
- [ ] **6.1**: Write unit tests for models, merging, transformation (including protocol choices), and proxying. `Task Type: New Feature`
- [ ] **6.2**: Write integration tests for APIs (including export with different client_type/protocol combinations) and proxy endpoints. `Task Type: New Feature`
- [ ] **6.3**: Update API documentation. `Task Type: Refactoring (Functional)`
- [ ] **6.4**: Add code comments. `Task Type: Refactoring (Functional)`

## Future Tasks

- [ ] UI/UX for Admin `MCPService` management (including an intuitive editor for the simplified `ClientConfigTemplates`).
- [ ] UI/UX for User MCP Setup and Service Instance management.
- [ ] Detailed error handling and reporting for the proxy layer.

## Implementation Plan

- Prioritize backend model updates (simplified `ClientConfigTemplates` structure) and core logic (merging, transformation logic aware of client-fixed protocol, initial proxy structure with distinct paths for SSE/Streamable HTTP).
- Develop Admin APIs for `MCPService` definition.
- Implement User APIs for managing setups and instances.
- Develop the core proxying runtime logic, ensuring handlers for `/sse` and `/mcp` paths correctly dispatch to underlying services.
- Implement the simplified export API.
- Testing will be conducted incrementally alongside development.

### Relevant Files

- `backend/model/mcp_service.go` (updated for simplified `ClientConfigTemplates` structure)
- `backend/model/user_config.go`
- `backend/model/config_service.go`
- `backend/api/handler/admin_mcp_service.go`
- `backend/api/handler/user_mcp_setup.go`
- `backend/api/handler/export.go` (updated export logic, no protocol param)
- `backend/api/handler/proxy.go` (or separate `proxy_sse.go`, `proxy_mcp.go` if cleaner, handling distinct `/sse` and `/mcp` paths)
- `backend/api/route/api-router.go` (routes for distinct proxy paths)
- `tests/` (updated test files for new logic, simplified export, and distinct proxy paths) 