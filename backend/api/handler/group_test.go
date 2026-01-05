package handler

import (
	"bytes"
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

func TestGroupCRUDHandlers(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	gin.SetMode(gin.TestMode)

	// Create
	createPayload := map[string]any{
		"name":             "group-a",
		"display_name":     "Group A",
		"description":      "test group",
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

func TestGroupMCPHandlerUnauthorized(t *testing.T) {
	teardown := setupGroupTestDB(t)
	defer teardown()

	gin.SetMode(gin.TestMode)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
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

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	req := newJSONRequest(t, http.MethodPost, "/group/group-tools/mcp", reqBody)
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
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "group-validate"}}
	ctx.Set("user_id", int64(1))

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)

	resp := decodeMCPResponse(t, recorder)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "mcp_name is required", resp.Error["message"])
}

func TestGroupMCPHandlerSearchToolsSuccess(t *testing.T) {
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
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "group-search"}}
	ctx.Set("user_id", int64(1))

	GroupMCPHandler(ctx)
	assert.Equal(t, http.StatusOK, recorder.Code)

	resp := decodeMCPResponse(t, recorder)
	// Response now returns YAML format
	toolsYAML, ok := resp.Result["tools_yaml"].(string)
	assert.True(t, ok)
	assert.Contains(t, toolsYAML, "alpha")
	assert.Contains(t, toolsYAML, "beta")

	toolCount, ok := resp.Result["tool_count"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(2), toolCount)

	currentTime, ok := resp.Result["current_time"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, currentTime)
}
