package model

import (
	"gorm.io/gorm"
	"time"
)

// ConfigType defines the type of configuration option
type ConfigType string

const (
	ConfigTypeString  ConfigType = "string"
	ConfigTypeNumber  ConfigType = "number"
	ConfigTypeBool    ConfigType = "boolean"
	ConfigTypeSelect  ConfigType = "select"
	ConfigTypeSecret  ConfigType = "secret"
	ConfigTypeJSON    ConfigType = "json"
	ConfigTypeTextarea ConfigType = "textarea"
)

// ConfigService represents a configuration option for an MCP service
type ConfigService struct {
	Id              int        `json:"id" gorm:"primaryKey"`
	ServiceId       int        `json:"service_id" gorm:"not null;index:idx_service_key"`
	Key             string     `json:"key" gorm:"size:100;not null;index:idx_service_key"`
	DisplayName     string     `json:"display_name" gorm:"size:255;not null"`
	Description     string     `json:"description" gorm:"type:text"`
	Type            ConfigType `json:"type" gorm:"size:50;not null;default:'string'"`
	DefaultValue    string     `json:"default_value" gorm:"type:text"`
	Options         string     `json:"options" gorm:"type:text"` // JSON array for select options
	Required        bool       `json:"required" gorm:"default:false"`
	AdvancedSetting bool       `json:"advanced_setting" gorm:"default:false"`
	Order           int        `json:"order" gorm:"default:0"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	Service         MCPService `json:"-" gorm:"foreignKey:ServiceId"`
}

// TableName sets the table name for the ConfigService model
func (c *ConfigService) TableName() string {
	return "config_services"
}

// GetConfigOptionsForService returns all configuration options for a specific service
func GetConfigOptionsForService(db *gorm.DB, serviceId int) ([]ConfigService, error) {
	var configs []ConfigService
	err := db.Where("service_id = ?", serviceId).Order("order asc").Find(&configs).Error
	return configs, err
}

// GetConfigOptionByID returns a specific configuration option by ID
func GetConfigOptionByID(db *gorm.DB, id int) (ConfigService, error) {
	var config ConfigService
	err := db.First(&config, id).Error
	return config, err
}

// GetConfigOptionByKey returns a specific configuration option by service ID and key
func GetConfigOptionByKey(db *gorm.DB, serviceId int, key string) (ConfigService, error) {
	var config ConfigService
	err := db.Where("service_id = ? AND key = ?", serviceId, key).First(&config).Error
	return config, err
}

// CreateConfigOption creates a new service configuration option
func CreateConfigOption(db *gorm.DB, config *ConfigService) error {
	return db.Create(config).Error
}

// UpdateConfigOption updates an existing service configuration option
func UpdateConfigOption(db *gorm.DB, config *ConfigService) error {
	return db.Save(config).Error
}

// DeleteConfigOption deletes a service configuration option
func DeleteConfigOption(db *gorm.DB, id int) error {
	return db.Delete(&ConfigService{}, id).Error
}

// DeleteConfigOptionsForService deletes all configuration options for a service
func DeleteConfigOptionsForService(db *gorm.DB, serviceId int) error {
	return db.Where("service_id = ?", serviceId).Delete(&ConfigService{}).Error
}

// GetAllConfigOptions returns all configuration options for all services
func GetAllConfigOptions(db *gorm.DB) ([]ConfigService, error) {
	var configs []ConfigService
	err := db.Order("service_id asc, order asc").Find(&configs).Error
	return configs, err
}

// GetConfigOptionsWithServiceDetails returns configuration options with their service details
func GetConfigOptionsWithServiceDetails(db *gorm.DB) ([]map[string]interface{}, error) {
	var configOptions []ConfigService
	if err := db.Order("service_id asc, order asc").Find(&configOptions).Error; err != nil {
		return nil, err
	}
	
	result := make([]map[string]interface{}, 0, len(configOptions))
	
	for _, config := range configOptions {
		var service MCPService
		
		if err := db.First(&service, config.ServiceId).Error; err != nil {
			continue
		}
		
		configMap := map[string]interface{}{
			"id":               config.Id,
			"service":          service,
			"key":              config.Key,
			"display_name":     config.DisplayName,
			"description":      config.Description,
			"type":             config.Type,
			"default_value":    config.DefaultValue,
			"options":          config.Options,
			"required":         config.Required,
			"advanced_setting": config.AdvancedSetting,
			"order":            config.Order,
			"created_at":       config.CreatedAt,
			"updated_at":       config.UpdatedAt,
		}
		
		result = append(result, configMap)
	}
	
	return result, nil
} 