package service

import (
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/burugo/thing"
	"github.com/stretchr/testify/assert"
)

func init() {
	common.JWTSecret = "test-jwt-secret-key-for-unit-tests"
	common.JWTRefreshSecret = "test-jwt-refresh-secret-key-for-unit-tests"
}

func TestGenerateToken(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 1},
		Username:  "testuser",
		Role:      1,
	}

	token, err := GenerateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_ValidToken(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 42},
		Username:  "alice",
		Role:      2,
	}

	token, err := GenerateToken(user)
	assert.NoError(t, err)

	claims, err := ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, 2, claims.Role)
	assert.Equal(t, "one-mcp", claims.Issuer)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	claims, err := ValidateToken("invalid-token-string")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateToken_TamperedToken(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 1},
		Username:  "testuser",
		Role:      1,
	}

	token, err := GenerateToken(user)
	assert.NoError(t, err)

	// Tamper with the token
	tamperedToken := token + "tampered"
	claims, err := ValidateToken(tamperedToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestGenerateRefreshToken(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 1},
		Username:  "testuser",
		Role:      1,
	}

	refreshToken, err := GenerateRefreshToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshToken)
}

func TestValidateRefreshToken_ValidToken(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 99},
		Username:  "bob",
		Role:      3,
	}

	refreshToken, err := GenerateRefreshToken(user)
	assert.NoError(t, err)

	claims, err := ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, int64(99), claims.UserID)
	assert.Equal(t, "bob", claims.Username)
	assert.Equal(t, 3, claims.Role)
}

func TestValidateRefreshToken_InvalidToken(t *testing.T) {
	claims, err := ValidateRefreshToken("invalid-refresh-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_WrongSecret(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 1},
		Username:  "testuser",
		Role:      1,
	}

	// Generate access token (uses JWTSecret)
	accessToken, err := GenerateToken(user)
	assert.NoError(t, err)

	// Try to validate access token as refresh token (uses JWTRefreshSecret)
	claims, err := ValidateRefreshToken(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestRefreshToken_Success(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 123},
		Username:  "refreshuser",
		Role:      1,
	}

	refreshToken, err := GenerateRefreshToken(user)
	assert.NoError(t, err)

	newAccessToken, err := RefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	// Validate the new access token
	claims, err := ValidateToken(newAccessToken)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), claims.UserID)
	assert.Equal(t, "refreshuser", claims.Username)
}

func TestRefreshToken_InvalidRefreshToken(t *testing.T) {
	newAccessToken, err := RefreshToken("invalid-refresh-token")
	assert.Error(t, err)
	assert.Empty(t, newAccessToken)
}

func TestJWTClaims_Expiration(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 1},
		Username:  "testuser",
		Role:      1,
	}

	token, err := GenerateToken(user)
	assert.NoError(t, err)

	claims, err := ValidateToken(token)
	assert.NoError(t, err)

	// Check that expiration is in the future (7 days from now)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
	assert.True(t, claims.ExpiresAt.Before(time.Now().Add(8*24*time.Hour)))
}

func TestTokensAreDifferent(t *testing.T) {
	user := &model.User{
		BaseModel: thing.BaseModel{ID: 1},
		Username:  "testuser",
		Role:      1,
	}

	accessToken, err := GenerateToken(user)
	assert.NoError(t, err)

	refreshToken, err := GenerateRefreshToken(user)
	assert.NoError(t, err)

	// Access and refresh tokens should be different
	assert.NotEqual(t, accessToken, refreshToken)
}
