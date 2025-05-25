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
	r.GET("/proxy/:serviceName/sse/*action", SSEProxyHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/not-exist-service/sse/someaction", nil)
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
	r.GET("/proxy/:serviceName/sse/*action", SSEProxyHandler)

	serviceName := "hello-world-simple"
	mcpService := &model.MCPService{
		Name:    serviceName,
		Type:    model.ServiceTypeStdio,
		Command: "dummy-cmd",
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
	req, _ := http.NewRequest("GET", "/proxy/hello-world-simple/sse/someaction", nil)
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
	router.GET("/proxy/:serviceName/sse/*action", SSEProxyHandler)
	router.POST("/proxy/:serviceName/sse/*action", SSEProxyHandler)

	serviceName := "exa-mcp-server-flow"
	stdioConf := model.StdioConfig{Command: "fake-exa-cmd", Args: []string{"arg1", "-f", "value"}}
	// Use standard json.Marshal via helper or directly
	// stdioConfJSON, err := stdioConfigToJSON(stdioConf) // This was for the whole struct
	// assert.NoError(t, err, "Failed to marshal stdioConf")

	argsBytes, err := json.Marshal(stdioConf.Args)
	assert.NoError(t, err, "Failed to marshal stdioConf.Args")
	argsJSON := string(argsBytes)

	mcpService := &model.MCPService{
		Name:        serviceName,
		Type:        model.ServiceTypeStdio,
		Command:     stdioConf.Command, // Populate from stdioConf used for test expectation
		ArgsJSON:    argsJSON,          // Correctly use JSON marshaled Args
		DisplayName: "Flow Test Service",
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
	reqGet, _ := http.NewRequest("GET", "/proxy/"+serviceName+"/sse/", nil)
	router.ServeHTTP(wGet, reqGet)

	assert.Equal(t, http.StatusOK, wGet.Code, "Initial GET should be 200 OK")
	assert.Equal(t, "text/event-stream", wGet.Header().Get("Content-Type"), "Content-Type should be text/event-stream")
	expectedEventData := "event: endpoint\\ndata: /message?sessionId=test-session-123\\n\\n"
	assert.Contains(t, wGet.Body.String(), expectedEventData, "Response should contain the endpoint event")
	assert.Contains(t, wGet.Body.String(), "event: message", "Response should contain initial message event")

	wPost := httptest.NewRecorder()
	postPath := "/proxy/" + serviceName + "/sse/message?sessionId=test-session-123"
	reqBody := strings.NewReader("{\"method\":\"initialize\",\"params\":{},\"jsonrpc\":\"2.0\",\"id\":0}")
	reqPost, _ := http.NewRequest("POST", postPath, reqBody)
	reqPost.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wPost, reqPost)

	assert.Equal(t, http.StatusAccepted, wPost.Code, "POST to message endpoint should be 202 Accepted")
	assert.Equal(t, "application/json", wPost.Header().Get("Content-Type"), "Content-Type should be application/json")
	expectedJsonRpcResponse := "{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":{\"protocolVersion\":\"test-pv\",\"serverInfo\":{\"name\":\"Mock MCP Server\"}}}"
	assert.JSONEq(t, expectedJsonRpcResponse, wPost.Body.String(), "Response body for POST should be correct JSON-RPC")

	wRedirect := httptest.NewRecorder()
	reqRedirect, _ := http.NewRequest("GET", "/proxy/"+serviceName+"/sse", nil)
	router.ServeHTTP(wRedirect, reqRedirect)

	assert.Equal(t, http.StatusMovedPermanently, wRedirect.Code, "GET to base without slash should 301")
	assert.Equal(t, "/proxy/"+serviceName+"/sse/", wRedirect.Header().Get("Location"), "Location header for 301 should be correct")
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
	router.GET("/proxy/:serviceName/sse/*action", SSEProxyHandler)

	serviceName := "user-specific-stdio-svc"
	baseStdioCommand := "base-cmd"
	baseStdioArgs := []string{"base-arg"}
	baseArgsJSON, _ := json.Marshal(baseStdioArgs)

	// defaultStdioConfig := model.StdioConfig{ // This variable is unused
	// 	Command: baseStdioCommand,
	// 	Args:    baseStdioArgs,
	// 	// Env: []string{"BASE_ENV=base_val"}, // DefaultEnvsJSON will provide this
	// }

	mcpDBService := &model.MCPService{
		Name:              serviceName,
		DisplayName:       "User Specific Stdio Service",
		Type:              model.ServiceTypeStdio,
		AllowUserOverride: true,
		Command:           baseStdioCommand,
		ArgsJSON:          string(baseArgsJSON),
		DefaultEnvsJSON:   `{"BASE_ENV":"base_val", "OVERRIDE_ME":"default_override"}`,
	}
	err := model.CreateService(mcpDBService)
	assert.NoError(t, err)

	// Get the service again to get its ID for UserConfig
	dbService, err := model.GetServiceByName(serviceName)
	assert.NoError(t, err)
	assert.NotNil(t, dbService)
	defer model.DeleteService(dbService.ID)

	// Setup user-specific config (ENV vars)
	// ConfigService entry for USER_ENV_VAR
	userEnvVarConfig := &model.ConfigService{
		ServiceID:   dbService.ID,
		Key:         "USER_ENV_VAR", // This is the ENV var name
		DisplayName: "User Specific Var",
		Type:        model.ConfigTypeString,
	}
	err = model.ConfigServiceDB.Save(userEnvVarConfig)
	assert.NoError(t, err)
	defer model.ConfigServiceDB.Delete(userEnvVarConfig)

	// UserConfig entry linking the user, service, and the ConfigService entry with the value
	userSpecificSetting := &model.UserConfig{
		UserID:    1,
		ServiceID: dbService.ID,
		ConfigID:  userEnvVarConfig.ID, // Link to the ConfigService entry for USER_ENV_VAR
		Value:     "user_value",
	}
	err = model.UserConfigDB.Save(userSpecificSetting)
	assert.NoError(t, err)
	defer model.UserConfigDB.Delete(userSpecificSetting)

	// ConfigService entry for OVERRIDE_ME (to test user override of default env)
	overrideEnvVarConfig := &model.ConfigService{
		ServiceID:   dbService.ID,
		Key:         "OVERRIDE_ME",
		DisplayName: "Override Var",
		Type:        model.ConfigTypeString,
	}
	err = model.ConfigServiceDB.Save(overrideEnvVarConfig)
	assert.NoError(t, err)
	defer model.ConfigServiceDB.Delete(overrideEnvVarConfig)

	userOverrideSetting := &model.UserConfig{
		UserID:    1,
		ServiceID: dbService.ID,
		ConfigID:  overrideEnvVarConfig.ID,
		Value:     "user_override_val",
	}
	err = model.UserConfigDB.Save(userOverrideSetting)
	assert.NoError(t, err)
	defer model.UserConfigDB.Delete(userOverrideSetting)

	// Mock NewStdioSSEHandlerUncached to capture the StdioConfig passed to it
	var capturedStdioConfig model.StdioConfig
	originalNewStdioSSEHandlerUncached := proxy.NewStdioSSEHandlerUncached
	proxy.NewStdioSSEHandlerUncached = func(ctx context.Context, mcpService *model.MCPService, effectiveStdioConfig model.StdioConfig) (http.Handler, error) {
		capturedStdioConfig = effectiveStdioConfig
		// Return a dummy handler, the focus is on capturing the config
		return &mockSSEHandler{}, nil
	}
	defer func() { proxy.NewStdioSSEHandlerUncached = originalNewStdioSSEHandlerUncached }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/"+serviceName+"/sse/someaction", nil)
	// Set user ID in context, as JWTAuth middleware would
	// req = req.WithContext(context.WithValue(req.Context(), "userID", int64(1))) // This way doesn't work with gin
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Assertions on the captured StdioConfig
	assert.Equal(t, baseStdioCommand, capturedStdioConfig.Command, "Command should be from DefaultAdminConfigValues")
	assert.Equal(t, baseStdioArgs, capturedStdioConfig.Args, "Args should be from DefaultAdminConfigValues")
	assert.Contains(t, capturedStdioConfig.Env, "BASE_ENV=base_val", "Should contain base env from DefaultEnvsJSON")
	assert.Contains(t, capturedStdioConfig.Env, "USER_ENV_VAR=user_value", "Should contain user-specific env")
	assert.Contains(t, capturedStdioConfig.Env, "OVERRIDE_ME=user_override_val", "User value should override default env")

	// Ensure the default value that was overridden is not present
	var foundOverriddenDefault bool
	for _, envVar := range capturedStdioConfig.Env {
		if envVar == "OVERRIDE_ME=default_override" {
			foundOverriddenDefault = true
			break
		}
	}
	assert.False(t, foundOverriddenDefault, "Default overridden value should not be present in final Env")
}
