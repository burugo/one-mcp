package model

// File represents a file record in the database
type File struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id"`
	Filename  string `json:"filename"`
	Link      string `json:"link"`
	CreatedAt int64  `json:"created_at"`
}
