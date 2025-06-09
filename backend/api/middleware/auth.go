package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"one-mcp/backend/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func authHelper(c *gin.Context, minRole int) {
	session := sessions.Default(c)
	username := session.Get("username")
	role := session.Get("role")
	id := session.Get("id")
	status := session.Get("status")
	authByToken := false
	if username == nil {
		// Check token
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无权进行此操作，未登录或 token 无效",
			})
			c.Abort()
			return
		}
		user := model.ValidateUserToken(token)
		if user != nil && user.Username != "" {
			// Token is valid
			username = user.Username
			role = user.Role
			id = user.ID
			status = user.Status
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无权进行此操作，token 无效",
			})
			c.Abort()
			return
		}
		authByToken = true
	}
	if status.(int) == common.UserStatusDisabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户已被封禁",
		})
		c.Abort()
		return
	}
	if role.(int) < minRole {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权进行此操作，权限不足",
		})
		c.Abort()
		return
	}
	c.Set("username", username)
	c.Set("role", role)
	c.Set("id", id)
	c.Set("authByToken", authByToken)
	c.Next()
}

func UserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, common.RoleCommonUser)
	}
}

// TokenAuth is a middleware for proxy endpoints that validates user tokens from URL parameters
// It supports both JWT Bearer tokens in headers and user tokens in URL query parameters
func TokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userID int64
		var username string
		var role int

		// First, try to get JWT token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString := parts[1]
				claims, err := service.ValidateToken(tokenString)
				if err == nil {
					// Check if token is blacklisted
					if common.RedisEnabled {
						blacklisted, _ := common.RDB.Exists(c, "jwt:blacklist:"+tokenString).Result()
						if blacklisted <= 0 {
							userID = claims.UserID
							username = claims.Username
							role = claims.Role
						}
					} else {
						userID = claims.UserID
						username = claims.Username
						role = claims.Role
					}
				}
			}
		}

		// If JWT validation failed or no JWT, try user token from URL query parameter
		if userID == 0 {
			userToken := c.Query("key")
			if userToken != "" {
				user := model.ValidateUserTokenByTokenString(userToken)
				if user != nil && user.Status == common.UserStatusEnabled {
					userID = user.ID
					username = user.Username
					role = user.Role
				}
			}
		}

		// If still no valid user found, continue without authentication
		// This allows the proxy to work in global mode if no valid authentication is provided
		if userID > 0 {
			c.Set("userID", userID)
			c.Set("user_id", userID) // Also set this for compatibility
			c.Set("username", username)
			c.Set("role", role)
			common.SysLog(fmt.Sprintf("[TokenAuth] Authenticated user %d (%s) for proxy request", userID, username))
		} else {
			common.SysLog("[TokenAuth] No valid authentication found, proceeding with global access")
		}

		c.Next()
	}
}

// JWTAuth is a middleware that validates JWT tokens
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Authorization header format must be Bearer {token}",
			})
			c.Abort()
			return
		}

		// Validate the token
		tokenString := parts[1]
		claims, err := service.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted
		if common.RedisEnabled {
			blacklisted, _ := common.RDB.Exists(c, "jwt:blacklist:"+tokenString).Result()
			if blacklisted > 0 {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "Token has been invalidated",
				})
				c.Abort()
				return
			}
		}

		// Set user information in the context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminAuth middleware verifies the user has admin role
// Note: This middleware assumes JWTAuth has already been called to set user info in context
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user has admin role
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Role information not found",
			})
			c.Abort()
			return
		}

		roleInt, ok := role.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		// Check if role is admin or higher
		if roleInt < common.RoleAdminUser {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Admin privileges required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RootAuth middleware verifies the user has root role
// Note: This middleware assumes JWTAuth has already been called to set user info in context
func RootAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user has root role
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Role information not found",
			})
			c.Abort()
			return
		}

		roleInt, ok := role.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		// Check if role is root
		if roleInt < common.RoleRootUser {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Root privileges required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// NoTokenAuth is a special middleware for endpoints that shouldn't use token authentication
// It's needed because some endpoints might already use session authentication
func NoTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip token auth for these endpoints
		c.Next()
	}
}

// TokenOnlyAuth You should always use this after normal auth middlewares.
func TokenOnlyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip JWT check, only check token
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无权进行此操作，未登录或 token 无效",
			})
			c.Abort()
			return
		}
		user := model.ValidateUserToken(token)
		if user == nil || user.Username == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无权进行此操作，token 无效",
			})
			c.Abort()
			return
		}
		if user.Status != common.UserStatusEnabled {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户已被封禁",
			})
			c.Abort()
			return
		}
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Set("id", user.ID)
		c.Next()
	}
}
