package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
)

// parseInt64 is a helper function to safely parse int64 from various numeric types or string.
// It's used for retrieving userID from gin.Context.
func parseInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case string:
		num, err := json.Number(v).Int64()
		if err == nil {
			return num, nil
		}
		// Fallback for plain integer strings if json.Number fails (e.g. not a valid JSON number but simple int string)
		var i int64
		_, scanErr := fmt.Sscan(v, &i)
		return i, scanErr
	default:
		return 0, fmt.Errorf("cannot parse type %T to int64", value)
	}
}

// tryGetOrCreateUserSpecificHandler attempts to find or create an SSE handler tailored for a specific user.
func tryGetOrCreateUserSpecificHandler(c *gin.Context, mcpDBService *model.MCPService, userID int64) (http.Handler, error) {
	userHandlerKey := fmt.Sprintf("user-%d-service-%d", userID, mcpDBService.ID)
	common.SysLog(fmt.Sprintf("[SSEProxyHandler] Attempting user-specific handler for key: %s", userHandlerKey))

	cachedHandler, found := proxy.GetCachedHandler(userHandlerKey)
	if found {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] Found cached user-specific handler for key: %s", userHandlerKey))
		return cachedHandler, nil
	}

	common.SysLog(fmt.Sprintf("[SSEProxyHandler] No cached user-specific handler for key: %s. Attempting creation.", userHandlerKey))
	var baseStdioConf model.StdioConfig
	if mcpDBService.DefaultAdminConfigValues != "" {
		if err := json.Unmarshal([]byte(mcpDBService.DefaultAdminConfigValues), &baseStdioConf); err != nil {
			// Log error but proceed, as baseStdioConf might still be partially usable or defaults will apply.
			common.SysError(fmt.Sprintf("[SSEProxyHandler] Error unmarshalling base StdioConfig for user-specific handler %s: %v", mcpDBService.Name, err))
		}
	} else {
		common.SysLog(fmt.Sprintf("WARN: [SSEProxyHandler] DefaultAdminConfigValues is empty for service %s. User-specific handler might lack base command/args.", mcpDBService.Name))
	}

	currentEnvMap := make(map[string]string)
	if mcpDBService.DefaultEnvsJSON != "" && mcpDBService.DefaultEnvsJSON != "{}" {
		if err := json.Unmarshal([]byte(mcpDBService.DefaultEnvsJSON), &currentEnvMap); err != nil {
			common.SysError(fmt.Sprintf("[SSEProxyHandler] Error unmarshalling DefaultEnvsJSON for %s (user-specific base): %v", mcpDBService.Name, err))
		}
	}

	userEnvs, userEnvErr := model.GetUserSpecificEnvs(userID, mcpDBService.ID)
	if userEnvErr != nil {
		common.SysError(fmt.Sprintf("[SSEProxyHandler] Error fetching user-specific ENVs for user %d, service %s: %v", userID, mcpDBService.Name, userEnvErr))
		// Potentially return error here if user ENVs are critical and failed to load
	}

	if len(userEnvs) > 0 {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] Merging %d user-specific ENVs for user %d, service %s", len(userEnvs), userID, mcpDBService.Name))
		for k, v := range userEnvs {
			currentEnvMap[k] = v
		}
	} else {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] No user-specific ENVs found for user %d, service %s. Using defaults.", userID, mcpDBService.Name))
	}

	var finalEnv []string
	for k, v := range currentEnvMap {
		finalEnv = append(finalEnv, fmt.Sprintf("%s=%s", k, v))
	}
	baseStdioConf.Env = finalEnv
	common.SysLog(fmt.Sprintf("[SSEProxyHandler] Effective StdioConfig for user %d, service %s: Command='%s', Args=%v, Env=%v", userID, mcpDBService.Name, baseStdioConf.Command, baseStdioConf.Args, baseStdioConf.Env))

	newHandler, creationErr := proxy.NewStdioSSEHandlerUncached(c.Request.Context(), mcpDBService, baseStdioConf)
	if creationErr == nil {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] Successfully created user-specific handler for key: %s. Caching.", userHandlerKey))
		proxy.CacheHandler(userHandlerKey, newHandler)
		return newHandler, nil
	}
	return nil, fmt.Errorf("failed to create user-specific handler for %s (user %d): %w", mcpDBService.Name, userID, creationErr)
}

// tryGetOrCreateGlobalHandler attempts to find or create a global SSE handler for the service.
func tryGetOrCreateGlobalHandler(c *gin.Context, mcpDBService *model.MCPService) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("[SSEProxyHandler] Attempting global handler for service %s", mcpDBService.Name))
	globalHandlerKey := fmt.Sprintf("global-service-%d", mcpDBService.ID)

	cachedGlobalHandler, found := proxy.GetCachedHandler(globalHandlerKey)
	if found {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] Found cached global handler for key: %s", globalHandlerKey))
		return cachedGlobalHandler, nil
	}

	common.SysLog(fmt.Sprintf("[SSEProxyHandler] No cached global handler for key: %s. Calling ServiceFactory.", globalHandlerKey))
	createdService, factoryErr := proxy.ServiceFactory(mcpDBService) // ServiceFactory caches Stdio-based global handlers
	if factoryErr != nil {
		return nil, fmt.Errorf("failed to get/create global handler for service '%s' from factory: %w", mcpDBService.Name, factoryErr)
	}

	if httpHandler, ok := createdService.(http.Handler); ok {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] Global handler obtained from ServiceFactory for %s.", mcpDBService.Name))
		return httpHandler, nil
	}
	return nil, fmt.Errorf("global service '%s' (type %s) from factory is not a valid http.Handler", mcpDBService.Name, mcpDBService.Type)
}

// SSEProxyHandler handles GET and POST /api/sse/:serviceName/*action
func SSEProxyHandler(c *gin.Context) {
	serviceName := c.Param("serviceName")
	actionPath := c.Param("action") // e.g., / or /some/path?query=val

	// Process path to be suitable for the underlying mcp-go SSEServer
	// It expects paths like "/" or "/message?sessionId=xyz"
	// The actionPath from Gin for /*action includes the leading slash if present.
	// If action is empty (e.g. /api/sse/serviceName), actionPath is "/"
	// If action is /foo (e.g. /api/sse/serviceName/foo), actionPath is "/foo"
	// If action is foo (e.g. /api/sse/serviceNamefoo), actionPath is "/foo" - this case is unlikely with /*action
	pathPart := actionPath
	if idx := strings.Index(actionPath, "?"); idx != -1 {
		pathPart = actionPath[:idx] // Extract path without query for logging/internal routing if needed
	}
	if pathPart == "" { // Should be at least "/" if param is matched
		pathPart = "/"
	}
	// Ensure leading slash for the request URL path passed to the handler
	if !strings.HasPrefix(pathPart, "/") {
		pathPart = "/" + pathPart
	}

	originalPathForRequest := c.Request.URL.Path // Preserve for logging
	c.Request.URL.Path = pathPart                // Set path for the proxied request

	common.SysLog(fmt.Sprintf("[SSEProxyHandler] Service: %s, Original ActionParam: %s, Processed Path: %s, Query: %s",
		serviceName, c.Param("action"), c.Request.URL.Path, c.Request.URL.RawQuery))

	mcpDBService, err := model.GetServiceByName(serviceName)
	if err != nil || mcpDBService == nil {
		common.SysError(fmt.Sprintf("[SSEProxyHandler] Service not found: %s, error: %v", serviceName, err))
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found: " + serviceName})
		return
	}
	if !mcpDBService.Enabled {
		common.SysLog(fmt.Sprintf("WARN: [SSEProxyHandler] Service not enabled: %s", serviceName))
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Service not enabled: " + serviceName})
		return
	}

	var targetHandler http.Handler
	var handlerErr error
	var userID int64

	if idVal, exists := c.Get("userID"); exists {
		parsedID, parseErr := parseInt64(idVal)
		if parseErr == nil {
			userID = parsedID
			common.SysLog(fmt.Sprintf("[SSEProxyHandler] Authenticated user %d identified for service %s", userID, serviceName))
		} else {
			common.SysLog(fmt.Sprintf("WARN: [SSEProxyHandler] userID found in context but failed to parse to int64: %v, type: %T, err: %v", idVal, idVal, parseErr))
		}
	} else {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] No authenticated user identified for service %s. Proceeding with global context.", serviceName))
	}

	if userID > 0 && mcpDBService.AllowUserOverride && mcpDBService.Type == model.ServiceTypeStdio {
		targetHandler, handlerErr = tryGetOrCreateUserSpecificHandler(c, mcpDBService, userID)
		if handlerErr != nil {
			common.SysError(fmt.Sprintf("[SSEProxyHandler] Error obtaining user-specific handler for %s (user %d): %v. Falling back to global.", serviceName, userID, handlerErr))
			// Clear handlerErr so global fallback logic doesn't use this error message if global succeeds
			handlerErr = nil
		}
	}

	if targetHandler == nil { // Fallback to Global Handler
		if userID > 0 && mcpDBService.AllowUserOverride && mcpDBService.Type == model.ServiceTypeStdio {
			common.SysLog(fmt.Sprintf("WARN: [SSEProxyHandler] User-specific handler attempt for service %s, user %d resulted in nil or error; falling back to global.", serviceName, userID))
		}
		targetHandler, handlerErr = tryGetOrCreateGlobalHandler(c, mcpDBService)
	}

	if targetHandler != nil {
		common.SysLog(fmt.Sprintf("[SSEProxyHandler] Serving request for service %s (original path %s, processed path %s) using obtained handler.", serviceName, originalPathForRequest, c.Request.URL.Path))
		targetHandler.ServeHTTP(c.Writer, c.Request)
	} else {
		finalErrMsg := "critical: unable to obtain any valid handler for service " + serviceName
		if handlerErr != nil {
			finalErrMsg = fmt.Sprintf("Service handler unavailable for %s: %s", serviceName, handlerErr.Error())
		}
		common.SysError(fmt.Sprintf("[SSEProxyHandler] Error: %s", finalErrMsg))
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": finalErrMsg})
	}
}

// HTTPProxyHandler handles ANY /api/http/:serviceName/*action
func HTTPProxyHandler(c *gin.Context) {
	serviceName := c.Param("serviceName")

	service, err := model.GetServiceByName(serviceName)
	if err != nil || service == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found"})
		return
	}

	if service.Type != model.ServiceTypeStreamableHTTP {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Service is not of type Streamable HTTP"})
		return
	}

	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"message": "HTTP proxy not implemented yet",
	})
}
