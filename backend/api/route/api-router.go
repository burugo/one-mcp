package route

import (
	"one-mcp/backend/api/handler"
	"one-mcp/backend/api/middleware"

	"github.com/gin-gonic/gin"
)

func SetApiRouter(route *gin.Engine) {
	apiRouter := route.Group("/api")
	apiRouter.Use(middleware.LangMiddleware())
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		// Public routes (no authentication required)
		apiRouter.GET("/status", handler.GetStatus)
		apiRouter.GET("/notice", handler.GetNotice)
		apiRouter.GET("/about", handler.GetAbout)
		apiRouter.GET("/verification", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), handler.ResetPassword)
		apiRouter.GET("/oauth/github", middleware.CriticalRateLimit(), handler.GitHubOAuth)
		apiRouter.GET("/oauth/google", middleware.CriticalRateLimit(), handler.GoogleOAuth)
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), handler.WeChatAuth)

		// Authentication routes
		authRoutes := apiRouter.Group("/auth")
		{
			authRoutes.POST("/login", middleware.CriticalRateLimit(), handler.Login)
			// authRoutes.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.Register)
			authRoutes.POST("/refresh", middleware.CriticalRateLimit(), handler.RefreshToken)
			authRoutes.POST("/logout", middleware.CriticalRateLimit(), handler.Logout)
		}

		// OAuth routes that require authentication
		authOauthRoutes := apiRouter.Group("/oauth")
		authOauthRoutes.Use(middleware.JWTAuth())
		{
			authOauthRoutes.GET("/github/bind", middleware.CriticalRateLimit(), handler.GitHubBind)
			authOauthRoutes.GET("/google/bind", middleware.CriticalRateLimit(), handler.GoogleBind)
			authOauthRoutes.GET("/wechat/bind", middleware.CriticalRateLimit(), handler.WeChatBind)
			authOauthRoutes.GET("/email/bind", middleware.CriticalRateLimit(), handler.EmailBind)
		}

		// User routes - keeping legacy endpoints for backwards compatibility
		apiRouter.POST("/user/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.Register)
		apiRouter.POST("/user/login", middleware.CriticalRateLimit(), handler.Login)
		apiRouter.GET("/user/logout", handler.Logout)

		// User routes that require authentication
		userRoute := apiRouter.Group("/user")
		{
			// Self-related endpoints (require authentication)
			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.JWTAuth())
			{
				selfRoute.GET("/self", handler.GetSelf)
				selfRoute.PUT("/self", handler.UpdateSelf)
				selfRoute.DELETE("/self", handler.DeleteSelf)
				selfRoute.GET("/token", handler.GenerateToken)
				selfRoute.POST("/change-password", handler.ChangePassword)
			}

			// Admin-only endpoints
			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.JWTAuth())   // First authenticate with JWT
			adminRoute.Use(middleware.AdminAuth()) // Then check admin privileges
			{
				adminRoute.GET("/", handler.GetAllUsers)
				adminRoute.GET("/search", handler.SearchUsers)
				adminRoute.GET("/:id", handler.GetUser)
				adminRoute.POST("/", handler.CreateUser)
				adminRoute.POST("/manage", handler.ManageUser)
				adminRoute.PUT("/", handler.UpdateUser)
				adminRoute.DELETE("/:id", handler.DeleteUser)
			}
		}

		// Option routes (Root admin only)
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.JWTAuth())  // First authenticate with JWT
		optionRoute.Use(middleware.RootAuth()) // Then check root privileges
		{
			optionRoute.GET("/", handler.GetOptions)
			optionRoute.PUT("/", handler.UpdateOption)
		}

		// MCP Service routes
		mcpServiceRoute := apiRouter.Group("/mcp_services")
		{
			// Public endpoints (read-only, require authentication)
			mcpServiceRoute.Use(middleware.JWTAuth())
			{
				mcpServiceRoute.POST("/:id/health/check", handler.CheckMCPServiceHealth)
			}

			// Admin-only endpoints (write operations)
			adminMCPServiceRoute := mcpServiceRoute.Group("/")
			adminMCPServiceRoute.Use(middleware.JWTAuth())   // First authenticate with JWT
			adminMCPServiceRoute.Use(middleware.AdminAuth()) // Then check admin privileges
			{
				adminMCPServiceRoute.PUT("/:id", handler.UpdateMCPService)
				adminMCPServiceRoute.POST("/:id/toggle", handler.ToggleMCPService)
			}
		}

		// Market API routes
		marketRoute := apiRouter.Group("/mcp_market")
		marketRoute.Use(middleware.JWTAuth())
		{
			marketRoute.GET("/search", handler.SearchMCPMarket)
			marketRoute.GET("/discover_env_vars", handler.DiscoverEnvVars)
			marketRoute.GET("/installed", handler.ListInstalledMCPServices)
			marketRoute.GET("/package_details", handler.GetPackageDetails)
			marketRoute.GET("/install_status/:id", handler.GetInstallationStatus)
			marketRoute.PATCH("/env_var", handler.PatchEnvVar)

			// Admin-only endpoints
			adminMarketRoute := marketRoute.Group("/")
			adminMarketRoute.Use(middleware.AdminAuth()) // JWTAuth already applied by parent group
			{
				adminMarketRoute.POST("/install_or_add_service", handler.InstallOrAddService)
				adminMarketRoute.POST("/batch-import", handler.StartBatchImport)
				adminMarketRoute.POST("/uninstall", handler.UninstallService)
				adminMarketRoute.POST("/custom_service", handler.CreateCustomService)
			}
		}

		// SSE endpoint for batch import progress (no middleware, handles auth internally)
		// This must be outside the marketRoute group to avoid JWTAuth middleware
		apiRouter.GET("/mcp_market/batch-import/progress/:task_id", handler.StreamBatchImportProgress)

		// User Config routes
		// configRoute := apiRouter.Group("/configs")
		// configRoute.Use(middleware.JWTAuth())
		// {
		// 	configRoute.GET("/", handler.GetUserConfigs)
		// 	configRoute.POST("/", handler.CreateUserConfig)
		// 	configRoute.GET("/:id", handler.GetUserConfig)
		// 	configRoute.PUT("/:id", handler.UpdateUserConfig)
		// 	configRoute.DELETE("/:id", handler.DeleteUserConfig)
		// 	configRoute.GET("/:id/:client", handler.ExportUserConfig)
		// }

		// 注册 SSE 代理路由
		// The *action will capture the rest of the path (e.g., /message, /files/somefile.txt)
		// apiRouter.GET("/sse/:serviceName/*action", handler.ProxyHandler)
		// apiRouter.POST("/sse/:serviceName/*action", handler.ProxyHandler) // Also handle POST requests

		// New SSE Proxy routes (incorrectly placed previously)
		// apiRouter.GET("/proxy/:serviceName/sse/*action", handler.ProxyHandler)
		// apiRouter.POST("/proxy/:serviceName/sse/*action", handler.ProxyHandler)
	}

	// Analytics routes
	analyticsRoute := apiRouter.Group("/analytics")
	analyticsRoute.Use(middleware.JWTAuth()) // Assuming analytics requires auth
	// Consider admin-only access if appropriate: analyticsRoute.Use(middleware.AdminAuth())
	{
		analyticsRoute.GET("/services/utilization", handler.GetServiceUtilization)
		analyticsRoute.GET("/services/metrics", handler.GetServiceMetrics)
		analyticsRoute.GET("/system/overview", handler.GetSystemOverview)
	}

	// Define routes under /proxy, outside the /api group
	proxyRouter := route.Group("/proxy")
	proxyRouter.Use(middleware.LangMiddleware()) // Apply similar general middlewares
	proxyRouter.Use(middleware.GlobalAPIRateLimit())
	proxyRouter.Use(middleware.TokenAuth()) // Add token-based authentication for proxy endpoints
	{
		// SSE proxy routes - for SSE endpoints and stdio->SSE conversion
		// proxyRouter.Any("/:serviceName/sse/*action", handler.ProxyHandler)

		// HTTP proxy routes - for native HTTP MCP services
		// proxyRouter.Any("/:serviceName/mcp/*action", handler.HTTPProxyHandler)

		// Legacy route removed to fix routing conflict with specific routes above
		proxyRouter.Any("/:serviceName/*action", handler.ProxyHandler)
	}
}
