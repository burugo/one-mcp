package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/stretchr/testify/require"
)

func TestDiscoverMCPOAuthUsesProtectedResourceMetadata(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource/mcp":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"resource":              server.URL + "/mcp",
				"authorization_servers": []string{server.URL},
				"scopes_supported":      []string{"mcp:read"},
			}))
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"issuer":                           server.URL,
				"authorization_endpoint":           server.URL + "/authorize",
				"token_endpoint":                   server.URL + "/token",
				"registration_endpoint":            server.URL + "/register",
				"response_types_supported":         []string{"code"},
				"code_challenge_methods_supported": []string{"S256"},
			}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := DiscoverMCPOAuth(context.Background(), server.URL+"/mcp", model.ServiceTypeStreamableHTTP, server.Client())

	require.NoError(t, err)
	require.True(t, result.OAuthAvailable)
	require.Equal(t, server.URL, result.AuthorizationServer)
	require.Equal(t, server.URL+"/.well-known/oauth-protected-resource/mcp", result.ProtectedResourceMetadataURL)
	require.Equal(t, []string{"mcp:read"}, result.Scopes)
	require.True(t, result.DynamicClientRegistrationSupported)
	require.True(t, result.PKCES256Supported)
	require.True(t, result.AutomaticAuthorizationSupported)
}

func TestDiscoverMCPOAuthPrefersChallengeMetadataAndScopes(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+server.URL+`/oauth/resource", scope="mcp:write"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/oauth/resource":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"resource":              server.URL + "/mcp",
				"authorization_servers": []string{server.URL},
				"scopes_supported":      []string{"mcp:read"},
			}))
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"issuer":                           server.URL,
				"authorization_endpoint":           server.URL + "/authorize",
				"token_endpoint":                   server.URL + "/token",
				"registration_endpoint":            server.URL + "/register",
				"response_types_supported":         []string{"code"},
				"code_challenge_methods_supported": []string{"S256"},
			}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := DiscoverMCPOAuth(context.Background(), server.URL+"/mcp", model.ServiceTypeStreamableHTTP, server.Client())

	require.NoError(t, err)
	require.True(t, result.OAuthAvailable)
	require.Equal(t, server.URL+"/oauth/resource", result.ProtectedResourceMetadataURL)
	require.Equal(t, []string{"mcp:write"}, result.Scopes)
}

func TestDiscoverMCPOAuthReportsNoOAuthWhenMetadataIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	result, err := DiscoverMCPOAuth(context.Background(), server.URL+"/mcp", model.ServiceTypeStreamableHTTP, server.Client())

	require.NoError(t, err)
	require.False(t, result.OAuthAvailable)
}

func TestProtectedResourceMetadataURLsUsePathThenRoot(t *testing.T) {
	urls, err := protectedResourceMetadataURLs("https://mcp.example.test/public/mcp", "")

	require.NoError(t, err)
	require.Equal(t, []string{
		"https://mcp.example.test/.well-known/oauth-protected-resource/public/mcp",
		"https://mcp.example.test/.well-known/oauth-protected-resource",
	}, urls)
}

func TestProtectedResourceMetadataURLsUseOnlyAdvertisedSameOriginURL(t *testing.T) {
	urls, err := protectedResourceMetadataURLs(
		"https://mcp.example.test/mcp",
		"https://mcp.example.test/custom/oauth-metadata",
	)

	require.NoError(t, err)
	require.Equal(t, []string{"https://mcp.example.test/custom/oauth-metadata"}, urls)
}

func TestProtectedResourceMetadataURLsRejectCrossOriginAdvertisement(t *testing.T) {
	_, err := protectedResourceMetadataURLs(
		"https://mcp.example.test/mcp",
		"https://attacker.example.test/oauth-metadata",
	)

	require.Error(t, err)
}

func TestValidateMCPOAuthProtectedResourceMetadataURLRejectsCrossOriginURL(t *testing.T) {
	err := ValidateMCPOAuthProtectedResourceMetadataURL(
		"https://mcp.example.test/mcp",
		"https://attacker.example.test/oauth-metadata",
	)

	require.Error(t, err)
}

func TestValidatePublicDiscoveryURLRejectsLocalTargets(t *testing.T) {
	require.Error(t, validatePublicDiscoveryURL("http://mcp.example.test/mcp"))
	require.Error(t, validatePublicDiscoveryURL("https://127.0.0.1/mcp"))
	require.Error(t, validatePublicDiscoveryURL("https://169.254.169.254/latest/meta-data"))
	require.NoError(t, validatePublicDiscoveryURL("https://mcp.example.test/mcp"))
}

func TestValidateMCPOAuthServerMetadataRejectsCrossOriginEndpoints(t *testing.T) {
	err := validateMCPOAuthServerMetadata(&transport.AuthServerMetadata{
		Issuer:                "https://accounts.example.test",
		AuthorizationEndpoint: "https://attacker.example.test/authorize",
		TokenEndpoint:         "https://accounts.example.test/token",
	})

	require.Error(t, err)
}
