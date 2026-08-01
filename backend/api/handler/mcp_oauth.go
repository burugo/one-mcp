package handler

import (
	"context"
	"errors"
	"fmt"
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
	authorizationURL, err := appservice.GetMCPOAuthManager().BeginAuthorization(authorizeCtx, serviceID)
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
		if serviceID := appservice.GetMCPOAuthManager().CancelAuthorization(state); serviceID > 0 {
			_ = appservice.SetMCPOAuthStatus(serviceID, appservice.MCPOAuthStatusAuthRequired)
		}
		c.Redirect(http.StatusFound, "/services?mcp_oauth=error&message="+url.QueryEscape(oauthError))
		return
	}
	if state == "" || code == "" {
		common.RespErrorStr(c, http.StatusBadRequest, "OAuth callback requires state and code")
		return
	}
	callbackCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	serviceID, err := appservice.GetMCPOAuthManager().CompleteAuthorization(callbackCtx, state, code)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, appservice.ErrInvalidMCPOAuthState) && !errors.Is(err, appservice.ErrExpiredMCPOAuthFlow) {
			status = http.StatusBadGateway
		}
		common.RespError(c, status, "complete MCP OAuth authorization failed", err)
		return
	}

	go reloadMCPService(serviceID)
	common.SysLog(fmt.Sprintf("OAuth authorization completed for MCP service %d", serviceID))
	c.Redirect(http.StatusFound, "/services?mcp_oauth=success")
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
