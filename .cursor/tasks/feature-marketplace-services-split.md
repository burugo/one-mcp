# Marketplace/Services 页面分工与重构

确保 Service Marketplace 只做"发现/安装"，Services 页面只做"管理/配置/统计"，并补齐所有相关交互和数据流。

## Completed Tasks
- [x] 现状分析与页面结构梳理 `ref-func`
- [x] 支持环境变量的查看与编辑（弹窗或内联） `new-feat`

## In Progress Tasks
- [ ] 移除 Marketplace 页面"已安装（Installed）"tab及相关逻辑 `ref-struct`
- [ ] 只保留 All/NPM tab，聚焦服务发现与安装 `ref-struct`
- [ ] 检查/清理 fetchInstalledServices 相关逻辑，确保只在 Services 页面使用 `ref-struct`
- [ ] Services 页面将服务数据源从 mock 改为 store 的已安装服务（fetchInstalledServices） `ref-func`
- [ ] 支持"全部/Active/Inactive"tab，按服务状态过滤展示 `ref-func`
- [ ] 每个服务卡片支持：启用/禁用、配置（环境变量）、卸载、查看详情 `new-feat`
- [ ] 支持服务健康状态展示（如"Active/Inactive/Unhealthy"） `new-feat`
- [ ] 支持服务的卸载操作（带确认弹窗） `new-feat`
- [ ] Usage Statistics 区块继续保留，数据可后续对接真实统计 `ref-func`
- [ ] 安装服务后自动刷新 Services 页面数据 `bug-fix`
- [ ] 配置/环境变量变更后自动保存并反馈 `bug-fix`
- [ ] 启用/禁用/卸载操作有明确的 UI 反馈和状态同步 `bug-fix`
- [ ] 检查/优化 store 状态管理，区分"可安装服务"与"已安装服务" `ref-func`
- [ ] 检查/补充相关 API（如 /installed、/uninstall、/update_env_vars） `bug-fix`
- [ ] Marketplace 页面更聚焦"发现/安装"，无管理入口 `ref-struct`
- [ ] Services 页面更聚焦"管理/配置"，无发现/安装入口（但可有"Add Service"按钮跳转 Marketplace） `ref-struct`
- [ ] 服务安装后自动协议转换与 SSE/HTTP 路由注册（数据库驱动） `new-feat`
- [ ] 新增：后端支持 PATCH /mcp_market/env_var，保存单个服务环境变量 `new-feat`
  - 参数：service_id, var_name, var_value
  - 验收标准：前端可单独保存任意变量，后端持久化并返回成功
- [ ] 后端所有服务相关 API 返回需补充 env_vars 字段，内容为该服务所有环境变量键值对 `bug-fix`
  - 验收标准：前端配置弹窗能正常显示和编辑环境变量
- [ ] 环境变量共享/私有机制后端实现（ConfigService 增加 is_shared 字段，接口权限校验与优先级逻辑） `new-feat`
- [ ] PATCH /api/mcp_market/env_var 支持单独保存变量，权限校验（仅管理员可切换共享，普通用户只能保存私有值） `new-feat`
- [ ] 前端 ServiceConfigModal 支持锁图标、共享状态切换、只读/可编辑渲染 `ref-func`
- [ ] 多用户场景下环境变量查询/保存优先级（用户私有 > 管理员共享 > 空） `bug-fix`

## Future Tasks
- [ ] Usage Statistics 区块对接真实后端数据 `new-feat`
- [ ] 服务健康状态与监控完善 `new-feat`

## Implementation Plan
- Marketplace 页面移除"已安装"tab，activeTab 只保留 All/NPM。
- Services 页面对接 store 的 fetchInstalledServices，按状态过滤展示。
- 服务卡片支持启用/禁用、配置、卸载、健康状态、环境变量管理。
- 安装/配置/卸载等操作有 UI 反馈，状态同步。
- Usage Statistics 区块后续对接真实数据。
- 环境变量无默认值，只有"管理员共享"与"用户私有"两种模式：
    - ConfigService 增加 is_shared 字段，后端接口 PATCH /env_var 支持权限校验。
    - 查询时优先级：用户私有 > 管理员共享 > 空。
    - 管理员可切换共享状态，普通用户只能填写自己的值，不能更改共享状态。
    - 前端 ServiceConfigModal 按权限渲染锁图标、只读/可编辑。
    - 多用户场景下确保数据隔离与权限安全。

### Relevant Files
- `frontend/src/components/market/ServiceMarketplace.tsx`
- `frontend/src/pages/ServicesPage.tsx`
- `frontend/src/store/marketStore.ts`
- `frontend/src/components/market/ServiceConfigModal.tsx`
- `backend/main.go`
- `backend/service/installer.go`
- `backend/api/handler/market.go`
- `backend/db/`  # 数据库相关 