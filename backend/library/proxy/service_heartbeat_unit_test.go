package proxy

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/mcp"
)

type fakeMcpClient struct {
	pingFn      func(context.Context) error
	closeCalled atomic.Bool
}

func (f *fakeMcpClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return &mcp.InitializeResult{}, nil
}

func (f *fakeMcpClient) Ping(ctx context.Context) error {
	if f.pingFn == nil {
		return nil
	}
	return f.pingFn(ctx)
}

func (f *fakeMcpClient) ListResourcesByPage(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return &mcp.ListResourcesResult{}, nil
}

func (f *fakeMcpClient) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return &mcp.ListResourcesResult{}, nil
}

func (f *fakeMcpClient) ListResourceTemplatesByPage(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return &mcp.ListResourceTemplatesResult{}, nil
}

func (f *fakeMcpClient) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return &mcp.ListResourceTemplatesResult{}, nil
}

func (f *fakeMcpClient) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{}, nil
}

func (f *fakeMcpClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	return nil
}

func (f *fakeMcpClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return nil
}

func (f *fakeMcpClient) ListPromptsByPage(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return &mcp.ListPromptsResult{}, nil
}

func (f *fakeMcpClient) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return &mcp.ListPromptsResult{}, nil
}

func (f *fakeMcpClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{}, nil
}

func (f *fakeMcpClient) ListToolsByPage(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}

func (f *fakeMcpClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}

func (f *fakeMcpClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{}, nil
}

func (f *fakeMcpClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return nil
}

func (f *fakeMcpClient) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return &mcp.CompleteResult{}, nil
}

func (f *fakeMcpClient) Close() error {
	f.closeCalled.Store(true)
	return nil
}

func (f *fakeMcpClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
}

func TestShouldInvalidateInstanceAfterCallError_ContextCanceledWithPingFailure(t *testing.T) {
	cli := pingableMcpClientFunc(func(ctx context.Context) error {
		return errors.New("ping failed")
	})

	if !shouldInvalidateInstanceAfterCallError(cli, context.Canceled) {
		t.Fatalf("expected invalidation when call error is canceled and ping fails")
	}
}

func TestShouldInvalidateInstanceAfterCallError_ContextCanceledWithPingSuccess(t *testing.T) {
	cli := pingableMcpClientFunc(func(ctx context.Context) error {
		return nil
	})

	if shouldInvalidateInstanceAfterCallError(cli, context.Canceled) {
		t.Fatalf("did not expect invalidation when call error is canceled but ping succeeds")
	}
}

type pingableMcpClientFunc func(context.Context) error

func (f pingableMcpClientFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestSharedMcpInstance_Heartbeat_RemovesCacheOnPingFailure(t *testing.T) {
	// Speed up loop for test and remove jitter.
	common.OptionMapRWMutex.Lock()
	common.OptionMap[common.OptionNetworkMcpHeartbeatInterval] = "10ms"
	common.OptionMap[common.OptionNetworkMcpHeartbeatTimeout] = "5ms"
	common.OptionMap[common.OptionNetworkMcpHeartbeatJitter] = "0s"
	common.OptionMapRWMutex.Unlock()

	// Reset caches
	sharedMCPServersMutex.Lock()
	sharedMCPServers = make(map[string]*SharedMcpInstance)
	sharedMCPServersMutex.Unlock()

	sseWrappersMutex.Lock()
	initializedSSEProxyWrappers = make(map[string]http.Handler)
	sseWrappersMutex.Unlock()

	httpWrappersMutex.Lock()
	initializedHTTPProxyWrappers = make(map[string]http.Handler)
	httpWrappersMutex.Unlock()

	fake := &fakeMcpClient{pingFn: func(ctx context.Context) error { return errors.New("boom") }}
	inst := &SharedMcpInstance{
		Client:      fake,
		cancel:      func() {},
		serviceID:   123,
		serviceName: "svc",
		serviceType: model.ServiceTypeStreamableHTTP,
		cacheKey:    "k1",
	}

	sharedMCPServersMutex.Lock()
	sharedMCPServers[inst.cacheKey] = inst
	sharedMCPServersMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inst.startMaintenanceLoops(ctx)

	deadline := time.After(500 * time.Millisecond)
	for {
		sharedMCPServersMutex.Lock()
		_, exists := sharedMCPServers[inst.cacheKey]
		sharedMCPServersMutex.Unlock()
		if !exists {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected instance to be removed from cache on heartbeat ping failure")
		case <-time.After(10 * time.Millisecond):
		}
	}
}
