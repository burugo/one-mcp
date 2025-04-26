package routes

import (
	"one-cmp/backend/internal/api/handlers"
	"one-cmp/backend/internal/api/middlewares"
	"one-cmp/backend/internal/common"

	"github.com/gin-gonic/gin"
)

// SetApiRouter sets up the API routes.
func SetApiRouter(router *gin.Engine) {
	router.Use(middlewares.RequestId())
	router.Use(middlewares.CORS())

	apiRouter := router.Group("/api")
	apiRouter.Use(middlewares.GlobalAPIRateLimit())
	{
		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", handlers.Register)
			userRoute.POST("/login", handlers.Login)
			userRoute.GET("/logout", handlers.Logout)

			authRoute := userRoute.Group("/")
			authRoute.Use(middlewares.AuthRequired())
			{
				authRoute.GET("/self", handlers.GetSelf)
				authRoute.PUT("/self", handlers.UpdateSelf)
				authRoute.DELETE("/self", handlers.DeleteSelf)
				authRoute.POST("/email/bind", handlers.EmailBind)
			}

			adminRoute := userRoute.Group("/")
			adminRoute.Use(middlewares.AdminRequired())
			{
				adminRoute.GET("/:id", handlers.GetUser)
				adminRoute.GET("/", handlers.GetAllUsers)
				adminRoute.POST("/", handlers.CreateUser)
				adminRoute.PUT("/:id", handlers.UpdateUser)
				adminRoute.DELETE("/:id", handlers.DeleteUser)
				adminRoute.POST("/manage", handlers.ManageUser)
			}
			userRoute.GET("/github/oauth", handlers.GitHubOAuth)
			userRoute.GET("/github/callback", handlers.GitHubCallback)
			userRoute.GET("/wechat/bind", handlers.WeChatBind)
			userRoute.GET("/wechat/login", handlers.WeChatLogin)
			userRoute.POST("/verification", handlers.SendVerificationCode)
			userRoute.GET("/reset_password", handlers.SendPasswordResetEmail)
			userRoute.POST("/reset_password", handlers.ResetPassword)
		}

		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middlewares.AdminRequired())
		{
			optionRoute.GET("/", handlers.GetAllOptions)
			optionRoute.PUT("/", handlers.UpdateOption)
		}

		fileRoute := apiRouter.Group("/file")
		{
			fileRoute.GET("/:id", handlers.DownloadFile)
			fileRoute.GET("/", handlers.GetAllFiles)
			fileRoute.POST("/", handlers.UploadFile)
			fileRoute.DELETE("/:id", handlers.DeleteFile)
			fileRoute.GET("/search", handlers.SearchFiles)
		}

		miscRoute := apiRouter.Group("/misc")
		{
			miscRoute.GET("/status", handlers.GetStatus)
			miscRoute.GET("/version", handlers.GetVersion)
			miscRoute.GET("/notice", handlers.GetNotice)
			miscRoute.GET("/about", handlers.GetAbout)
			miscRoute.GET("/home_page_link", handlers.GetHomePageLink)
		}
	}
}
