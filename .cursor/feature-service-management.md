# MCP服务管理

本文档详细描述了One MCP项目MCP服务管理功能的实现任务。

## 完成的任务

- [x] 3.1: 定义MCPService模型
- [x] 3.2: 将MCPService添加到AutoMigrate
- [x] 3.3: 实现MCPService模型的CRUD方法和API处理器
    - **Note**: CRUD操作需支持新的 `RequiredEnvVarsJSON`, `PackageManager`, `SourcePackageName`, `InstalledVersion` 字段。
    - GET API需返回这些新字段（尤其是 `RequiredEnvVarsJSON`）。
    - POST/PUT API需允许管理员定义/编辑 `RequiredEnvVarsJSON`。
- [x] 3.4: 实现toggle端点
    - **Note**: 已实现在 `ToggleMCPService` 处理器中，切换服务的启用/禁用状态。
    - 注册为 `POST /api/mcp_services/:id/toggle` 路由，仅管理员可访问。
    - 使用 `model.ToggleServiceEnabled(id)` 实现状态切换。
- [x] 3.5: 实现配置复制存根端点
    - **Note**: 已实现在 `GetMCPServiceConfig` 处理器中，获取特定客户端的服务配置模板。
    - 注册为 `GET /api/mcp_services/:id/config/:client` 路由，所有已认证用户可访问。
    - 能够基于模板和动态数据生成客户端特定的配置。

## 进行中的任务

暂无进行中的任务。

## 未来任务

- [x] 5.1: 在`library/proxy/`中定义基本服务结构
  - 已创建`service.go`定义服务接口和基本实现
  - 已创建`health_checker.go`实现健康检查管理器
  - 已创建`manager.go`实现服务管理器
- [x] 5.2: 实现基本健康检查逻辑
  - 实现了周期性健康检查
  - 添加了健康状态更新和查询接口
- [x] 5.3: 存储/更新健康状态
  - 为MCPService模型添加了`HealthStatus`, `LastHealthCheck`, `HealthDetails`字段
  - 实现了健康状态的存储和查询API

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
    HealthStatus         string    `db:"health_status" json:"health_status,omitempty"`
    LastHealthCheck      time.Time `db:"last_health_check" json:"last_health_check,omitempty"`
    HealthDetails        string    `db:"health_details" json:"health_details,omitempty"`
}
```

*(Note: `Config` field stores `ClientConfigTemplates` as a JSON string, as per `.cursor/feature-mcp-setup-management.md`. The `Type` field here is the underlying service type, not to be confused with `client_expected_protocol` or `our_proxy_protocol_for_this_client` which are part of `ClientConfigTemplates`.)*

### API处理器实现计划

已实现对MCPService模型的以下API处理器:

1. GET `/api/services` - 获取所有服务(所有已认证用户可访问)
2. GET `/api/services/:id` - 获取单个服务(所有已认证用户可访问)
3. POST `/api/services` - 创建新服务(仅管理员可访问)，支持定义环境变量和包管理器信息
4. PUT `/api/services/:id` - 更新服务(仅管理员可访问)，支持修改环境变量定义和包管理器信息
5. DELETE `/api/services/:id` - 删除服务(仅管理员可访问)
6. PUT `/api/services/:id/toggle` - 切换服务状态(仅管理员可访问)
7. GET `/api/services/:id/config/:client` - 复制服务配置(所有已认证用户可访问)

### 服务状态管理

服务状态管理包括:

1. 通过toggle端点启用/禁用服务
2. 实现健康检查逻辑监控服务状态
3. 存储和更新服务状态信息

## 相关文件

- `backend/model/mcp_service.go` - MCPService模型定义，已完善支持环境变量和包管理器信息
- `backend/api/handler/mcp_service.go` - 服务API处理器，已实现
- `backend/infrastructure/proxy/` (待实现) - 服务核心功能

# MCP 服务管理与全局客户端管理器

本文件跟踪 MCP 服务的安装、管理与全局客户端管理器的相关任务。

## Completed Tasks

- [x] 设计并实现全局 MCP 客户端管理器，支持服务注册、查询、卸载与进程管理 `Task Type: New Feature`
- [x] 修改 ListMCPServerTools、InstallNPMPackage、UninstallNPMPackage 等方法以集成全局管理器 `Task Type: Refactoring (Structural)`
- [x] 在 main.go 启动时自动加载数据库中已安装的 MCP 服务并注册到全局管理器 `Task Type: New Feature`
- [x] 实现服务卸载时自动移除并关闭对应 MCP 客户端进程 `Task Type: Bug Fix`
- [x] 为全局管理器添加单元测试，覆盖初始化、错误处理、移除等核心逻辑 `Task Type: New Feature`
- [x] 修复 RemoveClient 对 nil client 的 panic 问题 `Task Type: Bug Fix`

## In Progress Tasks

- [ ] 集成测试：mock 或真实 MCP server 端到端验证（后续可选） `Task Type: New Feature`

## Future Tasks

- [ ] 性能优化与资源回收机制 `Task Type: Refactoring (Functional)`
- [ ] 支持多种类型的 MCP 客户端（如 SSE/HTTP） `Task Type: New Feature`

## Implementation Plan

本阶段已完成：
- 统一管理所有 MCP 客户端实例，避免重复进程与资源泄漏。
- 支持服务的注册、查询、卸载与进程优雅关闭。
- 通过依赖注入与 Mock，提升了单元测试的可测性。
- 关键接口与主流程已通过测试验证。

### Relevant Files

- backend/library/market/client_manager.go - 全局 MCP 客户端管理器实现 ✅
- backend/library/market/npm.go - 相关方法集成全局管理器 ✅
- backend/library/market/installation.go - 服务状态更新与健康检查 ✅
- main.go - 启动与优雅关闭集成 ✅
- backend/library/market/client_manager_test.go - 单元测试 ✅ 