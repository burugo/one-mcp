package handlers

import (
	"encoding/json"
	"one-cmp/backend/internal/common"
	"one-cmp/backend/internal/library/db"
	"one-cmp/backend/internal/domain/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	if !common.PasswordLoginEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "管理员关闭了密码登录",
			"success": false,
		})
		return
	}
	var loginRequest LoginRequest
	err := json.NewDecoder(c.Request.Body).Decode(&loginRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数: " + err.Error()})
		return
	}
	username := loginRequest.Username
	password := loginRequest.Password
	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户名或密码无效"})
		return
	}
	user := model.User{
		Username: username,
		Password: password,
	}
	foundUser := model.User{}
	if err := db.DB.Where("username = ?", username).First(&foundUser).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "用户名或密码错误"})
		return
	}
	if ok := common.ValidatePasswordAndHash(password, foundUser.Password); !ok || foundUser.Status != model.UserStatusEnabled {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "用户名或密码错误，或用户已被封禁"})
		return
	}

	setupLogin(&foundUser, c)
}

// setup session & cookies and then return user info
func setupLogin(user *model.User, c *gin.Context) {
	session := sessions.Default(c)
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法保存会话信息: " + err.Error(), "success": false})
		return
	}
	cleanUser := model.User{
		Id:          user.Id,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Status:      user.Status,
		Email:       user.Email,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "",
		"success": true,
		"data":    cleanUser,
	})
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error(), "success": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "", "success": true})
}

func Register(c *gin.Context) {
	if !common.RegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"message": "管理员关闭了新用户注册", "success": false})
		return
	}
	if !common.PasswordRegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"message": "管理员关闭了通过密码进行注册", "success": false})
		return
	}
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数: " + err.Error()})
		return
	}
	if user.Username == "" || len(user.Username) > 12 || user.Password == "" || len(user.Password) < 8 || len(user.Password) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户名(<=12)或密码(8-20)格式不正确"})
		return
	}

	var existingUser int64
	db.DB.Model(&model.User{}).Where("username = ?", user.Username).Count(&existingUser)
	if existingUser > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户名已被占用"})
		return
	}

	if common.EmailVerificationEnabled {
		var reqBody map[string]string
		if err := json.Unmarshal([]byte{}, &reqBody); err != nil {
		}
		verificationCode := reqBody["verification_code"]
		if user.Email == "" || verificationCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "管理员开启了邮箱验证，请输入邮箱地址和验证码"})
			return
		}
		if !common.VerifyCodeWithKey(user.Email, verificationCode, common.EmailVerificationPurpose) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "验证码错误或已过期"})
			return
		}
		db.DB.Model(&model.User{}).Where("email = ?", user.Email).Count(&existingUser)
		if existingUser > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "邮箱地址已被占用"})
			return
		}
	}

	hashedPassword, hashErr := common.Password2Hash(user.Password)
	if hashErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "密码加密失败"})
		return
	}

	newUser := model.User{
		Username:    user.Username,
		Password:    hashedPassword,
		DisplayName: user.Username,
		Role:        model.RoleCommonUser,
		Status:      model.UserStatusEnabled,
	}
	if common.EmailVerificationEnabled {
		newUser.Email = user.Email
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "用户创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func GetAllUsers(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	var users []*model.User
	err := db.DB.Omit("password").Order("id desc").Limit(common.ItemsPerPage).Offset(p * common.ItemsPerPage).Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
}

func SearchUsers(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	keyword := c.Query("keyword")
	var users []*model.User
	likeKeyword := keyword + "%"
	err := db.DB.Omit("password").Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", likeKeyword, likeKeyword, likeKeyword).Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    users,
	})
}

func GetUser(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的用户 ID"})
		return
	}
	var user model.User
	err = db.DB.Omit("password").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
}

func GenerateToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "此功能已弃用",
	})
}

func GetSelf(c *gin.Context) {
	id := c.GetInt("id")
	var user model.User
	err := db.DB.Omit("password").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    user,
	})
}

func UpdateUser(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	var updatedUser model.User
	err := json.NewDecoder(c.Request.Body).Decode(&updatedUser)
	if err != nil || updatedUser.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数或用户 ID"})
		return
	}

	var originUser model.User
	if err := db.DB.First(&originUser, updatedUser.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		}
		return
	}

	originUser.DisplayName = updatedUser.DisplayName
	originUser.Role = updatedUser.Role
	originUser.Status = updatedUser.Status
	originUser.Email = updatedUser.Email

	if updatedUser.Password != "" {
		hashedPassword, hashErr := common.Password2Hash(updatedUser.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "密码加密失败"})
			return
		}
		originUser.Password = hashedPassword
	}

	if err := db.DB.Save(&originUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    originUser.Id,
	})
}

func UpdateSelf(c *gin.Context) {
	var userUpdates model.User
	err := json.NewDecoder(c.Request.Body).Decode(&userUpdates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数: " + err.Error()})
		return
	}

	id := c.GetInt("id")
	var currentUser model.User
	if err := db.DB.First(&currentUser, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		}
		return
	}

	updatePassword := false
	currentUser.DisplayName = userUpdates.DisplayName
	currentUser.Email = userUpdates.Email

	if userUpdates.Password != "" {
		hashedPassword, hashErr := common.Password2Hash(userUpdates.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "密码加密失败"})
			return
		}
		currentUser.Password = hashedPassword
		updatePassword = true
	}

	if err := db.DB.Save(&currentUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    currentUser.Id,
	})
}

func DeleteUser(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的用户 ID"})
		return
	}
	if err := db.DB.Delete(&model.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteSelf(c *gin.Context) {
	id := c.GetInt("id")
	if err := db.DB.Delete(&model.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func CreateUser(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	var user model.User
	err := json.NewDecoder(c.Request.Body).Decode(&user)
	if err != nil || user.Username == "" || user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}

	if len(user.Username) > 12 || len(user.Password) < 8 || len(user.Password) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户名(<=12)或密码(8-20)格式不正确"})
		return
	}

	var existingUser int64
	db.DB.Model(&model.User{}).Where("username = ?", user.Username).Count(&existingUser)
	if existingUser > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户名已被占用"})
		return
	}
	if user.Email != "" {
		db.DB.Model(&model.User{}).Where("email = ?", user.Email).Count(&existingUser)
		if existingUser > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "邮箱地址已被占用"})
			return
		}
	}

	hashedPassword, hashErr := common.Password2Hash(user.Password)
	if hashErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "密码加密失败"})
		return
	}

	newUser := model.User{
		Username:    user.Username,
		Password:    hashedPassword,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Status:      user.Status,
		Email:       user.Email,
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "用户创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    newUser.Id,
	})
}

type ManageRequest struct {
	Username string `json:"username"`
	Action   string `json:"action"`
}

func ManageUser(c *gin.Context) {
	requesterRole := c.GetInt("role")
	if requesterRole < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权执行此操作"})
		return
	}
	var req ManageRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数: " + err.Error()})
		return
	}

	var user model.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		}
		return
	}

	if requesterRole <= user.Role {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无法管理同级或更高级别的用户"})
		return
	}

	var updatedData map[string]interface{}
	processed := false
	switch req.Action {
	case "promote":
		if user.Role == model.RoleAdminUser {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户已经是管理员"})
			return
		}
		updatedData = map[string]interface{}{"role": model.RoleAdminUser}
		processed = true
	case "demote":
		if user.Role == model.RoleCommonUser {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户已经是普通用户"})
			return
		}
		updatedData = map[string]interface{}{"role": model.RoleCommonUser}
		processed = true
	case "enable":
		if user.Status == model.UserStatusEnabled {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户已经是启用状态"})
			return
		}
		updatedData = map[string]interface{}{"status": model.UserStatusEnabled}
		processed = true
	case "disable":
		if user.Status == model.UserStatusDisabled {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户已经是禁用状态"})
			return
		}
		updatedData = map[string]interface{}{"status": model.UserStatusDisabled}
		processed = true
	}

	if processed {
		if err := db.DB.Model(&user).Updates(updatedData).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "不支持的操作"})
	}
}

func EmailBind(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数: " + err.Error()})
		return
	}
	if req.Email == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "邮箱地址或验证码无效"})
		return
	}
	if !common.VerifyCodeWithKey(req.Email, req.Code, common.EmailVerificationPurpose) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "验证码错误或已过期"})
		return
	}
	var existingUser int64
	db.DB.Model(&model.User{}).Where("email = ?", req.Email).Count(&existingUser)
	if existingUser > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "邮箱地址已被占用"})
		return
	}

	id := c.GetInt("id")
	var currentUser model.User
	if err := db.DB.First(&currentUser, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		}
		return
	}

	currentUser.Email = req.Email
	if err := db.DB.Save(&currentUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
