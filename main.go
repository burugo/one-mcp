package main

import (
	"log"
	"os"
	"strconv"

	"one-mcp/backend/api/middlewares"
	"one-mcp/backend/api/routes"
	"one-mcp/backend/common"
	"one-mcp/backend/library/db"
	"one-mcp/backend/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	common.SetupGinLog()
	common.SysLog("One MCP Backend (from Gin Template) " + common.Version + " started")
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	// Initialize SQL Database
	err := db.InitDB()
	if err != nil {
		common.FatalLog(err)
	}
	defer func() {
		err := db.CloseDB()
		if err != nil {
			common.FatalLog(err)
		}
	}()

	// Initialize Redis
	err = common.InitRedisClient()
	if err != nil {
		common.FatalLog(err)
	}

	// Initialize options
	// model.InitOptionMap()
	// TODO: Resolve where InitOptionMap should live and be called. For now, it depends on db, so maybe in db package?

	// Initialize HTTP server
	server := gin.Default()
	//server.Use(gzip.Gzip(gzip.DefaultCompression))
	server.Use(middlewares.CORS())

	// Initialize session store
	if common.RedisEnabled {
		opt := common.ParseRedisOption()
		store, _ := redis.NewStore(opt.MinIdleConns, opt.Network, opt.Addr, opt.Password, common.SessionSecret)
		server.Use(sessions.Sessions("session", store))
	} else {
		store := cookie.NewStore([]byte(common.SessionSecret))
		server.Use(sessions.Sessions("session", store))
	}

	routes.SetRouter(server)
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	common.SysLog("Server listening on port: " + port)
	err = server.Run(":" + port)
	if err != nil {
		log.Fatal("failed to start server: " + err.Error())
	}
}
