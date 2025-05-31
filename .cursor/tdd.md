## Test-Driven Development (TDD)
**Final Verification:**
Always use: `go test -v ./tests | grep FAIL`

**Debugging & Handling Verbose Output:**
**Avoid** `go test -v ./...` directly in the terminal due to excessive output.
**Recommended Alternatives:**
*   **Specific Tests:** `go test -v ./... -run ^TestSpecificFunction$` (Fastest for pinpointing).
*   **Filter Full Output:** `go test -v ./... > test_output.log && grep -E '--- FAIL:|FAIL:' test_output.log` (Simple & portable for full runs).

## Project Startup and Logging Rules

**Service Startup:**
- Use `bash ./run.sh` to start the one-mcp service
- This script automatically loads .env, ensures PATH, kills port 3000 processes, and starts the Go backend in background
- Logs are output to `backend.log`

**Log Monitoring:**
- Use `tail -f backend.log` to monitor real-time logs
- Use `tail -n 50 backend.log` to view recent log entries
- Use `grep "ERROR\|WARN\|Failed" backend.log` to filter error messages

**Service Management:**
- Use `pkill -f one-mcp` to stop the service
- Check service status with `ps aux | grep one-mcp | grep -v grep`
- API status endpoint: `curl "http://localhost:3003/api/status"`

## Reference Implementation: Successful MCP Client Code

The following reference code shows a working MCP client implementation that successfully handles SSE and HTTP clients without timeout context issues:

```go
func newMCPClient(name string, conf *MCPClientConfigV2) (*Client, error) {
	clientInfo, pErr := parseMCPClientConfigV2(conf)
	if pErr != nil {
		return nil, pErr
	}
	switch v := clientInfo.(type) {
	case *StdioMCPClientConfig:
		envs := make([]string, 0, len(v.Env))
		for kk, vv := range v.Env {
			envs = append(envs, fmt.Sprintf("%s=%s", kk, vv))
		}
		mcpClient, err := client.NewStdioMCPClient(v.Command, envs, v.Args...)
		if err != nil {
			return nil, err
		}

		return &Client{
			name:    name,
			client:  mcpClient,
			options: conf.Options,
		}, nil
	case *SSEMCPClientConfig:
		var options []transport.ClientOption
		if len(v.Headers) > 0 {
			options = append(options, client.WithHeaders(v.Headers))
		}
		mcpClient, err := client.NewSSEMCPClient(v.URL, options...)
		if err != nil {
			return nil, err
		}
		return &Client{
			name:            name,
			needPing:        true,
			needManualStart: true,
			client:          mcpClient,
			options:         conf.Options,
		}, nil
	case *StreamableMCPClientConfig:
		var options []transport.StreamableHTTPCOption
		if len(v.Headers) > 0 {
			options = append(options, transport.WithHTTPHeaders(v.Headers))
		}
		if v.Timeout > 0 {
			options = append(options, transport.WithHTTPTimeout(v.Timeout))
		}
		mcpClient, err := client.NewStreamableHttpClient(v.URL, options...)
		if err != nil {
			return nil, err
		}
		return &Client{
			name:            name,
			needPing:        true,
			needManualStart: true,
			client:          mcpClient,
			options:         conf.Options,
		}, nil
	}
	return nil, errors.New("invalid client type")
}

func (c *Client) addToMCPServer(ctx context.Context, clientInfo mcp.Implementation, mcpServer *server.MCPServer) error {
	if c.needManualStart {
		err := c.client.Start(ctx)
		if err != nil {
			return err
		}
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo
	initRequest.Params.Capabilities = mcp.ClientCapabilities{
		Experimental: make(map[string]interface{}),
		Roots:        nil,
		Sampling:     nil,
	}
	_, err := c.client.Initialize(ctx, initRequest)
	if err != nil {
		return err
	}
	log.Printf("<%s> Successfully initialized MCP client", c.name)

	err = c.addToolsToServer(ctx, mcpServer)
	if err != nil {
		return err
	}
	_ = c.addPromptsToServer(ctx, mcpServer)
	_ = c.addResourcesToServer(ctx, mcpServer)
	_ = c.addResourceTemplatesToServer(ctx, mcpServer)

	if c.needPing {
		go c.startPingTask(ctx)
	}
	return nil
}
```

### Key Insights from Reference Code:
1. **needManualStart Flag**: SSE and HTTP clients are marked with `needManualStart: true`
2. **Same Context Usage**: Both `Start(ctx)` and `Initialize(ctx, initRequest)` use the same context
3. **No Timeout Context**: No separate timeout context is used for initialization
4. **Sequential Calls**: `Start()` is called first, then `Initialize()` with the same context
5. **Error Handling**: Simple error propagation without complex context management

This reference implementation should guide our fix for the SSE proxy transport issues. 