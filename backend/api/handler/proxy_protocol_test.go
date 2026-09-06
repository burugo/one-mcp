package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/burugo/thing"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

func TestProxySSERejectsModernProtocolBeforeForwardingOrCounting(t *testing.T) {
	teardown := setupTestEnvironmentForProxyHandler()
	defer teardown()
	proxy.ClearSSEProxyCache()
	defer proxy.ClearSSEProxyCache()
	service := &model.MCPService{
		Name: "sse-protocol-fixture", Type: model.ServiceTypeStreamableHTTP,
		Command: "https://unused.example.test/mcp", Enabled: true, RPDLimit: 1,
	}
	require.NoError(t, model.CreateService(service))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	quotaKey := fmt.Sprintf("user_request:%s:%d:4243:count", time.Now().Format("2006-01-02"), service.ID)
	require.NoError(t, thing.Cache().Set(ctx, quotaKey, "1", time.Hour))
	defer thing.Cache().Delete(context.Background(), quotaKey)
	router := gin.New()
	host := httptest.NewServer(router)
	defer host.Close()
	common.OptionMap["ServerAddress"] = host.URL
	server := mcpserver.NewMCPServer("sse-protocol-fixture", "1")
	sse, err := proxy.GetOrCreateProxyToSSEHandler(ctx, service, &proxy.SharedMcpInstance{Server: server})
	require.NoError(t, err)
	router.Use(func(c *gin.Context) { c.Set("userID", int64(4243)); c.Next() })
	router.GET("/proxy/:serviceName/sse", gin.WrapH(sse))
	router.POST("/proxy/:serviceName/*action", ProxyHandler)
	client, err := transport.NewSSE(host.URL + "/proxy/sse-protocol-fixture/sse")
	require.NoError(t, err)
	defer client.Close()
	require.NoError(t, client.Start(ctx))

	for _, tc := range []struct{ name, method string }{
		{"metadata", "server/discover"},
		{"header", "server/discover"},
		{"direct-call", "tools/call"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := transport.JSONRPCRequest{
				JSONRPC: "2.0", ID: mcp.NewRequestId(tc.name), Method: tc.method,
			}
			params := map[string]any{"name": "uncalled-tool"}
			if tc.name == "header" {
				request.Header = http.Header{mcp.HeaderProtocolVersion: {"2026-07-28"}}
			} else {
				params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": "2026-07-28"}
			}
			request.Params = params
			response, err := client.SendRequest(ctx, request)
			require.NoError(t, err, "a terminal JSON-RPC error must arrive on the existing SSE stream")
			require.NotNil(t, response.Error)
			require.Equal(t, mcp.NewRequestId(tc.name), response.ID)
			require.Equal(t, mcp.UNSUPPORTED_PROTOCOL_VERSION, response.Error.Code)
			data, err := json.Marshal(response.Error.Data)
			require.NoError(t, err)
			require.JSONEq(t, `{"requested":"2026-07-28","supported":["2025-11-25","2025-06-18","2025-03-26","2024-11-05"]}`, string(data))
		})
	}
	count, err := thing.Cache().Get(ctx, quotaKey)
	require.NoError(t, err)
	require.Equal(t, "1", count, "protocol rejections must not consume quota")
}
