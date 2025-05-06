package model

import (
	"gorm.io/gorm"
	"time"
)

// UserConfig represents a user's configuration for a specific service setting
type UserConfig struct {
	Id            int           `json:"id" gorm:"primaryKey"`
	UserId        int           `json:"user_id" gorm:"not null;index:idx_user_config"`
	ServiceId     int           `json:"service_id" gorm:"not null;index:idx_user_config"`
	ConfigId      int           `json:"config_id" gorm:"not null;index:idx_user_config"`
	Value         string        `json:"value" gorm:"type:text"`
	CreatedAt     time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	User          User          `json:"-" gorm:"foreignKey:UserId"`
	Service       MCPService    `json:"-" gorm:"foreignKey:ServiceId"`
	ConfigService ConfigService `json:"-" gorm:"foreignKey:ConfigId"`
}

// TableName sets the table name for the UserConfig model
func (c *UserConfig) TableName() string {
	return "user_configs"
}

// GetUserConfigsForUser returns all config values for a specific user
func GetUserConfigsForUser(db *gorm.DB, userId int) ([]UserConfig, error) {
	var configs []UserConfig
	err := db.Where("user_id = ?", userId).Find(&configs).Error
	return configs, err
}

// GetUserConfigsForService returns all config values for a specific user and service
func GetUserConfigsForService(db *gorm.DB, userId, serviceId int) ([]UserConfig, error) {
	var configs []UserConfig
	err := db.Where("user_id = ? AND service_id = ?", userId, serviceId).Find(&configs).Error
	return configs, err
}

// GetUserConfigValue returns a specific config value for a user
func GetUserConfigValue(db *gorm.DB, userId, configId int) (UserConfig, error) {
	var config UserConfig
	err := db.Where("user_id = ? AND config_id = ?", userId, configId).First(&config).Error
	return config, err
}

// SaveUserConfig creates or updates a user config value
func SaveUserConfig(db *gorm.DB, config *UserConfig) error {
	var existing UserConfig
	result := db.Where("user_id = ? AND config_id = ?", config.UserId, config.ConfigId).First(&existing)
	
	if result.Error == nil {
		// Update existing record
		existing.Value = config.Value
		return db.Save(&existing).Error
	} else if result.Error == gorm.ErrRecordNotFound {
		// Create new record
		return db.Create(config).Error
	} else {
		// Some other error
		return result.Error
	}
}

// DeleteUserConfig deletes a specific user config
func DeleteUserConfig(db *gorm.DB, userId, configId int) error {
	return db.Where("user_id = ? AND config_id = ?", userId, configId).Delete(&UserConfig{}).Error
}

// DeleteUserConfigsForService deletes all user configs for a specific service
func DeleteUserConfigsForService(db *gorm.DB, userId, serviceId int) error {
	return db.Where("user_id = ? AND service_id = ?", userId, serviceId).Delete(&UserConfig{}).Error
}

// GetUserConfigsWithDetails returns user configs with service and config details
func GetUserConfigsWithDetails(db *gorm.DB, userId int) ([]map[string]interface{}, error) {
	var userConfigs []UserConfig
	if err := db.Where("user_id = ?", userId).Find(&userConfigs).Error; err != nil {
		return nil, err
	}
	
	result := make([]map[string]interface{}, 0, len(userConfigs))
	
	for _, config := range userConfigs {
		var service MCPService
		var configService ConfigService
		
		if err := db.First(&service, config.ServiceId).Error; err != nil {
			continue
		}
		
		if err := db.First(&configService, config.ConfigId).Error; err != nil {
			continue
		}
		
		configMap := map[string]interface{}{
			"id":         config.Id,
			"user_id":    config.UserId,
			"service":    service,
			"config":     configService,
			"value":      config.Value,
			"created_at": config.CreatedAt,
			"updated_at": config.UpdatedAt,
		}
		
		result = append(result, configMap)
	}
	
	return result, nil
} 