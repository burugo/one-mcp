package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

func upgradeFixtureServer() *mcpserver.MCPServer {
	server := mcpserver.NewMCPServer("upgrade-fixture", "1", mcpserver.WithPaginationLimit(1), mcpserver.WithDescription("Fixture description"))
	for _, name := range []string{"alpha", "beta"} {
		server.AddTool(mcp.NewTool(name), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("called " + request.Params.Name), nil
		})
		server.AddPrompt(mcp.NewPrompt(name), func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{}, nil
		})
		read := func(_ context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{mcp.TextResourceContents{URI: request.Params.URI, Text: "fixture"}}, nil
		}
		server.AddResource(mcp.NewResource("test://"+name, name), read)
		server.AddResourceTemplate(mcp.NewResourceTemplate("test://"+name+"/{id}", name), read)
	}
	return server
}

func TestMCPUpgradeStdioHelper(t *testing.T) {
	if os.Getenv("ONE_MCP_UPGRADE_STDIO_HELPER") != "1" {
		t.Skip("subprocess fixture")
	}
	if err := mcpserver.ServeStdio(upgradeFixtureServer()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestMCPUpgradePreservesTransportAndLiveToolPolicy(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "upgrade.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	for _, serviceType := range []model.ServiceType{model.ServiceTypeStreamableHTTP, model.ServiceTypeSSE, model.ServiceTypeStdio} {
		t.Run(string(serviceType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			service := &model.MCPService{Name: "upgrade-fixture", Type: serviceType, Enabled: true}
			var offline atomic.Bool
			if serviceType == model.ServiceTypeStdio {
				command, err := os.Executable()
				require.NoError(t, err)
				service.Command = command
				args, err := json.Marshal([]string{"-test.run=^TestMCPUpgradeStdioHelper$"})
				require.NoError(t, err)
				service.ArgsJSON = string(args)
				service.DefaultEnvsJSON = `{"ONE_MCP_UPGRADE_STDIO_HELPER":"1"}`
			} else {
				var handler http.Handler
				if serviceType == model.ServiceTypeSSE {
					handler = mcpserver.NewSSEServer(upgradeFixtureServer())
				} else {
					handler = mcpserver.NewStreamableHTTPServer(upgradeFixtureServer())
				}
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if offline.Load() {
						http.Error(w, "upstream offline", http.StatusServiceUnavailable)
						return
					}
					handler.ServeHTTP(w, r)
				}))
				defer upstream.Close()
				service.Command = upstream.URL + "/mcp"
				if serviceType == model.ServiceTypeSSE {
					service.Command = upstream.URL + "/sse"
				}
			}
			require.NoError(t, model.CreateService(service))
			defer model.DeleteService(service.ID)
			cacheKey := fmt.Sprintf("upgrade-%d", service.ID)
			// Exercise the real transport constructor without starting unrelated
			// maintenance loops; this test drives healthy/offline Ping explicitly.
			runtimeCtx, runtimeCancel := context.WithCancel(ctx)
			defer runtimeCancel()
			server, upstreamClient, command, inventory, info, err := createActualMcpGoServerAndClientUncached(ctx, runtimeCtx, cacheKey, service, "upgrade")
			require.NoError(t, err)
			instance := &SharedMcpInstance{
				Server: server, Client: upstreamClient, Tools: inventory, ServerInfo: info,
				cancel: runtimeCancel, serviceID: service.ID, serviceName: service.Name,
				serviceType: service.Type, cacheKey: cacheKey, stdioCmd: command,
			}
			sharedMCPServersMutex.Lock()
			sharedMCPServers[cacheKey] = instance
			sharedMCPServersMutex.Unlock()
			defer func() {
				_ = instance.Shutdown(context.Background())
				sharedMCPServersMutex.Lock()
				delete(sharedMCPServers, cacheKey)
				sharedMCPServersMutex.Unlock()
			}()
			require.Len(t, instance.Tools, 2, "SDK must fetch all upstream pages")
			require.NoError(t, instance.Client.Ping(ctx))
			persisted, err := model.GetServiceByID(service.ID)
			require.NoError(t, err)
			require.Equal(t, "Fixture description", persisted.Description)

			// A real downstream SSE client observes list_changed and re-lists tools.
			downstream := httptest.NewServer(mcpserver.NewSSEServer(instance.Server))
			defer downstream.Close()
			client, err := mcpclient.NewSSEMCPClient(downstream.URL + "/sse")
			require.NoError(t, err)
			defer client.Close()
			notifications := make(chan struct{}, 4)
			client.OnNotification(func(notification mcp.JSONRPCNotification) {
				if notification.Method == "notifications/tools/list_changed" {
					notifications <- struct{}{}
				}
			})
			require.NoError(t, client.Start(ctx))
			init := mcp.InitializeRequest{}
			init.Params.ProtocolVersion = "2025-11-25"
			init.Params.ClientInfo = mcp.Implementation{Name: "upgrade-test", Version: "1"}
			_, err = client.Initialize(ctx, init)
			require.NoError(t, err)
			tools, err := client.ListTools(ctx, mcp.ListToolsRequest{})
			require.NoError(t, err)
			require.Len(t, tools.Tools, 2)
			prompts, err := client.ListPrompts(ctx, mcp.ListPromptsRequest{})
			require.NoError(t, err)
			require.Len(t, prompts.Prompts, 2)
			resources, err := client.ListResources(ctx, mcp.ListResourcesRequest{})
			require.NoError(t, err)
			require.Len(t, resources.Resources, 2)
			templates, err := client.ListResourceTemplates(ctx, mcp.ListResourceTemplatesRequest{})
			require.NoError(t, err)
			require.Len(t, templates.ResourceTemplates, 2)
			call := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "beta", Arguments: map[string]any{}}}
			result, err := client.CallTool(ctx, call)
			require.NoError(t, err)
			require.False(t, result.IsError)
			for _, enabled := range []bool{false, true} {
				_, err := model.SetMCPToolPolicy(service.ID, "beta", enabled, 1)
				require.NoError(t, err)
				require.NoError(t, ApplyMCPToolPolicy(service.ID, "beta", enabled))
				select {
				case <-notifications:
				case <-time.After(time.Second):
					t.Fatal("missing tools/list_changed notification")
				}
				tools, err = client.ListTools(ctx, mcp.ListToolsRequest{})
				require.NoError(t, err)
				if enabled {
					require.Len(t, tools.Tools, 2)
					result, err = client.CallTool(ctx, call)
					require.NoError(t, err)
					require.False(t, result.IsError)
				} else {
					require.Len(t, tools.Tools, 1)
					_, err = client.CallTool(ctx, call)
					require.ErrorContains(t, err, "not found")
				}
			}

			if serviceType == model.ServiceTypeStdio {
				require.NoError(t, instance.Client.Close())
			} else {
				offline.Store(true)
			}
			pingCtx, pingCancel := context.WithTimeout(ctx, time.Second)
			defer pingCancel()
			require.Error(t, instance.Client.Ping(pingCtx), "offline upstream must not be reported healthy")
		})
	}
}
