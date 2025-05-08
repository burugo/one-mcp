package model

import (
	"errors"
	"one-mcp/backend/common"
	mcperrors "one-mcp/backend/common/errors"
	"one-mcp/backend/common/i18n"

	"github.com/burugo/thing"
)

// Role constants
const (
	RoleGuestUser  = 0 // Or maybe remove if unused? For now, keep consistency with example.
	RoleCommonUser = 1
	RoleAdminUser  = 10
	// RoleRootUser   = 100 // Consider if needed, maybe just Admin is enough initially.
)

// Status constants
const (
	UserStatusPending  = 0 // Default, maybe needs verification?
	UserStatusEnabled  = 1
	UserStatusDisabled = 2
)

// User represents the user model in the database.
// Adapted from one-mcp/backend and example. Removed Token field (using JWT).
// Sensitive fields like Password should not be included in API responses.
type User struct {
	thing.BaseModel
	Username         string `json:"username" gorm:"uniqueIndex;size:12"`
	Password         string `json:"-" gorm:"size:100;not null"`
	DisplayName      string `json:"display_name" gorm:"index;size:20"`
	Role             int    `json:"role" gorm:"type:int;default:1"`
	Status           int    `json:"status" gorm:"type:int;default:1"`
	Email            string `json:"email" gorm:"index;size:50"`
	GitHubId         string `json:"-" gorm:"column:github_id;index"`
	WeChatId         string `json:"-" gorm:"column:wechat_id;index"`
	VerificationCode string `json:"verification_code" gorm:"-:all"`
	Token            string `json:"token" gorm:"index"`

	// Fields from example, consider if needed later:
	// LarkId           string `json:"lark_id" gorm:"column:lark_id;index"`
	// OidcId           string `json:"oidc_id" gorm:"column:oidc_id;index"`
	// Quota            int64  `json:"quota" gorm:"bigint;default:0"`
	// UsedQuota        int64  `json:"used_quota" gorm:"bigint;default:0;column:used_quota"` // used quota
	// RequestCount     int    `json:"request_count" gorm:"type:int;default:0;"`             // request number
	// Group            string `json:"group" gorm:"type:varchar(32);default:'default'"`
	// AffCode          string `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	// InviterId        int    `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
}

var UserDB *thing.Thing[*User]

// UserInit 用于在 InitDB 时初始化 UserDB
func UserInit() error {
	var err error
	UserDB, err = thing.Use[*User]()
	if err != nil {
		return err
	}
	return nil
}

func GetMaxUserId() int64 {
	users, err := UserDB.Order("id DESC").Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return 0
	}
	return users[0].ID
}

func GetAllUsers(startIdx int, num int) ([]*User, error) {
	return UserDB.Order("id DESC").Fetch(startIdx, num)
}

func SearchUsers(keyword string) ([]*User, error) {
	return UserDB.Where(
		"id = ? OR username LIKE ? OR email LIKE ? OR display_name LIKE ?",
		keyword, keyword+"%", keyword+"%", keyword+"%",
	).Order("id DESC").Fetch(0, 100)
}

// GetUserById 根据ID获取用户
// 添加lang参数支持i18n错误消息
func GetUserById(id int64, selectAll bool, lang string) (*User, error) {
	if id == 0 {
		return nil, i18n.New(mcperrors.ErrEmptyID, lang)
	}
	user, err := UserDB.ByID(id)
	if err != nil {
		return nil, i18n.Wrap(err, mcperrors.ErrUserNotFound, lang)
	}
	return user, nil
}

// DeleteUserById 根据ID删除用户
// 添加lang参数支持i18n错误消息
func DeleteUserById(id int64, lang string) error {
	if id == 0 {
		return i18n.New(mcperrors.ErrEmptyID, lang)
	}
	user, err := UserDB.ByID(id)
	if err != nil {
		return i18n.Wrap(err, mcperrors.ErrUserNotFound, lang)
	}
	return UserDB.Delete(user)
}

func (user *User) Insert() error {
	if user.Password != "" {
		var err error
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	return UserDB.Save(user)
}

func (user *User) Update(updatePassword bool) error {
	if updatePassword {
		var err error
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	return UserDB.Save(user)
}

func (user *User) Delete() error {
	if user.ID == 0 {
		return errors.New("id 为空！")
	}
	return UserDB.Delete(user)
}

func (user *User) ValidateAndFill() error {
	if user.Username == "" || user.Password == "" {
		return errors.New("用户名或密码为空")
	}
	users, err := UserDB.Where("username = ?", user.Username).Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return errors.New("用户名或密码错误，或用户已被封禁")
	}
	found := users[0]
	okay := common.ValidatePasswordAndHash(user.Password, found.Password)
	if !okay || found.Status != common.UserStatusEnabled {
		return errors.New("用户名或密码错误，或用户已被封禁")
	}
	*user = *found
	return nil
}

func (user *User) FillUserById() error {
	if user.ID == 0 {
		return errors.New("id 为空！")
	}
	found, err := UserDB.ByID(user.ID)
	if err != nil {
		return err
	}
	*user = *found
	return nil
}

func (user *User) FillUserByEmail() error {
	if user.Email == "" {
		return errors.New("email 为空！")
	}
	users, err := UserDB.Where("email = ?", user.Email).Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return errors.New("未找到用户")
	}
	*user = *users[0]
	return nil
}

func (user *User) FillUserByGitHubId() error {
	if user.GitHubId == "" {
		return errors.New("GitHub id 为空！")
	}
	users, err := UserDB.Where("github_id = ?", user.GitHubId).Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return errors.New("未找到用户")
	}
	*user = *users[0]
	return nil
}

func (user *User) FillUserByWeChatId() error {
	if user.WeChatId == "" {
		return errors.New("WeChat id 为空！")
	}
	users, err := UserDB.Where("wechat_id = ?", user.WeChatId).Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return errors.New("未找到用户")
	}
	*user = *users[0]
	return nil
}

func (user *User) FillUserByUsername() error {
	if user.Username == "" {
		return errors.New("username 为空！")
	}
	users, err := UserDB.Where("username = ?", user.Username).Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return errors.New("未找到用户")
	}
	*user = *users[0]
	return nil
}

func ValidateUserToken(token string) *User {
	// Stub implementation - always returns nil (invalid token) for now
	// This will be replaced with proper JWT validation later
	return nil
}

func IsEmailAlreadyTaken(email string) bool {
	users, err := UserDB.Where("email = ?", email).Fetch(0, 1)
	return err == nil && len(users) > 0
}

func IsWeChatIdAlreadyTaken(wechatId string) bool {
	users, err := UserDB.Where("wechat_id = ?", wechatId).Fetch(0, 1)
	return err == nil && len(users) > 0
}

func IsGitHubIdAlreadyTaken(githubId string) bool {
	users, err := UserDB.Where("github_id = ?", githubId).Fetch(0, 1)
	return err == nil && len(users) > 0
}

func IsUsernameAlreadyTaken(username string) bool {
	users, err := UserDB.Where("username = ?", username).Fetch(0, 1)
	return err == nil && len(users) > 0
}

func ResetUserPasswordByEmail(email string, password string) error {
	if email == "" || password == "" {
		return errors.New("邮箱地址或密码为空！")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	users, err := UserDB.Where("email = ?", email).Fetch(0, 1)
	if err != nil || len(users) == 0 {
		return errors.New("未找到用户")
	}
	user := users[0]
	user.Password = hashedPassword
	return UserDB.Save(user)
}
