package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTestEnvironmentForProxyHandler configures a test environment using an in-memory SQLite DB.
// It returns a teardown function to restore the original SQLite path.
func setupTestEnvironmentForProxyHandler() func() {
	originalPath := common.SQLitePath
	common.SQLitePath = ":memory:"

	// Initialize the database (which will use :memory: now)
	// InitDB will also handle migrations and initialize model.MCPServiceDB etc.
	err := model.InitDB()
	if err != nil {
		panic(fmt.Sprintf("model.InitDB() failed for :memory: in proxy_handler_test: %v", err))
	}

	return func() {
		common.SQLitePath = originalPath
		// Clear any global maps that might have been populated by InitDB
		common.OptionMap = make(map[string]string)
		// model.LoadedServicesMap = make(map[string]*model.MCPService) // If such a map exists and is populated by InitDB
	}
}

func TestSSEProxyHandler_ServiceNotFound(t *testing.T) {
	teardown := setupTestEnvironmentForProxyHandler()
	defer teardown()

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/sse/:serviceName/*action", SSEProxyHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sse/not-exist-service/someaction", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Service not found")
}

// mockSSEHandler 是一个简单的 SSE http.Handler
// 它会输出 event: message\ndata: Hello test message\n\n
type mockSSEHandler struct{}

func (h *mockSSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "event: message\\ndata: Hello test message\\n\\n")
}

func TestSSEProxyHandler_MockSSE_Simple(t *testing.T) {
	teardown := setupTestEnvironmentForProxyHandler()
	defer teardown()

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/sse/:serviceName/*action", SSEProxyHandler)

	serviceName := "hello-world-simple"
	mcpService := &model.MCPService{
		Name:                     serviceName,
		Type:                     model.ServiceTypeStdio,
		DefaultAdminConfigValues: `{"command":"dummy-cmd"}`,
	}
	err := model.CreateService(mcpService)
	assert.NoError(t, err)
	// Get the service again to ensure ID is populated for defer
	dbServiceForDefer, _ := model.GetServiceByName(serviceName)
	if dbServiceForDefer != nil {
		defer model.DeleteService(dbServiceForDefer.ID)
	}

	serviceManager := proxy.GetServiceManager()
	serviceManager.UnregisterService(context.Background(), dbServiceForDefer.ID)

	dbService, err := model.GetServiceByName(serviceName)
	assert.NoError(t, err)
	assert.NotNil(t, dbService)

	baseSvcForProxy := proxy.NewBaseService(dbService.ID, dbService.Name, model.ServiceTypeSSE)
	// Corrected call to NewSSESvc:
	simpleMockSvc := proxy.NewSSESvc(baseSvcForProxy, &mockSSEHandler{})
	serviceManager.SetService(dbService.ID, simpleMockSvc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sse/hello-world-simple/someaction", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "event: message")
	assert.Contains(t, body, "Hello test message")
}

// TODO: 可继续补充：
// - 服务类型不符
// - SSESvc 未初始化
// - 正常流式代理（需 mock SSESvc）

// mockMCPMasterHandler simulates the mcp-go server's HTTP responses for the full SSE flow.
type mockMCPMasterHandler struct {
	t *testing.T // To allow assertions within the handler if needed
}

func (h *mockMCPMasterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log the request path received by the mock handler for debugging
	// h.t.Logf("mockMCPMasterHandler received path: %s, method: %s", r.URL.Path, r.Method)

	if r.Method == "GET" && r.URL.Path == "/" { // Path seen by underlying handler after SSEProxyHandler
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Order of WriteHeader and Fprintf can matter. Usually Fprintf writes header if not set.
		// Explicitly setting WriteHeader(http.StatusOK) first is safer.
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "event: endpoint\\ndata: /message?sessionId=test-session-123\\n\\n")
		fmt.Fprintf(w, "event: message\\ndata: {\\\"jsonrpc\\\":\\\"2.0\\\",\\\"id\\\":0,\\\"result\\\":{\\\"protocolVersion\\\":\\\"test-pv\\\",\\\"serverInfo\\\":{\\\"name\\\":\\\"Mock MCP Server\\\"}}}\\n\\n")
		return
	}

	if r.Method == "POST" && r.URL.Path == "/message" && r.URL.Query().Get("sessionId") == "test-session-123" {
		// Optional: check request body
		// bodyBytes, _ := io.ReadAll(r.Body)
		// assert.Contains(h.t, string(bodyBytes), "\"method\":\"initialize\"")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202 Accepted for POSTs that initiate async work or just acknowledge
		fmt.Fprintf(w, "{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":{\"protocolVersion\":\"test-pv\",\"serverInfo\":{\"name\":\"Mock MCP Server\"}}}")
		return
	}

	// Fallback for unhandled paths/methods by the mock
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Mock MCP Master Handler: Path %s or method %s not handled", r.URL.Path, r.Method)
}

// Helper function to marshal StdioConfig to JSON string for tests
func stdioConfigToJSON(sc model.StdioConfig) (string, error) {
	bytes, err := json.Marshal(sc)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func TestSSEProxyHandler_MCPProtocolFlow(t *testing.T) {
	teardown := setupTestEnvironmentForProxyHandler()
	defer teardown()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/api/sse/:serviceName/*action", SSEProxyHandler)
	router.POST("/api/sse/:serviceName/*action", SSEProxyHandler)

	serviceName := "exa-mcp-server-flow"
	stdioConf := model.StdioConfig{Command: "fake-exa-cmd"}
	// Use standard json.Marshal via helper or directly
	stdioConfJSON, err := stdioConfigToJSON(stdioConf)
	assert.NoError(t, err, "Failed to marshal stdioConf")

	mcpService := &model.MCPService{
		Name:                     serviceName,
		Type:                     model.ServiceTypeStdio,
		DefaultAdminConfigValues: stdioConfJSON,
		DisplayName:              "Flow Test Service",
	}
	err = model.CreateService(mcpService)
	assert.NoError(t, err)
	// Get the service again to ensure ID is populated for defer
	dbServiceForDefer, _ := model.GetServiceByName(serviceName)
	if dbServiceForDefer != nil {
		defer model.DeleteService(dbServiceForDefer.ID)
	} else {
		// If service wasn't created, this indicates an issue in CreateService or GetServiceByName logic in test setup
		t.Errorf("Service %s was not found after creation for defer setup", serviceName)
	}

	serviceManager := proxy.GetServiceManager()
	if dbServiceForDefer != nil { // only unregister if it was created
		serviceManager.UnregisterService(context.Background(), dbServiceForDefer.ID)
	}

	dbService, err := model.GetServiceByName(serviceName)
	assert.NoError(t, err)
	assert.NotNil(t, dbService, "Service should exist in DB for test setup")

	baseSvcForProxy := proxy.NewBaseService(dbService.ID, dbService.Name, model.ServiceTypeSSE)
	masterMockHandler := &mockMCPMasterHandler{t: t}
	mockedProxiedSvc := proxy.NewSSESvc(baseSvcForProxy, masterMockHandler)
	serviceManager.SetService(dbService.ID, mockedProxiedSvc)

	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest("GET", "/api/sse/"+serviceName+"/", nil)
	router.ServeHTTP(wGet, reqGet)

	assert.Equal(t, http.StatusOK, wGet.Code, "Initial GET should be 200 OK")
	assert.Equal(t, "text/event-stream", wGet.Header().Get("Content-Type"), "Content-Type should be text/event-stream")
	expectedEventData := "event: endpoint\\ndata: /message?sessionId=test-session-123\\n\\n"
	assert.Contains(t, wGet.Body.String(), expectedEventData, "Response should contain the endpoint event")
	assert.Contains(t, wGet.Body.String(), "event: message", "Response should contain initial message event")

	wPost := httptest.NewRecorder()
	postPath := "/api/sse/" + serviceName + "/message?sessionId=test-session-123"
	reqBody := strings.NewReader("{\\\"method\\\":\\\"initialize\\\",\\\"params\\\":{},\\\"jsonrpc\\\":\\\"2.0\\\",\\\"id\\\":0}")
	reqPost, _ := http.NewRequest("POST", postPath, reqBody)
	reqPost.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wPost, reqPost)

	assert.Equal(t, http.StatusAccepted, wPost.Code, "POST to message endpoint should be 202 Accepted")
	assert.Equal(t, "application/json", wPost.Header().Get("Content-Type"), "Content-Type should be application/json")
	expectedJsonRpcResponse := "{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":{\"protocolVersion\":\"test-pv\",\"serverInfo\":{\"name\":\"Mock MCP Server\"}}}"
	assert.JSONEq(t, expectedJsonRpcResponse, wPost.Body.String(), "Response body for POST should be correct JSON-RPC")

	wRedirect := httptest.NewRecorder()
	reqRedirect, _ := http.NewRequest("GET", "/api/sse/"+serviceName, nil)
	router.ServeHTTP(wRedirect, reqRedirect)

	assert.Equal(t, http.StatusMovedPermanently, wRedirect.Code, "GET to base without slash should 301")
	assert.Equal(t, "/api/sse/"+serviceName+"/", wRedirect.Header().Get("Location"), "Location header for 301 should be correct")
}

// TestSSEProxyHandler_UserSpecific_CallsNewUncachedHandlerWithCorrectConfig verifies that when a user
// has specific configurations for an Stdio service that allows overrides,
// NewStdioSSEHandlerUncached is called with the correctly merged StdioConfig.
func TestSSEProxyHandler_UserSpecific_CallsNewUncachedHandlerWithCorrectConfig(t *testing.T) {
	teardown := setupTestEnvironmentForProxyHandler()
	defer teardown()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	// Simulate JWTAuth middleware setting userID
	router.Use(func(c *gin.Context) {
		c.Set("userID", int64(1)) // Assuming test user ID is 1
		c.Next()
	})
	router.GET("/api/sse/:serviceName/*action", SSEProxyHandler)

	// 1. Setup Database State
	// Create User (implicitly, userID=1 is used from context)
	// UserDB might not be directly needed if we just set userID in context

	// Create MCPService
	mcpServiceName := "user-specific-test-svc"
	defaultStdioConfig := model.StdioConfig{
		Command: "base-cmd",
		Args:    []string{"base-arg"},
		Env:     []string{"DEFAULT_CMD_ENV=cmd_val"}, // Env from command config
	}
	defaultStdioConfigJSON, _ := json.Marshal(defaultStdioConfig)
	defaultEnvsMap := map[string]string{"DEFAULT_JSON_ENV": "json_val", "USER_OVERRIDABLE_DEFAULT": "default_version"}
	defaultEnvsJSON, _ := json.Marshal(defaultEnvsMap)

	mcpSvc := &model.MCPService{
		Name:                     mcpServiceName,
		Type:                     model.ServiceTypeStdio,
		AllowUserOverride:        true,
		DefaultAdminConfigValues: string(defaultStdioConfigJSON),
		DefaultEnvsJSON:          string(defaultEnvsJSON),
		Enabled:                  true,
	}
	err := model.CreateService(mcpSvc)
	assert.NoError(t, err)
	dbMcpSvc, err := model.GetServiceByName(mcpServiceName)
	assert.NoError(t, err)
	assert.NotNil(t, dbMcpSvc)
	defer model.DeleteService(dbMcpSvc.ID)

	// Create ConfigService (the definition of the ENV var)
	configKeyName := "USER_API_KEY"
	cfgSvc := &model.ConfigService{
		ServiceID: dbMcpSvc.ID,
		Key:       configKeyName,
		Type:      model.ConfigTypeSecret,
	}
	err = model.CreateConfigOption(cfgSvc)
	assert.NoError(t, err)
	dbConfigSvc, err := model.GetConfigOptionByKey(dbMcpSvc.ID, configKeyName)
	assert.NoError(t, err)
	assert.NotNil(t, dbConfigSvc)
	// No defer needed for ConfigService as it would be deleted with MCPService or if schema had cascades

	// Create UserConfig (the user's specific value for the ENV var)
	userApiKeyVal := "user_secret_123"
	userCfg := &model.UserConfig{
		UserID:    1, // Matches userID set in Gin context
		ServiceID: dbMcpSvc.ID,
		ConfigID:  dbConfigSvc.ID,
		Value:     userApiKeyVal,
	}
	err = model.SaveUserConfig(userCfg)
	assert.NoError(t, err)

	// User override for a default env
	configKeyOverride := "USER_OVERRIDABLE_DEFAULT"
	cfgSvcOverride := &model.ConfigService{
		ServiceID: dbMcpSvc.ID,
		Key:       configKeyOverride,
		Type:      model.ConfigTypeString,
	}
	err = model.CreateConfigOption(cfgSvcOverride)
	assert.NoError(t, err)
	dbConfigSvcOverride, _ := model.GetConfigOptionByKey(dbMcpSvc.ID, configKeyOverride)

	userOverrideVal := "user_has_overridden"
	userCfgOverride := &model.UserConfig{
		UserID:    1,
		ServiceID: dbMcpSvc.ID,
		ConfigID:  dbConfigSvcOverride.ID,
		Value:     userOverrideVal,
	}
	err = model.SaveUserConfig(userCfgOverride)
	assert.NoError(t, err)

	// 2. Mock proxy.NewStdioSSEHandlerUncached
	originalNewStdioSSEHandlerUncached := proxy.NewStdioSSEHandlerUncached
	var capturedStdioConfig model.StdioConfig
	var newStdioSSEHandlerUncachedCalled bool

	proxy.NewStdioSSEHandlerUncached = func(ctx context.Context, mcpDBService *model.MCPService, effectiveStdioConfig model.StdioConfig) (http.Handler, error) {
		newStdioSSEHandlerUncachedCalled = true
		capturedStdioConfig = effectiveStdioConfig
		// Return a simple mock handler for the test to complete the HTTP flow
		return &mockSSEHandler{}, nil
	}
	defer func() {
		proxy.NewStdioSSEHandlerUncached = originalNewStdioSSEHandlerUncached
	}()

	// 3. Make the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sse/"+mcpServiceName+"/someaction", nil)
	router.ServeHTTP(w, req)

	// 4. Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, newStdioSSEHandlerUncachedCalled, "NewStdioSSEHandlerUncached should have been called")

	assert.Equal(t, "base-cmd", capturedStdioConfig.Command, "Command should be from DefaultAdminConfigValues")
	assert.Equal(t, []string{"base-arg"}, capturedStdioConfig.Args, "Args should be from DefaultAdminConfigValues")

	// Check ENVs: DEFAULT_CMD_ENV, DEFAULT_JSON_ENV, USER_API_KEY, USER_OVERRIDABLE_DEFAULT (overridden)
	expectedEnvsMap := map[string]string{
		"DEFAULT_CMD_ENV":  "cmd_val",
		"DEFAULT_JSON_ENV": "json_val",
		configKeyName:      userApiKeyVal,
		configKeyOverride:  userOverrideVal,
	}

	actualEnvsMap := make(map[string]string)
	for _, envStr := range capturedStdioConfig.Env {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) == 2 {
			actualEnvsMap[parts[0]] = parts[1]
		}
	}
	assert.Equal(t, expectedEnvsMap, actualEnvsMap, "Effective ENV variables are not correctly merged")

	// Verify that the user-specific handler is cached
	userHandlerKey := fmt.Sprintf("user-%d-service-%d", 1, dbMcpSvc.ID)
	cachedUserHandler, found := proxy.GetCachedHandler(userHandlerKey)
	assert.True(t, found, "User-specific handler should be cached")
	assert.NotNil(t, cachedUserHandler, "Cached user-specific handler should not be nil")
	_, ok := cachedUserHandler.(*mockSSEHandler)
	assert.True(t, ok, "Cached handler should be the one returned by the mock NewStdioSSEHandlerUncached")

}
