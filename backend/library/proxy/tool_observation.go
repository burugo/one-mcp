package proxy

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

type toolObserverKey struct{}

// ToolCallObserver receives the final synchronous tools/call outcome, including
// protocol errors and tool results with IsError set. It must not retain a Gin context.
type ToolCallObserver func(*mcp.CallToolResult, error)

// WithToolCallObserver attaches request-local observation to either HTTP transport.
// SSE's detached message context preserves it after the POST has returned 202.
func WithToolCallObserver(ctx context.Context, observer ToolCallObserver) context.Context {
	return context.WithValue(ctx, toolObserverKey{}, observer)
}

// WithToolCallObservation observes completion, not transport acknowledgement.
// Requests without an observer and non-tool methods have no observation side effects.
func WithToolCallObservation() mcpserver.ServerOption {
	hooks := &mcpserver.Hooks{}
	hooks.AddAfterCallTool(func(ctx context.Context, _ any, _ *mcp.CallToolRequest, result any) {
		observer, ok := ctx.Value(toolObserverKey{}).(ToolCallObserver)
		if !ok {
			return
		}
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok || toolResult == nil {
			observer(nil, fmt.Errorf("tools/call returned no synchronous tool result"))
			return
		}
		observer(toolResult, nil)
	})
	hooks.AddOnError(func(ctx context.Context, _ any, method mcp.MCPMethod, _ any, err error) {
		if method == mcp.MethodToolsCall {
			if observer, ok := ctx.Value(toolObserverKey{}).(ToolCallObserver); ok {
				observer(nil, err)
			}
		}
	})
	return mcpserver.WithHooks(hooks)
}
