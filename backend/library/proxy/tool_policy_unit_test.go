package proxy

import (
	"context"
	"path/filepath"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

type toolPolicyMCPClient struct {
	mcpclient.MCPClient
	tools []mcp.Tool
	calls int
}

func (c *toolPolicyMCPClient) ListTools(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{Tools: c.tools}, nil
}

func (c *toolPolicyMCPClient) CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	c.calls++
	return mcp.NewToolResultText("called"), nil
}

func TestProxyToolDiscoveryOmitsDisabledToolsButRetainsRawInventory(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "proxy-tool-policy.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	const serviceID int64 = 41
	_, err := model.SetMCPToolPolicy(serviceID, "blocked-tool", false, 1)
	require.NoError(t, err)
	client := &toolPolicyMCPClient{tools: []mcp.Tool{{Name: "allowed-tool"}, {Name: "blocked-tool"}}}
	server := mcpserver.NewMCPServer("policy-test", "1.0.0")

	inventory, err := addClientToolsToMCPServer(context.Background(), client, server, "policy-test", "policy-cache", serviceID, model.ServiceTypeStreamableHTTP)
	require.NoError(t, err)
	require.Equal(t, []mcp.Tool{{Name: "allowed-tool"}, {Name: "blocked-tool"}}, inventory)
	require.NotNil(t, server.GetTool("allowed-tool"))
	require.Nil(t, server.GetTool("blocked-tool"))
}

func TestApplyMCPToolPolicyUpdatesLiveInstancesAndRejectsStaleHandlers(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "live-tool-policy.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	const serviceID int64 = 42
	client := &toolPolicyMCPClient{tools: []mcp.Tool{{Name: "toggle-tool"}}}
	server := mcpserver.NewMCPServer("live-policy-test", "1.0.0")
	inventory, err := addClientToolsToMCPServer(context.Background(), client, server, "live-policy-test", "live-policy-cache", serviceID, model.ServiceTypeStreamableHTTP)
	require.NoError(t, err)
	staleHandler := server.GetTool("toggle-tool").Handler
	userServer := mcpserver.NewMCPServer("user-live-policy-test", "1.0.0")
	userInventory, err := addClientToolsToMCPServer(context.Background(), client, userServer, "live-policy-test", "user-live-policy-cache", serviceID, model.ServiceTypeStreamableHTTP)
	require.NoError(t, err)

	sharedMCPServersMutex.Lock()
	sharedMCPServers["live-policy-test"] = &SharedMcpInstance{
		Server:      server,
		Client:      client,
		Tools:       inventory,
		serviceID:   serviceID,
		serviceName: "live-policy-test",
		serviceType: model.ServiceTypeStreamableHTTP,
		cacheKey:    "live-policy-cache",
	}
	sharedMCPServers["user-live-policy-test"] = &SharedMcpInstance{
		Server:      userServer,
		Client:      client,
		Tools:       userInventory,
		serviceID:   serviceID,
		serviceName: "live-policy-test",
		serviceType: model.ServiceTypeStreamableHTTP,
		cacheKey:    "user-live-policy-cache",
	}
	sharedMCPServersMutex.Unlock()
	t.Cleanup(func() {
		sharedMCPServersMutex.Lock()
		delete(sharedMCPServers, "live-policy-test")
		delete(sharedMCPServers, "user-live-policy-test")
		sharedMCPServersMutex.Unlock()
	})

	_, err = model.SetMCPToolPolicy(serviceID, "toggle-tool", false, 1)
	require.NoError(t, err)
	require.NoError(t, ApplyMCPToolPolicy(serviceID, "toggle-tool", false))
	require.Nil(t, server.GetTool("toggle-tool"))
	require.Nil(t, userServer.GetTool("toggle-tool"))
	staleResult, err := staleHandler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "toggle-tool"}})
	require.NoError(t, err)
	require.True(t, staleResult.IsError)
	require.Zero(t, client.calls)

	_, err = model.SetMCPToolPolicy(serviceID, "toggle-tool", true, 1)
	require.NoError(t, err)
	require.NoError(t, ApplyMCPToolPolicy(serviceID, "toggle-tool", true))
	require.NotNil(t, server.GetTool("toggle-tool"))
	require.NotNil(t, userServer.GetTool("toggle-tool"))
	_, err = server.GetTool("toggle-tool").Handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "toggle-tool"}})
	require.NoError(t, err)
	require.Equal(t, 1, client.calls)
}
