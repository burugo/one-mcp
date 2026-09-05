# mcp-go v1.0.0 升级评估

评估日期：2026-09-05

当前项目版本：`github.com/mark3labs/mcp-go v0.57.0`

目标版本：`v1.0.0`

## 结论

**不建议在生产中只改 `go.mod` 直接升级。**

现有代码与 v1.0.0 在源码/API 层面兼容：已在隔离副本中把依赖替换为 v1.0.0，并用全新 Go build cache 执行 `go test ./...`，所有 Go package 通过。但 v1.0.0 将 `mcp.LATEST_PROTOCOL_VERSION` 从 `2025-11-25` 改为 `2026-07-28`，而 one-mcp 多处直接使用该常量初始化上游连接。一旦协商为新的无状态协议，`Client.Ping` 会直接返回成功，不再发出请求；one-mcp 当前基于 `Ping` 的健康检查、断线判断和自愈因此可能误报健康。

建议先做一个很小的选择，再升级：

1. **保守方案（推荐）**：将 one-mcp 上游 client 显式固定在 `mcp.ProtocolVersion20251125`，保留现有 session、`Ping` 和通知语义；然后升级依赖。
2. **采用 2026-07-28**：把健康检查改成真实 transport/application probe，并验证无 session 与 `subscriptions/listen` 对现有代理行为的影响后再上线。

## 上游变更范围

### v0.57.0 → v0.58.0

v0.58.0 是修复型版本，包括 Streamable HTTP `HEAD` 返回码、request-scoped elicitation 的 SSE 流归属、最终 SSE response flush、未声明 capability 时拒绝 resource subscribe，以及 JSON Schema 生成/转换修复。Release notes 未宣布 API 迁移要求。

- Release：<https://github.com/mark3labs/mcp-go/releases/tag/v0.58.0>
- 完整差异：<https://github.com/mark3labs/mcp-go/compare/v0.57.0...v0.58.0>

### v0.58.0 → v1.0.0

v1.0.0 的主要变化是支持 MCP `2026-07-28` specification，同时保留旧协议兼容。上游迁移文档明确说旧代码可保持不变，server 默认同时支持新旧两个时代，client `Initialize` 会先尝试 `server/discover`，不支持时回退到旧 `initialize` handshake。

- v1.0.0 release：<https://github.com/mark3labs/mcp-go/releases/tag/v1.0.0>
- 完整差异：<https://github.com/mark3labs/mcp-go/compare/v0.57.0...v1.0.0>
- 上游 2026-07-28 说明与 migration：<https://github.com/mark3labs/mcp-go/blob/v1.0.0/www/docs/pages/protocol-2026-07-28.mdx>
- 核心实现 PR：<https://github.com/mark3labs/mcp-go/pull/951>

2026-07-28 协议的行为变化包括：

- 从有 session 的 `initialize` / `notifications/initialized` 改为每个请求在 `_meta` 自描述，并通过 `server/discover` 发现 server。
- 现代协议不再使用 `Mcp-Session-Id`，`GET`/`DELETE` 的旧 session 语义被移除。
- `ping`、`logging/setLevel`、`resources/subscribe`、`resources/unsubscribe` 从现代协议移除；兼容 API 仍在，但现代连接上的行为已不同。
- 变更通知改为 `subscriptions/listen`；旧 `WithContinuousListening` 会自动将连接限定在旧协议。
- sampling / elicitation / roots 的 server-initiated request 迁移到 multi round-trip result。
- list/read result 增加 cache hints，默认仍要求 client 每次 revalidate。

协议常量和协商规则参见：

- <https://github.com/mark3labs/mcp-go/blob/v1.0.0/mcp/version.go>
- <https://github.com/mark3labs/mcp-go/blob/v1.0.0/client/client.go>
- <https://github.com/mark3labs/mcp-go/blob/v1.0.0/client/protocol.go>

## API 与依赖兼容性

### Go toolchain

v0.57.0 和 v1.0.0 的主 module 都声明 `go 1.25.5`，one-mcp 当前声明 `go 1.26.2`，因此没有新的 Go 版本阻塞。

- v0.57.0 `go.mod`：<https://github.com/mark3labs/mcp-go/blob/v0.57.0/go.mod>
- v1.0.0 `go.mod`：<https://github.com/mark3labs/mcp-go/blob/v1.0.0/go.mod>

v1.0.0 主 module 额外将 `github.com/rogpeppe/go-internal v1.14.1` 列为直接依赖，并增加 `golang.org/x/tools v0.26.0` 间接依赖。未发现与 one-mcp 现有 module graph 的版本冲突。

### one-mcp 实际使用的 API

隔离升级的编译和全量 Go 测试已覆盖项目当前用到的主要 API，包括：

- Client：`NewStdioMCPClientWithOptions`、`NewSSEMCPClient`、`NewStreamableHttpClient`、OAuth constructors、`Initialize`、`ListTools`、`CallTool`、`Ping`、`GetStderr`。
- Transport：HTTP client/header options、stdio `WithCommandFunc`、OAuth config/handler/token types。
- Server：`NewMCPServer`、`NewSSEServer`、`NewStreamableHTTPServer`、`AddTool`、`DeleteTools`、resource/prompt registration 和 capability options。
- MCP types：`Tool`、`ToolInputSchema`、list/call/read/prompt request/result、content types 和 `UnmarshalContent`。

上述签名在 v1.0.0 仍可编译。`MCPClient` interface 也保留原有方法，因此项目中的 fake client 无需为编译修改。

### 需要特别注意的行为

1. **`LATEST_PROTOCOL_VERSION` 不再是原协议。** one-mcp 在 proxy 和 npm/PyPI 工具发现中显式把 `InitializeRequest.Params.ProtocolVersion` 设为这个常量。升级后会主动尝试 2026-07-28，这不是单纯的内部替换。
2. **现代连接上 `Ping` 是 no-op。** v1.0.0 源码明确在 modern protocol 时直接返回 `nil`。one-mcp 用 `Ping` 更新 healthy/unhealthy 状态、触发网络服务重建，以及在 tool call timeout 后判断 transport 是否已坏。直接升级会使这些判断在 modern connection 上失效。
3. **OAuth challenge 有一处绕过 SDK 的手写初始化请求。** `backend/service/mcp_oauth_service.go` 直接把 `mcp.LATEST_PROTOCOL_VERSION` 填进 `initialize` body；升级后会形成 `initialize` + `2026-07-28` 的不合法组合，也不会经过 SDK 自动回退。即使多数鉴权中间件会先返回 401，也不应依赖所有上游都忽略请求语义；保守升级时这里也应固定为 `mcp.ProtocolVersion20251125`。
4. **初始化可多一次 probe。** `Initialize` 默认先发 `server/discover`；老 server 若正确返回 method-not-found，会立即回退。若老 server 忽略未知方法，上游默认最多等待 5 秒后再进行旧 handshake，会增加首次连接延迟。
5. **stdio stderr 处理变好但语义略变。** v1.0.0 由 transport 持续 drain 子进程 stderr，并通过 64 KiB drop-oldest ring buffer 提供 `GetStderr`，以避免 pipe-full deadlock。one-mcp 现有持续 scanner 仍能工作，但若未及时消费，旧 stderr 可被丢弃。
6. **server 端保留旧 client 兼容。** 现有 tool/resource/prompt 注册 API 不变，mcp-go 的 Streamable HTTP server 默认按每个请求的协议版本分流。因此 one-mcp 作为下游 MCP server 时，旧 client 不需同步升级。

相关一手源码：

- `Ping` 和 `Initialize`：<https://github.com/mark3labs/mcp-go/blob/v1.0.0/client/client.go>
- discover probe 和 5 秒默认超时：<https://github.com/mark3labs/mcp-go/blob/v1.0.0/client/protocol.go>
- stdio stderr drain：<https://github.com/mark3labs/mcp-go/blob/v1.0.0/client/transport/stdio.go>
- Streamable HTTP 新旧协议分流：<https://github.com/mark3labs/mcp-go/blob/v1.0.0/server/streamable_http.go>

## 验证记录

本次没有修改原工作区的 `go.mod`、`go.sum` 或业务代码。在临时隔离副本中执行：

```sh
go mod edit -require=github.com/mark3labs/mcp-go@v1.0.0
go mod tidy
GOCACHE="$(mktemp -d)" go test ./...
```

结果：根 module 及 `backend/api/handler`、`backend/common`、`backend/library/market`、`backend/library/proxy`、`backend/model`、`backend/service` 全部通过。这证明当前代码对 v1.0.0 **source-compatible**，但不能证明与真实新/旧 MCP server 的协议交互已通过运行时验证。

另对关键路径执行 `go test -race ./backend/library/proxy ./backend/service ./backend/api/handler`，三个 package 均通过。

## 建议的升级验收线

如采用保守方案，至少验证：

- stdio、SSE、Streamable HTTP 三种上游 transport 均能初始化、列工具和调工具。
- 断开上游后 `Ping` 健康检查仍能标记 unhealthy 并触发现有自愈。
- OAuth Streamable HTTP 初始化和 401/auth-required 判断不变。
- 下游旧版 client 与新版 client 都能连接 one-mcp；tool 动态禁用/启用的 `tools/list_changed` 行为仍正确。
- 重跑 `go test ./...` 和相关 race tests。
