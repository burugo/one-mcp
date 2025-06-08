package common

// GetGitHubClientId 获取GitHub客户端ID
func GetGitHubClientId() string {
	return OptionMap["GitHubClientId"]
}

// GetGitHubClientSecret 获取GitHub客户端密钥
func GetGitHubClientSecret() string {
	return OptionMap["GitHubClientSecret"]
}

// GetGitHubOAuthEnabled 获取GitHub OAuth是否启用
func GetGitHubOAuthEnabled() bool {
	return OptionMap["GitHubOAuthEnabled"] == "true"
}

// GetGoogleClientId 获取Google客户端ID
func GetGoogleClientId() string {
	return OptionMap["GoogleClientId"]
}

// GetGoogleClientSecret 获取Google客户端密钥
func GetGoogleClientSecret() string {
	return OptionMap["GoogleClientSecret"]
}

// GetGoogleOAuthEnabled 获取Google OAuth是否启用
func GetGoogleOAuthEnabled() bool {
	return OptionMap["GoogleOAuthEnabled"] == "true"
}

// GetServerAddress 获取服务器地址
func GetServerAddress() string {
	return OptionMap["ServerAddress"]
}

// GetSystemName 获取系统名称
func GetSystemName() string {
	return OptionMap["SystemName"]
}

// GetFooter 获取页脚信息
func GetFooter() string {
	return OptionMap["Footer"]
}

// GetHomePageLink 获取首页链接
func GetHomePageLink() string {
	return OptionMap["HomePageLink"]
}

// GetRegisterEnabled 获取注册是否启用
func GetRegisterEnabled() bool {
	return OptionMap["RegisterEnabled"] == "true"
}

// GetEmailVerificationEnabled 获取邮箱验证是否启用
func GetEmailVerificationEnabled() bool {
	return OptionMap["EmailVerificationEnabled"] == "true"
}

// GetWeChatAuthEnabled 获取微信认证是否启用
func GetWeChatAuthEnabled() bool {
	return OptionMap["WeChatAuthEnabled"] == "true"
}

// GetWeChatServerAddress 获取微信服务器地址
func GetWeChatServerAddress() string {
	return OptionMap["WeChatServerAddress"]
}

// GetWeChatServerToken 获取微信服务器令牌
func GetWeChatServerToken() string {
	return OptionMap["WeChatServerToken"]
}

// GetWeChatAccountQRCodeImageURL 获取微信账号二维码图片URL
func GetWeChatAccountQRCodeImageURL() string {
	return OptionMap["WeChatAccountQRCodeImageURL"]
}

// GetTurnstileCheckEnabled 获取Turnstile检查是否启用
func GetTurnstileCheckEnabled() bool {
	return OptionMap["TurnstileCheckEnabled"] == "true"
}

// GetTurnstileSiteKey 获取Turnstile站点密钥
func GetTurnstileSiteKey() string {
	return OptionMap["TurnstileSiteKey"]
}

// GetTurnstileSecretKey 获取Turnstile秘密密钥
func GetTurnstileSecretKey() string {
	return OptionMap["TurnstileSecretKey"]
}

// GetSMTPServer 获取SMTP服务器
func GetSMTPServer() string {
	return OptionMap["SMTPServer"]
}

// GetSMTPAccount 获取SMTP账号
func GetSMTPAccount() string {
	return OptionMap["SMTPAccount"]
}

// GetSMTPToken 获取SMTP令牌
func GetSMTPToken() string {
	return OptionMap["SMTPToken"]
}
