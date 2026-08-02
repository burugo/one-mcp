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
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateCustomServiceWithDiscoveredOAuthWaitsForAuthorization(t *testing.T) {
	originalSQLitePath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "custom-oauth.db")
	t.Cleanup(func() { common.SQLitePath = originalSQLitePath })
	require.NoError(t, model.InitDB())

	requestBody := []byte(`{
        "name":"Protected Remote MCP",
        "type":"streamableHttp",
        "url":"https://mcp.example.test/mcp",
        "oauth_enabled":true,
        "oauth_scopes":"mcp:read",
        "oauth_protected_resource_metadata_url":"https://mcp.example.test/.well-known/oauth-protected-resource/mcp"
    }`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/mcp_market/custom_service", bytes.NewReader(requestBody))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request = ctx.Request.WithContext(context.Background())
	ctx.Set("lang", "en")

	CreateCustomService(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response common.APIResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	data, ok := response.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, data["oauth_auth_required"])

	service, err := model.GetServiceByName("protected-remote-mcp")
	require.NoError(t, err)
	require.True(t, service.OAuthEnabled)
	require.Equal(t, "mcp:read", service.OAuthScopes)
	require.Equal(t, "auth_required", service.OAuthAuthStatus)
	oauthRecord, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	require.Equal(t, "https://mcp.example.test/.well-known/oauth-protected-resource/mcp", oauthRecord.ProtectedResourceMetadataURL)
}
