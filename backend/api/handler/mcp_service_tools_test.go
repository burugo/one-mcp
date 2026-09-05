package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"one-mcp/backend/api/middleware"
	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	appservice "one-mcp/backend/service"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMCPServiceTools_ReturnsToolsCacheWhenNotRunning(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = ":memory:"
	defer func() { common.SQLitePath = originalPath }()

	err := model.InitDB()
	assert.NoError(t, err)

	// Create enabled service in DB
	svc := &model.MCPService{
		Name:        "tools-cache-svc",
		DisplayName: "Tools Cache Svc",
		Type:        model.ServiceTypeStdio,
		Command:     "echo",
		ArgsJSON:    "[]",
		Enabled:     true,
	}
	err = model.CreateService(svc)
	assert.NoError(t, err)

	created, err := model.GetServiceByName("tools-cache-svc")
	assert.NoError(t, err)
	assert.NotNil(t, created)
	defer model.DeleteService(created.ID)

	// Seed tools cache (service may be not running / not registered)
	proxy.GetToolsCacheManager().SetServiceTools(created.ID, &proxy.ToolsCacheEntry{
		Tools: []mcp.Tool{
			{Name: "t1"},
			{Name: "t2"},
			{Name: "t3"},
			{Name: "t4"},
		},
		FetchedAt: time.Now(),
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/mcp_services/:id/tools", GetMCPServiceTools)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/mcp_services/"+fmt.Sprintf("%d", created.ID)+"/tools", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Tools []mcp.Tool `json:"tools"`
		} `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 4, len(resp.Data.Tools))
}

func TestMCPServiceToolPolicyPersistsAndIsReturnedWithToolSummary(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "tool-policy.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	svc := &model.MCPService{
		Name:        "tool-policy-svc",
		DisplayName: "Tool Policy Svc",
		Type:        model.ServiceTypeStdio,
		Command:     "echo",
		ArgsJSON:    "[]",
		Enabled:     true,
	}
	require.NoError(t, model.CreateService(svc))
	proxy.GetToolsCacheManager().SetServiceTools(svc.ID, &proxy.ToolsCacheEntry{
		Tools:     []mcp.Tool{{Name: "allowed-tool"}, {Name: "blocked-tool"}},
		FetchedAt: time.Now(),
	})
	t.Cleanup(func() { proxy.GetToolsCacheManager().DeleteServiceTools(svc.ID) })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/mcp_services/:id/tools/policy", func(c *gin.Context) {
		c.Set("user_id", int64(99))
		UpdateMCPToolPolicy(c)
	})
	r.GET("/api/mcp_services/:id/tools", GetMCPServiceTools)

	body := bytes.NewBufferString(`{"tool_name":"blocked-tool","enabled":false}`)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/mcp_services/%d/tools/policy", svc.ID), body)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	r.ServeHTTP(updateRecorder, updateReq)
	require.Equal(t, http.StatusOK, updateRecorder.Code, updateRecorder.Body.String())

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/mcp_services/%d/tools", svc.ID), nil)
	listRecorder := httptest.NewRecorder()
	r.ServeHTTP(listRecorder, listReq)
	require.Equal(t, http.StatusOK, listRecorder.Code, listRecorder.Body.String())

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Tools []struct {
				Name    string `json:"name"`
				Enabled bool   `json:"enabled"`
			} `json:"tools"`
			TotalToolCount    int `json:"total_tool_count"`
			EnabledToolCount  int `json:"enabled_tool_count"`
			DisabledToolCount int `json:"disabled_tool_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRecorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, 2, resp.Data.TotalToolCount)
	require.Equal(t, 1, resp.Data.EnabledToolCount)
	require.Equal(t, 1, resp.Data.DisabledToolCount)
	require.Equal(t, []struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}{{Name: "allowed-tool", Enabled: true}, {Name: "blocked-tool", Enabled: false}}, resp.Data.Tools)

	policy, err := model.GetMCPToolPolicy(svc.ID, "blocked-tool")
	require.NoError(t, err)
	require.False(t, policy.Enabled)
	require.Equal(t, int64(99), policy.UpdatedBy)
}

func TestMCPToolPolicyRouteEnforcesAdminAndRecordsJWTUser(t *testing.T) {
	originalPath := common.SQLitePath
	originalRedisEnabled := common.RedisEnabled
	common.SQLitePath = filepath.Join(t.TempDir(), "tool-policy-route-auth.db")
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.SQLitePath = originalPath
		common.RedisEnabled = originalRedisEnabled
	})
	require.NoError(t, model.InitDB())

	svc := &model.MCPService{Name: "tool-policy-route-auth-svc", Enabled: true}
	require.NoError(t, model.CreateService(svc))
	proxy.GetToolsCacheManager().SetServiceTools(svc.ID, &proxy.ToolsCacheEntry{Tools: []mcp.Tool{{Name: "route-auth-tool"}}})
	t.Cleanup(func() { proxy.GetToolsCacheManager().DeleteServiceTools(svc.ID) })
	admin := &model.User{Username: "tool-admin", Role: common.RoleAdminUser}
	user := &model.User{Username: "tool-user", Role: common.RoleCommonUser}
	require.NoError(t, model.UserDB.Save(admin))
	require.NoError(t, model.UserDB.Save(user))
	adminToken, err := appservice.GenerateToken(admin)
	require.NoError(t, err)
	userToken, err := appservice.GenerateToken(user)
	require.NoError(t, err)

	router := gin.New()
	route := router.Group("/api/mcp_services")
	route.Use(middleware.JWTAuth(), middleware.AdminAuth())
	route.PUT("/:id/tools/policy", UpdateMCPToolPolicy)
	requestPolicy := func(token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/mcp_services/%d/tools/policy", svc.ID),
			bytes.NewBufferString(`{"tool_name":"route-auth-tool","enabled":false}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	assert.Equal(t, http.StatusForbidden, requestPolicy(userToken).Code)
	adminResponse := requestPolicy(adminToken)
	require.Equal(t, http.StatusOK, adminResponse.Code, adminResponse.Body.String())
	policy, err := model.GetMCPToolPolicy(svc.ID, "route-auth-tool")
	require.NoError(t, err)
	assert.Equal(t, admin.ID, policy.UpdatedBy)
}

func TestUpdateMCPToolPolicyRequiresAuthenticatedAdministratorContext(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "tool-policy-auth.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	svc := &model.MCPService{Name: "tool-policy-auth-svc", Enabled: true}
	require.NoError(t, model.CreateService(svc))
	proxy.GetToolsCacheManager().SetServiceTools(svc.ID, &proxy.ToolsCacheEntry{Tools: []mcp.Tool{{Name: "unauth-tool"}}})
	t.Cleanup(func() { proxy.GetToolsCacheManager().DeleteServiceTools(svc.ID) })

	router := gin.New()
	router.PUT("/api/mcp_services/:id/tools/policy", UpdateMCPToolPolicy)
	request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/mcp_services/%d/tools/policy", svc.ID),
		bytes.NewBufferString(`{"tool_name":"unauth-tool","enabled":false}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	_, err := model.GetMCPToolPolicy(svc.ID, "unauth-tool")
	assert.ErrorIs(t, err, model.ErrMCPToolPolicyNotFound)
}

func TestDisablingMCPServicePreservesToolPolicies(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "tool-policy-disable-service.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	svc := &model.MCPService{Name: "tool-policy-disable-svc", Enabled: true}
	require.NoError(t, model.CreateService(svc))
	_, err := model.SetMCPToolPolicy(svc.ID, "preserved-disabled-tool", false, 1)
	require.NoError(t, err)
	require.NoError(t, model.ToggleServiceEnabled(svc.ID))

	policy, err := model.GetMCPToolPolicy(svc.ID, "preserved-disabled-tool")
	require.NoError(t, err)
	require.False(t, policy.Enabled)
}
