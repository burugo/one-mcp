# Model 层 context 重构与多语言支持

本任务旨在将所有需要 i18n 的 model 层方法签名统一加 context.Context，支持多用户多语言并发。

## Completed Tasks

- [x] 现有 model 层方法签名梳理
- [x] 批量修改 model 层方法签名，增加 context.Context `Task Type: Refactoring (Structural)`
- [x] 批量修改 handler 层调用，传递 c.Request.Context() `Task Type: Refactoring (Structural)`
- [x] 设计并注册 Gin 中间件注入 lang `Task Type: New Feature`
- [x] model 层通过 ctx.Value("lang") 获取语言 `Task Type: Refactoring (Functional)`
- [x] 修正/补充测试用例 `Task Type: Refactoring (Functional)`
- [x] 回归测试 `Task Type: Bug Fix`

## In Progress Tasks

- [ ] 文档和注释同步更新 `Task Type: Refactoring (Functional)`

## Future Tasks

- [ ] 文档和注释同步更新 `Task Type: Refactoring (Functional)`

## Implementation Plan

- 先从 user.go 相关方法和 handler 开始，逐步推广到其他 model。
- 中间件优先实现，便于后续 handler 层统一传递。
- 每步重构后及时回归测试，确保无回归。

### Relevant Files

- backend/model/user.go - 用户模型及相关方法
- backend/api/handler/user.go - 用户相关 handler
- backend/common/i18n.go - 国际化相关
- tests/basic_test.go - 测试用例 