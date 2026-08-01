package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

const (
	MCPOAuthStatusNotConfigured = "not_configured"
	MCPOAuthStatusAuthRequired  = "auth_required"
	MCPOAuthStatusAuthorized    = "authorized"
)

var (
	ErrMCPOAuthNotEnabled   = errors.New("mcp_oauth_not_enabled")
	ErrInvalidMCPOAuthState = errors.New("invalid_mcp_oauth_state")
	ErrExpiredMCPOAuthFlow  = errors.New("expired_mcp_oauth_flow")
)

type MCPOAuthConfigInput struct {
	ClientID                     string
	ClientSecret                 string
	Scopes                       []string
	AuthServerMetadataURL        string
	ProtectedResourceMetadataURL string
}

type MCPOAuthStatus struct {
	Enabled    bool   `json:"enabled"`
	Status     string `json:"status"`
	Authorized bool   `json:"authorized"`
	ClientID   string `json:"client_id,omitempty"`
	Scopes     string `json:"scopes,omitempty"`
	Callback   string `json:"callback_url"`
}

type MCPOAuthManager struct {
	mu    sync.Mutex
	flows map[string]*mcpOAuthFlow
}

type mcpOAuthFlow struct {
	ServiceID    int64
	State        string
	CodeVerifier string
	Handler      *transport.OAuthHandler
	ExpiresAt    time.Time
}

var globalMCPOAuthManager = NewMCPOAuthManager()

func NewMCPOAuthManager() *MCPOAuthManager {
	return &MCPOAuthManager{flows: make(map[string]*mcpOAuthFlow)}
}

func GetMCPOAuthManager() *MCPOAuthManager {
	return globalMCPOAuthManager
}

func (m *MCPOAuthManager) Configure(ctx context.Context, serviceID int64, input MCPOAuthConfigInput) (*MCPOAuthStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return nil, err
	}
	if service.Type != model.ServiceTypeSSE && service.Type != model.ServiceTypeStreamableHTTP {
		return nil, fmt.Errorf("OAuth is only supported for remote MCP services")
	}
	if err := validateSecureOAuthURL(service.Command, "remote MCP URL"); err != nil {
		return nil, err
	}
	if err := validateMCPOAuthMetadataURL(input.AuthServerMetadataURL); err != nil {
		return nil, err
	}
	if err := validateMCPOAuthMetadataURL(input.ProtectedResourceMetadataURL); err != nil {
		return nil, err
	}

	record, err := getOrCreateMCPOAuthRecord(serviceID)
	if err != nil {
		return nil, err
	}
	if input.ClientID != "" {
		if record.ClientID != "" && record.ClientID != strings.TrimSpace(input.ClientID) {
			record.EncryptedToken = ""
		}
		record.ClientID = strings.TrimSpace(input.ClientID)
	}
	if input.AuthServerMetadataURL != "" {
		metadataURL := strings.TrimSpace(input.AuthServerMetadataURL)
		if record.AuthServerMetadataURL != "" && record.AuthServerMetadataURL != metadataURL {
			record.EncryptedToken = ""
		}
		record.AuthServerMetadataURL = metadataURL
	}
	if input.ProtectedResourceMetadataURL != "" {
		metadataURL := strings.TrimSpace(input.ProtectedResourceMetadataURL)
		if record.ProtectedResourceMetadataURL != "" && record.ProtectedResourceMetadataURL != metadataURL {
			record.EncryptedToken = ""
		}
		record.ProtectedResourceMetadataURL = metadataURL
	}
	if input.ClientSecret != "" {
		record.EncryptedClientSecret, err = common.EncryptSecret(input.ClientSecret)
		if err != nil {
			return nil, err
		}
	}
	if err := model.SaveMCPOAuth(record); err != nil {
		return nil, err
	}

	service.OAuthEnabled = true
	service.OAuthScopes = strings.Join(input.Scopes, " ")
	service.OAuthAuthStatus = MCPOAuthStatusAuthRequired
	if err := model.UpdateService(service); err != nil {
		return nil, err
	}
	return m.Status(serviceID)
}

func (m *MCPOAuthManager) Status(serviceID int64) (*MCPOAuthStatus, error) {
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return nil, err
	}
	record, err := model.GetMCPOAuthByServiceID(serviceID)
	if err != nil && !errors.Is(err, model.ErrMCPOAuthNotFound) {
		return nil, err
	}
	status := &MCPOAuthStatus{
		Enabled:  service.OAuthEnabled,
		Status:   service.OAuthAuthStatus,
		Scopes:   service.OAuthScopes,
		Callback: MCPOAuthCallbackURL(),
	}
	if status.Status == "" {
		status.Status = MCPOAuthStatusNotConfigured
	}
	if record != nil {
		status.ClientID = record.ClientID
		status.Authorized = record.EncryptedToken != "" && service.OAuthAuthStatus == MCPOAuthStatusAuthorized
	}
	return status, nil
}

func (m *MCPOAuthManager) Disable(serviceID int64) error {
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return err
	}
	if err := model.DeleteMCPOAuthByServiceID(serviceID); err != nil {
		return err
	}
	m.mu.Lock()
	for state, flow := range m.flows {
		if flow != nil && flow.ServiceID == serviceID {
			delete(m.flows, state)
		}
	}
	m.mu.Unlock()
	service.OAuthEnabled = false
	service.OAuthScopes = ""
	service.OAuthAuthStatus = MCPOAuthStatusNotConfigured
	return model.UpdateService(service)
}

func (m *MCPOAuthManager) BeginAuthorization(ctx context.Context, serviceID int64) (string, error) {
	m.pruneExpiredFlows()
	if err := validateSecureOAuthURL(MCPOAuthCallbackURL(), "OAuth callback URL"); err != nil {
		return "", err
	}
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return "", err
	}
	if !service.OAuthEnabled {
		return "", ErrMCPOAuthNotEnabled
	}

	handler, record, err := newMCPOAuthHandler(service)
	if err != nil {
		return "", err
	}
	metadata, err := handler.GetServerMetadata(ctx)
	if err != nil {
		return "", err
	}
	if err := validateMCPOAuthServerMetadata(metadata); err != nil {
		return "", err
	}
	if record.ClientID == "" {
		if err := handler.RegisterClient(ctx, "one-mcp"); err != nil {
			return "", err
		}
		record.ClientID = handler.GetClientID()
		if secret := handler.GetClientSecret(); secret != "" {
			record.EncryptedClientSecret, err = common.EncryptSecret(secret)
			if err != nil {
				return "", err
			}
		}
		if err := model.SaveMCPOAuth(record); err != nil {
			return "", err
		}
	}

	state, err := mcpclient.GenerateState()
	if err != nil {
		return "", err
	}
	verifier, err := mcpclient.GenerateCodeVerifier()
	if err != nil {
		return "", err
	}
	authorizationURL, err := handler.GetAuthorizationURL(ctx, state, mcpclient.GenerateCodeChallenge(verifier))
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	m.flows[state] = &mcpOAuthFlow{
		ServiceID:    serviceID,
		State:        state,
		CodeVerifier: verifier,
		Handler:      handler,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	m.mu.Unlock()
	return authorizationURL, nil
}

func (m *MCPOAuthManager) CancelAuthorization(state string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	flow := m.flows[state]
	delete(m.flows, state)
	if flow == nil {
		return 0
	}
	return flow.ServiceID
}

func (m *MCPOAuthManager) pruneExpiredFlows() {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for state, flow := range m.flows {
		if flow == nil || now.After(flow.ExpiresAt) {
			delete(m.flows, state)
		}
	}
}

func (m *MCPOAuthManager) CompleteAuthorization(ctx context.Context, state, code string) (int64, error) {
	m.mu.Lock()
	flow, ok := m.flows[state]
	if ok {
		delete(m.flows, state)
	}
	m.mu.Unlock()
	if !ok {
		return 0, ErrInvalidMCPOAuthState
	}
	if time.Now().After(flow.ExpiresAt) {
		return flow.ServiceID, ErrExpiredMCPOAuthFlow
	}
	if flow.Handler == nil {
		service, err := model.GetServiceByID(flow.ServiceID)
		if err != nil {
			return flow.ServiceID, err
		}
		flow.Handler, _, err = newMCPOAuthHandler(service)
		if err != nil {
			return flow.ServiceID, err
		}
		flow.Handler.SetExpectedState(flow.State)
	}
	if err := flow.Handler.ProcessAuthorizationResponse(ctx, code, state, flow.CodeVerifier); err != nil {
		_ = SetMCPOAuthStatus(flow.ServiceID, MCPOAuthStatusAuthRequired)
		return flow.ServiceID, err
	}
	return flow.ServiceID, SetMCPOAuthStatus(flow.ServiceID, MCPOAuthStatusAuthorized)
}

func MCPOAuthCallbackURL() string {
	common.OptionMapRWMutex.RLock()
	serverAddress := strings.TrimSpace(common.GetServerAddress())
	common.OptionMapRWMutex.RUnlock()
	if serverAddress == "" {
		serverAddress = fmt.Sprintf("http://localhost:%d", *common.Port)
	}
	return strings.TrimRight(serverAddress, "/") + "/api/mcp_oauth/callback"
}

func BuildMCPOAuthConfig(service *model.MCPService, httpClient *http.Client) (transport.OAuthConfig, error) {
	if service == nil || !service.OAuthEnabled {
		return transport.OAuthConfig{}, ErrMCPOAuthNotEnabled
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	record, err := getOrCreateMCPOAuthRecord(service.ID)
	if err != nil {
		return transport.OAuthConfig{}, err
	}
	clientSecret, err := common.DecryptSecret(record.EncryptedClientSecret)
	if err != nil {
		return transport.OAuthConfig{}, err
	}
	return transport.OAuthConfig{
		ClientID:                     record.ClientID,
		ClientSecret:                 clientSecret,
		ClientURI:                    strings.TrimSuffix(MCPOAuthCallbackURL(), "/api/mcp_oauth/callback"),
		RedirectURI:                  MCPOAuthCallbackURL(),
		Scopes:                       strings.Fields(service.OAuthScopes),
		TokenStore:                   newDBMCPOAuthTokenStore(service.ID),
		AuthServerMetadataURL:        record.AuthServerMetadataURL,
		ProtectedResourceMetadataURL: record.ProtectedResourceMetadataURL,
		PKCEEnabled:                  true,
		HTTPClient:                   httpClient,
	}, nil
}

func SetMCPOAuthStatus(serviceID int64, status string) error {
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return err
	}
	if !service.OAuthEnabled {
		return nil
	}
	if service.OAuthAuthStatus == status {
		return nil
	}
	service.OAuthAuthStatus = status
	return model.UpdateService(service)
}

func IsMCPOAuthAuthorizationRequired(err error) bool {
	return mcpclient.IsOAuthAuthorizationRequiredError(err) || mcpclient.IsAuthorizationRequiredError(err) || errors.Is(err, transport.ErrOAuthAuthorizationRequired) || errors.Is(err, transport.ErrAuthorizationRequired)
}

func newMCPOAuthHandler(service *model.MCPService) (*transport.OAuthHandler, *model.MCPOAuth, error) {
	record, err := getOrCreateMCPOAuthRecord(service.ID)
	if err != nil {
		return nil, nil, err
	}
	config, err := BuildMCPOAuthConfig(service, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		return nil, nil, err
	}
	handler := transport.NewOAuthHandler(config)
	handler.SetBaseURL(service.Command)
	return handler, record, nil
}

func getOrCreateMCPOAuthRecord(serviceID int64) (*model.MCPOAuth, error) {
	record, err := model.GetMCPOAuthByServiceID(serviceID)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, model.ErrMCPOAuthNotFound) {
		return nil, err
	}
	record = &model.MCPOAuth{ServiceID: serviceID}
	if err := model.SaveMCPOAuth(record); err != nil {
		return nil, err
	}
	return record, nil
}

type dbMCPOAuthTokenStore struct {
	serviceID int64
}

func newDBMCPOAuthTokenStore(serviceID int64) *dbMCPOAuthTokenStore {
	return &dbMCPOAuthTokenStore{serviceID: serviceID}
}

func (s *dbMCPOAuthTokenStore) GetToken(ctx context.Context) (*transport.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	record, err := model.GetMCPOAuthByServiceID(s.serviceID)
	if err != nil {
		if errors.Is(err, model.ErrMCPOAuthNotFound) {
			return nil, transport.ErrNoToken
		}
		return nil, err
	}
	if record.EncryptedToken == "" {
		return nil, transport.ErrNoToken
	}
	plaintext, err := common.DecryptSecret(record.EncryptedToken)
	if err != nil {
		return nil, err
	}
	var token transport.Token
	if err := json.Unmarshal([]byte(plaintext), &token); err != nil {
		return nil, fmt.Errorf("decode OAuth token: %w", err)
	}
	return &token, nil
}

func (s *dbMCPOAuthTokenStore) SaveToken(ctx context.Context, token *transport.Token) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("OAuth token is nil")
	}
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode OAuth token: %w", err)
	}
	encrypted, err := common.EncryptSecret(string(data))
	if err != nil {
		return err
	}
	record, err := getOrCreateMCPOAuthRecord(s.serviceID)
	if err != nil {
		return err
	}
	record.EncryptedToken = encrypted
	if err := model.SaveMCPOAuth(record); err != nil {
		return err
	}
	return SetMCPOAuthStatus(s.serviceID, MCPOAuthStatusAuthorized)
}

func validateMCPOAuthMetadataURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	return validateSecureOAuthURL(rawURL, "OAuth metadata URL")
}

func validateSecureOAuthURL(rawURL, label string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid %s", label)
	}
	if parsed.Scheme != "https" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" {
		return fmt.Errorf("%s requires HTTPS for non-local hosts", label)
	}
	return nil
}

func validateMCPOAuthServerMetadata(metadata *transport.AuthServerMetadata) error {
	if metadata == nil {
		return fmt.Errorf("OAuth server metadata is unavailable")
	}
	if err := validateSecureOAuthURL(metadata.AuthorizationEndpoint, "OAuth authorization endpoint"); err != nil {
		return err
	}
	if err := validateSecureOAuthURL(metadata.TokenEndpoint, "OAuth token endpoint"); err != nil {
		return err
	}
	if metadata.RegistrationEndpoint != "" {
		if err := validateSecureOAuthURL(metadata.RegistrationEndpoint, "OAuth registration endpoint"); err != nil {
			return err
		}
	}
	return nil
}
