# MCP服务管理

本文档详细描述了One MCP项目MCP服务管理功能的实现任务。

## 完成的任务

- [x] 3.1: 定义MCPService模型
- [x] 3.2: 将MCPService添加到AutoMigrate

## 进行中的任务

暂无进行中的任务。

## 未来任务

- [ ] 3.3: 实现MCPService模型的CRUD方法和API处理器
    - **Note**: CRUD操作需支持新的 `RequiredEnvVarsJSON`, `PackageManager`, `SourcePackageName`, `InstalledVersion` 字段。
    - GET API需返回这些新字段（尤其是 `RequiredEnvVarsJSON`）。
    - POST/PUT API需允许管理员定义/编辑 `RequiredEnvVarsJSON`。
- [ ] 3.4: 实现toggle端点
- [ ] 3.5: 实现配置复制存根端点
- [ ] 5.1: 在`library/proxy/`中定义基本服务结构
- [ ] 5.2: 实现基本健康检查逻辑(占位符)
- [ ] 5.3: 存储/更新健康状态(占位符，例如MCPService中的新字段)

### Marketplace & Installation APIs (New Section for Future Tasks)
- [ ] **MKT-1**: Design and Implement `GET /api/mcp_market/search` API `Task Type: New Feature`
    - Purpose: Search services from npm, PyPI, and recommended list.
    - Params: `query`, `sources`, pagination.
    - Response: List of service candidates with details (`name`, `version`, `description`, `package_name`, `package_manager`, `source_url`, `icon_url`, `is_installed`).
- [ ] **MKT-2**: Design and Implement `POST /api/mcp_market/install_or_add_service` API `Task Type: New Feature`
    - Purpose: Install `stdio` services via `npx`/`uvx` OR add predefined `remote`/`stdio` services.
    - Params: `sourceType` ("marketplace" or "predefined"), package details if marketplace, `mcpServiceID` if predefined, `userProvidedEnvVars`.
    - Logic: Installs package, auto-creates/updates `MCPService` (for new marketplace stdio), creates `ConfigService` for user, stores env vars.
- [ ] **MKT-3**: Design and Implement `GET /api/mcp_market/discover_env_vars` API `Task Type: Research/New Feature`
    - Purpose: Attempt to discover required env vars from npm/PyPI package pages/readmes for `stdio` services.
    - Params: `packageName`, `packageManager`.
    - Response: Suggested env vars list.
- [ ] **MKT-4**: Design and Implement `GET /api/mcp_market/installed` API `Task Type: New Feature`
    - Purpose: List user's configured service instances (linked to `ConfigService`).
    - Response: List of services with `packageName`, `installedVersion`, `packageManager`, `status`, actions.
- [ ] **MKT-5**: Design and Implement `POST /api/mcp_market/uninstall` API `Task Type: New Feature`
    - Purpose: Uninstall `stdio` package and remove related `ConfigService`.
    - Params: `packageName`, `packageManager` (or `configServiceID`).
- [ ] **MKT-6**: (Optional) Design and Implement `GET /api/mcp_market/install/status/:task_id` API `Task Type: New Feature`
    - Purpose: Poll status for asynchronous installations.

## 实现计划

### MCPService模型

已成功定义MCPService模型并添加到AutoMigrate。模型结构如下:

```go
type MCPService struct {
    ID                   int64     `db:"id" json:"id"`
    Name                 string    `db:"name" json:"name"`
    Type                 string    `db:"type" json:"type"` // e.g., "stdio", "remote", "sse", "streamable_http"
    Description          string    `db:"description" json:"description"`
    Enabled              bool      `db:"enabled" json:"enabled"`
    Config               string    `db:"config" json:"config"` // Stores ClientConfigTemplates JSON string
    RequiredEnvVarsJSON  string    `db:"required_env_vars_json" json:"required_env_vars_json,omitempty"` // JSON array of {name, description, is_secret, optional, default_value}
    PackageManager       string    `db:"package_manager" json:"package_manager,omitempty"` // e.g., "npm", "pypi", filled if installed from marketplace
    SourcePackageName    string    `db:"source_package_name" json:"source_package_name,omitempty"` // e.g., "@org/package", filled if installed from marketplace
    InstalledVersion     string    `db:"installed_version" json:"installed_version,omitempty"` // Version installed from marketplace
    CreatedAt            time.Time `db:"created_at" json:"created_at"`
    UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}
```

*(Note: `Config` field stores `ClientConfigTemplates` as a JSON string, as per `.cursor/feature-mcp-setup-management.md`. The `Type` field here is the underlying service type, not to be confused with `client_expected_protocol` or `our_proxy_protocol_for_this_client` which are part of `ClientConfigTemplates`.)*

### API处理器实现计划

将为MCPService模型实现以下API处理器:

1. GET `/api/services` - 获取所有服务(所有已认证用户可访问)
2. GET `/api/services/:id` - 获取单个服务(所有已认证用户可访问)
3. POST `/api/services` - 创建新服务(仅管理员可访问)
4. PUT `/api/services/:id` - 更新服务(仅管理员可访问)
5. DELETE `/api/services/:id` - 删除服务(仅管理员可访问)
6. PUT `/api/services/:id/toggle` - 切换服务状态(仅管理员可访问)
7. GET `/api/services/:id/config/:client` - 复制服务配置(所有已认证用户可访问)

### 服务状态管理

服务状态管理包括:

1. 通过toggle端点启用/禁用服务
2. 实现健康检查逻辑监控服务状态
3. 存储和更新服务状态信息

## 相关文件

- `backend/model/mcpservice.go` - MCPService模型定义
- `backend/api/handler/service.go` (待实现) - 服务API处理器
- `backend/infrastructure/proxy/` (待实现) - 服务核心功能 