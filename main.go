package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"strconv"

	"one-mcp/backend/api/middleware"
	"one-mcp/backend/api/route"
	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

//go:embed web/build
var buildFS embed.FS

//go:embed web/build/index.html
var indexPage []byte

func main() {
	flag.Parse()
	if *common.PrintVersion {
		println(common.Version)
		os.Exit(0)
	}
	if *common.PrintHelpFlag {
		common.PrintHelp()
		os.Exit(0)
	}
	common.SetupGinLog()
	common.SysLog("One MCP Backend (from Gin Template) " + common.Version + " started")
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	// Initialize SQL Database
	err := model.InitDB()
	if err != nil {
		common.FatalLog(err)
	}
	defer func() {
		err := model.CloseDB()
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
	model.InitOptionMap()

	// Initialize HTTP server
	server := gin.Default()
	//server.Use(gzip.Gzip(gzip.DefaultCompression))
	server.Use(middleware.CORS())

	// Initialize session store
	if common.RedisEnabled {
		opt := common.ParseRedisOption()
		store, _ := redis.NewStore(opt.MinIdleConns, opt.Network, opt.Addr, opt.Password, common.SessionSecret)
		server.Use(sessions.Sessions("session", store))
	} else {
		store := cookie.NewStore([]byte(common.SessionSecret))
		server.Use(sessions.Sessions("session", store))
	}

	route.SetRouter(server, buildFS, indexPage)
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
