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
	Name                  string          `db:"name"`
	DisplayName           string          `db:"display_name"`
	Description           string          `db:"description"`
	Category              ServiceCategory `db:"category"`
	Icon                  string          `db:"icon"`
	DefaultOn             bool            `db:"default_on"`
	AdminOnly             bool            `db:"admin_only"`
	OrderNum              int             `db:"order_num"`
	Enabled               bool            `db:"enabled"`
	Type                  ServiceType     `db:"type"`
	Command               string          `db:"command"`
	ArgsJSON              string          `db:"args_json"`
	AllowUserOverride     bool            `db:"allow_user_override"`     // Whether users can override admin settings
	ClientConfigTemplates string          `db:"client_config_templates"` // JSON map of client_type to template details
	RequiredEnvVarsJSON   string          `db:"required_env_vars_json"`  // JSON array of environment variables required by the service
	PackageManager        string          `db:"package_manager"`         // For marketplace services: npm, pypi
	SourcePackageName     string          `db:"source_package_name"`     // For marketplace services: package name in the repository
	InstalledVersion      string          `db:"installed_version"`       // For marketplace services: currently installed version
	HealthStatus          string          `db:"health_status"`           // 健康状态: unknown, healthy, unhealthy, starting, stopped
	LastHealthCheck       time.Time       `db:"last_health_check"`       // 最后健康检查时间
	HealthDetails         string          `db:"health_details"`          // 健康详情的JSON字符串
	DefaultEnvsJSON       string          `db:"default_envs_json"`       // JSON string for default environment variables map[string]string
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
	return MCPServiceDB.Where("enabled = ?", true).Order("category ASC, order_num ASC").All()
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

// const exaMCPServiceStdioSchema = `{"type":"object","properties":{"command":{"type":"string"},"args":{"type":"array","items":{"type":"string"}},"env":{"type":"array","items":{"type":"string","pattern":"^[^=]+=[^=]+$"}}},"required":["command"]}`

// SeedDefaultServices ensures default services like "exa-mcp-server" exist.
// func SeedDefaultServices() error {
// 	common.SysLog("Seeding default services...")
// 	serviceName := "exa-mcp-server"
// 	existingService, _ := GetServiceByName(serviceName) // Ignore error, just check if nil

// 	// StdioConfig for exa-mcp-server
// 	// Assuming it's run via npx and is an npm package.
// 	// If it's a direct command, Command would be "exa-mcp-server" and ArgsJSON "[]" or specific args.
// 	exaCommand := "npx"
// 	exaArgsJSON := `["-y", "exa-mcp-server"]` // Note: Stored as a JSON string

// 	defaultExaEnvs := map[string]string{
// 		"PORT": "0", // Example default environment variable
// 	}
// 	defaultExaEnvsJSONBytes, err := json.Marshal(defaultExaEnvs)
// 	if err != nil {
// 		common.SysError(fmt.Sprintf("Failed to marshal DefaultEnvsJSON for %s: %v", serviceName, err))
// 		// Decide if we should return error or continue
// 	}
// 	defaultExaEnvsJSON := string(defaultExaEnvsJSONBytes)

// 	if existingService == nil {
// 		common.SysLog(fmt.Sprintf("Service %s not found, creating...", serviceName))
// 		newService := &MCPService{
// 			Name:                     serviceName,
// 			DisplayName:              "Exa Server (Stdio)",
// 			Description:              "Exa MCP Server for search and agents.",
// 			Category:                 CategoryAI,
// 			Icon:                     "/static/exa.png",
// 			DefaultOn:                true,
// 			AdminOnly:                false,
// 			OrderNum:                 10,
// 			Enabled:                  true,
// 			Type:                     ServiceTypeStdio,
// 			Command:                  exaCommand,
// 			ArgsJSON:                 exaArgsJSON,
// 			// AdminConfigSchema:        exaMCPServiceStdioSchema, // Removed
// 			// DefaultAdminConfigValues: string(stdioConfJSON), // Removed
// 			// UserConfigSchema:         `{"type":"object","properties":{"API_KEY":{"type":"string","description":"Your Exa API Key (user-specific)"}}}`, // Removed
// 			AllowUserOverride:        true,
// 			ClientConfigTemplates:    "{}",
// 			RequiredEnvVarsJSON:      "[]", // Example: `[{"name":"EXA_API_KEY", "description":"Your Exa API Key", "is_secret":true}]`
// 			DefaultEnvsJSON:          defaultExaEnvsJSON,
// 			PackageManager:           "npm", // Assuming it's an npm package for this example
// 			SourcePackageName:        "exa-mcp-server",
// 			InstalledVersion:         "N/A", // This would be set upon actual installation
// 		}
// 		if err := MCPServiceDB.Save(newService); err != nil {
// 			common.SysError(fmt.Sprintf("Failed to create service %s: %v", serviceName, err))
// 			return err
// 		}
// 		common.SysLog(fmt.Sprintf("Service %s created successfully.", serviceName))
// 	} else {
// 		common.SysLog(fmt.Sprintf("Service %s already exists. Updating if necessary...", serviceName))
// 		updateNeeded := false
// 		if existingService.Type != ServiceTypeStdio {
// 			existingService.Type = ServiceTypeStdio
// 			updateNeeded = true
// 			common.SysLog(fmt.Sprintf("Updated Type for service %s to Stdio", serviceName))
// 		}
// 		// Update command and args if they differ from the new defaults
// 		if existingService.Command != exaCommand {
// 			existingService.Command = exaCommand
// 			updateNeeded = true
// 			common.SysLog(fmt.Sprintf("Updated Command for service %s to %s", serviceName, exaCommand))
// 		}
// 		if existingService.ArgsJSON != exaArgsJSON {
// 			existingService.ArgsJSON = exaArgsJSON
// 			updateNeeded = true
// 			common.SysLog(fmt.Sprintf("Updated ArgsJSON for service %s", serviceName))
// 		}
// 		if existingService.DefaultEnvsJSON != defaultExaEnvsJSON {
// 			existingService.DefaultEnvsJSON = defaultExaEnvsJSON
// 			updateNeeded = true
// 			common.SysLog(fmt.Sprintf("Updated DefaultEnvsJSON for service %s", serviceName))
// 		}
// 		if existingService.PackageManager != "npm" { // Example, align with your typical setup
// 			existingService.PackageManager = "npm"
// 			updateNeeded = true
// 			common.SysLog(fmt.Sprintf("Updated PackageManager for service %s to npm", serviceName))
// 		}
// 		if existingService.SourcePackageName != "exa-mcp-server" {
// 			existingService.SourcePackageName = "exa-mcp-server"
// 			updateNeeded = true
// 			common.SysLog(fmt.Sprintf("Updated SourcePackageName for service %s", serviceName))
// 		}
// 		// Remove schema fields if they still exist (they shouldn't after struct change, but good for migration from old state)
// 		// if existingService.AdminConfigSchema != "" { // This field will be gone from struct
// 		// 	// existingService.AdminConfigSchema = "" // No need to set, will be zero value
// 		// 	updateNeeded = true
// 		// }
// 		// if existingService.DefaultAdminConfigValues != "" { // This field will be gone
// 		// 	// existingService.DefaultAdminConfigValues = ""
// 		// 	updateNeeded = true
// 		// }
// 		// if existingService.UserConfigSchema != "" { // This field will be gone
// 		// 	// existingService.UserConfigSchema = ""
// 		// 	updateNeeded = true
// 		// }

// 		if existingService.DisplayName != "Exa Server (Stdio)" {
// 			existingService.DisplayName = "Exa Server (Stdio)"
// 			updateNeeded = true
// 		}
// 		if existingService.Description != "Exa MCP Server for search and agents." {
// 			existingService.Description = "Exa MCP Server for search and agents."
// 			updateNeeded = true
// 		}
// 		if existingService.Icon != "/static/exa.png" {
// 			existingService.Icon = "/static/exa.png"
// 			updateNeeded = true
// 		}
// 		if existingService.Category != CategoryAI {
// 			existingService.Category = CategoryAI
// 			updateNeeded = true
// 		}

// 		if updateNeeded {
// 			if err := MCPServiceDB.Save(existingService); err != nil {
// 				common.SysError(fmt.Sprintf("Failed to update service %s: %v", serviceName, err))
// 				return err
// 			}
// 			common.SysLog(fmt.Sprintf("Service %s updated successfully.", serviceName))
// 		} else {
// 			common.SysLog(fmt.Sprintf("No updates needed for service %s.", serviceName))
// 		}
// 	}
// 	return nil
// }
