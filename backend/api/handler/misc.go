package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
)

func GetStatus(c *gin.Context) {
	// Get current language from context (set by LangMiddleware)
	lang := c.GetString("lang")
	if lang == "" {
		lang = "zh-CN" // Default language
	}

	common.RespSuccess(c, gin.H{
		"version":            common.Version,
		"start_time":         common.StartTime,
		"email_verification": common.GetEmailVerificationEnabled(),
		"github_oauth":       common.GetGitHubOAuthEnabled(),
		"github_client_id":   common.GetGitHubClientId(),
		"google_oauth":       common.GetGoogleOAuthEnabled(),
		"google_client_id":   common.GetGoogleClientId(),
		"system_name":        common.GetSystemName(),
		"home_page_link":     common.GetHomePageLink(),
		"footer_html":        common.GetFooter(),
		"wechat_qrcode":      common.GetWeChatAccountQRCodeImageURL(),
		"wechat_login":       common.GetWeChatAuthEnabled(),
		"server_address":     common.GetServerAddress(),
		"turnstile_check":    common.GetTurnstileCheckEnabled(),
		"turnstile_site_key": common.GetTurnstileSiteKey(),
		"current_language":   lang,
	})
	return
}

func GetNotice(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	common.RespSuccess(c, common.OptionMap["Notice"])
	return
}

func GetAbout(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	common.RespSuccess(c, common.OptionMap["About"])
	return
}

func SendEmailVerification(c *gin.Context) {
	email := c.Query("email")
	if err := common.Validate.Var(email, "required,email"); err != nil {
		common.RespErrorStr(c, http.StatusOK, "无效的参数")
		return
	}
	if model.IsEmailAlreadyTaken(email) {
		common.RespErrorStr(c, http.StatusOK, "邮箱地址已被占用")
		return
	}
	code := common.GenerateVerificationCode(6)
	common.RegisterVerificationCodeWithKey(email, code, common.EmailVerificationPurpose)
	subject := fmt.Sprintf("%s邮箱验证邮件", common.GetSystemName())
	content := fmt.Sprintf("<p>您好，你正在进行%s邮箱验证。</p>"+
		"<p>您的验证码为: <strong>%s</strong></p>"+
		"<p>验证码 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.GetSystemName(), code, common.VerificationValidMinutes)
	err := common.SendEmail(subject, email, content)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	common.RespSuccessStr(c, "")
	return
}

func SendPasswordResetEmail(c *gin.Context) {
	email := c.Query("email")
	if err := common.Validate.Var(email, "required,email"); err != nil {
		common.RespErrorStr(c, http.StatusOK, "无效的参数")
		return
	}
	if !model.IsEmailAlreadyTaken(email) {
		common.RespErrorStr(c, http.StatusOK, "该邮箱地址未注册")
		return
	}
	code := common.GenerateVerificationCode(0)
	common.RegisterVerificationCodeWithKey(email, code, common.PasswordResetPurpose)
	link := fmt.Sprintf("%s/user/reset?email=%s&token=%s", common.GetServerAddress(), email, code)
	subject := fmt.Sprintf("%s密码重置", common.GetSystemName())
	content := fmt.Sprintf("<p>您好，你正在进行%s密码重置。</p>"+
		"<p>点击<a href='%s'>此处</a>进行密码重置。</p>"+
		"<p>重置链接 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.GetSystemName(), link, common.VerificationValidMinutes)
	err := common.SendEmail(subject, email, content)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	common.RespSuccessStr(c, "")
	return
}

type PasswordResetRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func ResetPassword(c *gin.Context) {
	var req PasswordResetRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if req.Email == "" || req.Token == "" {
		common.RespErrorStr(c, http.StatusOK, "无效的参数")
		return
	}
	if !common.VerifyCodeWithKey(req.Email, req.Token, common.PasswordResetPurpose) {
		common.RespErrorStr(c, http.StatusOK, "重置链接非法或已过期")
		return
	}
	password := common.GenerateVerificationCode(12)
	err = model.ResetUserPasswordByEmail(req.Email, password)
	if err != nil {
		common.RespErrorStr(c, http.StatusOK, err.Error())
		return
	}
	common.DeleteKey(req.Email, common.PasswordResetPurpose)
	common.RespSuccess(c, password)
	return
}
