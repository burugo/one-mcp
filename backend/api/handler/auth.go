package handler

import (
	"net/http"
	"time"

	// "time" // No longer needed here as Logout logic is separate

	// "one-mcp/backend/common" // No longer needed here if Redis logic is self-contained in Logout
	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"one-mcp/backend/service"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents the request body for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserLoginResponseDTO REMOVED
// type UserLoginResponseDTO struct {
// 	ID          int64  `json:"id"`
// 	Username    string `json:"username"`
// 	DisplayName string `json:"display_name"`
// 	Email       string `json:"email"`
// 	Role        int    `json:"role"`
// 	Status      int    `json:"status"`
// }

// LoginResponse represents the response for login
type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         *model.User `json:"user"` // Changed back to *model.User
}

// Login handles user login and returns JWT tokens
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	user := &model.User{
		Username: req.Username,
		Password: req.Password,
	}
	if err := user.ValidateAndFill(); err != nil {
		common.RespErrorStr(c, http.StatusUnauthorized, err.Error())
		return
	}

	accessToken, err := service.GenerateToken(user)
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to generate access token: "+err.Error())
		return
	}

	refreshToken, err := service.GenerateRefreshToken(user)
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to generate refresh token: "+err.Error())
		return
	}

	common.RespSuccessWithMsg(c, "Login successful", LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user, // Now directly using *model.User
	})
}

// RefreshTokenRequest represents the request body for refreshing a token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents the response for refreshing a token
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// RefreshToken handles refreshing an expired access token
func RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Refresh the token
	newToken, err := service.RefreshToken(req.RefreshToken)
	if err != nil {
		common.RespErrorStr(c, http.StatusUnauthorized, "Invalid refresh token: "+err.Error())
		return
	}

	// Return the new access token
	common.RespSuccess(c, RefreshTokenResponse{
		AccessToken: newToken,
	})
}

// LogoutRequest represents the request body for logout
type LogoutRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// Logout handles user logout by blacklisting the token
func Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate the token first
	claims, err := service.ValidateToken(req.AccessToken)
	if err != nil {
		common.RespErrorStr(c, http.StatusUnauthorized, "Invalid token: "+err.Error())
		return
	}

	// Blacklist the token if Redis is enabled
	if common.RedisEnabled {
		// Calculate remaining time until expiration
		expirationTime := time.Unix(claims.ExpiresAt.Unix(), 0)
		ttl := expirationTime.Sub(time.Now())

		// Add token to blacklist
		err := common.RDB.Set(c, "jwt:blacklist:"+req.AccessToken, true, ttl).Err()
		if err != nil {
			common.RespErrorStr(c, http.StatusInternalServerError, "Failed to blacklist token: "+err.Error())
			return
		}
	}

	common.RespSuccessStr(c, "Logged out successfully")
}

// RegisterRequest represents the request body for registration
type RegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	DisplayName string `json:"display_name"`
	Captcha     string `json:"captcha"`
	CaptchaID   string `json:"captcha_id"`
}

// Register handles user registration
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// // Verify captcha if enabled
	// if common.RegisterCaptchaEnabled {
	// 	if req.Captcha == "" || req.CaptchaID == "" {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"success": false,
	// 			"message": "Captcha is required",
	// 		})
	// 		return
	// 	}

	// 	if !common.VerifyCodeWithKey(req.CaptchaID, req.Captcha, common.CaptchaPurpose) {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"success": false,
	// 			"message": "Invalid captcha",
	// 		})
	// 		return
	// 	}
	// }

	if model.IsUsernameAlreadyTaken(req.Username) {
		common.RespErrorStr(c, http.StatusConflict, "Username already exists")
		return
	}

	if model.IsEmailAlreadyTaken(req.Email) {
		common.RespErrorStr(c, http.StatusConflict, "Email already exists")
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	user := &model.User{
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		DisplayName: displayName,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}

	if err := user.Insert(); err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	accessToken, err := service.GenerateToken(user)
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to generate access token after registration: "+err.Error())
		return
	}

	refreshToken, err := service.GenerateRefreshToken(user)
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to generate refresh token after registration: "+err.Error())
		return
	}

	common.RespSuccessWithMsg(c, "Registration successful", LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user, // Now directly using *model.User
	})
}
