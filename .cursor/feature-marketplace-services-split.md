# Marketplace/Services 页面分工与重构

确保 Service Marketplace 只做"发现/安装"，Services 页面只做"管理/配置/统计"，并补齐所有相关交互和数据流。

## Completed Tasks
- [x] 现状分析与页面结构梳理 `ref-func`

## In Progress Tasks
- [ ] 移除 Marketplace 页面"已安装（Installed）"tab及相关逻辑 `ref-struct`
- [ ] 只保留 All/NPM tab，聚焦服务发现与安装 `ref-struct`
- [ ] 检查/清理 fetchInstalledServices 相关逻辑，确保只在 Services 页面使用 `ref-struct`
- [ ] Services 页面将服务数据源从 mock 改为 store 的已安装服务（fetchInstalledServices） `ref-func`
- [ ] 支持"全部/Active/Inactive"tab，按服务状态过滤展示 `ref-func`
- [ ] 每个服务卡片支持：启用/禁用、配置（环境变量）、卸载、查看详情 `new-feat`
- [ ] 支持服务健康状态展示（如"Active/Inactive/Unhealthy"） `new-feat`
- [ ] 支持环境变量的查看与编辑（弹窗或内联） `new-feat`
- [ ] 支持服务的卸载操作（带确认弹窗） `new-feat`
- [ ] Usage Statistics 区块继续保留，数据可后续对接真实统计 `ref-func`
- [ ] 安装服务后自动刷新 Services 页面数据 `bug-fix`
- [ ] 配置/环境变量变更后自动保存并反馈 `bug-fix`
- [ ] 启用/禁用/卸载操作有明确的 UI 反馈和状态同步 `bug-fix`
- [ ] 检查/优化 store 状态管理，区分"可安装服务"与"已安装服务" `ref-func`
- [ ] 检查/补充相关 API（如 /installed、/uninstall、/update_env_vars） `bug-fix`
- [ ] Marketplace 页面更聚焦"发现/安装"，无管理入口 `ref-struct`
- [ ] Services 页面更聚焦"管理/配置"，无发现/安装入口（但可有"Add Service"按钮跳转 Marketplace） `ref-struct`

### 新增：服务安装后自动协议转换与 SSE/HTTP 路由注册（数据库驱动） `new-feat`
- [ ] 服务安装后，数据库插入/更新服务记录（含类型、env、状态等）。
- [ ] 主进程定期（或监听变更）从数据库拉取所有已安装服务，遍历注册：
  ```go
  func syncServicesFromDB() {
      services := db.GetAllInstalledServices()
      for _, svc := range services {
          if !isRegistered(svc.Name) {
              client := newMCPClient(svc)
              registerSSEEndpoint(svc.Name, client)
          }
      }
  }
  func main() {
      syncServicesFromDB() // 启动时同步
      go func() {
          for {
              time.Sleep(time.Minute)
              syncServicesFromDB()
          }
      }()
      // ... 启动 HTTP server 等
  }
  ```
- [ ] 每个 client 自动注册 SSE endpoint，如 `/firecrawl-mcp/message`。
- [ ] 支持热加载/动态注册（定时/监听 DB 变更）。
- [ ] 支持 graceful shutdown，关闭所有 client。
- [ ] 验收标准：
  - 服务安装后数据库有记录，主进程自动注册 SSE/HTTP endpoint。
  - /firecrawl-mcp/message 等 endpoint 能立即 remote 访问。
  - 支持多服务、并发、热加载（定时/监听 DB）。

## Future Tasks
- [ ] Usage Statistics 区块对接真实后端数据 `new-feat`
- [ ] 服务健康状态与监控完善 `new-feat`

## Implementation Plan
- Marketplace 页面移除"已安装"tab，activeTab 只保留 All/NPM。
- Services 页面对接 store 的 fetchInstalledServices，按状态过滤展示。
- 服务卡片支持启用/禁用、配置、卸载、健康状态、环境变量管理。
- 安装/配置/卸载等操作有 UI 反馈，状态同步。
- Usage Statistics 区块后续对接真实数据。

### Relevant Files
- `frontend/src/components/market/ServiceMarketplace.tsx`
- `frontend/src/pages/ServicesPage.tsx`
- `frontend/src/store/marketStore.ts`
- `backend/main.go`
- `backend/service/installer.go`
- `backend/db/`  # 数据库相关 