package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	common.SysLog(fmt.Sprintf("[ProxyHandler] Attempting user-specific handler for key: %s", userHandlerKey))

	cachedHandler, found := proxy.GetCachedHandler(userHandlerKey)
	if found {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Found cached user-specific handler for key: %s", userHandlerKey))
		return cachedHandler, nil
	}

	common.SysLog(fmt.Sprintf("[ProxyHandler] No cached user-specific handler for key: %s. Attempting creation.", userHandlerKey))
	var baseStdioConf model.StdioConfig

	// Populate baseStdioConf.Command and baseStdioConf.Args from mcpDBService
	baseStdioConf.Command = mcpDBService.Command
	if mcpDBService.Command == "" {
		// This is a critical issue if we expect a command for Stdio type for user-specific handler creation
		common.SysError(fmt.Sprintf("[ProxyHandler] MCPService.Command is empty for service %s (ID: %d) when creating user-specific handler.", mcpDBService.Name, mcpDBService.ID))
		// Depending on strictness, might return error here. For now, proceeding will likely fail in NewStdioSSEHandlerUncached.
	}

	if mcpDBService.ArgsJSON != "" {
		if err := json.Unmarshal([]byte(mcpDBService.ArgsJSON), &baseStdioConf.Args); err != nil {
			common.SysError(fmt.Sprintf("[ProxyHandler] Error unmarshalling ArgsJSON for service %s (user-specific base): %v. Args will be empty.", mcpDBService.Name, err))
			baseStdioConf.Args = []string{} // Ensure Args is empty on error
		}
	} else {
		baseStdioConf.Args = []string{} // Ensure Args is empty if ArgsJSON is empty
	}

	// Initialize Env, will be populated by DefaultEnvsJSON and user-specific overrides
	baseStdioConf.Env = []string{}

	currentEnvMap := make(map[string]string)
	// Populate currentEnvMap from DefaultEnvsJSON first
	if mcpDBService.DefaultEnvsJSON != "" && mcpDBService.DefaultEnvsJSON != "{}" {
		if err := json.Unmarshal([]byte(mcpDBService.DefaultEnvsJSON), &currentEnvMap); err != nil {
			common.SysError(fmt.Sprintf("[ProxyHandler] Error unmarshalling DefaultEnvsJSON for %s (user-specific base): %v", mcpDBService.Name, err))
			// Continue with an empty map if unmarshal fails
			currentEnvMap = make(map[string]string)
		}
	}

	// Fetch and merge user-specific ENVs
	userEnvs, userEnvErr := model.GetUserSpecificEnvs(userID, mcpDBService.ID)
	if userEnvErr != nil {
		common.SysError(fmt.Sprintf("[ProxyHandler] Error fetching user-specific ENVs for user %d, service %s: %v", userID, mcpDBService.Name, userEnvErr))
		// Potentially return error here if user ENVs are critical and failed to load
	}

	if len(userEnvs) > 0 {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Merging %d user-specific ENVs for user %d, service %s", len(userEnvs), userID, mcpDBService.Name))
		for k, v := range userEnvs {
			currentEnvMap[k] = v // User-specific ENVs override DefaultEnvsJSON
		}
	} else {
		common.SysLog(fmt.Sprintf("[ProxyHandler] No user-specific ENVs found for user %d, service %s. Using defaults from DefaultEnvsJSON if any.", userID, mcpDBService.Name))
	}

	// Convert the final map to the KEY=VALUE string slice format for StdioConfig.Env
	// This part (populating baseStdioConf.Env) is for logging or if baseStdioConf were passed directly.
	// Since we are modifying a copy of mcpDBService.DefaultEnvsJSON, createMcpGoServer will use that.
	// Keeping it for the log message for now.
	tempEnvSlice := []string{}
	for k, v := range currentEnvMap {
		tempEnvSlice = append(tempEnvSlice, fmt.Sprintf("%s=%s", k, v))
	}

	// Create a copy of mcpDBService to avoid modifying the original from the database/cache
	// and to pass user-specific configurations.
	userSpecificMcpService := *mcpDBService // Shallow copy

	// Marshal the merged env map back to JSON and set it on the service copy
	mergedEnvsJSONBytes, marshalErr := json.Marshal(currentEnvMap)
	if marshalErr != nil {
		common.SysError(fmt.Sprintf("[ProxyHandler] Error marshalling merged ENVs for user %d, service %s: %v. Proceeding with original DefaultEnvsJSON.", userID, mcpDBService.Name, marshalErr))
		// If marshalling fails, userSpecificMcpService will retain the original DefaultEnvsJSON.
		// The log below will show the intended envs, but the actual process might get defaults.
	} else {
		userSpecificMcpService.DefaultEnvsJSON = string(mergedEnvsJSONBytes)
		common.SysLog(fmt.Sprintf("[ProxyHandler] Updated DefaultEnvsJSON for user-specific call for %s (user %d): %s", mcpDBService.Name, userID, userSpecificMcpService.DefaultEnvsJSON))
	}

	common.SysLog(fmt.Sprintf("[ProxyHandler] Effective StdioConfig (intended for user %d, service %s): Command='%s', Args=%v, Env=%v. Actual envs based on (potentially updated) DefaultEnvsJSON.", userID, mcpDBService.Name, baseStdioConf.Command, baseStdioConf.Args, tempEnvSlice))

	// Pass the potentially modified service copy
	handler, err := proxy.NewProxyToSSEHandlerUncached(c.Request.Context(), &userSpecificMcpService)
	if err == nil {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Successfully created user-specific handler for key: %s. Caching.", userHandlerKey))
		proxy.CacheHandler(userHandlerKey, handler)
		return handler, nil
	}
	return nil, fmt.Errorf("failed to create user-specific handler for %s (user %d): %w", mcpDBService.Name, userID, err)
}

// tryGetOrCreateGlobalHandler attempts to find or create a global SSE handler for the service.
func tryGetOrCreateGlobalHandler(c *gin.Context, mcpDBService *model.MCPService) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("[ProxyHandler] Attempting global handler for service %s", mcpDBService.Name))
	globalHandlerKey := fmt.Sprintf("global-service-%d", mcpDBService.ID)

	cachedGlobalHandler, found := proxy.GetCachedHandler(globalHandlerKey)
	if found {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Found cached global handler for key: %s", globalHandlerKey))
		return cachedGlobalHandler, nil
	}

	common.SysLog(fmt.Sprintf("[ProxyHandler] No cached global handler for key: %s. Calling ServiceFactory.", globalHandlerKey))
	createdService, factoryErr := proxy.ServiceFactory(mcpDBService) // ServiceFactory caches Stdio-based global handlers
	if factoryErr != nil {
		return nil, fmt.Errorf("failed to get/create global handler for service '%s' from factory: %w", mcpDBService.Name, factoryErr)
	}

	if httpHandler, ok := createdService.(http.Handler); ok {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Global handler obtained from ServiceFactory for %s.", mcpDBService.Name))
		return httpHandler, nil
	}
	return nil, fmt.Errorf("global service '%s' (type %s) from factory is not a valid http.Handler", mcpDBService.Name, mcpDBService.Type)
}

// ProxyHandler handles GET and POST /proxy/:serviceName/*action
func ProxyHandler(c *gin.Context) {
	serviceName := c.Param("serviceName")
	action := c.Param("action") // sse or mcp
	common.SysLog(fmt.Sprintf("[ProxyHandler] Service: %s, Action: %s, Processed Path: %s, Query: %s",
		serviceName, action, c.Request.URL.Path, c.Request.URL.RawQuery))

	mcpDBService, err := model.GetServiceByName(serviceName)
	if err != nil || mcpDBService == nil {
		common.SysError(fmt.Sprintf("[ProxyHandler] Service not found: %s, error: %v", serviceName, err))
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found: " + serviceName})
		return
	}
	if !mcpDBService.Enabled {
		common.SysLog(fmt.Sprintf("WARN: [ProxyHandler] Service not enabled: %s", serviceName))
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
			common.SysLog(fmt.Sprintf("[ProxyHandler] Authenticated user %d identified for service %s", userID, serviceName))
		} else {
			common.SysLog(fmt.Sprintf("WARN: [ProxyHandler] userID found in context but failed to parse to int64: %v, type: %T, err: %v", idVal, idVal, parseErr))
		}
	} else {
		common.SysLog(fmt.Sprintf("[ProxyHandler] No authenticated user identified for service %s. Proceeding with global context.", serviceName))
	}

	if userID > 0 && mcpDBService.AllowUserOverride && mcpDBService.Type == model.ServiceTypeStdio {
		targetHandler, handlerErr = tryGetOrCreateUserSpecificHandler(c, mcpDBService, userID)
		if handlerErr != nil {
			common.SysError(fmt.Sprintf("[ProxyHandler] Error obtaining user-specific handler for %s (user %d): %v. Falling back to global.", serviceName, userID, handlerErr))
			// Clear handlerErr so global fallback logic doesn't use this error message if global succeeds
			handlerErr = nil
		}
	}

	if targetHandler == nil { // Fallback to Global Handler
		if userID > 0 && mcpDBService.AllowUserOverride && mcpDBService.Type == model.ServiceTypeStdio {
			common.SysLog(fmt.Sprintf("WARN: [ProxyHandler] User-specific handler attempt for service %s, user %d resulted in nil or error; falling back to global.", serviceName, userID))
		}
		targetHandler, handlerErr = tryGetOrCreateGlobalHandler(c, mcpDBService)
	}

	if targetHandler != nil {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Serving request for service %s (processed path %s) using obtained handler.", serviceName, c.Request.URL.Path))
		targetHandler.ServeHTTP(c.Writer, c.Request)
	} else {
		finalErrMsg := "critical: unable to obtain any valid handler for service " + serviceName
		if handlerErr != nil {
			finalErrMsg = fmt.Sprintf("Service handler unavailable for %s: %s", serviceName, handlerErr.Error())
		}
		common.SysError(fmt.Sprintf("[ProxyHandler] Error: %s", finalErrMsg))
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": finalErrMsg})
	}
}

// HTTPProxyHandler handles ANY /proxy/:serviceName/mcp/*action
func HTTPProxyHandler(c *gin.Context) {
	serviceName := c.Param("serviceName")

	originalPathForRequest := c.Request.URL.Path // Preserve for logging

	common.SysLog(fmt.Sprintf("[HTTPProxyHandler] Service: %s, Original ActionParam: %s, Processed Path: %s, Query: %s",
		serviceName, c.Param("action"), c.Request.URL.Path, c.Request.URL.RawQuery))

	mcpDBService, err := model.GetServiceByName(serviceName)
	if err != nil || mcpDBService == nil {
		common.SysError(fmt.Sprintf("[HTTPProxyHandler] Service not found: %s, error: %v", serviceName, err))
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found: " + serviceName})
		return
	}
	if !mcpDBService.Enabled {
		common.SysLog(fmt.Sprintf("WARN: [HTTPProxyHandler] Service not enabled: %s", serviceName))
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Service not enabled: " + serviceName})
		return
	}

	// HTTP proxy only handles StreamableHTTP type services
	if mcpDBService.Type != model.ServiceTypeStreamableHTTP {
		common.SysError(fmt.Sprintf("[HTTPProxyHandler] Service %s is not of type StreamableHTTP (actual: %s)", serviceName, mcpDBService.Type))
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Service is not of type Streamable HTTP"})
		return
	}

	var targetHandler http.Handler
	var handlerErr error

	// For HTTP services, we use the global handler approach since user-specific configs
	// are typically handled at the HTTP level rather than process level
	targetHandler, handlerErr = tryGetOrCreateGlobalHandler(c, mcpDBService)

	if targetHandler != nil {
		common.SysLog(fmt.Sprintf("[HTTPProxyHandler] Serving request for service %s (original path %s, processed path %s) using obtained handler.", serviceName, originalPathForRequest, c.Request.URL.Path))
		targetHandler.ServeHTTP(c.Writer, c.Request)
	} else {
		finalErrMsg := "critical: unable to obtain any valid handler for service " + serviceName
		if handlerErr != nil {
			finalErrMsg = fmt.Sprintf("Service handler unavailable for %s: %s", serviceName, handlerErr.Error())
		}
		common.SysError(fmt.Sprintf("[HTTPProxyHandler] Error: %s", finalErrMsg))
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": finalErrMsg})
	}
}
