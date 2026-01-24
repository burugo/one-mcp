package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
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
	r.GET("/api/mcp-services/:id/tools", GetMCPServiceTools)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/mcp-services/"+fmt.Sprintf("%d", created.ID)+"/tools", nil)
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
