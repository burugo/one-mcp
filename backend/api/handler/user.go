package handler

import (
	"encoding/json"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"strconv"

	"one-mcp/backend/common/i18n"

	"github.com/burugo/thing"
	"github.com/gin-gonic/gin"
)

func GetAllUsers(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	users, err := model.GetAllUsers(p*common.ItemsPerPage, common.ItemsPerPage)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
	return
}

func SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	users, err := model.SearchUsers(keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
	return
}

func GetUser(c *gin.Context) {
	lang := c.GetString("lang")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user, err := model.GetUserById(int64(id), false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	myRole := c.GetInt("role")
	if myRole <= user.Role {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("no_permission_get_same_or_higher_user", lang),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
	return
}

func GenerateToken(c *gin.Context) {
	lang := c.GetString("lang")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type",
		})
		return
	}
	user, err := model.GetUserById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.Token = model.GenerateUserToken()

	if err := user.Update(false); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user.Token,
	})
	return
}

func GetSelf(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type",
		})
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
	return
}

// UserUpdateRequestPayload defines the structure for an update user request.
// It includes fields that can be updated by an admin, especially Password.
type UserUpdateRequestPayload struct {
	ID          uint    `json:"id" validate:"required"`
	Username    string  `json:"username" validate:"required,min=3,max=20"`
	DisplayName string  `json:"display_name" validate:"required,max=50"`
	Email       *string `json:"email" validate:"omitempty,email"`    // Pointer type to distinguish "not provided" from "empty string"
	Role        *int    `json:"role" validate:"omitempty,gte=0"`     // Pointer type to distinguish "not provided" from "0"
	Password    string  `json:"password" validate:"omitempty,min=6"` // Keep as string, empty means "don't change"
}

func UpdateUser(c *gin.Context) {
	lang := c.GetString("lang")
	var requestPayload UserUpdateRequestPayload

	if err := c.ShouldBindJSON(&requestPayload); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("invalid_param", lang) + ": " + err.Error(),
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	myID, ok := currentUserID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type in context",
		})
		return
	}

	actualRequestPassword := requestPayload.Password

	originUser, err := model.GetUserById(int64(requestPayload.ID), false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	myRole := c.GetInt("role")

	if myRole <= originUser.Role && int64(originUser.ID) != myID {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("no_permission_update_same_or_higher_user", lang),
		})
		return
	}

	// Role permission checks - only apply if role is explicitly provided in the request
	if requestPayload.Role != nil {
		requestedRole := *requestPayload.Role // Dereference the pointer to get the actual role value

		// Only proceed with role checks if the role is actually different from current
		if requestedRole != originUser.Role {
			if int64(originUser.ID) == myID { // Trying to change own role
				if requestedRole > myRole {
					c.JSON(http.StatusOK, gin.H{
						"success": false,
						"message": i18n.Translate("no_permission_promote_self_to_higher_role", lang),
					})
					return
				}
			} else { // Trying to change another user's role
				// Cannot set another user's role to be higher than my own role, unless I am root.
				if requestedRole > myRole && myRole != common.RoleRootUser {
					c.JSON(http.StatusOK, gin.H{
						"success": false,
						"message": i18n.Translate("no_permission_promote_other_to_higher_role", lang),
					})
					return
				}
			}
			// General check: if not root, cannot assign any role (to self or other) that is greater than myRole.
			if requestedRole > myRole && myRole != common.RoleRootUser {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": i18n.Translate("no_permission_set_role_higher_than_own", lang),
				})
				return
			}
		}
	}

	// Apply changes to the original user object
	originUser.Username = requestPayload.Username
	originUser.DisplayName = requestPayload.DisplayName

	// Only update email if explicitly provided in the request
	if requestPayload.Email != nil {
		originUser.Email = *requestPayload.Email // Dereference pointer to get actual email value
	}

	// Only update role if explicitly provided in the request and different from current
	if requestPayload.Role != nil && *requestPayload.Role != originUser.Role {
		originUser.Role = *requestPayload.Role // Dereference pointer to get actual role value
	}

	updatePassword := actualRequestPassword != ""
	if updatePassword {
		originUser.Password = actualRequestPassword
	}

	if err := originUser.Update(updatePassword); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateSelf(c *gin.Context) {
	lang := c.GetString("lang")
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("invalid_param", lang),
		})
		return
	}
	if user.Password == "" {
		user.Password = "$I_LOVE_U" // make Validator happy :)
	}
	if err := common.Validate.Struct(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("invalid_input", lang) + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type",
		})
		return
	}

	// 获取当前用户完整信息，避免覆盖其他字段
	currentUser, err := model.GetUserById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 只更新允许修改的字段
	currentUser.Username = user.Username
	currentUser.DisplayName = user.DisplayName
	currentUser.Email = user.Email

	updatePassword := false
	if user.Password != "" && user.Password != "$I_LOVE_U" {
		currentUser.Password = user.Password
		updatePassword = true
	}

	if err := currentUser.Update(updatePassword); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteUser(c *gin.Context) {
	lang := c.GetString("lang")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	originUser, err := model.GetUserById(int64(id), false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	myRole := c.GetInt("role")
	if myRole <= originUser.Role {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("no_permission_delete_same_or_higher_user", lang),
		})
		return
	}
	err = model.DeleteUserById(int64(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	// 删除成功，返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

// ChangePasswordRequest represents the request for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// ChangePassword handles password change for the current user
func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	userId, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type",
		})
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 验证当前密码
	if !common.ValidatePasswordAndHash(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "current_password_incorrect",
		})
		return
	}

	// 检查是否为 OAuth 用户（不应该能够修改密码）
	if user.GitHubId != "" || user.GoogleId != "" || user.WeChatId != "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "oauth_user_cannot_change_password",
		})
		return
	}

	// 更新密码
	user.Password = req.NewPassword
	if err := user.Update(true); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "password_changed_successfully",
	})
	return
}

func DeleteSelf(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type",
		})
		return
	}
	err := model.DeleteUserById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

// UserCreateRequestPayload defines the structure for a create user request.
// It includes all fields that can be set when creating a new user, especially Password.
type UserCreateRequestPayload struct {
	Username    string `json:"username" validate:"required,min=3,max=20"`
	Password    string `json:"password" validate:"required,min=6"`
	DisplayName string `json:"display_name" validate:"omitempty,max=50"`
	Email       string `json:"email" validate:"omitempty,email"`
	Role        int    `json:"role" validate:"omitempty,gte=0"`
}

func CreateUser(c *gin.Context) {
	lang := c.GetString("lang")
	var requestPayload UserCreateRequestPayload

	if err := c.ShouldBindJSON(&requestPayload); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("invalid_param", lang) + ": " + err.Error(),
		})
		return
	}

	if requestPayload.DisplayName == "" {
		requestPayload.DisplayName = requestPayload.Username
	}

	myRole := c.GetInt("role")
	if requestPayload.Role >= myRole {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("cannot_create_user_with_higher_or_equal_role", lang),
		})
		return
	}

	// Set default role to common user if not provided
	userRole := requestPayload.Role
	if userRole == 0 {
		userRole = common.RoleCommonUser
	}

	// Create a clean user object with the parsed data
	cleanUser := model.User{
		Username:    requestPayload.Username,
		Password:    requestPayload.Password,
		DisplayName: requestPayload.DisplayName,
		Email:       requestPayload.Email,
		Role:        userRole,
	}

	if err := cleanUser.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

type ManageRequest struct {
	Username string `json:"username"`
	Action   string `json:"action"`
}

// ManageUser Only admin user can do this
func ManageUser(c *gin.Context) {
	lang := c.GetString("lang")
	var req ManageRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("invalid_param", lang),
		})
		return
	}

	// Use specialized function for admin operations (not affected by user status)
	user, err := model.GetUserByUsernameForAdmin(req.Username)
	if err != nil {
		if err.Error() == "user_not_found" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("user_not_found", lang),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": err.Error(),
			})
		}
		return
	}

	myRole := c.GetInt("role")
	if myRole <= user.Role && myRole != common.RoleRootUser {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("no_permission_update_same_or_higher_user", lang),
		})
		return
	}
	switch req.Action {
	case "disable":
		user.Status = common.UserStatusDisabled
		if user.Role == common.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("cannot_disable_root_user", lang),
			})
			return
		}
	case "enable":
		user.Status = common.UserStatusEnabled
	case "delete":
		if user.Role == common.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("cannot_delete_root_user", lang),
			})
			return
		}
		if err := user.Delete(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "promote":
		if myRole != common.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("admin_cannot_promote_to_admin", lang),
			})
			return
		}
		if user.Role >= common.RoleAdminUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("user_already_admin", lang),
			})
			return
		}
		user.Role = common.RoleAdminUser
	case "demote":
		if user.Role == common.RoleRootUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("cannot_demote_root_user", lang),
			})
			return
		}
		if user.Role == common.RoleCommonUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.Translate("user_already_common", lang),
			})
			return
		}
		user.Role = common.RoleCommonUser
	}

	// Only save if action wasn't delete
	if req.Action != "delete" {
		if err := user.Update(false); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	clearUser := model.User{
		BaseModel: thing.BaseModel{ID: user.ID}, // Use found user's ID
		Role:      user.Role,
		Status:    user.Status,
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    clearUser,
	})
	return
}

func EmailBind(c *gin.Context) {
	lang := c.GetString("lang")
	email := c.Query("email")
	code := c.Query("code")
	if !common.VerifyCodeWithKey(email, code, common.EmailVerificationPurpose) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate("invalid_or_expired_code", lang),
		})
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "user_id not found in context",
		})
		return
	}
	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid user_id type",
		})
		return
	}
	user := model.User{
		BaseModel: thing.BaseModel{ID: id},
	}
	err := user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.Email = email
	// no need to check if this email already taken, because we have used verification code to check it
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}
