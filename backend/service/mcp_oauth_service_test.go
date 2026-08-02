package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	_, _, err = manager.CompleteAuthorization(context.Background(), "wrong-state", "code")
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
	service.OAuthEnabled = false
	service.OAuthAuthStatus = MCPOAuthStatusAuthRequired
	require.NoError(t, model.UpdateService(service))
	handler.SetExpectedState("expected-state")
	manager := NewMCPOAuthManager()
	manager.flows["expected-state"] = &mcpOAuthFlow{
		ServiceID:    service.ID,
		State:        "expected-state",
		CodeVerifier: "code-verifier",
		Handler:      handler,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	completedServiceID, _, err := manager.CompleteAuthorization(context.Background(), "expected-state", "auth-code")
	require.NoError(t, err)
	require.Equal(t, service.ID, completedServiceID)

	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.True(t, updatedService.OAuthEnabled)
	require.Equal(t, MCPOAuthStatusAuthorized, updatedService.OAuthAuthStatus)
	token, err := newDBMCPOAuthTokenStore(service.ID).GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, "persisted-access", token.AccessToken)
	require.Equal(t, "persisted-refresh", token.RefreshToken)
}

func TestBeginAuthorizationDefaultsToAllAdvertisedScopes(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 9)
	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/oauth-authorization-server" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"issuer":"` + "http://" + r.Host + `",
			"authorization_endpoint":"` + "http://" + r.Host + `/authorize",
			"token_endpoint":"` + "http://" + r.Host + `/token",
			"response_types_supported":["code"],
			"code_challenge_methods_supported":["S256"],
			"scopes_supported":["category.read","offline_access","todo.create","todo.read"]
		}`))
	}))
	defer oauthHTTP.Close()

	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	record.ClientID = "client-id"
	record.AuthServerMetadataURL = oauthHTTP.URL + "/.well-known/oauth-authorization-server"
	require.NoError(t, model.SaveMCPOAuth(record))

	authorizationURL, err := NewMCPOAuthManager().BeginAuthorization(context.Background(), service.ID, "http://localhost:5173/services")
	require.NoError(t, err)
	parsedAuthorizationURL, err := url.Parse(authorizationURL)
	require.NoError(t, err)
	require.Equal(t, "category.read offline_access todo.create todo.read", parsedAuthorizationURL.Query().Get("scope"))

	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.Equal(t, "category.read offline_access todo.create todo.read", updatedService.OAuthScopes)
}

func TestConfigurePreservesExistingAuthorizationUntilReplacementSucceeds(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 10)
	store := newDBMCPOAuthTokenStore(service.ID)
	require.NoError(t, store.SaveToken(context.Background(), &transport.Token{
		AccessToken:  "existing-access",
		RefreshToken: "existing-refresh",
		TokenType:    "Bearer",
	}))
	originalRecord, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	originalEncryptedToken := originalRecord.EncryptedToken

	status, err := NewMCPOAuthManager().Configure(context.Background(), service.ID, MCPOAuthConfigInput{
		Scopes: []string{"todo.read", "todo.write"},
	})

	require.NoError(t, err)
	require.True(t, status.Authorized)
	require.Equal(t, MCPOAuthStatusAuthorized, status.Status)
	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.Equal(t, MCPOAuthStatusAuthorized, updatedService.OAuthAuthStatus)
	updatedRecord, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	require.Equal(t, originalEncryptedToken, updatedRecord.EncryptedToken)
}

func TestFailedReauthorizationPreservesExistingAuthorization(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 11)
	store := newDBMCPOAuthTokenStore(service.ID)
	require.NoError(t, store.SaveToken(context.Background(), &transport.Token{
		AccessToken: "existing-access",
		TokenType:   "Bearer",
	}))

	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + "http://" + r.Host + `","authorization_endpoint":"` + "http://" + r.Host + `/authorize","token_endpoint":"` + "http://" + r.Host + `/token","response_types_supported":["code"]}`))
		case "/token":
			http.Error(w, `{"error":"access_denied"}`, http.StatusBadRequest)
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
	handler.SetExpectedState("reauthorize-state")
	manager := NewMCPOAuthManager()
	manager.flows["reauthorize-state"] = &mcpOAuthFlow{
		ServiceID:    service.ID,
		State:        "reauthorize-state",
		CodeVerifier: "code-verifier",
		Handler:      handler,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	_, _, err = manager.CompleteAuthorization(context.Background(), "reauthorize-state", "rejected-code")

	require.Error(t, err)
	updatedService, getServiceErr := model.GetServiceByID(service.ID)
	require.NoError(t, getServiceErr)
	require.Equal(t, MCPOAuthStatusAuthorized, updatedService.OAuthAuthStatus)
	token, tokenErr := store.GetToken(context.Background())
	require.NoError(t, tokenErr)
	require.Equal(t, "existing-access", token.AccessToken)
}

func TestMCPOAuthConfigureRejectsInsecureRemoteURL(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 4)
	service.Command = "http://mcp.example.test/mcp"
	require.NoError(t, model.UpdateService(service))

	_, err := NewMCPOAuthManager().Configure(context.Background(), service.ID, MCPOAuthConfigInput{})

	require.ErrorContains(t, err, "requires HTTPS")
}

func TestMCPOAuthDisableClearsCredentialsAndPreservesAuthorizationConfiguration(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 5)
	service.OAuthScopes = "todo.read todo.write"
	require.NoError(t, model.UpdateService(service))
	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	record.ClientID = "client-id"
	record.EncryptedClientSecret, err = common.EncryptSecret("client-secret")
	require.NoError(t, err)
	record.AuthServerMetadataURL = "https://auth.example.test/.well-known/oauth-authorization-server"
	record.ProtectedResourceMetadataURL = "https://mcp.example.test/.well-known/oauth-protected-resource/mcp"
	require.NoError(t, model.SaveMCPOAuth(record))
	store := newDBMCPOAuthTokenStore(service.ID)
	require.NoError(t, store.SaveToken(context.Background(), &transport.Token{AccessToken: "secret", TokenType: "Bearer"}))

	require.NoError(t, NewMCPOAuthManager().Disable(service.ID))

	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.False(t, updatedService.OAuthEnabled)
	require.Equal(t, MCPOAuthStatusAuthRequired, updatedService.OAuthAuthStatus)
	require.Equal(t, "todo.read todo.write", updatedService.OAuthScopes)
	updatedRecord, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	require.Empty(t, updatedRecord.ClientID)
	require.Empty(t, updatedRecord.EncryptedClientSecret)
	require.Empty(t, updatedRecord.EncryptedToken)
	require.Equal(t, "https://auth.example.test/.well-known/oauth-authorization-server", updatedRecord.AuthServerMetadataURL)
	require.Equal(t, "https://mcp.example.test/.well-known/oauth-protected-resource/mcp", updatedRecord.ProtectedResourceMetadataURL)
}

func TestBeginAuthorizationAllowsDisabledConfiguredServiceWithoutEnablingIt(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 12)
	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + "http://" + r.Host + `","authorization_endpoint":"` + "http://" + r.Host + `/authorize","token_endpoint":"` + "http://" + r.Host + `/token","registration_endpoint":"` + "http://" + r.Host + `/register","response_types_supported":["code"],"code_challenge_methods_supported":["S256"]}`))
		case "/register":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"client_id":"new-client-id"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauthHTTP.Close()
	record, err := model.GetMCPOAuthByServiceID(service.ID)
	require.NoError(t, err)
	record.AuthServerMetadataURL = oauthHTTP.URL + "/.well-known/oauth-authorization-server"
	require.NoError(t, model.SaveMCPOAuth(record))
	require.NoError(t, NewMCPOAuthManager().Disable(service.ID))

	manager := NewMCPOAuthManager()
	authorizationURL, err := manager.BeginAuthorization(context.Background(), service.ID, "http://localhost:5173/services")

	require.NoError(t, err)
	require.Contains(t, authorizationURL, oauthHTTP.URL+"/authorize")
	updatedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.False(t, updatedService.OAuthEnabled)
	require.Equal(t, MCPOAuthStatusAuthRequired, updatedService.OAuthAuthStatus)
	parsedAuthorizationURL, err := url.Parse(authorizationURL)
	require.NoError(t, err)
	cancelledServiceID, _ := manager.CancelAuthorization(parsedAuthorizationURL.Query().Get("state"))
	require.Equal(t, service.ID, cancelledServiceID)
	require.NoError(t, RestoreMCPOAuthStatus(cancelledServiceID))
	updatedService, err = model.GetServiceByID(service.ID)
	require.NoError(t, err)
	require.False(t, updatedService.OAuthEnabled)
	require.Equal(t, MCPOAuthStatusAuthRequired, updatedService.OAuthAuthStatus)
}

func TestFailedAuthorizationKeepsDisabledServiceDisabled(t *testing.T) {
	setupMCPOAuthTestDB(t)
	service := createMCPOAuthTestService(t, 13)
	oauthHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + "http://" + r.Host + `","authorization_endpoint":"` + "http://" + r.Host + `/authorize","token_endpoint":"` + "http://" + r.Host + `/token","response_types_supported":["code"]}`))
		case "/token":
			http.Error(w, `{"error":"access_denied"}`, http.StatusBadRequest)
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
	handler.SetExpectedState("disabled-state")
	require.NoError(t, NewMCPOAuthManager().Disable(service.ID))
	manager := NewMCPOAuthManager()
	manager.flows["disabled-state"] = &mcpOAuthFlow{
		ServiceID:    service.ID,
		State:        "disabled-state",
		CodeVerifier: "code-verifier",
		Handler:      handler,
		ExpiresAt:    time.Now().Add(time.Minute),
	}

	_, _, err = manager.CompleteAuthorization(context.Background(), "disabled-state", "rejected-code")

	require.Error(t, err)
	updatedService, getServiceErr := model.GetServiceByID(service.ID)
	require.NoError(t, getServiceErr)
	require.False(t, updatedService.OAuthEnabled)
	require.Equal(t, MCPOAuthStatusAuthRequired, updatedService.OAuthAuthStatus)
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
