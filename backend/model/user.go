package model

import "time"

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
// Adapted from gin-template and example. Removed Token field (using JWT).
// Sensitive fields like Password should not be included in API responses.
type User struct {
	Id          int       `json:"id" gorm:"primaryKey"`
	Username    string    `json:"username" gorm:"uniqueIndex;size:12"` // Added size constraint from example
	Password    string    `json:"-" gorm:"size:100;not null"`          // json:"-" to prevent sending it out, increased size
	DisplayName string    `json:"display_name" gorm:"index;size:20"`   // Added size constraint
	Role        int       `json:"role" gorm:"type:int;default:1"`      // Simplified Role: RoleCommonUser, RoleAdminUser
	Status      int       `json:"status" gorm:"type:int;default:1"`    // Status: UserStatusEnabled, UserStatusDisabled
	Email       string    `json:"email" gorm:"index;size:50"`          // Added size constraint
	GitHubId    string    `json:"-" gorm:"column:github_id;index"`     // Assuming sensitive or internal, hiding from json
	WeChatId    string    `json:"-" gorm:"column:wechat_id;index"`     // Assuming sensitive or internal, hiding from json
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

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