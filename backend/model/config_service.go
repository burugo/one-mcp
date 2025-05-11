package model

import (
	"errors"

	"github.com/burugo/thing"
)

// ErrRecordNotFound is used when a record is not found
var ErrRecordNotFound = errors.New("config_service_not_found")

// ConfigType defines the type of configuration option
type ConfigType string

const (
	ConfigTypeString   ConfigType = "string"
	ConfigTypeNumber   ConfigType = "number"
	ConfigTypeBool     ConfigType = "boolean"
	ConfigTypeSelect   ConfigType = "select"
	ConfigTypeSecret   ConfigType = "secret"
	ConfigTypeJSON     ConfigType = "json"
	ConfigTypeTextarea ConfigType = "textarea"
)

// ConfigService represents a configuration option for an MCP service
type ConfigService struct {
	thing.BaseModel
	ServiceID       int64      `db:"service_id,index:idx_service_key"`
	Key             string     `db:"key,index:idx_service_key"`
	DisplayName     string     `db:"display_name"`
	Description     string     `db:"description"`
	Type            ConfigType `db:"type"`
	DefaultValue    string     `db:"default_value"`
	Options         string     `db:"options"` // JSON array for select options
	Required        bool       `db:"required"`
	AdvancedSetting bool       `db:"advanced_setting"`
	OrderNum        int        `db:"order_num"`
}

// TableName sets the table name for the ConfigService model
func (c *ConfigService) TableName() string {
	return "config_services"
}

var ConfigServiceDB *thing.Thing[*ConfigService]

// ConfigServiceInit initializes the ConfigServiceDB
func ConfigServiceInit() error {
	var err error
	ConfigServiceDB, err = thing.Use[*ConfigService]()
	if err != nil {
		return err
	}
	return nil
}

// GetConfigOptionsForService returns all configuration options for a specific service
func GetConfigOptionsForService(serviceID int64) ([]*ConfigService, error) {
	return ConfigServiceDB.Where("service_id = ?", serviceID).Order("order_num ASC").All()
}

// GetConfigOptionByID returns a specific configuration option by ID
func GetConfigOptionByID(id int64) (*ConfigService, error) {
	return ConfigServiceDB.ByID(id)
}

// GetConfigOptionByKey returns a specific configuration option by service ID and key
func GetConfigOptionByKey(serviceID int64, key string) (*ConfigService, error) {
	configs, err := ConfigServiceDB.Where("service_id = ? AND key = ?", serviceID, key).Fetch(0, 1)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, ErrRecordNotFound
	}
	return configs[0], nil
}

// CreateConfigOption creates a new service configuration option
func CreateConfigOption(config *ConfigService) error {
	return ConfigServiceDB.Save(config)
}

// UpdateConfigOption updates an existing service configuration option
func UpdateConfigOption(config *ConfigService) error {
	return ConfigServiceDB.Save(config)
}

// DeleteConfigOption deletes a service configuration option
func DeleteConfigOption(id int64) error {
	config, err := ConfigServiceDB.ByID(id)
	if err != nil {
		return err
	}
	return ConfigServiceDB.Delete(config)
}

// DeleteConfigOptionsForService deletes all configuration options for a service
func DeleteConfigOptionsForService(serviceID int64) error {
	configs, err := ConfigServiceDB.Where("service_id = ?", serviceID).All()
	if err != nil {
		return err
	}

	for _, config := range configs {
		if err := ConfigServiceDB.Delete(config); err != nil {
			return err
		}
	}

	return nil
}

// GetAllConfigOptions returns all configuration options for all services
func GetAllConfigOptions() ([]*ConfigService, error) {
	return ConfigServiceDB.Order("service_id ASC, order_num ASC").All()
}
