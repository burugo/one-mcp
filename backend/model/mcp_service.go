package model

import (
	"time"

	"gorm.io/gorm"
)

// ServiceCategory represents different categories of MCP services
type ServiceCategory string

const (
	CategorySearch  ServiceCategory = "search"
	CategoryFetch   ServiceCategory = "fetch"
	CategoryAI      ServiceCategory = "ai"
	CategoryUtil    ServiceCategory = "utility"
	CategoryStorage ServiceCategory = "storage"
)

// MCPService represents an MCP service that can be enabled or configured
type MCPService struct {
	Id          int             `json:"id" gorm:"primaryKey"`
	Name        string          `json:"name" gorm:"size:255;not null;uniqueIndex"`
	DisplayName string          `json:"display_name" gorm:"size:255;not null"`
	Description string          `json:"description" gorm:"type:text"`
	Category    ServiceCategory `json:"category" gorm:"size:50;not null"`
	Icon        string          `json:"icon" gorm:"size:255"`
	DefaultOn   bool            `json:"default_on" gorm:"default:false"`
	AdminOnly   bool            `json:"admin_only" gorm:"default:false"`
	OrderNum    int             `json:"order_num" gorm:"column:order_num;default:0"`
	Enabled     bool            `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName sets the table name for the MCPService model
func (s *MCPService) TableName() string {
	return "mcp_services"
}

// GetAllServices returns all MCP services
func GetAllServices(db *gorm.DB) ([]MCPService, error) {
	var services []MCPService
	err := db.Order("category asc, order asc").Find(&services).Error
	return services, err
}

// GetEnabledServices returns all enabled MCP services
func GetEnabledServices(db *gorm.DB) ([]MCPService, error) {
	var services []MCPService
	err := db.Where("enabled = ?", true).Order("category asc, order asc").Find(&services).Error
	return services, err
}

// GetServiceByID retrieves a specific service by ID
func GetServiceByID(db *gorm.DB, id int) (MCPService, error) {
	var service MCPService
	err := db.First(&service, id).Error
	return service, err
}

// GetServiceByName retrieves a specific service by name
func GetServiceByName(db *gorm.DB, name string) (MCPService, error) {
	var service MCPService
	err := db.Where("name = ?", name).First(&service).Error
	return service, err
}

// CreateService creates a new MCP service
func CreateService(db *gorm.DB, service *MCPService) error {
	return db.Create(service).Error
}

// UpdateService updates an existing MCP service
func UpdateService(db *gorm.DB, service *MCPService) error {
	return db.Save(service).Error
}

// DeleteService deletes an MCP service
func DeleteService(db *gorm.DB, id int) error {
	return db.Delete(&MCPService{}, id).Error
}

// ToggleServiceEnabled toggles the enabled status of a service
func ToggleServiceEnabled(db *gorm.DB, id int) error {
	var service MCPService
	if err := db.First(&service, id).Error; err != nil {
		return err
	}

	service.Enabled = !service.Enabled
	return db.Save(&service).Error
}

// GetServicesWithConfig returns services with their configuration options
func GetServicesWithConfig(db *gorm.DB) ([]map[string]interface{}, error) {
	var services []MCPService
	if err := db.Order("category asc, order asc").Find(&services).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(services))

	for _, service := range services {
		var configs []ConfigService
		if err := db.Where("service_id = ?", service.Id).Order("order asc").Find(&configs).Error; err != nil {
			return nil, err
		}

		serviceMap := map[string]interface{}{
			"service": service,
			"configs": configs,
		}

		result = append(result, serviceMap)
	}

	return result, nil
}
