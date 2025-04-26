package model

// ConfigService is the join table between UserConfig and MCPService.
// Based on the technical architecture document.
type ConfigService struct {
	Id         int `json:"id" gorm:"primaryKey"`
	ConfigId   int `json:"config_id" gorm:"uniqueIndex:idx_config_service;not null"` // Foreign key to UserConfig.Id
	ServiceId  int `json:"service_id" gorm:"uniqueIndex:idx_config_service;not null"` // Foreign key to MCPService.Id

	// You might not need the actual structs here unless you query through this table directly often.
	// UserConfig UserConfig `gorm:"foreignKey:ConfigId"`
	// MCPService MCPService `gorm:"foreignKey:ServiceId"`
}

// Optional: Define table name explicitly if needed
// func (ConfigService) TableName() string {
// 	 return "config_services"
// } 