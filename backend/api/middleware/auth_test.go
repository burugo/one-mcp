package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
	common.JWTSecret = "test-jwt-secret-for-middleware-tests"
	common.JWTRefreshSecret = "test-jwt-refresh-secret-for-middleware-tests"
	common.RedisEnabled = false
}

func setupTestRouter() *gin.Engine {
	router := gin.New()
	return router
}

func TestJWTAuth_NoAuthorizationHeader(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Authorization header is required")
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Bearer")
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestJWTAuth_ValidToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", JWTAuth(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})

	// Generate a valid token using the service package
	token := generateTestToken(42, "testuser", common.RoleCommonUser)

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "testuser")
}

// generateTestToken creates a JWT token for testing purposes using the same method as service package
func generateTestToken(userID int64, username string, role int) string {
	claims := service.JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
	}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	claims.NotBefore = jwt.NewNumericDate(time.Now())
	claims.Issuer = "one-mcp"
	claims.Subject = username

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(common.JWTSecret))
	return tokenString
}

func TestAdminAuth_NoRole(t *testing.T) {
	router := setupTestRouter()
	router.GET("/admin", AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "Role information not found")
}

func TestAdminAuth_InvalidRoleType(t *testing.T) {
	router := setupTestRouter()
	router.GET("/admin", func(c *gin.Context) {
		c.Set("role", "not-an-int")
		c.Next()
	}, AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid role format")
}

func TestAdminAuth_InsufficientPrivileges(t *testing.T) {
	router := setupTestRouter()
	router.GET("/admin", func(c *gin.Context) {
		c.Set("role", common.RoleCommonUser)
		c.Next()
	}, AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "Admin privileges required")
}

func TestAdminAuth_Success(t *testing.T) {
	router := setupTestRouter()
	router.GET("/admin", func(c *gin.Context) {
		c.Set("role", common.RoleAdminUser)
		c.Next()
	}, AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestRootAuth_NoRole(t *testing.T) {
	router := setupTestRouter()
	router.GET("/root", RootAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/root", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "Role information not found")
}

func TestRootAuth_InvalidRoleType(t *testing.T) {
	router := setupTestRouter()
	router.GET("/root", func(c *gin.Context) {
		c.Set("role", "not-an-int")
		c.Next()
	}, RootAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/root", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid role format")
}

func TestRootAuth_InsufficientPrivileges_CommonUser(t *testing.T) {
	router := setupTestRouter()
	router.GET("/root", func(c *gin.Context) {
		c.Set("role", common.RoleCommonUser)
		c.Next()
	}, RootAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/root", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "Root privileges required")
}

func TestRootAuth_InsufficientPrivileges_AdminUser(t *testing.T) {
	router := setupTestRouter()
	router.GET("/root", func(c *gin.Context) {
		c.Set("role", common.RoleAdminUser)
		c.Next()
	}, RootAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/root", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "Root privileges required")
}

func TestRootAuth_Success(t *testing.T) {
	router := setupTestRouter()
	router.GET("/root", func(c *gin.Context) {
		c.Set("role", common.RoleRootUser)
		c.Next()
	}, RootAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/root", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestTokenAuth_NoToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/proxy", TokenAuth(), func(c *gin.Context) {
		userID, exists := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"has_user":   exists,
			"user_id":    userID,
		})
	})

	req, _ := http.NewRequest("GET", "/proxy", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// TokenAuth allows requests without auth (global access mode)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestTokenAuth_InvalidBearerToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/proxy", TokenAuth(), func(c *gin.Context) {
		_, exists := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"has_user": exists,
		})
	})

	req, _ := http.NewRequest("GET", "/proxy", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// TokenAuth continues without authentication on invalid token
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestTokenAuth_InvalidQueryKey(t *testing.T) {
	router := setupTestRouter()
	router.GET("/proxy", TokenAuth(), func(c *gin.Context) {
		_, exists := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"has_user": exists,
		})
	})

	req, _ := http.NewRequest("GET", "/proxy?key=invalid-key", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// TokenAuth continues without authentication on invalid key
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestNoTokenAuth(t *testing.T) {
	router := setupTestRouter()
	router.GET("/public", NoTokenAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/public", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestTokenOnlyAuth_NoToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/token-only", TokenOnlyAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/token-only", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestTokenOnlyAuth_InvalidToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/token-only", TokenOnlyAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/token-only", nil)
	req.Header.Set("Authorization", "invalid-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "token 无效")
}
