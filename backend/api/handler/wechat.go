package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type wechatLoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func getWeChatIdByCode(code string) (string, error) {
	if code == "" {
		return "", errors.New("无效的参数")
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/wechat/user?code=%s", common.GetWeChatServerAddress(), code), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", common.GetWeChatServerToken())
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()
	var res wechatLoginResponse
	err = json.NewDecoder(httpResponse.Body).Decode(&res)
	if err != nil {
		return "", err
	}
	if !res.Success {
		return "", errors.New(res.Message)
	}
	if res.Data == "" {
		return "", errors.New("验证码错误或已过期")
	}
	return res.Data, nil
}

func WeChatAuth(c *gin.Context) {
	if !common.GetWeChatAuthEnabled() {
		common.RespErrorStr(c, http.StatusOK, "管理员未开启通过微信登录以及注册")
		return
	}
	code := c.Query("code")
	wechatId, err := getWeChatIdByCode(code)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	user := model.User{
		WeChatId: wechatId,
	}
	if model.IsWeChatIdAlreadyTaken(wechatId) {
		err := user.FillUserByWeChatId()
		if err != nil {
			common.RespErrorStr(c, http.StatusOK, err.Error())
			return
		}
	} else {
		if common.GetRegisterEnabled() {
			user.Username = "wechat_" + strconv.Itoa(int(model.GetMaxUserId()+1))
			user.DisplayName = "WeChat User"
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
	// setupLogin(&user, c) // TODO: implement or replace with actual login handler
}

func WeChatBind(c *gin.Context) {
	if !common.GetWeChatAuthEnabled() {
		common.RespErrorStr(c, http.StatusOK, "管理员未开启通过微信登录以及注册")
		return
	}
	code := c.Query("code")
	wechatId, err := getWeChatIdByCode(code)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	if model.IsWeChatIdAlreadyTaken(wechatId) {
		common.RespErrorStr(c, http.StatusOK, "该微信账号已被绑定")
		return
	}
	id := c.GetInt("id")
	user := model.User{}
	user.ID = int64(id)
	err = user.FillUserById()
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	user.WeChatId = wechatId
	err = user.Update(false)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	common.RespSuccessStr(c, "")
	return
}
