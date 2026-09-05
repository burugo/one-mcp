# 禁用工具的 MCP 错误响应

核对日期：2026-09-05。当前依赖为 mcp-go v0.57.0，按 MCP 2025-11-25 核对。

## 选择

管理员全局禁用会把工具从可用列表移除。客户端引用旧工具名时，按工具不可用返回 JSON-RPC error；不把它报告为已经执行过的工具失败。

Streamable HTTP `/mcp` 返回 HTTP 200、`Content-Type: application/json`：

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "error": {
    "code": -32602,
    "message": "tool \"web_search_exa\" is disabled by administrator"
  }
}
```

`id` 原样保留客户端的数字或字符串。旧 SSE `/message` 则以 HTTP 202 空响应确认接收，JSON-RPC error 投递到已有会话的事件流。没有 ID 的通知不发送 JSON-RPC response。

拒绝发生在上游创建、配额检查和调用统计之前；写入 warning 日志，包含工具、用户和禁用原因，不记录 arguments。

## 官方依据与边界

- [Tools / Error Handling](https://modelcontextprotocol.io/specification/2025-11-25/server/tools#error-handling)：unknown tools 属于协议错误，官方例子使用 `-32602`；API、输入值校验和业务执行失败使用 `result.isError: true`。规范没有专门定义 tool-disabled 错误码，将已移除工具映射为 unavailable 是本项目的语义选择。
- [JSON-RPC 2.0 Response](https://www.jsonrpc.org/specification#response_object)：响应 ID MUST 与请求相同；不能把可识别的请求 ID 替换成 null。`-32600` 表示请求对象非法，不适合结构正确的禁用工具调用。
- [Streamable HTTP](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports#sending-messages-to-the-server)：请求可返回单个 application/json 对象或 SSE。规范没有逐字要求所有 JSON-RPC error 一律 HTTP 200；这里采用现用 SDK 的行为。
- [mcp-go 工具调用处理](https://github.com/mark3labs/mcp-go/blob/v0.57.0/server/server.go#L1936-L1951)：工具不存在或被过滤时返回 `INVALID_PARAMS` 并保留请求 ID；[HTTP 编码路径](https://github.com/mark3labs/mcp-go/blob/v0.57.0/server/streamable_http.go#L852-L859) 将此 RPC 响应作为 HTTP 200 JSON 返回。
- [Authorization / Scope Challenge Handling](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#scope-challenge-handling)：403 可用于 scope/权限不足，并配合 `WWW-Authenticate` challenge；不能笼统说 403 违反 MCP。本功能是全局工具可用性控制，不是要求用户重新授权更多 scope。

## 验证

回归测试覆盖 HTTP 数字、字符串及大整数 ID，JSON-RPC error 与 result 互斥、配额耗尽时仍先拒绝禁用调用、无新增调用统计、日志隐私，以及真实旧 SSE 会话上的错误事件投递。
