package errors

// 用户相关错误码
const (
	// 通用错误
	ErrInternalServer = "ERR_INTERNAL_SERVER"
	ErrInvalidParam   = "ERR_INVALID_PARAM"

	// 用户错误
	ErrEmptyID            = "ERR_EMPTY_ID"
	ErrUserNotFound       = "ERR_USER_NOT_FOUND"
	ErrEmptyCredentials   = "ERR_EMPTY_CREDENTIALS"
	ErrInvalidCredentials = "ERR_INVALID_CREDENTIALS"
	ErrUserDisabled       = "ERR_USER_DISABLED"
	ErrEmptyEmail         = "ERR_EMPTY_EMAIL"
	ErrEmptyUsername      = "ERR_EMPTY_USERNAME"
	ErrEmailTaken         = "ERR_EMAIL_TAKEN"
	ErrUsernameTaken      = "ERR_USERNAME_TAKEN"
	ErrEmptyPassword      = "ERR_EMPTY_PASSWORD"
	ErrGithubIDEmpty      = "ERR_GITHUB_ID_EMPTY"
	ErrWeChatIDEmpty      = "ERR_WECHAT_ID_EMPTY"
)

// 服务相关错误码
const (
	ErrServiceNotFound = "ERR_SERVICE_NOT_FOUND"
	ErrServiceDisabled = "ERR_SERVICE_DISABLED"
)

// 配置相关错误码
const (
	ErrConfigNotFound  = "ERR_CONFIG_NOT_FOUND"
	ErrConfigOwnership = "ERR_CONFIG_OWNERSHIP"
)
