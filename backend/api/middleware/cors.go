package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://one-mcp/backend.vercel.app", "http://localhost:3000/", "http://localhost:5173"}
	// It's often better to allow all headers and methods during development, or be very specific.
	// config.AllowAllOrigins = true // Consider for local dev if issues persist
	config.AllowHeaders = append(config.AllowHeaders, "Authorization", "X-Requested-With", "X-Request-Id") // Add any custom headers your frontend might send
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowCredentials = true
	return cors.New(config)
}
