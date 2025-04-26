package model

import (
	_ "gorm.io/driver/sqlite" // Keep underscore import if needed by gorm
)

// File represents a file stored in the system
type File struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"index"`
	Filename  string `json:"filename" gorm:"index;size:255"`
	Link      string `json:"link" gorm:"uniqueIndex;size:100"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

// --- Helper functions moved to service/file_service.go ---
