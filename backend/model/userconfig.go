package model

import "time"

// UserConfig represents a named configuration combination created by a user.
// Based on the technical architecture document.
type UserConfig struct {
	Id          int       `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Description string    `json:"description" gorm:"size:255"`
	UserId      int       `json:"user_id" gorm:"index;not null"` // Foreign key to User.Id
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relation (optional, depending on how you query)
	// Services []*MCPService `json:"services,omitempty" gorm:"many2many:config_services;"`
} 