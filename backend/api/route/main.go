package route

import (
	"embed"
	"github.com/gin-gonic/gin"
	"one-mcp/backend/api/middleware"
)

func SetRouter(route *gin.Engine, buildFS embed.FS, indexPage []byte) {
	// Apply gzip middleware to the entire application
	route.Use(middleware.GzipDecodeMiddleware()) // Decode gzipped requests
	route.Use(middleware.GzipEncodeMiddleware()) // Compress responses with gzip
	
	SetApiRouter(route)
	setWebRouter(route, buildFS, indexPage)
}
