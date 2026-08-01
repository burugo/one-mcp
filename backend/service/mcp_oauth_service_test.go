package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/stretchr/testify/require"
)

func TestMCPOAuthTokenStoreRoundTrip(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 1)
	store := newDBMCPOAuthTokenStore(service.ID)

	want := &transport.Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		Scope:        "read write",
	}
	require.NoError(t, store.SaveToken(context.Background(), want))

	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	require.NotContains(t, record.EncryptedToken, want.AccessToken)
	require.NotContains(t, record.EncryptedToken, want.RefreshToken)

	got, err := store.GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, want.AccessToken, got.AccessToken)
	require.Equal(t, want.RefreshToken, got.RefreshToken)
	require.Equal(t, want.Scope, got.Scope)
}

func TestMCPOAuthCallbackRejectsInvalidState(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 2)
	callbackCalled := false
	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + "http://" + r.Host + `","authorization_endpoint":"` + "http://" + r.Host + `/authorize","token_endpoint":"` + "http://" + r.Host + `/token","response_types_supported":["code"]}`))
		case "/token":
			callbackCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"should-not-be-issued","token_type":"Bearer"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthHTTP.Close()

	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	record.ClientID = "client-id"
	record.AuthServerMetadataURL = oauthHTTP.URL + "/.well-known/oauth-authorization-server"
	require.NoError(t, model.SaveMCPOAuth(record))

	manager := NewMCPOAuthManager()
	manager.flows["expected-state"] = &mcpOAuthFlow{
		ServiceID:    service.ID,
		State:        "expected-state",
		CodeVerifier: "code-verifier",
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	_, err = manager.CompleteAuthorization(context.Background(), "wrong-state", "code")
	require.ErrorIs(t, err, ErrInvalidMCPOAuthState)
	require.False(t, callbackCalled)
}

func TestMCPOAuthCallbackPersistsTokenAndAuthorizesService(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 3)
	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + "http://" + r.Host + `","authorization_endpoint":"` + "http://" + r.Host + `/authorize","token_endpoint":"` + "http://" + r.Host + `/token","response_types_supported":["code"]}`))
		case "/token":
			require.Equal(t, "authorization_code", r.FormValue("grant_type"))
			require.Equal(t, "code-verifier", r.FormValue("code_verifier"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"persisted-access","token_type":"Bearer","refresh_token":"persisted-refresh","expires_in":3600}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthHTTP.Close()

	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	record.ClientID = "client-id"
	record.AuthServerMetadataURL = oauthHTTP.URL + "/.well-known/oauth-authorization-server"
	require.NoError(t, model.SaveMCPOAuth(record))

	handler, _, err := newMCPOAuthHandler(service)
	require.NoError(t, err)
	handler.SetExpectedState("expected-state")
	manager := NewMCPOAuthManager()
	manager.flows["expected-state"] = &mcpOAuthFlow{
		ServiceID:    service.ID,
		State:        "expected-state",
		CodeVerifier: "code-verifier",
		Handler:      handler,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	completedServiceID, err := manager.CompleteAuthorization(context.Background(), "expected-state", "auth-code")
	require.NoError(t, err)
	require.Equal(t, service.ID, completedServiceID)

	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.Equal(t, MCPOAuthStatusAuthorized, updatedService.OAuthAuthStatus)
	token, err := newDBMCPOAuthTokenStore(service.ID).GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, "persisted-access", token.AccessToken)
	require.Equal(t, "persisted-refresh", token.RefreshToken)
}

func TestMCPOAuthConfigureRejectsInsecureRemoteURL(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 4)
	service.Command = "http://mcp.example.test/mcp"
	require.NoError(t, model.UpdateService(service))

	_, err := NewMCPOAuthManager().Configure(context.Background(), service.ID, MCPOAuthConfigInput{})

	require.ErrorContains(t, err, "requires HTTPS")
}

func TestMCPOAuthDisableDeletesCredentials(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 5)
	store := newDBMCPOAuthTokenStore(service.ID)
	require.NoError(t, store.SaveToken(context.Background(), &transport.Token{AccessToken: "secret", TokenType: "Bearer"}))

	require.NoError(t, NewMCPOAuthManager().Disable(service.ID))

	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.False(t, updatedService.OAuthEnabled)
	require.Equal(t, MCPOAuthStatusNotConfigured, updatedService.OAuthAuthStatus)
	_, err = model.GetMCPOAuthByServiceID(service.ID)
	require.ErrorIs(t, err, model.ErrMCPOAuthNotFound)
}

func TestValidateMCPOAuthServerMetadataRejectsInsecureEndpoints(t *testing.T) {
	err := validateMCPOAuthServerMetadata(&transport.AuthServerMetadata{
		AuthorizationEndpoint: "https://auth.example.test/authorize",
		TokenEndpoint:         "http://auth.example.test/token",
	})

	require.ErrorContains(t, err, "requires HTTPS")
}

func TestMCPOAuthRefreshPersistsNewAccessToken(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 6)
	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + "http://" + r.Host + `","authorization_endpoint":"` + "http://" + r.Host + `/authorize","token_endpoint":"` + "http://" + r.Host + `/token","response_types_supported":["code"]}`))
		case "/token":
			require.Equal(t, "refresh_token", r.FormValue("grant_type"))
			require.Equal(t, "old-refresh", r.FormValue("refresh_token"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"refreshed-access","token_type":"Bearer","expires_in":3600}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthHTTP.Close()

	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	record.ClientID = "client-id"
	record.AuthServerMetadataURL = oauthHTTP.URL + "/.well-known/oauth-authorization-server"
	require.NoError(t, model.SaveMCPOAuth(record))
	store := newDBMCPOAuthTokenStore(service.ID)
	require.NoError(t, saveTokenWithoutStatus(context.Background(), store, &transport.Token{
		AccessToken:  "expired-access",
		TokenType:    "Bearer",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}))

	handler, _, err := newMCPOAuthHandler(service)
	require.NoError(t, err)
	header, err := handler.GetAuthorizationHeader(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Bearer refreshed-access", header)

	token, err := store.GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, "refreshed-access", token.AccessToken)
	require.Equal(t, "old-refresh", token.RefreshToken)
}

func saveTokenWithoutStatus(ctx context.Context, store *dbMCPOAuthTokenStore, token *transport.Token) error {
	service, err := model.GetServiceByID(store.serviceID)
	if err != nil {
		return err
	}
	service.OAuthEnabled = false
	if err := model.UpdateService(service); err != nil {
		return err
	}
	if err := store.SaveToken(ctx, token); err != nil {
		return err
	}
	service.OAuthEnabled = true
	service.OAuthAuthStatus = MCPOAuthStatusAuthorized
	return model.UpdateService(service)
}

func setupMCPOAuthTestDB(t *testing.T) {
	t.Helper()
	originalPath := common.SQLitePath
	originalSecret := common.JWTSecret
	common.SQLitePath = t.TempDir() + "/oauth.db"
	common.JWTSecret = "stable-test-secret"
	require.NoError(t, model.InitDB())
	t.Cleanup(func() {
		common.SQLitePath = originalPath
		common.JWTSecret = originalSecret
	})
}

func createMCPOAuthTestService(t *testing.T, suffix int64) *model.MCPService {
	t.Helper()
	service := &model.MCPService{
		Name:            "oauth-service-" + string(rune('a'+suffix)),
		DisplayName:     "OAuth Service",
		Type:            model.ServiceTypeStreamableHTTP,
		Command:         "https://mcp.example.test/mcp",
		Enabled:         true,
		OAuthEnabled:    true,
		OAuthAuthStatus: MCPOAuthStatusAuthRequired,
	}
	require.NoError(t, model.CreateService(service))
	require.NoError(t, model.SaveMCPOAuth(&model.MCPOAuth{ServiceID: service.ID}))
	return service
}
