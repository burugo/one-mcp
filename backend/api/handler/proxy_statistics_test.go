package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/burugo/thing"
	"github.com/gin-gonic/gin"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statisticsProxyHost(t *testing.T, service *model.MCPService, server *mcpserver.MCPServer) string {
	t.Helper()
	instance := &proxy.SharedMcpInstance{Server: server}
	originalFactory := proxy.GetOrCreateSharedMcpInstanceWithKey
	proxy.GetOrCreateSharedMcpInstanceWithKey = func(_ context.Context, requested *model.MCPService, _, _, _ string) (*proxy.SharedMcpInstance, error) {
		if requested.ID != service.ID {
			return nil, errors.New("unexpected service")
		}
		return instance, nil
	}
	t.Cleanup(func() { proxy.GetOrCreateSharedMcpInstanceWithKey = originalFactory })
	router := gin.New()
	router.Use(func(c *gin.Context) {
		userID, err := strconv.ParseInt(c.GetHeader("X-Test-User-ID"), 10, 64)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("userID", userID)
		c.Next()
	})
	router.Any("/proxy/:serviceName/*action", ProxyHandler)
	host := httptest.NewServer(router)
	t.Cleanup(host.Close)
	common.OptionMap["ServerAddress"] = host.URL
	manager := proxy.GetServiceManager()
	manager.SetService(service.ID, proxy.NewMonitoredProxiedService(proxy.NewBaseService(service.ID, service.Name, service.Type), instance, service))
	t.Cleanup(func() { _ = manager.UnregisterService(context.Background(), service.ID) })
	return host.URL + "/proxy/" + service.Name
}

func statisticsClient(t *testing.T, ctx context.Context, baseURL, endpoint string, userID int64) *mcpclient.Client {
	t.Helper()
	headers := map[string]string{"X-Test-User-ID": fmt.Sprint(userID)}
	var client *mcpclient.Client
	var err error
	if endpoint == "sse" {
		client, err = mcpclient.NewSSEMCPClient(baseURL+"/sse", mcpclient.WithHeaders(headers))
	} else {
		client, err = mcpclient.NewStreamableHttpClient(baseURL+"/mcp", transport.WithHTTPHeaders(headers))
	}
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Start(ctx))
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = "2025-11-25"
	init.Params.ClientInfo = mcp.Implementation{Name: "statistics-test", Version: "1"}
	_, err = client.Initialize(ctx, init)
	require.NoError(t, err)
	_, err = client.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	return client
}

func TestProxyStatisticsFollowToolCompletion(t *testing.T) {
	originalPath, originalOptions := common.SQLitePath, common.OptionMap
	common.SQLitePath = fmt.Sprintf("file:statistics-%d?mode=memory&cache=shared", time.Now().UnixNano())
	defer func() { common.SQLitePath, common.OptionMap = originalPath, originalOptions }()
	require.NoError(t, model.InitDB())
	common.OptionMap = map[string]string{}
	statsDB, err := model.GetProxyRequestStatThing()
	require.NoError(t, err)
	for _, endpoint := range []string{"mcp", "sse"} {
		for _, outcome := range []string{"success", "tool_error", "protocol_error"} {
			t.Run(endpoint+"/"+outcome, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				userID := time.Now().UnixNano()
				service := &model.MCPService{Name: fmt.Sprintf("stats-%d", userID), Enabled: true, Type: model.ServiceTypeStreamableHTTP}
				require.NoError(t, model.CreateService(service))
				defer model.DeleteService(service.ID)
				entered, release := make(chan struct{}), make(chan struct{})
				var releaseOnce sync.Once
				unblock := func() { releaseOnce.Do(func() { close(release) }) }
				defer unblock()
				server := mcpserver.NewMCPServer("stats-fixture", "1", proxy.WithToolCallObservation())
				server.AddTool(mcp.NewTool("controlled"), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					close(entered)
					select {
					case <-release:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					switch outcome {
					case "tool_error":
						return mcp.NewToolResultError("business failure"), nil
					case "protocol_error":
						return nil, errors.New("upstream failure")
					default:
						return mcp.NewToolResultText("finished"), nil
					}
				})
				baseURL := statisticsProxyHost(t, service, server)
				quotaKey := fmt.Sprintf("user_request:%s:%d:%d:count", time.Now().Format("2006-01-02"), service.ID, userID)
				t.Cleanup(func() {
					stats, err := statsDB.Where("service_id = ? AND user_id = ?", service.ID, userID).All()
					require.NoError(t, err)
					for _, stat := range stats {
						require.NoError(t, statsDB.Delete(stat))
					}
					_ = thing.Cache().Delete(context.Background(), quotaKey)
				})
				client := statisticsClient(t, ctx, baseURL, endpoint, userID)
				done := make(chan error, 1)
				go func() {
					_, err := client.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "controlled", Arguments: map[string]any{"secret": "private-statistics-argument"}}})
					done <- err
				}()
				select {
				case <-entered:
				case <-ctx.Done():
					t.Fatal("tool was not dispatched")
				}
				assert.Never(t, func() bool {
					stats, err := statsDB.Where("service_id = ? AND user_id = ?", service.ID, userID).All()
					return err != nil || len(stats) != 0
				}, 50*time.Millisecond, 5*time.Millisecond, "initialize/list and SSE acknowledgement are not completed tool calls")
				logs, _, err := model.GetMCPLogs(ctx, &service.ID, &service.Name, nil, nil, 1, 100)
				require.NoError(t, err)
				require.Empty(t, logs, "access logs must wait for tool completion")
				unblock()
				callErr := <-done
				if outcome == "protocol_error" {
					require.Error(t, callErr)
				} else {
					require.NoError(t, callErr)
				}
				require.Eventually(t, func() bool {
					count, err := thing.Cache().Get(ctx, quotaKey)
					return err == nil && count == "1"
				}, time.Second, 5*time.Millisecond)
				stats, err := statsDB.Where("service_id = ? AND user_id = ?", service.ID, userID).All()
				require.NoError(t, err)
				require.Len(t, stats, 1)
				assert.Equal(t, outcome == "success", stats[0].Success)
				assert.GreaterOrEqual(t, stats[0].ResponseTimeMs, int64(50))
				expectedStatus := http.StatusOK
				if endpoint == "sse" {
					expectedStatus = http.StatusAccepted
				}
				assert.Equal(t, expectedStatus, stats[0].StatusCode)
				logs, _, err = model.GetMCPLogs(ctx, &service.ID, &service.Name, nil, nil, 1, 100)
				require.NoError(t, err)
				require.Len(t, logs, 1)
				expectedLevel, expectedOutcome := model.MCPLogLevelInfo, "OK"
				if outcome != "success" {
					expectedLevel, expectedOutcome = model.MCPLogLevelError, "FAILED"
				}
				assert.Equal(t, expectedLevel, logs[0].Level)
				assert.Contains(t, logs[0].Message, "MCP tool call "+expectedOutcome)
				assert.Contains(t, logs[0].Message, "tool=controlled")
				assert.Contains(t, logs[0].Message, fmt.Sprintf("user=%d", userID))
				assert.NotContains(t, logs[0].Message, "private-statistics-argument")
				require.NoError(t, model.MCPLogDB.Delete(logs[0]))
			})
		}
	}
}

func TestProxyStatisticsKeepConcurrentSessionsIsolated(t *testing.T) {
	originalPath, originalOptions := common.SQLitePath, common.OptionMap
	common.SQLitePath = fmt.Sprintf("file:statistics-sessions-%d?mode=memory&cache=shared", time.Now().UnixNano())
	defer func() { common.SQLitePath, common.OptionMap = originalPath, originalOptions }()
	require.NoError(t, model.InitDB())
	common.OptionMap = map[string]string{}
	statsDB, err := model.GetProxyRequestStatThing()
	require.NoError(t, err)
	for _, endpoint := range []string{"mcp", "sse"} {
		t.Run(endpoint, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			service := &model.MCPService{Name: fmt.Sprintf("concurrent-statistics-%s-%d", endpoint, time.Now().UnixNano()), Enabled: true, Type: model.ServiceTypeStreamableHTTP}
			require.NoError(t, model.CreateService(service))
			defer model.DeleteService(service.ID)
			users := []int64{time.Now().UnixNano(), time.Now().UnixNano() + 1}
			names := []string{"first", "second"}
			entered := []chan struct{}{make(chan struct{}), make(chan struct{})}
			release := []chan struct{}{make(chan struct{}), make(chan struct{})}
			var releaseOnce [2]sync.Once
			unblock := func(index int) { releaseOnce[index].Do(func() { close(release[index]) }) }
			defer unblock(0)
			defer unblock(1)
			server := mcpserver.NewMCPServer("concurrent-statistics", "1", proxy.WithToolCallObservation())
			for index, name := range names {
				server.AddTool(mcp.NewTool(name), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					close(entered[index])
					select {
					case <-release[index]:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					if index == 1 {
						return mcp.NewToolResultError("second call failed"), nil
					}
					return mcp.NewToolResultText("first call finished"), nil
				})
			}
			baseURL := statisticsProxyHost(t, service, server)
			clients := []*mcpclient.Client{
				statisticsClient(t, ctx, baseURL, endpoint, users[0]),
				statisticsClient(t, ctx, baseURL, endpoint, users[1]),
			}
			quotaKeys := make([]string, 2)
			for index, userID := range users {
				quotaKeys[index] = fmt.Sprintf("user_request:%s:%d:%d:count", time.Now().Format("2006-01-02"), service.ID, userID)
				key := quotaKeys[index]
				t.Cleanup(func() { _ = thing.Cache().Delete(context.Background(), key) })
			}
			type callOutcome struct {
				result *mcp.CallToolResult
				err    error
			}
			done := []chan callOutcome{make(chan callOutcome, 1), make(chan callOutcome, 1)}
			for index, client := range clients {
				go func() {
					result, err := client.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: names[index], Arguments: map[string]any{"secret": "private-session-argument"}}})
					done[index] <- callOutcome{result, err}
				}()
				select {
				case <-entered[index]:
				case <-ctx.Done():
					t.Fatal("tool was not dispatched")
				}
			}
			logs, _, err := model.GetMCPLogs(ctx, &service.ID, &service.Name, nil, nil, 1, 100)
			require.NoError(t, err)
			require.Empty(t, logs)
			// Finish in reverse order while the first user's session is still running.
			for completed, index := range []int{1, 0} {
				unblock(index)
				outcome := <-done[index]
				require.NoError(t, outcome.err)
				require.NotNil(t, outcome.result)
				require.Equal(t, index == 1, outcome.result.IsError)
				require.Eventually(t, func() bool {
					count, err := thing.Cache().Get(ctx, quotaKeys[index])
					return err == nil && count == "1"
				}, time.Second, 5*time.Millisecond)
				stats, err := statsDB.Where("service_id = ? AND service_name = ?", service.ID, service.Name).All()
				require.NoError(t, err)
				require.Len(t, stats, completed+1)
				for _, stat := range stats {
					if stat.UserID == users[1] {
						assert.False(t, stat.Success)
					} else {
						assert.Equal(t, users[0], stat.UserID)
						assert.True(t, stat.Success)
					}
				}
				if completed == 0 {
					assert.Equal(t, users[1], stats[0].UserID, "first user must not be charged by the second session's completion")
					count, _ := thing.Cache().Get(ctx, quotaKeys[0])
					assert.Empty(t, count)
				}
				logs, _, err = model.GetMCPLogs(ctx, &service.ID, &service.Name, nil, nil, 1, 100)
				require.NoError(t, err)
				require.Len(t, logs, completed+1)
				var matchingLogs int
				for _, entry := range logs {
					assert.NotContains(t, entry.Message, "private-session-argument")
					if strings.Contains(entry.Message, fmt.Sprintf("user=%d", users[index])) {
						matchingLogs++
						assert.Contains(t, entry.Message, "tool="+names[index])
						if index == 1 {
							assert.Equal(t, model.MCPLogLevelError, entry.Level)
							assert.Contains(t, entry.Message, "MCP tool call FAILED")
						} else {
							assert.Equal(t, model.MCPLogLevelInfo, entry.Level)
							assert.Contains(t, entry.Message, "MCP tool call OK")
						}
					}
				}
				assert.Equal(t, 1, matchingLogs)
			}
			stats, err := statsDB.Where("service_id = ? AND service_name = ?", service.ID, service.Name).All()
			require.NoError(t, err)
			for _, stat := range stats {
				require.NoError(t, statsDB.Delete(stat))
			}
			for _, entry := range logs {
				require.NoError(t, model.MCPLogDB.Delete(entry))
			}
		})
	}
}
