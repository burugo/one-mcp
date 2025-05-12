package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"os"
	"strconv"
	"text/template"

	"github.com/gin-gonic/gin"
)

// GetAllMCPServices godoc
// @Summary 获取所有MCP服务
// @Description 返回所有MCP服务的列表，包括环境变量定义和包管理器信息
// @Tags MCP Services
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} object
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services [get]
func GetAllMCPServices(c *gin.Context) {
	lang := c.GetString("lang")
	services, err := model.GetAllServices()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_service_list_failed", lang), err)
		return
	}

	// 使用Thing ORM的ToJSON进行序列化
	jsonBytes, err := model.MCPServiceDB.ToJSON(services)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("serialize_service_failed", lang), err)
		return
	}
	c.Data(http.StatusOK, "application/json", jsonBytes)
}

// GetMCPService godoc
// @Summary 获取单个MCP服务
// @Description 根据ID返回一个MCP服务的详情，包括环境变量定义和包管理器信息
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Security ApiKeyAuth
// @Success 200 {object} object
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id} [get]
func GetMCPService(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	jsonBytes, err := model.MCPServiceDB.ToJSON(service)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("serialize_service_failed", lang), err)
		return
	}
	c.Data(http.StatusOK, "application/json", jsonBytes)
}

// CreateMCPService godoc
// @Summary 创建新的MCP服务
// @Description 创建一个新的MCP服务，支持定义环境变量和包管理器信息
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param service body object true "服务信息"
// @Security ApiKeyAuth
// @Success 200 {object} object
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services [post]
func CreateMCPService(c *gin.Context) {
	lang := c.GetString("lang")
	var service model.MCPService
	if err := c.ShouldBindJSON(&service); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}

	// 基本验证
	if service.Name == "" || service.DisplayName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("name_and_display_name_required", lang))
		return
	}

	// 验证服务类型
	if !isValidServiceType(service.Type) {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_service_type", lang))
		return
	}

	// 验证RequiredEnvVarsJSON (如果提供)
	if service.RequiredEnvVarsJSON != "" {
		if err := validateRequiredEnvVarsJSON(service.RequiredEnvVarsJSON); err != nil {
			common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_env_vars_json", lang), err)
			return
		}
	}

	// 如果是marketplace服务（stdio类型且PackageManager不为空），验证相关字段
	if service.Type == model.ServiceTypeStdio && service.PackageManager != "" {
		if service.SourcePackageName == "" {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("source_package_name_required", lang))
			return
		}

		// 如果需要安装包，还可以添加更多验证...
	}

	if err := model.CreateService(&service); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("create_service_failed", lang), err)
		return
	}

	jsonBytes, err := model.MCPServiceDB.ToJSON(&service)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("serialize_service_failed", lang), err)
		return
	}
	c.Data(http.StatusOK, "application/json", jsonBytes)
}

// UpdateMCPService godoc
// @Summary 更新MCP服务
// @Description 更新现有的MCP服务，支持修改环境变量定义和包管理器信息
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Param service body object true "服务信息"
// @Security ApiKeyAuth
// @Success 200 {object} object
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id} [put]
func UpdateMCPService(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	// 保存原始值用于比较
	oldPackageManager := service.PackageManager
	oldSourcePackageName := service.SourcePackageName

	if err := c.ShouldBindJSON(service); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}

	// 基本验证
	if service.Name == "" || service.DisplayName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("name_and_display_name_required", lang))
		return
	}

	// 验证服务类型
	if !isValidServiceType(service.Type) {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_service_type", lang))
		return
	}

	// 验证RequiredEnvVarsJSON (如果提供)
	if service.RequiredEnvVarsJSON != "" {
		if err := validateRequiredEnvVarsJSON(service.RequiredEnvVarsJSON); err != nil {
			common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_env_vars_json", lang), err)
			return
		}
	}

	// 如果是marketplace服务（stdio类型且PackageManager不为空），验证相关字段
	if service.Type == model.ServiceTypeStdio && service.PackageManager != "" {
		if service.SourcePackageName == "" {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("source_package_name_required", lang))
			return
		}

		// 检查是否修改了关键包信息，可能需要重新安装
		if oldPackageManager != service.PackageManager || oldSourcePackageName != service.SourcePackageName {
			// 这里可以添加处理逻辑或警告...
		}
	}

	if err := model.UpdateService(service); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("update_service_failed", lang), err)
		return
	}

	jsonBytes, err := model.MCPServiceDB.ToJSON(service)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("serialize_service_failed", lang), err)
		return
	}
	c.Data(http.StatusOK, "application/json", jsonBytes)
}

// DeleteMCPService godoc
// @Summary 删除MCP服务
// @Description 删除一个MCP服务
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id} [delete]
func DeleteMCPService(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	// 获取服务信息，用于检查是否是marketplace安装的服务
	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	// 如果是安装的包，可能需要卸载物理包
	// 注意：这里只是检查，实际卸载逻辑需要在专门的API中实现
	if service.Type == model.ServiceTypeStdio && service.PackageManager != "" && service.SourcePackageName != "" {
		// 可以添加警告或特殊处理...
	}

	// 删除服务
	if err := model.DeleteService(id); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("delete_service_failed", lang), err)
		return
	}

	common.RespSuccessStr(c, i18n.Translate("service_deleted_successfully", lang))
}

// ToggleMCPService godoc
// @Summary 切换MCP服务启用状态
// @Description 切换MCP服务的启用/禁用状态
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id}/toggle [post]
func ToggleMCPService(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	// 尝试获取服务，确认它存在
	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	// 切换启用状态
	if err := model.ToggleServiceEnabled(id); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("toggle_service_status_failed", lang), err)
		return
	}

	status := i18n.Translate("enabled", lang)
	if !service.Enabled {
		status = i18n.Translate("disabled", lang)
	}

	common.RespSuccessStr(c, i18n.Translate("service_toggle_success", lang)+status)
}

// GetMCPServiceConfig godoc
// @Summary 获取特定客户端的MCP服务配置模板
// @Description 根据服务ID和客户端类型，返回适用于该客户端的服务配置模板
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Param client path string true "客户端类型，例如 'cherry_studio', 'cursor_sse', 'cursor_streamable' 等"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id}/config/{client} [get]
func GetMCPServiceConfig(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	clientType := c.Param("client")

	// 检查参数有效性
	if clientType == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("client_type_required", lang))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	// 获取服务信息
	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	var templateDetail *model.ClientTemplateDetail

	// 首先尝试从服务的ClientConfigTemplates获取模板
	if service.ClientConfigTemplates != "" {
		// 尝试从服务模型获取模板
		templateDetail, err = service.GetClientTemplateDetail(clientType)
		if err != nil && err.Error() != "mcp_service_template_not_found" {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_template_failed", lang), err)
			return
		}
	}

	// 如果服务模型中没有找到模板，尝试从配置文件加载
	if templateDetail == nil {
		// 从配置文件加载客户端模板
		templateDetail, err = loadClientTemplateFromConfig(clientType)
		if err != nil {
			common.RespError(c, http.StatusNotFound, i18n.Translate("client_template_not_found", lang), err)
			return
		}
	}

	// 构建响应数据
	response := map[string]interface{}{
		"service_id":               service.ID,
		"service_name":             service.Name,
		"client_type":              clientType,
		"template_string":          templateDetail.TemplateString,
		"client_expected_protocol": templateDetail.ClientExpectedProtocol,
		"our_proxy_protocol":       templateDetail.ClientExpectedProtocol,
		"display_name":             templateDetail.DisplayName,
	}

	// 示例实例ID（用于演示）
	instanceID := "instance_" + strconv.FormatInt(service.ID, 10)

	// 示例基础URL（实际应从配置或请求中获取）
	baseURL := "https://router.mcp.so"

	// 处理模板
	tmpl, err := template.New("config").Parse(templateDetail.TemplateString)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("template_parse_failed", lang), err)
		return
	}

	// 准备模板数据
	templateData := map[string]interface{}{
		"Name":                   service.Name,
		"ID":                     service.ID,
		"InstanceId":             instanceID,
		"BaseUrl":                baseURL,
		"ClientExpectedProtocol": templateDetail.ClientExpectedProtocol,
		"OurProxyProtocol":       templateDetail.ClientExpectedProtocol,
	}

	// 渲染模板
	var renderedConfig bytes.Buffer
	if err := tmpl.Execute(&renderedConfig, templateData); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("template_render_failed", lang), err)
		return
	}

	// 将渲染后的配置添加到响应中
	response["rendered_config"] = renderedConfig.String()

	common.RespSuccess(c, response)
}

// 从配置文件加载客户端模板
func loadClientTemplateFromConfig(clientType string) (*model.ClientTemplateDetail, error) {
	// 读取配置文件
	configPath := "config/client_templates.json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client templates config: %w", err)
	}

	// 解析配置
	var templates map[string]model.ClientTemplateDetail
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, fmt.Errorf("failed to parse client templates config: %w", err)
	}

	// 获取特定客户端类型的模板
	template, exists := templates[clientType]
	if !exists {
		return nil, fmt.Errorf("template for client type '%s' not found in config", clientType)
	}

	return &template, nil
}

// 辅助函数：验证服务类型
func isValidServiceType(sType model.ServiceType) bool {
	return sType == model.ServiceTypeStdio ||
		sType == model.ServiceTypeSSE ||
		sType == model.ServiceTypeStreamableHTTP
}

// 辅助函数：验证RequiredEnvVarsJSON格式
func validateRequiredEnvVarsJSON(envVarsJSON string) error {
	if envVarsJSON == "" {
		return nil
	}

	var envVars []model.EnvVarDefinition
	if err := json.Unmarshal([]byte(envVarsJSON), &envVars); err != nil {
		return err
	}

	// 验证每个环境变量是否有name字段
	for _, envVar := range envVars {
		if envVar.Name == "" {
			return errors.New("missing name field in env var definition")
		}
	}

	return nil
}

// GetMCPServiceHealth godoc
// @Summary 获取MCP服务的健康状态
// @Description 获取指定MCP服务的健康状态信息
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id}/health [get]
func GetMCPServiceHealth(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	// 获取服务信息
	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	// 准备健康状态响应
	healthData := map[string]interface{}{
		"service_id":     service.ID,
		"service_name":   service.Name,
		"health_status":  service.HealthStatus,
		"last_checked":   service.LastHealthCheck,
		"health_details": nil,
	}

	// 如果有健康详情，解析并添加到响应中
	if service.HealthDetails != "" {
		var healthDetails map[string]interface{}
		if err := json.Unmarshal([]byte(service.HealthDetails), &healthDetails); err == nil {
			healthData["health_details"] = healthDetails
		}
	}

	common.RespSuccess(c, healthData)
}

// CheckMCPServiceHealth godoc
// @Summary 检查MCP服务的健康状态
// @Description 强制检查指定MCP服务的健康状态，并返回最新结果
// @Tags MCP Services
// @Accept json
// @Produce json
// @Param id path int true "服务ID"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_services/{id}/health/check [post]
func CheckMCPServiceHealth(c *gin.Context) {
	lang := c.GetString("lang")
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	// 获取服务信息
	service, err := model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	// 获取服务管理器
	serviceManager := proxy.GetServiceManager()

	// 检查服务是否已经注册
	_, err = serviceManager.GetService(id)
	if err == proxy.ErrServiceNotFound {
		// 服务尚未注册，尝试注册
		ctx := c.Request.Context()
		if err := serviceManager.RegisterService(ctx, service); err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("register_service_failed", lang), err)
			return
		}
	}

	// 强制检查健康状态
	health, err := serviceManager.ForceCheckServiceHealth(id)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("check_service_health_failed", lang), err)
		return
	}

	// 更新数据库中的健康状态
	if err := serviceManager.UpdateMCPServiceHealth(id); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("update_service_health_failed", lang), err)
		return
	}

	// 构建响应
	healthData := map[string]interface{}{
		"service_id":     service.ID,
		"service_name":   service.Name,
		"health_status":  string(health.Status),
		"last_checked":   health.LastChecked,
		"health_details": health,
	}

	common.RespSuccess(c, healthData)
}
