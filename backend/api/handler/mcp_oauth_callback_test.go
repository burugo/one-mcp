package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/model"
	appservice "one-mcp/backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMCPOAuthCallbackRedirectsToLocalDevFrontend(t *testing.T) {
	originalSQLitePath := common.SQLitePath
	originalJWTSecret := common.JWTSecret
	common.SQLitePath = filepath.Join(t.TempDir(), "oauth-callback.db")
	common.JWTSecret = "oauth-callback-test-secret"
	t.Cleanup(func() {
		common.SQLitePath = originalSQLitePath
		common.JWTSecret = originalJWTSecret
	})
	require.NoError(t, model.InitDB())

	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + oauthServerURL(r) + `","authorization_endpoint":"` + oauthServerURL(r) + `/authorize","token_endpoint":"` + oauthServerURL(r) + `/token","response_types_supported":["code"],"code_challenge_methods_supported":["S256"]}`))
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"callback-access-token","token_type":"Bearer"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthServer.Close()

	mcpService := &model.MCPService{
		Name:            "oauth-callback-service",
		DisplayName:     "OAuth Callback Service",
		Type:            model.ServiceTypeStreamableHTTP,
		Command:         "https://mcp.example.test/mcp",
		OAuthEnabled:    true,
		OAuthAuthStatus: "auth_required",
	}
	require.NoError(t, model.CreateService(mcpService))
	require.NoError(t, model.SaveMCPOAuth(&model.MCPOAuth{
		ServiceID:             mcpService.ID,
		ClientID:              "client-id",
		AuthServerMetadataURL: oauthServer.URL + "/.well-known/oauth-authorization-server",
	}))

	authorizeRecorder := httptest.NewRecorder()
	authorizeContext, _ := gin.CreateTestContext(authorizeRecorder)
	serviceID := strconv.FormatInt(mcpService.ID, 10)
	authorizeContext.Params = gin.Params{{Key: "id", Value: serviceID}}
	authorizeContext.Request = httptest.NewRequest(http.MethodPost, "/api/mcp_services/"+serviceID+"/oauth/authorize", nil)
	authorizeContext.Request.Header.Set("Origin", "http://localhost:5173")
	AuthorizeMCPOAuth(authorizeContext)
	require.Equal(t, http.StatusOK, authorizeRecorder.Code)

	var authorizeResponse struct {
		Success bool `json:"success"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(authorizeRecorder.Body.Bytes(), &authorizeResponse))
	require.True(t, authorizeResponse.Success)
	authorizationURL, err := url.Parse(authorizeResponse.Data.AuthorizationURL)
	require.NoError(t, err)
	state := authorizationURL.Query().Get("state")
	require.NotEmpty(t, state)

	callbackRecorder := httptest.NewRecorder()
	callbackContext, _ := gin.CreateTestContext(callbackRecorder)
	callbackContext.Request = httptest.NewRequest(http.MethodGet, "/api/mcp_oauth/callback?code=auth-code&state="+url.QueryEscape(state), nil)
	MCPOAuthCallback(callbackContext)

	require.Equal(t, http.StatusOK, callbackRecorder.Code)
	require.Contains(t, callbackRecorder.Header().Get("Content-Type"), "text/html")
	require.Contains(t, callbackRecorder.Body.String(), `window.location.replace("http://localhost:5173/services?mcp_oauth=success")`)
	require.Contains(t, callbackRecorder.Body.String(), `http-equiv="refresh"`)

	errorAuthorizeRecorder := httptest.NewRecorder()
	errorAuthorizeContext, _ := gin.CreateTestContext(errorAuthorizeRecorder)
	errorAuthorizeContext.Params = gin.Params{{Key: "id", Value: serviceID}}
	errorAuthorizeContext.Request = httptest.NewRequest(http.MethodPost, "/api/mcp_services/"+serviceID+"/oauth/authorize", nil)
	errorAuthorizeContext.Request.Header.Set("Origin", "http://localhost:5173")
	AuthorizeMCPOAuth(errorAuthorizeContext)
	require.Equal(t, http.StatusOK, errorAuthorizeRecorder.Code)
	require.NoError(t, json.Unmarshal(errorAuthorizeRecorder.Body.Bytes(), &authorizeResponse))
	errorAuthorizationURL, err := url.Parse(authorizeResponse.Data.AuthorizationURL)
	require.NoError(t, err)

	errorCallbackRecorder := httptest.NewRecorder()
	errorCallbackContext, _ := gin.CreateTestContext(errorCallbackRecorder)
	errorCallbackContext.Request = httptest.NewRequest(http.MethodGet, "/api/mcp_oauth/callback?error=access_denied&state="+url.QueryEscape(errorAuthorizationURL.Query().Get("state")), nil)
	MCPOAuthCallback(errorCallbackContext)

	require.Equal(t, http.StatusOK, errorCallbackRecorder.Code)
	require.Contains(t, errorCallbackRecorder.Body.String(), `window.location.replace("http://localhost:5173/services?mcp_oauth=error\u0026message=access_denied")`)
}

func TestMCPOAuthInvalidStateStillReturnsToSignedFrontendURL(t *testing.T) {
	state, err := appservice.NewMCPOAuthState("http://localhost:5173/services")
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/mcp_oauth/callback?code=auth-code&state="+url.QueryEscape(state), nil)
	MCPOAuthCallback(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `http://localhost:5173/services?mcp_oauth=error`)
	require.Contains(t, recorder.Body.String(), `invalid_mcp_oauth_state`)
}

func TestMCPOAuthFrontendReturnURLRejectsArbitraryOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/mcp_services/1/oauth/authorize", nil)
	request.Header.Set("Origin", "https://attacker.example.test")

	require.Equal(t, "http://localhost:3000/services", mcpOAuthFrontendReturnURL(request))
}

func oauthServerURL(r *http.Request) string {
	return "http://" + r.Host
}
