package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	TemplateString         string `json:"template_string"`
	ClientExpectedProtocol string `json:"client_expected_protocol"`
	DisplayName            string `json:"display_name"`
}

// EnvVarDefinition defines a required environment variable
type EnvVarDefinition struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsSecret     bool   `json:"is_secret"`
	Optional     bool   `json:"optional"`
	DefaultValue string `json:"default_value"`
}

// MCPService represents an MCP service that can be enabled or configured
type MCPService struct {
	thing.BaseModel
	Name                  string          `db:"name" json:"name"`
	DisplayName           string          `db:"display_name" json:"display_name"`
	Description           string          `db:"description" json:"description"`
	Category              ServiceCategory `db:"category"`
	Icon                  string          `db:"icon"`
	DefaultOn             bool            `db:"default_on"`
	AdminOnly             bool            `db:"admin_only"`
	OrderNum              int             `db:"order_num"`
	Enabled               bool            `db:"enabled"`
	Type                  ServiceType     `db:"type"`
	Command               string          `json:"command,omitempty" db:"command"`
	ArgsJSON              string          `json:"args_json,omitempty" db:"args_json,default:'{}'"`
	AllowUserOverride     bool            `db:"allow_user_override"`     // Whether users can override admin settings
	ClientConfigTemplates string          `db:"client_config_templates"` // JSON map of client_type to template details
	RequiredEnvVarsJSON   string          `db:"required_env_vars_json"`  // JSON array of environment variables required by the service
	PackageManager        string          `db:"package_manager"`         // For marketplace services: npm, pypi
	SourcePackageName     string          `db:"source_package_name"`     // For marketplace services: package name in the repository
	InstalledVersion      string          `db:"installed_version"`       // For marketplace services: currently installed version
	InstallerUserID       int64           `db:"installer_user_id"`       // 记录安装者的用户ID
	HealthStatus          string          `db:"-"`                       // 健康状态: unknown, healthy, unhealthy, starting, stopped
	LastHealthCheck       time.Time       `db:"-"`                       // 最后健康检查时间
	HealthDetails         string          `db:"-"`                       // 健康详情的JSON字符串
	DefaultEnvsJSON       string          `json:"default_envs_json,omitempty" db:"default_envs_json,default:'{}'"`
	HeadersJSON           string          `json:"headers_json,omitempty" db:"headers_json,default:'{}'"` // JSON string for custom request headers map[string]string
	RPDLimit              int             `json:"rpd_limit,omitempty" db:"rpd_limit,default:0"`          // 每日请求次数限制(0表示不限制)
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
		return nil, errors.New("mcp_service_template_not_found")
	}

	return &detail, nil
}

// SetRequiredEnvVars sets the RequiredEnvVarsJSON field from a slice of EnvVarDefinition
func (s *MCPService) SetRequiredEnvVars(envVars []EnvVarDefinition) error {
	if len(envVars) == 0 {
		s.RequiredEnvVarsJSON = ""
		return nil
	}

	data, err := json.Marshal(envVars)
	if err != nil {
		return err
	}
	s.RequiredEnvVarsJSON = string(data)
	return nil
}

// GetRequiredEnvVars returns the RequiredEnvVarsJSON as a slice of EnvVarDefinition
func (s *MCPService) GetRequiredEnvVars() ([]EnvVarDefinition, error) {
	if s.RequiredEnvVarsJSON == "" {
		return []EnvVarDefinition{}, nil
	}

	var envVars []EnvVarDefinition
	err := json.Unmarshal([]byte(s.RequiredEnvVarsJSON), &envVars)
	if err != nil {
		return nil, err
	}
	return envVars, nil
}

var MCPServiceDB *thing.Thing[*MCPService]

// MCPServiceInit initializes the MCPServiceDB
func MCPServiceInit() error {
	var err error
	MCPServiceDB, err = thing.Use[*MCPService]()
	if err != nil {
		return fmt.Errorf("failed to initialize MCPServiceDB: %w", err)
	}
	return nil
}

// GetAllServices returns all MCP services
func GetAllServices() ([]*MCPService, error) {
	return MCPServiceDB.Order("category ASC, order_num ASC").All()
}

// GetEnabledServices returns all enabled MCP services
func GetEnabledServices() ([]*MCPService, error) {
	// Filter out soft-deleted services as well
	return MCPServiceDB.Where("enabled = ? AND deleted = ?", true, false).Order("category ASC, order_num ASC").All()
}

// GetInstalledServices returns all installed MCP services (regardless of enabled status)
func GetInstalledServices() ([]*MCPService, error) {
	// Only return non-deleted services
	return MCPServiceDB.Where("deleted = ?", false).Order("category ASC, order_num ASC").All()
}

// GetServiceByID retrieves a specific service by ID
func GetServiceByID(id int64) (*MCPService, error) {
	return MCPServiceDB.ByID(id)
}

// GetServiceByName retrieves a specific service by name
func GetServiceByName(name string) (*MCPService, error) {
	return MCPServiceDB.Where("name = ?", name).First()
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
	service, err := GetServiceByID(id)
	if err != nil {
		return err
	}
	return MCPServiceDB.Delete(service)
}

// ToggleServiceEnabled toggles the enabled status of a service
func ToggleServiceEnabled(id int64) error {
	service, err := GetServiceByID(id)
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

// GetServicesByPackageDetails retrieves services by package details
func GetServicesByPackageDetails(packageManager, packageName string) ([]*MCPService, error) {
	return MCPServiceDB.Where("package_manager = ? AND source_package_name = ?", packageManager, packageName).All()
}

// StdioConfig holds the configuration for an Stdio MCP service
type StdioConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env"` // Stored as "KEY=VALUE" strings
}

// GetMCPServiceThing returns the initialized Thing ORM instance for MCPService.
// It ensures MCPServiceInit is called if the instance is not yet available.
func GetMCPServiceThing() (*thing.Thing[*MCPService], error) {
	if MCPServiceDB == nil {
		// This typically should not happen if InitDB -> MCPServiceInit was successful at startup.
		// However, as a safeguard or for contexts where InitDB might not have fully run (e.g. specific tests without full app init):
		if err := MCPServiceInit(); err != nil {
			return nil, fmt.Errorf("failed to explicitly initialize MCPServiceDB in GetMCPServiceThing: %w", err)
		}
		if MCPServiceDB == nil {
			// If still nil after explicit init, something is seriously wrong.
			return nil, errors.New("MCPServiceDB is nil even after explicit re-initialization attempt in GetMCPServiceThing")
		}
	}
	return MCPServiceDB, nil
}
