# 用户特定服务 ENV 支持与 handler 动态管理

支持每个用户为服务配置独立 ENV，动态创建并缓存 handler，提升多租户能力。

## Completed Tasks

- [x] 支持用户特定 ENV 查询与合并 `new-feat`
- [x] handler 动态创建与缓存（用户+服务维度）`new-feat`
- [x] 单元测试覆盖用户特定 handler 路径与配置合并 `test`
- [x] SSEProxyHandler 结构重构，逻辑分离 `ref-func`

## In Progress Tasks

- [ ] handler 生命周期与资源释放机制设计 `design`
- [ ] handler 缓存过期与自动清理实现 `new-feat`

## Future Tasks

- [ ] 支持更多服务类型的用户特定配置（如 SSE/HTTP）`new-feat`
- [ ] 管理端 ENV 配置可视化与批量管理 `new-feat`
- [ ] 性能监控与资源占用统计 `improve`

## Implementation Plan

- 用户请求时，优先查找用户特定 handler，无则动态创建并缓存。
- ENV 合并顺序：用户配置 > 服务默认 ENV > 服务默认配置 ENV。
- handler 缓存采用 `user-{userID}-service-{serviceID}` 及 `global-service-{serviceID}` 区分。
- 单元测试通过 mock handler 及 mock `proxy.NewStdioSSEHandlerUncached` 验证分支与合并逻辑。
- 后续需设计 handler 生命周期管理与缓存过期策略，避免资源泄漏。

### Relevant Files

- backend/api/handler/proxy_handler.go - 主 handler 逻辑与重构
- backend/api/handler/proxy_handler_test.go - 单元测试
- backend/library/proxy/service.go - handler 创建与缓存
- backend/model/user_config.go, config_service.go - 配置模型 