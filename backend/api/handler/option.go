package handler

import (
	"encoding/json"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"one-mcp/backend/service"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	options, err := model.OptionDB.All()
	if err != nil {
		common.RespErrorStr(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.RespSuccess(c, options)
	return
}

func UpdateOption(c *gin.Context) {
	var option model.Option
	err := json.NewDecoder(c.Request.Body).Decode(&option)
	if err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, "无效的参数")
		return
	}
	switch option.Key {
	case "ServerAddress":
		proxy.ClearSSEProxyCache()
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GetGitHubClientId() == "" {
			common.RespErrorStr(c, http.StatusOK, "无法启用 GitHub OAuth，请先填入 GitHub Client ID 以及 GitHub Client Secret！")
			return
		}
	case "GoogleOAuthEnabled":
		if option.Value == "true" && common.GetGoogleClientId() == "" {
			common.RespErrorStr(c, http.StatusOK, "无法启用 Google OAuth，请先填入 Google Client ID 以及 Google Client Secret！")
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.GetWeChatServerAddress() == "" {
			common.RespErrorStr(c, http.StatusOK, "无法启用微信登录，请先填入微信登录相关配置信息！")
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.GetTurnstileSiteKey() == "" {
			common.RespErrorStr(c, http.StatusOK, "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！")
			return
		}
	case common.OptionStdioServiceStartupStrategy:
		if option.Value != common.StrategyStartOnBoot && option.Value != common.StrategyStartOnDemand {
			common.RespErrorStr(c, http.StatusBadRequest, "Invalid startup strategy, only 'boot' or 'demand' are supported")
			return
		}
	}
	err = service.UpdateOption(option.Key, option.Value)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	common.RespSuccessStr(c, "")
	return
}
