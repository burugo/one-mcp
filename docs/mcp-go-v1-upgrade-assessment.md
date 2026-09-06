# mcp-go v1.0.0 升级评估

评估日期：2026-09-05；工作区升级验证更新：2026-09-06

升级前版本：`github.com/mark3labs/mcp-go v0.57.0`

当前工作区版本：`v1.0.0`（重构和升级尚未提交）

## 结论

**不建议在生产中只改 `go.mod` 直接升级。**

工作区已执行升级并完成重构，真实客户端测试发现的协议问题已修复。上下游保持 legacy 协议；HTTP 使用 SDK 原生版本限制，旧 SSE 按用户确认增加最小入口校验，不引入协议转换层。旧 HTTP+SSE（`/sse` + `/message`）只保留兼容支持；这不等于移除 Streamable HTTP `/mcp` 的 SSE 流式响应。

现有代码与 v1.0.0 在源码/API 层面兼容：已在隔离副本中把依赖替换为 v1.0.0，并用全新 Go build cache 执行 `go test ./...`，所有 Go package 通过。但 v1.0.0 将 `mcp.LATEST_PROTOCOL_VERSION` 从 `2025-11-25` 改为 `2026-07-28`，而 one-mcp 多处直接使用该常量初始化上游连接。一旦协商为新的无状态协议，`Client.Ping` 会直接返回成功，不再发出请求；one-mcp 当前基于 `Ping` 的健康检查、断线判断和自愈因此可能误报健康。

初次评估的两个选择：

1. **保守方案（本轮采用）**：将 one-mcp 上游 client 显式固定在 `mcp.ProtocolVersion20251125`，下游也仅支持 legacy 协议，保留现有 session、`Ping` 和通知语义；然后升级依赖。实测证明只固定上游不够。
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

## 初次隔离评估记录（2026-09-05）

初次评估没有修改原工作区的 `go.mod`、`go.sum` 或业务代码，在临时隔离副本中执行：

```sh
go mod edit -require=github.com/mark3labs/mcp-go@v1.0.0
go mod tidy
GOCACHE="$(mktemp -d)" go test ./...
```

结果：根 module 及 `backend/api/handler`、`backend/common`、`backend/library/market`、`backend/library/proxy`、`backend/model`、`backend/service` 全部通过。这证明当前代码对 v1.0.0 **source-compatible**，但不能证明与真实新/旧 MCP server 的协议交互已通过运行时验证。

另对关键路径执行 `go test -race ./backend/library/proxy ./backend/service ./backend/api/handler`，三个 package 均通过。

## 工作区重构与升级验证（2026-09-06）

已先提交工具全局禁用及协议报错修复：`6c5a686 feat(tools): add global administrator policies`。以下重构和依赖升级仍未提交；未部署、未重启运行中的 backend。

### 重构范围

- 直接读取 typed `ServerInfo`，删除两个 JSON marshal/unmarshal 辅助函数。
- 删除 tools/prompts/resources/templates 四类列表的外层分页循环；SDK 的四个 `List*` 方法已遍历所有页。
- 工具注册改为一次批量 policy 读取，保留完整 inventory 和调用时实时校验。
- 请求统计复用预检解析结果，删除第二次 request body 读取和 JSON decode。
- 弹窗计数从 tools 派生，删除重复 state；列表使用新的 total/enabled 字段，移除旧计数字段回退。
- 删除 Vite 完整环境变量调试输出，避免构建日志泄露配置。

### 真实测试发现

1. **离线上游被 Ping 误报健康。** HTTP、SSE、stdio fixtures 均复现新版协议下 `Ping()` 直接成功；显式固定 proxy 上游 `2025-11-25` 后，离线检查恢复报错。npm/PyPI 初始化同步固定版本，但没有执行外网安装验证。
2. **OAuth 手写 handshake 版本不合法。** fixture 仅接受 `initialize + 2025-11-25` 才返回 401 challenge；固定版本后 discovery 测试恢复通过。
3. **下游 modern `_meta` 透传至 legacy 上游导致 header mismatch。** 仅固定上游不足以保持行为；普通 HTTP 和 Group HTTP 使用 SDK 原生 `WithStreamableHTTPProtocolVersions(mcp.LegacyProtocolVersions()...)` 限制协商，普通 HTTP 新/旧客户端实际调用通过。
4. **SSE 声明支持列表不等于实际版本限制。** `WithSSEContextFunc + WithSupportedProtocolVersions` 能改变 discover 声明，但 SDK `server/request_handler.go` 只检查全局合法版本，`client/protocol.go` 在 discover 成功后也不根据 `SupportedVersions` 降级。真实 SSE 新客户端因此仍返回 `2026-07-28`。经用户确认，在 `/message` 检查 header/`_meta` 的现代版本声明，用 SDK `UnsupportedProtocolVersionError`（`-32022`，含 `supported`/`requested`）经已有 SSE session 返回，POST 为 202 空体；新客户端随后正常协商 `2025-11-25`。没有 ID 的通知不回 RPC 响应；拒绝发生在上游创建、配额和统计之前。没有新增协议转换层。

### 验证结果与边界

- `backend/library/proxy/mcp_upgrade_test.go`：真实 HTTP/SSE/stdio 上游的初始化、四类完整分页、工具调用、SSE 通知、动态禁用/恢复及离线 Ping 通过。stdio fixture 使用 Go 测试子进程，无需外部命令或外网。
- `TestProxyHandlerPreservesAllowedCallsAndStatistics`：HTTP/SSE legacy 以及两种新客户端协商 legacy 的调用参数和统计全部通过。fixture 使用真实上下游传输，只替换共享 instance 的创建，隔离与本测试无关的后台心跳；统计记录按独立用户隔离并清理，避免全局 ORM 生命周期影响重复执行。
- `TestProxySSERejectsModernProtocolBeforeForwardingOrCounting`：metadata、header-only 及绕过 discover 的直接工具调用均收到正确错误码和 supported/requested，配额耗尽时仍先返回协议错误且不增加配额。
- 最新 `go test ./...` 全部通过。新增回归已先复现失败，再验证修复；没有跳过原 SSE 协商失败用例。
- 最新 `go test -race ./backend/library/proxy ./backend/api/handler ./backend/service` 三包通过；新增传输/统计回归另以 `-race -count=3` 重复通过。测试 fixture 隔离后台心跳，避免在修改全局配置时与维护 goroutine 竞争；生产维护逻辑未改动。
- 前端相关 16 个测试、TypeScript 检查和 Vite build 通过。全量前端为 126 passed、6 failed；在提交 `6c5a686` 的隔离基线副本中复现相同六个失败（App loading、marketStore env 默认值、四个 clipboard mocks），未扩展修复这些既有问题。

### Review 验收矩阵

| 验收项 | 状态 | 证据 |
| --- | --- | --- |
| A1 raw inventory、批量 policy、四类分页 | covered | policy 测试及每页一项的真实 fixtures |
| A2 派生计数与独立 switch | covered | Modal/ServicesPage 交互测试 |
| A3 单次解析、参数转发与统计 | covered | 允许调用每次增加一条统计，initialize/list 不计数 |
| A4 上下游保持 legacy 协议 | covered | 四处上游、HTTP 原生限制、SSE 最小入口校验及新客户端协商断言 |
| A5 下游互通、三类上游与 offline Ping | covered | HTTP/SSE 新旧客户端、三类真实上游、显式 offline Ping |
| A6 真通知、禁用、恢复 | covered | 真实 SSE list_changed、重新 list 及 call |
| A7 OAuth challenge | covered | 严格 legacy initialize fixture |
| A8 配置与参数不泄露 | covered | 删除 Vite env 输出；拒绝日志不含 arguments |

A1–A8 均已覆盖；Group HTTP 的版本配置和 npm/PyPI 初始化仍主要依据代码核对，未执行 npm/PyPI 外网安装或部署验证。

## 建议的升级验收线

如采用保守方案，至少验证：

- stdio、SSE、Streamable HTTP 三种上游 transport 均能初始化、列工具和调工具。
- 断开上游后 `Ping` 健康检查仍能标记 unhealthy 并触发现有自愈。
- OAuth Streamable HTTP 初始化和 401/auth-required 判断不变。
- 下游旧版 client 与新版 client 都能连接 one-mcp；tool 动态禁用/启用的 `tools/list_changed` 行为仍正确。
- 重跑 `go test ./...` 和相关 race tests。
