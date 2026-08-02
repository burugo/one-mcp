package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	appservice "one-mcp/backend/service"

	"github.com/gin-gonic/gin"
)

func DiscoverMCPOAuth(c *gin.Context) {
	var request struct {
		URL  string `json:"url" binding:"required"`
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.RespError(c, http.StatusBadRequest, "invalid MCP OAuth discovery request", err)
		return
	}

	var serviceType model.ServiceType
	switch request.Type {
	case "sse":
		serviceType = model.ServiceTypeSSE
	case "streamableHttp":
		serviceType = model.ServiceTypeStreamableHTTP
	default:
		common.RespErrorStr(c, http.StatusBadRequest, "OAuth discovery only supports remote MCP services")
		return
	}

	discoveryCtx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()
	result, err := appservice.DiscoverMCPOAuth(discoveryCtx, strings.TrimSpace(request.URL), serviceType, nil)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, "discover MCP OAuth failed", err)
		return
	}
	common.RespSuccess(c, result)
}

func GetMCPOAuthStatus(c *gin.Context) {
	serviceID, ok := mcpOAuthServiceID(c)
	if !ok {
		return
	}
	status, err := appservice.GetMCPOAuthManager().Status(serviceID)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "get MCP OAuth status failed", err)
		return
	}
	common.RespSuccess(c, status)
}

func ConfigureMCPOAuth(c *gin.Context) {
	serviceID, ok := mcpOAuthServiceID(c)
	if !ok {
		return
	}
	var request struct {
		ClientID                     string `json:"client_id"`
		ClientSecret                 string `json:"client_secret"`
		Scopes                       string `json:"scopes"`
		AuthServerMetadataURL        string `json:"auth_server_metadata_url"`
		ProtectedResourceMetadataURL string `json:"protected_resource_metadata_url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.RespError(c, http.StatusBadRequest, "invalid MCP OAuth configuration", err)
		return
	}
	status, err := appservice.GetMCPOAuthManager().Configure(c.Request.Context(), serviceID, appservice.MCPOAuthConfigInput{
		ClientID:                     request.ClientID,
		ClientSecret:                 request.ClientSecret,
		Scopes:                       strings.Fields(request.Scopes),
		AuthServerMetadataURL:        request.AuthServerMetadataURL,
		ProtectedResourceMetadataURL: request.ProtectedResourceMetadataURL,
	})
	if err != nil {
		common.RespError(c, http.StatusBadRequest, "configure MCP OAuth failed", err)
		return
	}
	common.SysLog(fmt.Sprintf("Admin %s configured OAuth for MCP service %d", c.GetString("username"), serviceID))
	go reloadMCPService(serviceID)
	common.RespSuccess(c, status)
}

func DisableMCPOAuth(c *gin.Context) {
	serviceID, ok := mcpOAuthServiceID(c)
	if !ok {
		return
	}
	if err := appservice.GetMCPOAuthManager().Disable(serviceID); err != nil {
		common.RespError(c, http.StatusInternalServerError, "disable MCP OAuth failed", err)
		return
	}
	common.SysLog(fmt.Sprintf("Admin %s disabled OAuth for MCP service %d", c.GetString("username"), serviceID))
	go reloadMCPService(serviceID)
	common.RespSuccessStr(c, "MCP OAuth disabled")
}

func AuthorizeMCPOAuth(c *gin.Context) {
	serviceID, ok := mcpOAuthServiceID(c)
	if !ok {
		return
	}
	authorizeCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	returnURL := mcpOAuthFrontendReturnURL(c.Request)
	authorizationURL, err := appservice.GetMCPOAuthManager().BeginAuthorization(authorizeCtx, serviceID, returnURL)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, "start MCP OAuth authorization failed", err)
		return
	}
	common.SysLog(fmt.Sprintf("Admin %s started OAuth authorization for MCP service %d", c.GetString("username"), serviceID))
	common.RespSuccess(c, gin.H{"authorization_url": authorizationURL})
}

func MCPOAuthCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	if oauthError := c.Query("error"); oauthError != "" {
		serviceID, returnURL := appservice.GetMCPOAuthManager().CancelAuthorization(state)
		if serviceID > 0 {
			_ = appservice.RestoreMCPOAuthStatus(serviceID)
		}
		writeMCPOAuthRedirectPage(c, mcpOAuthResultURL(resolveMCPOAuthReturnURL(returnURL, state), "error", oauthError))
		return
	}
	if state == "" || code == "" {
		common.RespErrorStr(c, http.StatusBadRequest, "OAuth callback requires state and code")
		return
	}
	callbackCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	serviceID, returnURL, err := appservice.GetMCPOAuthManager().CompleteAuthorization(callbackCtx, state, code)
	if err != nil {
		message := "complete MCP OAuth authorization failed"
		if errors.Is(err, appservice.ErrInvalidMCPOAuthState) || errors.Is(err, appservice.ErrExpiredMCPOAuthFlow) {
			message = err.Error()
		}
		writeMCPOAuthRedirectPage(c, mcpOAuthResultURL(resolveMCPOAuthReturnURL(returnURL, state), "error", message))
		return
	}

	go reloadMCPService(serviceID)
	common.SysLog(fmt.Sprintf("OAuth authorization completed for MCP service %d", serviceID))
	writeMCPOAuthRedirectPage(c, mcpOAuthResultURL(resolveMCPOAuthReturnURL(returnURL, state), "success", ""))
}

func resolveMCPOAuthReturnURL(flowReturnURL, state string) string {
	if strings.TrimSpace(flowReturnURL) != "" {
		return flowReturnURL
	}
	if returnURL, ok := appservice.MCPOAuthReturnURLFromState(state); ok {
		return returnURL
	}
	return ""
}

func writeMCPOAuthRedirectPage(c *gin.Context, destination string) {
	destinationJSON, _ := json.Marshal(destination)
	escapedDestination := html.EscapeString(destination)
	body := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta http-equiv="refresh" content="0;url=%s">
  <title>OAuth Redirect</title>
</head>
<body>
  <p>Returning to One MCP…</p>
  <p><a href="%s">Continue</a></p>
  <script>window.location.replace(%s);</script>
</body>
</html>`, escapedDestination, escapedDestination, destinationJSON)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(body))
}

func mcpOAuthFrontendReturnURL(request *http.Request) string {
	common.OptionMapRWMutex.RLock()
	serverAddress := strings.TrimSpace(common.GetServerAddress())
	common.OptionMapRWMutex.RUnlock()
	if serverAddress == "" {
		serverAddress = fmt.Sprintf("http://localhost:%d", *common.Port)
	}

	serverURL, err := url.Parse(serverAddress)
	if err != nil || serverURL.Scheme == "" || serverURL.Host == "" {
		serverURL, _ = url.Parse(fmt.Sprintf("http://localhost:%d", *common.Port))
	}
	frontendOrigin := serverURL.Scheme + "://" + serverURL.Host

	if origin := strings.TrimSpace(request.Header.Get("Origin")); origin != "" {
		originURL, err := url.Parse(origin)
		if err == nil && isLocalViteOrigin(serverURL, originURL) {
			frontendOrigin = originURL.Scheme + "://" + originURL.Host
		}
	}
	return strings.TrimRight(frontendOrigin, "/") + "/services"
}

func isLocalViteOrigin(serverURL, originURL *url.URL) bool {
	if serverURL == nil || originURL == nil || originURL.Scheme != "http" || originURL.Port() != "5173" {
		return false
	}
	serverHost := strings.ToLower(serverURL.Hostname())
	originHost := strings.ToLower(originURL.Hostname())
	return isLoopbackHostname(serverHost) && isLoopbackHostname(originHost)
}

func isLoopbackHostname(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func mcpOAuthResultURL(returnURL, result, message string) string {
	if strings.TrimSpace(returnURL) == "" {
		returnURL = mcpOAuthFrontendReturnURL(&http.Request{Header: make(http.Header)})
	}
	parsed, err := url.Parse(returnURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		parsed, _ = url.Parse(mcpOAuthFrontendReturnURL(&http.Request{Header: make(http.Header)}))
	}
	query := parsed.Query()
	query.Set("mcp_oauth", result)
	if message != "" {
		query.Set("message", message)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func mcpOAuthServiceID(c *gin.Context) (int64, bool) {
	serviceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || serviceID <= 0 {
		common.RespError(c, http.StatusBadRequest, "invalid service ID", err)
		return 0, false
	}
	return serviceID, true
}

func reloadMCPService(serviceID int64) {
	mcpService, err := model.GetServiceByID(serviceID)
	if err != nil {
		common.SysError(fmt.Sprintf("Failed to reload OAuth-enabled service %d after callback: %v", serviceID, err))
		return
	}
	manager := proxy.GetServiceManager()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = manager.UnregisterService(ctx, mcpService.ID)
	if !mcpService.Enabled || mcpService.Deleted {
		return
	}
	if err := manager.RegisterService(ctx, mcpService); err != nil {
		common.SysError(fmt.Sprintf("Failed to reload OAuth MCP service %s: %v", mcpService.Name, err))
	}
}
