package model

import (
	"encoding/json"
	"errors"

	"github.com/burugo/thing"
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

// ServiceType represents the underlying type of an MCP service
type ServiceType string

const (
	ServiceTypeStdio          ServiceType = "stdio"
	ServiceTypeSSE            ServiceType = "sse"
	ServiceTypeStreamableHTTP ServiceType = "streamable_http"
)

// ClientTemplateDetail contains template info for a specific client type
type ClientTemplateDetail struct {
	TemplateString                string `json:"template_string"`
	ClientExpectedProtocol        string `json:"client_expected_protocol"`
	OurProxyProtocolForThisClient string `json:"our_proxy_protocol_for_this_client"`
}

// MCPService represents an MCP service that can be enabled or configured
type MCPService struct {
	thing.BaseModel
	Name                     string          `db:"name,unique"`
	DisplayName              string          `db:"display_name"`
	Description              string          `db:"description"`
	Category                 ServiceCategory `db:"category"`
	Icon                     string          `db:"icon"`
	DefaultOn                bool            `db:"default_on"`
	AdminOnly                bool            `db:"admin_only"`
	OrderNum                 int             `db:"order_num"`
	Enabled                  bool            `db:"enabled"`
	Type                     ServiceType     `db:"type"`                        // New: Underlying type (stdio, sse, streamable_http)
	AdminConfigSchema        string          `db:"admin_config_schema"`         // New: JSON schema for admin configuration
	DefaultAdminConfigValues string          `db:"default_admin_config_values"` // New: Default values for admin configuration
	UserConfigSchema         string          `db:"user_config_schema"`          // New: JSON schema for user configuration
	AllowUserOverride        bool            `db:"allow_user_override"`         // New: Whether users can override admin settings
	ClientConfigTemplates    string          `db:"client_config_templates"`     // New: JSON map of client_type to template details
}

// TableName sets the table name for the MCPService model
func (s *MCPService) TableName() string {
	return "mcp_services"
}

// SetClientConfigTemplates sets the ClientConfigTemplates field from a map
func (s *MCPService) SetClientConfigTemplates(templates map[string]ClientTemplateDetail) error {
	data, err := json.Marshal(templates)
	if err != nil {
		return err
	}
	s.ClientConfigTemplates = string(data)
	return nil
}

// GetClientConfigTemplates returns the ClientConfigTemplates as a map
func (s *MCPService) GetClientConfigTemplates() (map[string]ClientTemplateDetail, error) {
	if s.ClientConfigTemplates == "" {
		return make(map[string]ClientTemplateDetail), nil
	}

	var templates map[string]ClientTemplateDetail
	err := json.Unmarshal([]byte(s.ClientConfigTemplates), &templates)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// GetClientTemplateDetail returns the template detail for a specific client type
func (s *MCPService) GetClientTemplateDetail(clientType string) (*ClientTemplateDetail, error) {
	templates, err := s.GetClientConfigTemplates()
	if err != nil {
		return nil, err
	}

	detail, exists := templates[clientType]
	if !exists {
		return nil, errors.New("no template found for client type: " + clientType)
	}

	return &detail, nil
}

var MCPServiceDB *thing.Thing[*MCPService]

// MCPServiceInit initializes the MCPServiceDB
func MCPServiceInit() error {
	var err error
	MCPServiceDB, err = thing.Use[*MCPService]()
	if err != nil {
		return err
	}
	return nil
}

// GetAllServices returns all MCP services
func GetAllServices() ([]*MCPService, error) {
	return MCPServiceDB.Order("category ASC, order_num ASC").All()
}

// GetEnabledServices returns all enabled MCP services
func GetEnabledServices() ([]*MCPService, error) {
	return MCPServiceDB.Where("enabled = ?", true).Order("category ASC, order_num ASC").All()
}

// GetServiceByID retrieves a specific service by ID
func GetServiceByID(id int64) (*MCPService, error) {
	return MCPServiceDB.ByID(id)
}

// GetServiceByName retrieves a specific service by name
func GetServiceByName(name string) (*MCPService, error) {
	services, err := MCPServiceDB.Where("name = ?", name).Fetch(0, 1)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, errors.New("service not found")
	}
	return services[0], nil
}

// CreateService creates a new MCP service
func CreateService(service *MCPService) error {
	return MCPServiceDB.Save(service)
}

// UpdateService updates an existing MCP service
func UpdateService(service *MCPService) error {
	return MCPServiceDB.Save(service)
}

// DeleteService deletes an MCP service
func DeleteService(id int64) error {
	service, err := MCPServiceDB.ByID(id)
	if err != nil {
		return err
	}
	return MCPServiceDB.Delete(service)
}

// ToggleServiceEnabled toggles the enabled status of a service
func ToggleServiceEnabled(id int64) error {
	service, err := MCPServiceDB.ByID(id)
	if err != nil {
		return err
	}

	service.Enabled = !service.Enabled
	return MCPServiceDB.Save(service)
}

// GetServicesWithConfig returns services with their configuration options
func GetServicesWithConfig() ([]map[string]interface{}, error) {
	services, err := MCPServiceDB.Order("category ASC, order_num ASC").All()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(services))

	for _, service := range services {
		configs, err := ConfigServiceDB.Where("service_id = ?", service.ID).Order("order_num ASC").All()
		if err != nil {
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
