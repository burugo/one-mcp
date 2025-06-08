package handler

import (
	"encoding/json"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"one-mcp/backend/service"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	options, err := model.OptionDB.All()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
	return
}

func UpdateOption(c *gin.Context) {
	var option model.Option
	err := json.NewDecoder(c.Request.Body).Decode(&option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GetGitHubClientId() == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 GitHub OAuth，请先填入 GitHub Client ID 以及 GitHub Client Secret！",
			})
			return
		}
	case "GoogleOAuthEnabled":
		if option.Value == "true" && common.GetGoogleClientId() == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Google OAuth，请先填入 Google Client ID 以及 Google Client Secret！",
			})
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.GetWeChatServerAddress() == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用微信登录，请先填入微信登录相关配置信息！",
			})
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.GetTurnstileSiteKey() == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！",
			})
			return
		}
	}
	err = service.UpdateOption(option.Key, option.Value)
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
