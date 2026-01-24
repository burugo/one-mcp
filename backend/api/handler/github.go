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

type GitHubOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type GitHubUser struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func getGitHubUserInfoByCode(code string) (*GitHubUser, error) {
	if code == "" {
		return nil, errors.New("无效的参数")
	}
	values := map[string]string{"client_id": common.GetGitHubClientId(), "client_secret": common.GetGitHubClientSecret(), "code": code}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(jsonData))
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
		return nil, errors.New("无法连接至 GitHub 服务器，请稍后重试！")
	}
	defer res.Body.Close()
	var oAuthResponse GitHubOAuthResponse
	err = json.NewDecoder(res.Body).Decode(&oAuthResponse)
	if err != nil {
		return nil, err
	}
	req, err = http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oAuthResponse.AccessToken))
	res2, err := client.Do(req)
	if err != nil {
		common.SysLog(err.Error())
		return nil, errors.New("无法连接至 GitHub 服务器，请稍后重试！")
	}
	defer res2.Body.Close()
	var githubUser GitHubUser
	err = json.NewDecoder(res2.Body).Decode(&githubUser)
	if err != nil {
		return nil, err
	}
	if githubUser.Login == "" {
		return nil, errors.New("返回值非法，用户字段为空，请稍后重试！")
	}
	return &githubUser, nil
}

func GitHubOAuth(c *gin.Context) {
	// Check if user is already logged in via JWT
	if userID, exists := c.Get("user_id"); exists && userID != nil {
		GitHubBind(c)
		return
	}

	if !common.GetGitHubOAuthEnabled() {
		common.RespErrorStr(c, http.StatusOK, "管理员未开启通过 GitHub 登录以及注册")
		return
	}
	code := c.Query("code")
	githubUser, err := getGitHubUserInfoByCode(code)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	user := model.User{
		GitHubId: githubUser.Login,
	}
	if model.IsGitHubIdAlreadyTaken(user.GitHubId) {
		err := user.FillUserByGitHubId()
		if err != nil {
			common.RespErrorStr(c, http.StatusOK, err.Error())
			return
		}
	} else {
		if common.GetRegisterEnabled() {
			user.Username = "github_" + strconv.Itoa(int(model.GetMaxUserId()+1))
			if githubUser.Name != "" {
				user.DisplayName = githubUser.Name
			} else {
				user.DisplayName = "GitHub User"
			}
			user.Email = githubUser.Email
			user.Role = common.RoleCommonUser
			user.Status = common.UserStatusEnabled

			if err := user.Insert(); err != nil {
				common.RespErrorStr(c, http.StatusOK, err.Error())
				return
			}
		} else {
			common.RespErrorStr(c, http.StatusOK, "管理员关闭了新用户注册")
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		common.RespErrorStr(c, http.StatusOK, "用户已被封禁")
		return
	}
	// Generate JWT tokens
	accessToken, err := service.GenerateToken(&user)
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to generate access token: "+err.Error())
		return
	}

	refreshToken, err := service.GenerateRefreshToken(&user)
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, "Failed to generate refresh token: "+err.Error())
		return
	}

	common.RespSuccessWithMsg(c, "GitHub OAuth login successful", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          &user,
	})
}

func GitHubBind(c *gin.Context) {
	if !common.GetGitHubOAuthEnabled() {
		common.RespErrorStr(c, http.StatusOK, "管理员未开启通过 GitHub 登录以及注册")
		return
	}
	code := c.Query("code")
	githubUser, err := getGitHubUserInfoByCode(code)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	user := model.User{
		GitHubId: githubUser.Login,
	}
	if model.IsGitHubIdAlreadyTaken(user.GitHubId) {
		common.RespErrorStr(c, http.StatusOK, "该 GitHub 账户已被绑定")
		return
	}
	// Get user ID from JWT context
	userID, exists := c.Get("user_id")
	if !exists {
		common.RespErrorStr(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	id, ok := userID.(int64)
	if !ok {
		common.RespErrorStr(c, http.StatusInternalServerError, "用户ID格式错误")
		return
	}

	user.ID = id
	err = user.FillUserById()
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	user.GitHubId = githubUser.Login
	err = user.Update(false)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	common.RespSuccessStr(c, "GitHub账户绑定成功")
}
