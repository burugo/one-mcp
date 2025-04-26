package routes

import (
	"github.com/gin-gonic/gin"

	"one-cmp/backend/internal/api/middlewares"
)

// SetRouter sets up all the API routes.
// Removed buildFS and indexPage parameters as frontend is separate.
func SetRouter(router *gin.Engine) {
	// Set up API routes
	SetApiRouter(router)

	// Removed web router setup
	// setWebRouter(router, buildFS, indexPage)
}

// Serve is a placeholder or helper function, might not be needed
func Serve(router *gin.Engine) {
	// This function seems unused in the original main.go call structure.
	// Perhaps intended for a different setup.
	// We can remove or repurpose it if necessary.
}
