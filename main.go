package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"one-mcp/backend/api/middleware"
	"one-mcp/backend/api/route"
	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist
var buildFS embed.FS

//go:embed frontend/dist/index.html
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
	// Initialize Redis
	err := common.InitRedisClient()
	if err != nil {
		common.FatalLog(err)
	}
	// Initialize SQL Database
	err = model.InitDB()
	if err != nil {
		common.FatalLog(err)
	}
	defer func() {
		err := model.CloseDB()
		if err != nil {
			common.FatalLog(err)
		}
	}()

	// Seed default services
	// if err := model.SeedDefaultServices(); err != nil {
	// 	common.SysError(fmt.Sprintf("Failed to seed default services: %v", err))
	// 	// Depending on severity, might os.Exit(1) or just log
	// }

	// Initialize service manager
	serviceManager := proxy.GetServiceManager()
	go func() {
		if err := serviceManager.Initialize(context.Background()); err != nil {
			common.SysLog("Failed to initialize service manager: " + err.Error())
		} else {
			common.SysLog("Service manager initialized successfully")
		}
	}()

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
	server.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{
				"success": false,
				"message": "API route not found",
			})
		} else {
			c.File("./frontend/dist/index.html")
		}
	})

	port := strconv.Itoa(*common.Port)
	common.SysLog("Server listening on port: " + port)

	// Setup graceful shutdown
	setupGracefulShutdown()

	err = server.Run(":" + port)
	if err != nil {
		log.Fatal("failed to start server: " + err.Error())
	}
}

// setupGracefulShutdown registers signal handlers to ensure clean shutdown
func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		common.SysLog("Shutting down...")

		// 关闭服务管理器
		serviceManager := proxy.GetServiceManager()
		if err := serviceManager.Shutdown(context.Background()); err != nil {
			common.SysLog("Error shutting down service manager: " + err.Error())
		} else {
			common.SysLog("Service manager shut down successfully")
		}

		// 关闭其他资源...

		os.Exit(0)
	}()
}
