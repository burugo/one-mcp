package service

import (
	"errors"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/burugo/thing"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in the JWT
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(user *model.User) (string, error) {
	// Create token with claims
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 7)), // Token expires in 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "one-mcp",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(common.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken creates a refresh token
func GenerateRefreshToken(user *model.User) (string, error) {
	// Create token with claims
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Refresh token expires in 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "one-mcp",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(common.JWTRefreshSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates the JWT token
func ValidateToken(tokenString string) (*JWTClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(common.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Check if the token is valid and extract the claims
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken validates the refresh token
func ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(common.JWTRefreshSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Check if the token is valid and extract the claims
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// RefreshToken validates a refresh token and generates a new access token
func RefreshToken(refreshToken string) (string, error) {
	// Validate the refresh token
	claims, err := ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Create a user object from the claims
	user := &model.User{
		BaseModel: thing.BaseModel{ID: claims.UserID},
		Username:  claims.Username,
		Role:      claims.Role,
	}

	// Generate a new access token
	newToken, err := GenerateToken(user)
	if err != nil {
		return "", err
	}

	return newToken, nil
}
