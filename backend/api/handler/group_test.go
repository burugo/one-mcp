package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type groupResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	ServiceIDsJSON string `json:"service_ids_json"`
	Enabled        bool   `json:"enabled"`
}

type mcpResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Result  map[string]any `json:"result"`
	Error   map[string]any `json:"error"`
}

func setupGroupTestDB(t *testing.T) func() {
	t.Helper()
	originalSQLitePath := common.SQLitePath
	dbPath := filepath.Join(t.TempDir(), "group_test.db")
	common.SQLitePath = dbPath

	err := model.InitDB()
	assert.NoError(t, err)

	return func() {
		common.SQLitePath = originalSQLitePath
	}
}

func newJSONRequest(t *testing.T, method string, path string, payload any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	assert.NoError(t, err)
	req, err := http.NewRequest(method, path, bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	assert.NoError(t, err)
	return resp
}

func decodeMCPResponse(t *testing.T, recorder *httptest.ResponseRecorder) mcpResponse {
	t.Helper()
	var resp mcpResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	assert.NoError(t, err)
	return resp
}

func initializeGroupSession(t *testing.T, groupName string, userID int64) (string, mcpResponse) {
	t.Helper()
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "group-test",
				"version": "0.0.0",
			},
			"capabilities": map[string]any{},
		},
	}
	req := newJSONRequest(t, http.MethodPost, "/group/"+groupName+"/mcp", reqBody)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: groupName}}
	ctx.Set("user_id", userID)

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)
	sessionID := recorder.Header().Get("Mcp-Session-Id")
	assert.NotEmpty(t, sessionID)
	return sessionID, decodeMCPResponse(t, recorder)
}

func TestGroupCRUDHandlers(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	gin.SetMode(gin.TestMode)

	// Create
	createPayload := map[string]any{
		"name":             "group-a",
		"display_name":     "Group A",
		"service_ids_json": "[1,2]",
	}
	createReq := newJSONRequest(t, http.MethodPost, "/api/groups", createPayload)
	createRecorder := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(createRecorder)
	createCtx.Request = createReq
	createCtx.Set("user_id", int64(1))
	createCtx.Set("lang", "en")

	CreateGroup(createCtx)
	assert.Equal(t, http.StatusOK, createRecorder.Code)

	createResp := decodeAPIResponse(t, createRecorder)
	assert.True(t, createResp.Success)

	var createdGroup groupResponse
	err := json.Unmarshal(createResp.Data, &createdGroup)
	assert.NoError(t, err)
	assert.Equal(t, "group-a", createdGroup.Name)
	assert.Equal(t, "Group A", createdGroup.DisplayName)

	// List
	listRecorder := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRecorder)
	listCtx.Request, _ = http.NewRequest(http.MethodGet, "/api/groups", nil)
	listCtx.Set("user_id", int64(1))
	GetGroups(listCtx)
	assert.Equal(t, http.StatusOK, listRecorder.Code)

	listResp := decodeAPIResponse(t, listRecorder)
	var groups []groupResponse
	err = json.Unmarshal(listResp.Data, &groups)
	assert.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, createdGroup.ID, groups[0].ID)

	// Update
	updatePayload := map[string]any{
		"display_name": "Group A Updated",
		"enabled":      false,
	}
	updateReq := newJSONRequest(t, http.MethodPut, "/api/groups/1", updatePayload)
	updateRecorder := httptest.NewRecorder()
	updateCtx, _ := gin.CreateTestContext(updateRecorder)
	updateCtx.Request = updateReq
	updateCtx.Params = gin.Params{{Key: "id", Value: "1"}}
	updateCtx.Set("user_id", int64(1))
	updateCtx.Set("lang", "en")

	UpdateGroup(updateCtx)
	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	updateResp := decodeAPIResponse(t, updateRecorder)
	var updatedGroup groupResponse
	err = json.Unmarshal(updateResp.Data, &updatedGroup)
	assert.NoError(t, err)
	assert.Equal(t, "Group A Updated", updatedGroup.DisplayName)
	assert.False(t, updatedGroup.Enabled)

	// Delete
	deleteRecorder := httptest.NewRecorder()
	deleteCtx, _ := gin.CreateTestContext(deleteRecorder)
	deleteCtx.Request, _ = http.NewRequest(http.MethodDelete, "/api/groups/1", nil)
	deleteCtx.Params = gin.Params{{Key: "id", Value: "1"}}
	deleteCtx.Set("user_id", int64(1))
	deleteCtx.Set("lang", "en")

	DeleteGroup(deleteCtx)
	assert.Equal(t, http.StatusOK, deleteRecorder.Code)

	// List after delete
	listAfterRecorder := httptest.NewRecorder()
	listAfterCtx, _ := gin.CreateTestContext(listAfterRecorder)
	listAfterCtx.Request, _ = http.NewRequest(http.MethodGet, "/api/groups", nil)
	listAfterCtx.Set("user_id", int64(1))
	GetGroups(listAfterCtx)
	assert.Equal(t, http.StatusOK, listAfterRecorder.Code)

	listAfterResp := decodeAPIResponse(t, listAfterRecorder)
	var groupsAfter []groupResponse
	err = json.Unmarshal(listAfterResp.Data, &groupsAfter)
	assert.NoError(t, err)
	assert.Len(t, groupsAfter, 0)
}

func TestGroupDescriptionReflectsCurrentServiceDescription(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	service := &model.MCPService{
		Name:        "dynamic-description-service",
		DisplayName: "Dynamic Description Service",
		Description: "Initial service description",
		Type:        model.ServiceTypeSSE,
		Command:     "http://127.0.0.1:1/sse",
		Enabled:     true,
	}
	require.NoError(t, model.CreateService(service))

	group := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "dynamic-description-group",
		DisplayName: "Dynamic Description Group",
		Description: "This group contains the following MCP services:\n- dynamic-description-service: Initial service description",
		Enabled:     true,
	}
	group.SetServiceIDs([]int64{service.ID})
	require.NoError(t, group.Insert())
	customGroup := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "custom-description-group",
		DisplayName: "Custom Description Group",
		Description: "Custom group instructions",
		Enabled:     true,
	}
	customGroup.SetServiceIDs([]int64{service.ID})
	require.NoError(t, customGroup.Insert())

	_, initialInitializeResp := initializeGroupSession(t, group.Name, group.UserID)
	initialInstructions, ok := initialInitializeResp.Result["instructions"].(string)
	require.True(t, ok)
	assert.Contains(t, initialInstructions, "Initial service description")

	service.Description = "Current service description"
	require.NoError(t, model.UpdateService(service))

	listRecorder := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRecorder)
	listCtx.Request, _ = http.NewRequest(http.MethodGet, "/api/groups", nil)
	listCtx.Set("user_id", int64(1))
	GetGroups(listCtx)
	require.Equal(t, http.StatusOK, listRecorder.Code)

	listResp := decodeAPIResponse(t, listRecorder)
	var groups []groupResponse
	require.NoError(t, json.Unmarshal(listResp.Data, &groups))
	require.Len(t, groups, 2)
	groupsByName := make(map[string]groupResponse, len(groups))
	for _, currentGroup := range groups {
		groupsByName[currentGroup.Name] = currentGroup
	}
	assert.Equal(t, "This group contains the following MCP services:\n- dynamic-description-service: Current service description", groupsByName[group.Name].Description)
	assert.Equal(t, "Custom group instructions", groupsByName[customGroup.Name].Description)

	persistedGroup, err := model.MCPServiceGroupDB.ByID(group.ID)
	require.NoError(t, err)
	assert.Contains(t, persistedGroup.Description, "Initial service description")
	assert.NotContains(t, persistedGroup.Description, "Current service description")

	_, initializeResp := initializeGroupSession(t, group.Name, group.UserID)
	instructions, ok := initializeResp.Result["instructions"].(string)
	require.True(t, ok)
	assert.Contains(t, instructions, "Current service description")
	assert.NotContains(t, instructions, "Initial service description")

	_, customInitializeResp := initializeGroupSession(t, customGroup.Name, customGroup.UserID)
	assert.Equal(t, "Custom group instructions", customInitializeResp.Result["instructions"])
}

func TestGroupMCPHandlerUnauthorized(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	gin.SetMode(gin.TestMode)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "group-test",
				"version": "0.0.0",
			},
			"capabilities": map[string]any{},
		},
	}
	req := newJSONRequest(t, http.MethodPost, "/group/test/mcp", reqBody)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "test"}}

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestGroupMCPHandlerToolsList(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	group := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "group-tools",
		DisplayName: "Group Tools",
		Enabled:     true,
	}
	group.SetServiceIDs([]int64{})
	err := group.Insert()
	assert.NoError(t, err)

	sessionID, _ := initializeGroupSession(t, "group-tools", 1)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	req := newJSONRequest(t, http.MethodPost, "/group/group-tools/mcp", reqBody)
	req.Header.Set("Mcp-Session-Id", sessionID)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "group-tools"}}
	ctx.Set("user_id", int64(1))

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)

	resp := decodeMCPResponse(t, recorder)
	tools, ok := resp.Result["tools"].([]any)
	assert.True(t, ok)
	assert.Len(t, tools, 2)
}

func TestGroupMCPHandlerExcludesDisabledServices(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	enabledService := &model.MCPService{
		Name:        "enabled-group-service",
		DisplayName: "Enabled Group Service",
		Description: "Enabled service description",
		Type:        model.ServiceTypeStdio,
		Command:     "echo",
		ArgsJSON:    `[]`,
		Enabled:     true,
	}
	require.NoError(t, model.CreateService(enabledService))

	disabledService := &model.MCPService{
		Name:        "disabled-group-service",
		DisplayName: "Disabled Group Service",
		Description: "Disabled service description",
		Type:        model.ServiceTypeStdio,
		Command:     "echo",
		ArgsJSON:    `[]`,
		Enabled:     false,
	}
	require.NoError(t, model.CreateService(disabledService))

	group := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "group-disabled-filter",
		DisplayName: "Group Disabled Filter",
		Enabled:     true,
	}
	group.SetServiceIDs([]int64{enabledService.ID, disabledService.ID})
	require.NoError(t, group.Insert())

	proxy.GetToolsCacheManager().SetServiceTools(disabledService.ID, &proxy.ToolsCacheEntry{
		Tools: []mcp.Tool{{Name: "stale-disabled-tool"}},
	})
	defer proxy.GetToolsCacheManager().DeleteServiceTools(disabledService.ID)

	assert.Equal(t, []string{enabledService.Name}, getGroupServiceNames(group))
	assert.NotContains(t, group.EffectiveDescription(), disabledService.Name)
	_, err := searchGroupTools(context.Background(), group, &groupSearchArgs{MCPName: disabledService.Name})
	assert.Error(t, err)

	disabledFingerprint := groupHandlerFingerprint(group)
	disabledService.Enabled = true
	require.NoError(t, model.UpdateService(disabledService))
	assert.NotEqual(t, disabledFingerprint, groupHandlerFingerprint(group))
	disabledService.Enabled = false
	require.NoError(t, model.UpdateService(disabledService))

	sessionID, _ := initializeGroupSession(t, group.Name, group.UserID)

	toolsReq := newJSONRequest(t, http.MethodPost, "/group/"+group.Name+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	toolsReq.Header.Set("Mcp-Session-Id", sessionID)
	toolsRecorder := httptest.NewRecorder()
	toolsCtx, _ := gin.CreateTestContext(toolsRecorder)
	toolsCtx.Request = toolsReq
	toolsCtx.Params = gin.Params{{Key: "name", Value: group.Name}}
	toolsCtx.Set("user_id", group.UserID)
	GroupMCPHandler(toolsCtx)
	require.Equal(t, http.StatusOK, toolsRecorder.Code)

	toolsResp := decodeMCPResponse(t, toolsRecorder)
	tools, ok := toolsResp.Result["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 2)
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		require.True(t, ok)
		inputSchema, ok := tool["inputSchema"].(map[string]any)
		require.True(t, ok)
		properties, ok := inputSchema["properties"].(map[string]any)
		require.True(t, ok)
		mcpName, ok := properties["mcp_name"].(map[string]any)
		require.True(t, ok)
		serviceNames, ok := mcpName["enum"].([]any)
		require.True(t, ok)
		assert.Equal(t, []any{enabledService.Name}, serviceNames)
	}

	resourcesReq := newJSONRequest(t, http.MethodPost, "/group/"+group.Name+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "resources/list",
	})
	resourcesReq.Header.Set("Mcp-Session-Id", sessionID)
	resourcesRecorder := httptest.NewRecorder()
	resourcesCtx, _ := gin.CreateTestContext(resourcesRecorder)
	resourcesCtx.Request = resourcesReq
	resourcesCtx.Params = gin.Params{{Key: "name", Value: group.Name}}
	resourcesCtx.Set("user_id", group.UserID)
	GroupMCPHandler(resourcesCtx)
	require.Equal(t, http.StatusOK, resourcesRecorder.Code)

	resourcesResp := decodeMCPResponse(t, resourcesRecorder)
	resources, ok := resourcesResp.Result["resources"].([]any)
	require.True(t, ok)
	require.Len(t, resources, 1)
	resource, ok := resources[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, enabledService.Name, resource["name"])
}

func TestGroupMCPHandlerSearchToolsValidation(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	group := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "group-validate",
		DisplayName: "Group Validate",
		Enabled:     true,
	}
	group.SetServiceIDs([]int64{})
	err := group.Insert()
	assert.NoError(t, err)

	sessionID, _ := initializeGroupSession(t, "group-validate", 1)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "search_tools",
			"arguments": map[string]any{},
		},
	}
	req := newJSONRequest(t, http.MethodPost, "/group/group-validate/mcp", reqBody)
	req.Header.Set("Mcp-Session-Id", sessionID)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "group-validate"}}
	ctx.Set("user_id", int64(1))

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)

	resp := decodeMCPResponse(t, recorder)
	assert.Nil(t, resp.Error)
	assert.Equal(t, true, resp.Result["isError"])
	content, ok := resp.Result["content"].([]any)
	assert.True(t, ok)
	assert.NotEmpty(t, content)
}

func TestGroupMCPHandlerSearchToolsOmitsGloballyDisabledTools(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	svc := &model.MCPService{
		Name:        "svc-search",
		DisplayName: "Svc Search",
		Type:        model.ServiceTypeStdio,
		Command:     "echo",
		ArgsJSON:    `[]`,
		Enabled:     true,
	}
	err := model.CreateService(svc)
	assert.NoError(t, err)

	dbService, err := model.GetServiceByName("svc-search")
	assert.NoError(t, err)
	assert.NotNil(t, dbService)

	group := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "group-search",
		DisplayName: "Group Search",
		Enabled:     true,
	}
	group.SetServiceIDs([]int64{dbService.ID})
	err = group.Insert()
	assert.NoError(t, err)

	cache := proxy.GetToolsCacheManager()
	cache.SetServiceTools(dbService.ID, &proxy.ToolsCacheEntry{
		Tools: []mcp.Tool{
			{
				Name:        "alpha",
				Description: "alpha tool",
				InputSchema: mcp.ToolInputSchema{Type: "object"},
			},
			{
				Name:        "beta",
				Description: "beta tool",
				InputSchema: mcp.ToolInputSchema{Type: "object"},
			},
		},
	})
	defer cache.DeleteServiceTools(dbService.ID)
	_, err = model.SetMCPToolPolicy(dbService.ID, "beta", false, 1)
	require.NoError(t, err)

	sessionID, _ := initializeGroupSession(t, "group-search", 1)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "search_tools",
			"arguments": map[string]any{
				"mcp_name":  "svc-search",
				"tool_name": "alpha",
				"limit":     10,
			},
		},
	}
	req := newJSONRequest(t, http.MethodPost, "/group/group-search/mcp", reqBody)
	req.Header.Set("Mcp-Session-Id", sessionID)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "group-search"}}
	ctx.Set("user_id", int64(1))

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)

	resp := decodeMCPResponse(t, recorder)
	assert.Nil(t, resp.Error)

	// content is an array with text containing tools and current_time
	content, ok := resp.Result["content"].([]any)
	assert.True(t, ok)
	assert.NotEmpty(t, content)
	firstContent, ok := content[0].(map[string]any)
	assert.True(t, ok)
	toolsYAML, ok := firstContent["text"].(string)
	assert.True(t, ok)
	assert.Contains(t, toolsYAML, "alpha")
	assert.NotContains(t, toolsYAML, "beta")
	assert.Contains(t, toolsYAML, "current_time:")
}

func TestGroupMCPHandlerRejectsGloballyDisabledToolExecution(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	svc := &model.MCPService{
		Name:        "svc-blocked-execute",
		DisplayName: "Svc Blocked Execute",
		Type:        model.ServiceTypeStdio,
		Command:     "echo",
		ArgsJSON:    `[]`,
		Enabled:     true,
	}
	require.NoError(t, model.CreateService(svc))
	_, err := model.SetMCPToolPolicy(svc.ID, "dangerous-tool", false, 1)
	require.NoError(t, err)

	group := &model.MCPServiceGroup{UserID: 1, Name: "group-blocked-execute", DisplayName: "Group Blocked Execute", Enabled: true}
	group.SetServiceIDs([]int64{svc.ID})
	require.NoError(t, group.Insert())

	sessionID, _ := initializeGroupSession(t, group.Name, group.UserID)
	req := newJSONRequest(t, http.MethodPost, "/group/"+group.Name+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "execute_tool",
			"arguments": map[string]any{
				"mcp_name":  svc.Name,
				"tool_name": "dangerous-tool",
				"arguments": map[string]any{},
			},
		},
	})
	req.Header.Set("Mcp-Session-Id", sessionID)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: group.Name}}
	ctx.Set("user_id", group.UserID)

	GroupMCPHandler(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
	resp := decodeMCPResponse(t, recorder)
	require.Equal(t, true, resp.Result["isError"])
	content, ok := resp.Result["content"].([]any)
	require.True(t, ok)
	require.Contains(t, content[0].(map[string]any)["text"], `tool "dangerous-tool" is disabled by administrator`)
}

func TestGroupMCPHandlerInvalidSessionReturnsNotFound(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	group := &model.MCPServiceGroup{
		UserID:      1,
		Name:        "group-invalid-session",
		DisplayName: "Group Invalid Session",
		Enabled:     true,
	}
	group.SetServiceIDs([]int64{})
	err := group.Insert()
	assert.NoError(t, err)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	req := newJSONRequest(t, http.MethodPost, "/group/group-invalid-session/mcp", reqBody)
	req.Header.Set("Mcp-Session-Id", "invalid-session-id")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "group-invalid-session"}}
	ctx.Set("user_id", int64(1))

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
