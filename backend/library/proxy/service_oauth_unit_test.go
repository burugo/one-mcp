package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"one-mcp/backend/service"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/stretchr/testify/require"
)

func TestMonitoredProxiedServiceSkipsReinitWhenOAuthAuthorizationRequired(t *testing.T) {
	dbConfig := &model.MCPService{
		Name:            "protected-service",
		Type:            model.ServiceTypeStreamableHTTP,
		Enabled:         true,
		OAuthEnabled:    true,
		OAuthAuthStatus: service.MCPOAuthStatusAuthRequired,
	}
	monitored := NewMonitoredProxiedService(
		NewBaseService(42, dbConfig.Name, dbConfig.Type),
		nil,
		dbConfig,
	)

	health, err := monitored.CheckHealth(context.Background())

	require.Error(t, err)
	require.Equal(t, StatusUnhealthy, health.Status)
	require.Equal(t, service.MCPOAuthStatusAuthRequired, health.ErrorMessage)
}

func TestServiceFactorySkipsInitializationWhenOAuthAuthorizationRequired(t *testing.T) {
	dbConfig := &model.MCPService{
		Name:            "protected-service",
		Type:            model.ServiceTypeStreamableHTTP,
		Enabled:         true,
		OAuthEnabled:    true,
		OAuthAuthStatus: service.MCPOAuthStatusAuthRequired,
	}

	createdService, err := ServiceFactory(dbConfig)

	require.NoError(t, err)
	monitored, ok := createdService.(*MonitoredProxiedService)
	require.True(t, ok)
	require.Nil(t, monitored.sharedInstance)
	require.Equal(t, service.MCPOAuthStatusAuthRequired, monitored.GetHealth().ErrorMessage)
}

func TestOAuthEnabledServiceMarksAuthorizationRequiredAfter401(t *testing.T) {
	originalPath := common.SQLitePath
	originalSecret := common.JWTSecret
	common.SQLitePath = t.TempDir() + "/oauth-401.db"
	common.JWTSecret = "oauth-401-secret"
	require.NoError(t, model.InitDB())
	t.Cleanup(func() {
		common.SQLitePath = originalPath
		common.JWTSecret = originalSecret
		sharedMCPServersMutex.Lock()
		sharedMCPServers = make(map[string]*SharedMcpInstance)
		sharedMCPServersMutex.Unlock()
	})

	protectedMCP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer stale-access", r.Header.Get("Authorization"))
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+"http://"+r.Host+`/.well-known/oauth-protected-resource"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer protectedMCP.Close()

	dbConfig := &model.MCPService{
		Name:            "oauth-401-service",
		DisplayName:     "OAuth 401 Service",
		Type:            model.ServiceTypeStreamableHTTP,
		Command:         protectedMCP.URL + "/mcp",
		Enabled:         true,
		OAuthEnabled:    true,
		OAuthAuthStatus: service.MCPOAuthStatusAuthorized,
	}
	require.NoError(t, model.CreateService(dbConfig))
	require.NoError(t, model.SaveMCPOAuth(&model.MCPOAuth{ServiceID: dbConfig.ID, ClientID: "client-id"}))
	oauthConfig, err := service.BuildMCPOAuthConfig(dbConfig, protectedMCP.Client())
	require.NoError(t, err)
	dbConfig.OAuthEnabled = false
	require.NoError(t, model.UpdateService(dbConfig))
	require.NoError(t, oauthConfig.TokenStore.SaveToken(context.Background(), &transport.Token{
		AccessToken: "stale-access",
		TokenType:   "Bearer",
	}))
	dbConfig.OAuthEnabled = true
	dbConfig.OAuthAuthStatus = service.MCPOAuthStatusAuthorized
	require.NoError(t, model.UpdateService(dbConfig))

	createdService, err := ServiceFactory(dbConfig)

	require.NoError(t, err)
	monitored, ok := createdService.(*MonitoredProxiedService)
	require.True(t, ok)
	require.Nil(t, monitored.sharedInstance)
	updatedService, err := model.GetServiceByID(dbConfig.ID)
	require.NoError(t, err)
	require.Equal(t, service.MCPOAuthStatusAuthRequired, updatedService.OAuthAuthStatus)
}
