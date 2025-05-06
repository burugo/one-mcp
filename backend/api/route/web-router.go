package route

import (
	"embed"
	"one-mcp/backend/common"
	"one-mcp/backend/api/middleware"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"net/http"
)

func setWebRouter(route *gin.Engine, buildFS embed.FS, indexPage []byte) {
	route.Use(middleware.GlobalWebRateLimit())
	route.Use(middleware.Cache())
	route.Use(static.Serve("/", common.EmbedFolder(buildFS, "web/build")))
	route.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
	})
}
