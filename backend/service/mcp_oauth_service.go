package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
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

type MCPOAuthDiscovery struct {
	OAuthAvailable                     bool     `json:"oauth_available"`
	AutomaticAuthorizationSupported    bool     `json:"automatic_authorization_supported"`
	AuthorizationServer                string   `json:"authorization_server,omitempty"`
	ProtectedResourceMetadataURL       string   `json:"protected_resource_metadata_url,omitempty"`
	Scopes                             []string `json:"scopes,omitempty"`
	DynamicClientRegistrationSupported bool     `json:"dynamic_client_registration_supported"`
	PKCES256Supported                  bool     `json:"pkce_s256_supported"`
}

type MCPOAuthManager struct {
	mu    sync.Mutex
	flows map[string]*mcpOAuthFlow
}

type mcpOAuthFlow struct {
	ServiceID    int64
	State        string
	CodeVerifier string
	ReturnURL    string
	Handler      *transport.OAuthHandler
	ExpiresAt    time.Time
}

type mcpOAuthStatePayload struct {
	Nonce     string `json:"nonce"`
	ReturnURL string `json:"return_url"`
	ExpiresAt int64  `json:"expires_at"`
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
	hadToken := record.EncryptedToken != ""
	if input.ClientID != "" {
		record.ClientID = strings.TrimSpace(input.ClientID)
	}
	if input.AuthServerMetadataURL != "" {
		metadataURL := strings.TrimSpace(input.AuthServerMetadataURL)
		record.AuthServerMetadataURL = metadataURL
	}
	if input.ProtectedResourceMetadataURL != "" {
		metadataURL := strings.TrimSpace(input.ProtectedResourceMetadataURL)
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
	if hadToken {
		service.OAuthAuthStatus = MCPOAuthStatusAuthorized
	} else {
		service.OAuthAuthStatus = MCPOAuthStatusAuthRequired
	}
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
	record, err := model.GetMCPOAuthByServiceID(serviceID)
	if err != nil && !errors.Is(err, model.ErrMCPOAuthNotFound) {
		return err
	}
	if record != nil {
		record.ClientID = ""
		record.EncryptedClientSecret = ""
		record.EncryptedToken = ""
		if err := model.SaveMCPOAuth(record); err != nil {
			return err
		}
	}
	m.mu.Lock()
	for state, flow := range m.flows {
		if flow != nil && flow.ServiceID == serviceID {
			delete(m.flows, state)
		}
	}
	m.mu.Unlock()
	service.OAuthEnabled = false
	service.OAuthAuthStatus = MCPOAuthStatusAuthRequired
	return model.UpdateService(service)
}

func (m *MCPOAuthManager) BeginAuthorization(ctx context.Context, serviceID int64, returnURL string) (string, error) {
	m.pruneExpiredFlows()
	if err := validateSecureOAuthURL(MCPOAuthCallbackURL(), "OAuth callback URL"); err != nil {
		return "", err
	}
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return "", err
	}
	if !service.OAuthEnabled {
		if _, err := model.GetMCPOAuthByServiceID(serviceID); err != nil {
			if errors.Is(err, model.ErrMCPOAuthNotFound) {
				return "", ErrMCPOAuthNotEnabled
			}
			return "", err
		}
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
	if strings.TrimSpace(service.OAuthScopes) == "" && len(metadata.ScopesSupported) > 0 {
		service.OAuthScopes = strings.Join(metadata.ScopesSupported, " ")
		if err := model.UpdateService(service); err != nil {
			return "", err
		}
		handler, record, err = newMCPOAuthHandler(service)
		if err != nil {
			return "", err
		}
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

	state, err := NewMCPOAuthState(returnURL)
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
		ReturnURL:    returnURL,
		Handler:      handler,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	m.mu.Unlock()
	return authorizationURL, nil
}

func (m *MCPOAuthManager) CancelAuthorization(state string) (int64, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	flow := m.flows[state]
	delete(m.flows, state)
	if flow == nil {
		return 0, ""
	}
	return flow.ServiceID, flow.ReturnURL
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

func (m *MCPOAuthManager) CompleteAuthorization(ctx context.Context, state, code string) (int64, string, error) {
	m.mu.Lock()
	flow, ok := m.flows[state]
	if ok {
		delete(m.flows, state)
	}
	m.mu.Unlock()
	if !ok {
		return 0, "", ErrInvalidMCPOAuthState
	}
	if time.Now().After(flow.ExpiresAt) {
		return flow.ServiceID, flow.ReturnURL, ErrExpiredMCPOAuthFlow
	}
	if flow.Handler == nil {
		service, err := model.GetServiceByID(flow.ServiceID)
		if err != nil {
			return flow.ServiceID, flow.ReturnURL, err
		}
		flow.Handler, _, err = newMCPOAuthHandler(service)
		if err != nil {
			return flow.ServiceID, flow.ReturnURL, err
		}
		flow.Handler.SetExpectedState(flow.State)
	}
	if err := flow.Handler.ProcessAuthorizationResponse(ctx, code, state, flow.CodeVerifier); err != nil {
		_ = RestoreMCPOAuthStatus(flow.ServiceID)
		return flow.ServiceID, flow.ReturnURL, err
	}
	return flow.ServiceID, flow.ReturnURL, enableMCPOAuthAfterAuthorization(flow.ServiceID)
}

func NewMCPOAuthState(returnURL string) (string, error) {
	parsedReturnURL, err := url.Parse(returnURL)
	if err != nil || parsedReturnURL.Scheme == "" || parsedReturnURL.Host == "" || parsedReturnURL.User != nil {
		return "", fmt.Errorf("invalid MCP OAuth return URL")
	}
	if parsedReturnURL.Scheme != "http" && parsedReturnURL.Scheme != "https" {
		return "", fmt.Errorf("invalid MCP OAuth return URL scheme")
	}
	nonce, err := mcpclient.GenerateState()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(mcpOAuthStatePayload{
		Nonce:     nonce,
		ReturnURL: returnURL,
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signMCPOAuthState(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func MCPOAuthReturnURLFromState(state string) (string, bool) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return "", false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, signMCPOAuthState(parts[0])) {
		return "", false
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	var payload mcpOAuthStatePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil || payload.ExpiresAt < time.Now().Unix() {
		return "", false
	}
	parsedReturnURL, err := url.Parse(payload.ReturnURL)
	if err != nil || parsedReturnURL.Scheme == "" || parsedReturnURL.Host == "" || parsedReturnURL.User != nil {
		return "", false
	}
	if parsedReturnURL.Scheme != "http" && parsedReturnURL.Scheme != "https" {
		return "", false
	}
	return payload.ReturnURL, true
}

func signMCPOAuthState(payload string) []byte {
	mac := hmac.New(sha256.New, []byte(common.JWTSecret))
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
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

type mcpProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

var (
	resourceMetadataPattern = regexp.MustCompile(`(?i)resource_metadata\s*=\s*"([^"]+)"`)
	scopeChallengePattern   = regexp.MustCompile(`(?i)(?:^|[,\s])scope\s*=\s*"([^"]*)"`)
)

func DiscoverMCPOAuth(ctx context.Context, endpoint string, serviceType model.ServiceType, httpClient *http.Client) (*MCPOAuthDiscovery, error) {
	if err := validateSecureOAuthURL(endpoint, "remote MCP URL"); err != nil {
		return nil, err
	}
	if httpClient == nil {
		var err error
		httpClient, err = newMCPOAuthDiscoveryHTTPClient(endpoint)
		if err != nil {
			return nil, err
		}
	}

	metadataURL, challengeScopes := discoverMCPOAuthChallenge(ctx, endpoint, serviceType, httpClient)
	protectedResource, protectedResourceMetadataURL, err := discoverProtectedResourceMetadata(ctx, endpoint, metadataURL, httpClient)
	if err != nil {
		return nil, err
	}
	if protectedResource == nil {
		return &MCPOAuthDiscovery{}, nil
	}
	if len(protectedResource.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("protected resource metadata does not list an authorization server")
	}

	metadata, err := discoverAuthorizationServerMetadata(ctx, protectedResource.AuthorizationServers[0], httpClient)
	if err != nil {
		return nil, err
	}
	if err := validateMCPOAuthServerMetadata(metadata); err != nil {
		return nil, err
	}

	scopes := challengeScopes
	if len(scopes) == 0 {
		scopes = append([]string(nil), protectedResource.ScopesSupported...)
	}
	result := &MCPOAuthDiscovery{
		OAuthAvailable:                     true,
		AuthorizationServer:                metadata.Issuer,
		ProtectedResourceMetadataURL:       protectedResourceMetadataURL,
		Scopes:                             scopes,
		DynamicClientRegistrationSupported: metadata.RegistrationEndpoint != "",
	}
	for _, method := range metadata.CodeChallengeMethodsSupported {
		if method == "S256" {
			result.PKCES256Supported = true
			break
		}
	}
	result.AutomaticAuthorizationSupported = result.DynamicClientRegistrationSupported && result.PKCES256Supported
	return result, nil
}

func discoverMCPOAuthChallenge(ctx context.Context, endpoint string, serviceType model.ServiceType, httpClient *http.Client) (string, []string) {
	method := http.MethodPost
	requestBody := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":%q,"capabilities":{},"clientInfo":{"name":"one-mcp-oauth-discovery","version":"1"}}}`, mcp.LATEST_PROTOCOL_VERSION)
	var body io.Reader = bytes.NewBufferString(requestBody)
	if serviceType == model.ServiceTypeSSE {
		method = http.MethodGet
		body = nil
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return "", nil
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return "", nil
	}

	var metadataURL string
	var scopes []string
	for _, value := range resp.Header.Values("WWW-Authenticate") {
		if matches := resourceMetadataPattern.FindStringSubmatch(value); len(matches) == 2 {
			metadataURL = matches[1]
		}
		if matches := scopeChallengePattern.FindStringSubmatch(value); len(matches) == 2 {
			scopes = strings.Fields(matches[1])
		}
	}
	return metadataURL, scopes
}

func discoverProtectedResourceMetadata(ctx context.Context, endpoint, advertisedURL string, httpClient *http.Client) (*mcpProtectedResourceMetadata, string, error) {
	candidates, err := protectedResourceMetadataURLs(endpoint, advertisedURL)
	if err != nil {
		return nil, "", err
	}
	for _, candidate := range candidates {
		var metadata mcpProtectedResourceMetadata
		found, err := fetchMCPOAuthJSON(ctx, httpClient, candidate, &metadata)
		if err != nil {
			if advertisedURL != "" {
				return nil, "", err
			}
			continue
		}
		if !found {
			continue
		}
		if metadata.Resource != endpoint {
			return nil, "", fmt.Errorf("protected resource metadata resource %q does not match endpoint %q", metadata.Resource, endpoint)
		}
		return &metadata, candidate, nil
	}
	return nil, "", nil
}

func protectedResourceMetadataURLs(endpoint, advertisedURL string) ([]string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	origin := parsed.Scheme + "://" + parsed.Host
	urls := make([]string, 0, 3)
	if advertisedURL != "" {
		advertised, err := url.Parse(advertisedURL)
		if err != nil || !strings.EqualFold(advertised.Scheme, parsed.Scheme) || !strings.EqualFold(advertised.Host, parsed.Host) {
			return nil, fmt.Errorf("advertised protected resource metadata URL must use the MCP endpoint origin")
		}
		return []string{advertised.String()}, nil
	}
	pathCandidate := origin + "/.well-known/oauth-protected-resource"
	if parsed.EscapedPath() != "" && parsed.EscapedPath() != "/" {
		pathCandidate += parsed.EscapedPath()
	}
	if parsed.RawQuery != "" {
		pathCandidate += "?" + parsed.RawQuery
	}
	urls = appendUniqueURL(urls, pathCandidate)
	urls = appendUniqueURL(urls, origin+"/.well-known/oauth-protected-resource")
	return urls, nil
}

func ValidateMCPOAuthProtectedResourceMetadataURL(endpoint, metadataURL string) error {
	if strings.TrimSpace(metadataURL) == "" {
		return nil
	}
	if err := validateSecureOAuthURL(endpoint, "remote MCP URL"); err != nil {
		return err
	}
	if err := validateSecureOAuthURL(metadataURL, "protected resource metadata URL"); err != nil {
		return err
	}
	_, err := protectedResourceMetadataURLs(endpoint, metadataURL)
	return err
}

func discoverAuthorizationServerMetadata(ctx context.Context, issuer string, httpClient *http.Client) (*transport.AuthServerMetadata, error) {
	if err := validateSecureOAuthURL(issuer, "OAuth authorization server issuer"); err != nil {
		return nil, err
	}
	candidates, err := authorizationServerMetadataURLs(issuer)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		var metadata transport.AuthServerMetadata
		found, err := fetchMCPOAuthJSON(ctx, httpClient, candidate, &metadata)
		if err != nil {
			continue
		}
		if !found {
			continue
		}
		if metadata.Issuer != issuer {
			return nil, fmt.Errorf("authorization server metadata issuer %q does not match %q", metadata.Issuer, issuer)
		}
		return &metadata, nil
	}
	return nil, fmt.Errorf("authorization server metadata was not found")
}

func authorizationServerMetadataURLs(issuer string) ([]string, error) {
	parsed, err := url.Parse(issuer)
	if err != nil {
		return nil, err
	}
	origin := parsed.Scheme + "://" + parsed.Host
	path := parsed.EscapedPath()
	if path == "/" {
		path = ""
	}
	urls := []string{
		origin + "/.well-known/oauth-authorization-server" + path,
		origin + "/.well-known/openid-configuration" + path,
	}
	if path != "" {
		urls = append(urls, strings.TrimRight(issuer, "/")+"/.well-known/openid-configuration")
	}
	return urls, nil
}

func fetchMCPOAuthJSON(ctx context.Context, httpClient *http.Client, target string, destination any) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("OAuth discovery request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("OAuth discovery endpoint %q returned status %d", target, resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, 1<<20)
	if err := json.NewDecoder(limited).Decode(destination); err != nil {
		return false, fmt.Errorf("decode OAuth discovery metadata: %w", err)
	}
	return true, nil
}

func appendUniqueURL(urls []string, candidate string) []string {
	for _, existing := range urls {
		if existing == candidate {
			return urls
		}
	}
	return append(urls, candidate)
}

func newMCPOAuthDiscoveryHTTPClient(endpoint string) (*http.Client, error) {
	if err := validatePublicDiscoveryURL(endpoint); err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if isPublicDiscoveryIP(ip) {
					return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				}
			}
			return nil, fmt.Errorf("OAuth discovery target resolves to a non-public IP address")
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many OAuth discovery redirects")
			}
			if err := validatePublicDiscoveryURL(req.URL.String()); err != nil {
				return err
			}
			if len(via) > 0 && !sameURLOrigin(via[0].URL, req.URL) {
				return fmt.Errorf("OAuth discovery redirects must remain on the original origin")
			}
			return nil
		},
	}, nil
}

func sameURLOrigin(left, right *url.URL) bool {
	return left != nil && right != nil && strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func validatePublicDiscoveryURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("OAuth discovery requires a public HTTPS URL without user information")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !isPublicDiscoveryIP(ip) {
		return fmt.Errorf("OAuth discovery target must not use a private or local IP address")
	}
	return nil
}

func isPublicDiscoveryIP(ip net.IP) bool {
	return ip != nil && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
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
	return buildMCPOAuthConfig(service, record, httpClient)
}

func buildMCPOAuthConfig(service *model.MCPService, record *model.MCPOAuth, httpClient *http.Client) (transport.OAuthConfig, error) {
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

func enableMCPOAuthAfterAuthorization(serviceID int64) error {
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		return err
	}
	service.OAuthEnabled = true
	service.OAuthAuthStatus = MCPOAuthStatusAuthorized
	return model.UpdateService(service)
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

func RestoreMCPOAuthStatus(serviceID int64) error {
	record, err := model.GetMCPOAuthByServiceID(serviceID)
	if err != nil && !errors.Is(err, model.ErrMCPOAuthNotFound) {
		return err
	}
	if record != nil && record.EncryptedToken != "" {
		return SetMCPOAuthStatus(serviceID, MCPOAuthStatusAuthorized)
	}
	return SetMCPOAuthStatus(serviceID, MCPOAuthStatusAuthRequired)
}

func IsMCPOAuthAuthorizationRequired(err error) bool {
	return mcpclient.IsOAuthAuthorizationRequiredError(err) || mcpclient.IsAuthorizationRequiredError(err) || errors.Is(err, transport.ErrOAuthAuthorizationRequired) || errors.Is(err, transport.ErrAuthorizationRequired)
}

func newMCPOAuthHandler(service *model.MCPService) (*transport.OAuthHandler, *model.MCPOAuth, error) {
	record, err := getOrCreateMCPOAuthRecord(service.ID)
	if err != nil {
		return nil, nil, err
	}
	config, err := buildMCPOAuthConfig(service, record, &http.Client{Timeout: 30 * time.Second})
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
	if metadata.Issuer != "" {
		issuer, err := url.Parse(metadata.Issuer)
		if err != nil {
			return fmt.Errorf("invalid OAuth authorization server issuer")
		}
		for label, endpoint := range map[string]string{
			"authorization endpoint": metadata.AuthorizationEndpoint,
			"token endpoint":         metadata.TokenEndpoint,
			"registration endpoint":  metadata.RegistrationEndpoint,
		} {
			if endpoint == "" {
				continue
			}
			parsedEndpoint, err := url.Parse(endpoint)
			if err != nil || !sameURLOrigin(issuer, parsedEndpoint) {
				return fmt.Errorf("OAuth %s must use the authorization server issuer origin", label)
			}
		}
	}
	return nil
}
