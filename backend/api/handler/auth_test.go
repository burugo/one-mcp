package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Helper function to initialize the database for tests
func setupTestDB(t *testing.T) func() {
	originalSQLitePath := common.SQLitePath
	testDBPath := "./test_auth_handler.db" // Use a specific name for this test suite
	common.SQLitePath = testDBPath
	_ = os.Remove(testDBPath) // Clean up previous test DB if it exists

	// InitDB will configure thing and run migrations (including AutoMigrate for all models)
	err := model.InitDB()
	assert.NoError(t, err, "model.InitDB() failed during test setup")

	// Teardown function to clean up
	return func() {
		// Attempt to close DB if model.CloseDB() is available and needed
		// if model.CloseDB != nil { model.CloseDB() }
		_ = os.Remove(testDBPath)
		common.SQLitePath = originalSQLitePath // Restore original path
	}
}

func TestLogin_RootUser_CorrectCredentials(t *testing.T) {
	teardown := setupTestDB(t)
	defer teardown()

	// Explicitly create/ensure root user for this test for determinism,
	// even if InitDB might create one under certain conditions.
	hashedPassword, errHash := common.Password2Hash("123456")
	assert.NoError(t, errHash, "Failed to hash password")

	rootUser := &model.User{
		Username:    "root",
		Password:    hashedPassword,
		Role:        common.RoleRootUser,      // Corrected to common.RoleRootUser
		Status:      common.UserStatusEnabled, // Corrected to common.UserStatusEnabled (was already correct in model)
		Email:       "root@example.com",
		DisplayName: "Test Root",
	}
	// Ensure no other user with this username exists from a previous bad state or InitDB itself
	users, _ := model.UserDB.Where("username = ?", "root").Fetch(0, 1) // Corrected to Fetch
	if len(users) > 0 {
		existingUser := users[0]
		errDel := model.UserDB.Delete(existingUser)
		assert.NoError(t, errDel, "Failed to delete pre-existing root user")
	}
	errSave := model.UserDB.Save(rootUser)
	assert.NoError(t, errSave, "Failed to create root user for test")

	loginPayload := LoginRequest{
		Username: "root",
		Password: "123456",
	}
	jsonValue, _ := json.Marshal(loginPayload)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonValue))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP status OK")

	var loginResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}

	errUnmarshal := json.Unmarshal(w.Body.Bytes(), &loginResp)
	assert.NoError(t, errUnmarshal, "Failed to unmarshal response body")
	assert.True(t, loginResp.Success, "Expected login success to be true")
	assert.NotEmpty(t, loginResp.Data.AccessToken, "Expected access token to be non-empty")
}

func TestLogin_RootUser_IncorrectCredentials(t *testing.T) {
	teardown := setupTestDB(t)
	defer teardown()

	hashedPassword, errHash := common.Password2Hash("123456")
	fmt.Println("hashedPassword", hashedPassword)
	assert.NoError(t, errHash, "Failed to hash password")

	rootUser := &model.User{
		Username:    "root",
		Password:    hashedPassword,
		Role:        common.RoleRootUser, // Corrected to common.RoleRootUser
		Status:      common.UserStatusEnabled,
		Email:       "root@example.com",
		DisplayName: "Test Root",
	}
	// Ensure no other user with this username exists
	users, _ := model.UserDB.Where("username = ?", "root").Fetch(0, 1) // Corrected to Fetch
	if len(users) > 0 {
		existingUser := users[0]
		errDel := model.UserDB.Delete(existingUser)
		assert.NoError(t, errDel, "Failed to delete pre-existing root user")
	}
	errSave := model.UserDB.Save(rootUser)
	assert.NoError(t, errSave, "Failed to create root user for test")

	loginPayload := LoginRequest{
		Username: "root",
		Password: "wrongpassword",
	}
	jsonValue, _ := json.Marshal(loginPayload)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonValue))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected HTTP status Unauthorized")

	var errorResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	errUnmarshal := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, errUnmarshal, "Failed to unmarshal error response body")
	assert.False(t, errorResp.Success, "Expected login success to be false")
	assert.Equal(t, "invalid_username_or_password", errorResp.Message, "Expected specific error message")
}

func TestLogin_NonExistentUser(t *testing.T) {
	teardown := setupTestDB(t)
	defer teardown()

	loginPayload := LoginRequest{
		Username: "nosuchuser",
		Password: "anypassword",
	}
	jsonValue, _ := json.Marshal(loginPayload)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonValue))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected HTTP status Unauthorized for non-existent user")

	var errorResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	errUnmarshal := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, errUnmarshal, "Failed to unmarshal error response body")
	assert.False(t, errorResp.Success, "Expected login success to be false")
	assert.Equal(t, "invalid_username_or_password", errorResp.Message, "Expected specific error message for non-existent user")
}

// You might need to add this to common/constants.go or a similar place if not already defined
// var DatabaseType string = "sqlite" // Example, adjust based on your actual setup

// You might need to adjust model.InitModels() or ensure it can be called multiple times / in a test context.
// For example, by making thing.Setup idempotent or providing a reset mechanism for tests.
