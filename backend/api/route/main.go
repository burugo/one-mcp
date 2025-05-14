package route

import (
	"embed"
	"one-mcp/backend/api/middleware"

	"github.com/gin-gonic/gin"
)

func SetRouter(route *gin.Engine, buildFS embed.FS, indexPage []byte) {
	// Apply gzip middleware to the entire application
	route.Use(middleware.GzipDecodeMiddleware()) // Decode gzipped requests
	route.Use(middleware.GzipEncodeMiddleware()) // Compress responses with gzip

	// Apply CORS middleware globally
	route.Use(middleware.CORS())

	SetApiRouter(route)
	setWebRouter(route, buildFS, indexPage)
}
