package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/burugo/thing"
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

// tryGetOrCreateUserSpecificHandler attempts to find or create a handler tailored for a specific user.
// proxyType should be "sseproxy" or "httpproxy"
func tryGetOrCreateUserSpecificHandler(c *gin.Context, mcpDBService *model.MCPService, userID int64, proxyType string) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("[ProxyHandler] Attempting user-specific handler for user %d, service %s with proxy type %s", userID, mcpDBService.Name, proxyType))

	// Prepare user-specific environment variables
	currentEnvMap := make(map[string]string)
	// Populate currentEnvMap from DefaultEnvsJSON first
	if mcpDBService.DefaultEnvsJSON != "" && mcpDBService.DefaultEnvsJSON != "{}" {
		if err := json.Unmarshal([]byte(mcpDBService.DefaultEnvsJSON), &currentEnvMap); err != nil {
			common.SysError(fmt.Sprintf("[ProxyHandler] Error unmarshalling DefaultEnvsJSON for %s (user-specific): %v", mcpDBService.Name, err))
			currentEnvMap = make(map[string]string)
		}
	}

	// Fetch and merge user-specific ENVs
	userEnvs, userEnvErr := model.GetUserSpecificEnvs(userID, mcpDBService.ID)
	if userEnvErr != nil {
		common.SysError(fmt.Sprintf("[ProxyHandler] Error fetching user-specific ENVs for user %d, service %s: %v", userID, mcpDBService.Name, userEnvErr))
	}

	if len(userEnvs) > 0 {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Merging %d user-specific ENVs for user %d, service %s", len(userEnvs), userID, mcpDBService.Name))
		for k, v := range userEnvs {
			currentEnvMap[k] = v // User-specific ENVs override DefaultEnvsJSON
		}
	} else {
		common.SysLog(fmt.Sprintf("[ProxyHandler] No user-specific ENVs found for user %d, service %s. Using defaults from DefaultEnvsJSON if any.", userID, mcpDBService.Name))
	}

	// Marshal the merged env map back to JSON
	mergedEnvsJSONBytes, marshalErr := json.Marshal(currentEnvMap)
	if marshalErr != nil {
		common.SysError(fmt.Sprintf("[ProxyHandler] Error marshalling merged ENVs for user %d, service %s: %v. Proceeding with original DefaultEnvsJSON.", userID, mcpDBService.Name, marshalErr))
		mergedEnvsJSONBytes = []byte(mcpDBService.DefaultEnvsJSON)
	}
	mergedEnvsJSON := string(mergedEnvsJSONBytes)

	// Create user-specific shared MCP instance
	ctx := c.Request.Context()
	userSharedCacheKey := fmt.Sprintf("user-%d-service-%d-shared", userID, mcpDBService.ID)
	instanceNameDetail := fmt.Sprintf("user-%d-shared-svc-%d", userID, mcpDBService.ID)

	sharedInst, err := proxy.GetOrCreateSharedMcpInstanceWithKey(ctx, mcpDBService, userSharedCacheKey, instanceNameDetail, mergedEnvsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create user-specific shared MCP instance for %s (user %d): %w", mcpDBService.Name, userID, err)
	}

	var targetHandler http.Handler
	switch proxyType {
	case "sseproxy":
		targetHandler, err = proxy.GetOrCreateProxyToSSEHandler(ctx, mcpDBService, sharedInst)
		if err != nil {
			return nil, fmt.Errorf("failed to create user-specific SSE proxy handler for %s (user %d): %w", mcpDBService.Name, userID, err)
		}
	case "httpproxy":
		targetHandler, err = proxy.GetOrCreateProxyToHTTPHandler(ctx, mcpDBService, sharedInst)
		if err != nil {
			return nil, fmt.Errorf("failed to create user-specific HTTP proxy handler for %s (user %d): %w", mcpDBService.Name, userID, err)
		}
	default:
		return nil, fmt.Errorf("unsupported proxy type for user-specific handler: %s", proxyType)
	}

	common.SysLog(fmt.Sprintf("[ProxyHandler] Successfully created user-specific %s handler for %s (user %d)", proxyType, mcpDBService.Name, userID))
	return targetHandler, nil
}

// tryGetOrCreateGlobalHandler attempts to find or create a global handler for the service.
// proxyType should be "sseproxy" or "httpproxy"
func tryGetOrCreateGlobalHandler(c *gin.Context, mcpDBService *model.MCPService, proxyType string) (http.Handler, error) {
	common.SysLog(fmt.Sprintf("[ProxyHandler] Attempting global handler for service %s with proxy type %s", mcpDBService.Name, proxyType))

	// Use unified global cache key and standardized parameters (same as ServiceFactory)
	ctx := c.Request.Context()
	globalSharedCacheKey := fmt.Sprintf("global-service-%d-shared", mcpDBService.ID)
	instanceNameDetail := fmt.Sprintf("global-shared-svc-%d", mcpDBService.ID)
	effectiveEnvs := mcpDBService.DefaultEnvsJSON

	sharedInst, err := proxy.GetOrCreateSharedMcpInstanceWithKey(ctx, mcpDBService, globalSharedCacheKey, instanceNameDetail, effectiveEnvs)
	if err != nil {
		return nil, fmt.Errorf("failed to create shared MCP instance for %s: %w", mcpDBService.Name, err)
	}

	var targetHandler http.Handler
	switch proxyType {
	case "sseproxy":
		targetHandler, err = proxy.GetOrCreateProxyToSSEHandler(ctx, mcpDBService, sharedInst)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSE proxy handler for %s: %w", mcpDBService.Name, err)
		}
	case "httpproxy":
		targetHandler, err = proxy.GetOrCreateProxyToHTTPHandler(ctx, mcpDBService, sharedInst)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP proxy handler for %s: %w", mcpDBService.Name, err)
		}
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	common.SysLog(fmt.Sprintf("[ProxyHandler] Successfully created global %s handler for %s", proxyType, mcpDBService.Name))
	return targetHandler, nil
}

// ProxyHandler handles GET and POST /proxy/:serviceName/*action
func ProxyHandler(c *gin.Context) {
	serviceName := c.Param("serviceName")
	action := c.Param("action") // This captures the path after /proxy/:serviceName
	requestPath := c.Request.URL.Path
	requestMethod := c.Request.Method

	common.SysLog(fmt.Sprintf("[ProxyHandler] Service: %s, Action: %s, Method: %s, Path: %s, Query: %s",
		serviceName, action, requestMethod, requestPath, c.Request.URL.RawQuery))

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
		// Determine proxy type based on action (SSE vs Streamable endpoint routing)
		proxyType := "sseproxy" // default to SSE
		if action == "/mcp" {
			proxyType = "httpproxy" // Streamable endpoint
		}
		// Note: Both /sse and /message are SSE type endpoints and use sseproxy

		targetHandler, handlerErr = tryGetOrCreateUserSpecificHandler(c, mcpDBService, userID, proxyType)
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

		// Determine proxy type based on action (SSE vs Streamable endpoint routing)
		proxyType := "sseproxy" // default to SSE for /sse and /message endpoints
		if action == "/mcp" {
			proxyType = "httpproxy" // Streamable endpoint uses HTTP proxy
		}
		// Additional routing validation for better error messages
		if action != "/sse" && action != "/message" && action != "/mcp" &&
			!strings.HasPrefix(action, "/sse/") && !strings.HasPrefix(action, "/message/") && !strings.HasPrefix(action, "/mcp/") {
			common.SysLog(fmt.Sprintf("WARN: [ProxyHandler] Unrecognized action %s for service %s, defaulting to SSE proxy", action, serviceName))
		}

		targetHandler, handlerErr = tryGetOrCreateGlobalHandler(c, mcpDBService, proxyType)
	}

	if targetHandler != nil {
		common.SysLog(fmt.Sprintf("[ProxyHandler] Serving request for service %s (processed path %s) using obtained handler.", serviceName, c.Request.URL.Path))

		// Unified logic for determining if this request should be recorded for statistics
		shouldRecordStat := false
		requestTypeForStat := ""
		methodForStat := ""

		if requestMethod == http.MethodPost {
			if action == "/message" || action == "/mcp" {
				if c.Request.Body != nil {
					bodyBytes, err := io.ReadAll(c.Request.Body)
					if err != nil {
						c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					} else {
						c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

						var parsedBody map[string]interface{}
						if errUnmarshal := json.Unmarshal(bodyBytes, &parsedBody); errUnmarshal != nil {
						} else {
							if actualMethod, ok := parsedBody["method"].(string); ok && actualMethod == "tools/call" {
								shouldRecordStat = true
								methodForStat = "tools/call"
								if action == "/message" {
									requestTypeForStat = "sse"
								} else { // action == "/mcp"
									requestTypeForStat = "http"
								}
							}
						}
					}
				} else { // c.Request.Body == nil
				}
			} else {
			}
		} else {
		}

		if shouldRecordStat {
			startTime := time.Now()

			// It's important to serve the request using the potentially restored body
			targetHandler.ServeHTTP(c.Writer, c.Request)

			duration := time.Since(startTime)
			statusCode := c.Writer.Status()
			success := statusCode >= 200 && statusCode < 300

			// Record the statistic to database
			go model.RecordRequestStat(
				mcpDBService.ID,
				mcpDBService.Name, // Service Name
				userID,
				model.ProxyRequestType(requestTypeForStat),
				methodForStat,
				requestPath,
				duration.Milliseconds(),
				statusCode,
				success,
			)

			// Record daily request count to cache only if status is 200 or 202
			if statusCode == http.StatusOK || statusCode == http.StatusAccepted {
				go func() {
					cacheClient := thing.Cache()
					if cacheClient == nil {
						common.SysError(fmt.Sprintf("[ProxyHandler-CACHE] Cache client is nil for service %s", serviceName))
						return
					}

					today := time.Now().Format("2006-01-02")
					cacheKey := fmt.Sprintf("request:%s:%d:count", today, mcpDBService.ID)

					ctx := context.Background()

					// Get current count first
					currentValue, err := cacheClient.Get(ctx, cacheKey)
					var count int64 = 1
					if err == nil {
						if currentCount, parseErr := strconv.ParseInt(currentValue, 10, 64); parseErr == nil {
							count = currentCount + 1
						}
					}

					// Set the incremented value with expiration
					err = cacheClient.Set(ctx, cacheKey, strconv.FormatInt(count, 10), 24*time.Hour)
					if err != nil {
						common.SysError(fmt.Sprintf("[ProxyHandler-CACHE] Error setting daily count for service %s: %v", serviceName, err))
						return
					}

					if count == 1 {
						common.SysLog(fmt.Sprintf("[ProxyHandler-CACHE] Created daily count key %s for service %s", cacheKey, serviceName))
					}

					common.SysLog(fmt.Sprintf("[ProxyHandler-CACHE] Daily count for service %s: %d", serviceName, count))
				}()
			} else {
				common.SysLog(fmt.Sprintf("[ProxyHandler-CACHE] Daily count for service %s not incremented due to status code: %d", serviceName, statusCode))
			}

		} else {
			// If not recording stats, just serve the request
			// If body was read for a non-stat HTTP/MCP call, it should have been restored already.
			targetHandler.ServeHTTP(c.Writer, c.Request)
		}

	} else {
		finalErrMsg := "critical: unable to obtain any valid handler for service " + serviceName
		if handlerErr != nil {
			finalErrMsg = fmt.Sprintf("Service handler unavailable for %s: %s", serviceName, handlerErr.Error())
		}
		common.SysError(fmt.Sprintf("[ProxyHandler] Error: %s", finalErrMsg))
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": finalErrMsg})
	}
}
