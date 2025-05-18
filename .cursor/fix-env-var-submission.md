# Fix Environment Variable Submission

Ensure environment variables entered in the modal are correctly submitted during service installation.

## Completed Tasks
- [x] Initial analysis of data flow `ref-func`
- [x] Previous fix for re-enabling install button `bug-fix`
- [x] Add detailed logging to trace env var propagation from modal to API request `bug-fix`
- [x] Identify and fix the point of variable loss or incorrect merging if any `bug-fix`
- [x] Confirm correct HTTP request payload includes all accumulated and newly input variables `bug-fix`
- [x] 移除所有调试日志，恢复生产代码 `ref-func`
- [x] 验证后端 /api/mcp_market/installed 能正确返回 firecrawl-mcp，数据链路闭环 `bug-fix`

## In Progress Tasks
- [ ] 无

## Future Tasks
- [ ] N/A

## Implementation Plan

（已全部完成，详见上方 Completed Tasks）

### 验收说明
- 用户输入的环境变量已能完整传递到后端，/api/mcp_market/installed 能正确返回 firecrawl-mcp。
- 安装流程无卡死，按钮状态切换正常。
- 代码已无调试日志，状态管理健全。
- 功能闭环，满足最初需求。

### Relevant Files
- `frontend/src/components/market/ServiceMarketplace.tsx`
- `frontend/src/components/market/EnvVarInputModal.tsx`
- `frontend/src/store/marketStore.ts` 