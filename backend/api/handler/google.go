package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"one-mcp/backend/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type GoogleOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

func getGoogleUserInfoByCode(code string) (*GoogleUser, error) {
	if code == "" {
		return nil, errors.New("无效的参数")
	}

	// Exchange code for access token
	values := map[string]string{
		"client_id":     common.GetGoogleClientId(),
		"client_secret": common.GetGoogleClientSecret(),
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  fmt.Sprintf("%s/oauth/google", common.GetServerAddress()),
	}

	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		common.SysLog(err.Error())
		return nil, errors.New("无法连接至 Google 服务器，请稍后重试！")
	}
	defer res.Body.Close()

	var oAuthResponse GoogleOAuthResponse
	err = json.NewDecoder(res.Body).Decode(&oAuthResponse)
	if err != nil {
		return nil, err
	}

	// Get user info using access token
	req, err = http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oAuthResponse.AccessToken))

	res2, err := client.Do(req)
	if err != nil {
		common.SysLog(err.Error())
		return nil, errors.New("无法连接至 Google 服务器，请稍后重试！")
	}
	defer res2.Body.Close()

	var googleUser GoogleUser
	err = json.NewDecoder(res2.Body).Decode(&googleUser)
	if err != nil {
		return nil, err
	}

	if googleUser.ID == "" {
		return nil, errors.New("返回值非法，用户字段为空，请稍后重试！")
	}

	return &googleUser, nil
}

func GoogleOAuth(c *gin.Context) {
	if !common.GetGoogleOAuthEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过 Google 登录以及注册",
		})
		return
	}

	code := c.Query("code")
	googleUser, err := getGoogleUserInfoByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	user := model.User{
		GoogleId: googleUser.ID,
	}

	if model.IsGoogleIdAlreadyTaken(user.GoogleId) {
		err := user.FillUserByGoogleId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	} else {
		if common.GetRegisterEnabled() {
			user.Username = "google_" + strconv.Itoa(int(model.GetMaxUserId()+1))
			if googleUser.Name != "" {
				user.DisplayName = googleUser.Name
			} else {
				user.DisplayName = "Google User"
			}
			user.Email = googleUser.Email
			user.Role = common.RoleCommonUser
			user.Status = common.UserStatusEnabled

			if err := user.Insert(); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}

	// Generate JWT tokens
	accessToken, err := service.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to generate access token: " + err.Error(),
		})
		return
	}

	refreshToken, err := service.GenerateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to generate refresh token: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Google OAuth login successful",
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user":          &user,
		},
	})
}

func GoogleBind(c *gin.Context) {
	if !common.GetGoogleOAuthEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过 Google 登录以及注册",
		})
		return
	}

	code := c.Query("code")
	googleUser, err := getGoogleUserInfoByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	user := model.User{
		GoogleId: googleUser.ID,
	}

	if model.IsGoogleIdAlreadyTaken(user.GoogleId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Google 账户已被绑定",
		})
		return
	}

	// Get user ID from JWT context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户未登录",
		})
		return
	}

	id, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "用户ID格式错误",
		})
		return
	}

	user.ID = id
	err = user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	user.GoogleId = googleUser.ID
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
		"message": "Google账户绑定成功",
	})
}
