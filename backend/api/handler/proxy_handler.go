package handler

import (
	"net/http"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"strings"

	"github.com/gin-gonic/gin"
)

// SSEProxyHandler handles GET /api/sse/:serviceName/*action
func SSEProxyHandler(c *gin.Context) {
	serviceName := c.Param("serviceName")
	action := c.Param("action")

	if action == "" {
		action = "/" // Ensure /api/sse/server-name maps to /
	}

	if action != "" && !strings.HasPrefix(action, "/") {
		action = "/" + action
	}

	service, err := model.GetServiceByName(serviceName)
	if err != nil || service == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found"})
		return
	}

	serviceManager := proxy.GetServiceManager()
	sseSvc, err := serviceManager.GetSSEServiceByName(serviceName)
	if err != nil || sseSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "SSE service not available: " + err.Error()})
		return
	}

	c.Request.URL.Path = action

	sseSvc.ServeHTTP(c.Writer, c.Request)
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
