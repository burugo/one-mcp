package route

import (
	"one-mcp/backend/api/handler"
	"one-mcp/backend/api/middleware"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(route *gin.Engine) {
	apiRouter := route.Group("/api")
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
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), handler.WeChatAuth)

		// Authentication routes
		authRoutes := apiRouter.Group("/auth")
		{
			authRoutes.POST("/login", middleware.CriticalRateLimit(), handler.Login)
			authRoutes.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), handler.Register)
			authRoutes.POST("/refresh", middleware.CriticalRateLimit(), handler.RefreshToken)
			authRoutes.POST("/logout", middleware.CriticalRateLimit(), handler.Logout)
		}

		// OAuth routes that require authentication
		authOauthRoutes := apiRouter.Group("/oauth")
		authOauthRoutes.Use(middleware.JWTAuth())
		{
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
			}

			// Admin-only endpoints
			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth())
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
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", handler.GetOptions)
			optionRoute.PUT("/", handler.UpdateOption)
		}

		// MCP Service routes
		serviceRoute := apiRouter.Group("/services")
		{
			// Public endpoints (read-only, require authentication)
			serviceRoute.Use(middleware.JWTAuth())
			{
				serviceRoute.GET("/", handler.GetAllServices)
				serviceRoute.GET("/:id", handler.GetService)
				serviceRoute.GET("/:id/config/:client", handler.GetServiceConfig)
			}

			// Admin-only endpoints (write operations)
			adminServiceRoute := serviceRoute.Group("/")
			adminServiceRoute.Use(middleware.AdminAuth())
			{
				adminServiceRoute.POST("/", handler.CreateService)
				adminServiceRoute.PUT("/:id", handler.UpdateService)
				adminServiceRoute.DELETE("/:id", handler.DeleteService)
				adminServiceRoute.POST("/:id/toggle", handler.ToggleService)
			}
		}

		// User Config routes
		configRoute := apiRouter.Group("/configs")
		configRoute.Use(middleware.JWTAuth())
		{
			configRoute.GET("/", handler.GetUserConfigs)
			configRoute.POST("/", handler.CreateUserConfig)
			configRoute.GET("/:id", handler.GetUserConfig)
			configRoute.PUT("/:id", handler.UpdateUserConfig)
			configRoute.DELETE("/:id", handler.DeleteUserConfig)
			configRoute.GET("/:id/:client", handler.ExportUserConfig)
		}
	}
}
