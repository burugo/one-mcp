package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/library/market"
	"one-mcp/backend/model"
	"strconv"
	"strings"
	"time"

	"log"

	"github.com/gin-gonic/gin"
)

// GetPackageDetails godoc
// @Summary 获取包详情
// @Description 获取指定包的详细信息
// @Tags Market
// @Accept json
// @Produce json
// @Param package_name query string true "包名"
// @Param package_manager query string true "包管理器，例如：npm"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/package_details [get]
func GetPackageDetails(c *gin.Context) {
	lang := c.GetString("lang")
	packageName := c.Query("package_name")
	packageManager := c.Query("package_manager")

	// 参数验证
	if packageName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_name_required", lang))
		return
	}

	if packageManager == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_manager_required", lang))
		return
	}

	// 添加一个超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 根据包管理器类型获取详情
	switch packageManager {
	case "npm":
		details, err := market.GetNPMPackageDetails(ctx, packageName)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_npm_package_details_failed", lang), err)
			return
		}

		// 检查是否已安装
		isInstalled := false
		services, err := model.GetServicesByPackageDetails(packageManager, packageName)
		if err == nil && len(services) > 0 {
			isInstalled = true
		}

		// 获取README内容
		readme, err := market.GetNPMPackageReadme(ctx, packageName)
		if err != nil {
			// 获取README失败不是致命错误，只记录日志
			common.SysLog("Error getting README for " + packageName + ": " + err.Error())
		}

		// 尝试从README中提取MCP配置
		mcpConfig, _ := market.ExtractMCPConfig(details, readme)

		// 猜测可能的环境变量
		var envVars []string

		// 首先从MCP配置中提取环境变量
		if mcpConfig != nil {
			envVars = market.GetEnvVarsFromMCPConfig(mcpConfig)
		}

		// 如果MCP配置中没有找到环境变量，则从README中猜测
		if len(envVars) == 0 {
			envVars = market.GuessMCPEnvVarsFromReadme(readme)
		}

		// 构建环境变量定义
		var envVarDefinitions []model.EnvVarDefinition
		for _, env := range envVars {
			definition := model.EnvVarDefinition{
				Name:        env,
				Description: "From package configuration",
				IsSecret:    strings.Contains(strings.ToLower(env), "token") || strings.Contains(strings.ToLower(env), "key") || strings.Contains(strings.ToLower(env), "secret"),
				Optional:    false,
			}
			envVarDefinitions = append(envVarDefinitions, definition)
		}

		// 构建响应
		response := map[string]interface{}{
			"details":      details,
			"is_installed": isInstalled,
			"env_vars":     envVarDefinitions,
			"mcp_config":   mcpConfig,
			"readme":       readme,
		}

		common.RespSuccess(c, response)
		return

	default:
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("unsupported_package_manager", lang))
		return
	}
}

// DiscoverEnvVars godoc
// @Summary 发现环境变量
// @Description 尝试从包的信息中发现可能需要的环境变量
// @Tags Market
// @Accept json
// @Produce json
// @Param package_name query string true "包名"
// @Param package_manager query string true "包管理器，例如：npm"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/discover_env_vars [get]
func DiscoverEnvVars(c *gin.Context) {
	lang := c.GetString("lang")
	packageName := c.Query("package_name")
	packageManager := c.Query("package_manager")

	// 参数验证
	if packageName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_name_required", lang))
		return
	}

	if packageManager == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_manager_required", lang))
		return
	}

	// 添加一个超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 根据包管理器类型发现环境变量
	var envVars []string

	switch packageManager {
	case "npm":
		// 获取包详情
		details, err := market.GetNPMPackageDetails(ctx, packageName)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_npm_package_details_failed", lang), err)
			return
		}

		// 获取README内容
		readme, err := market.GetNPMPackageReadme(ctx, packageName)
		if err != nil {
			// 获取README失败不是致命错误，只记录日志
			common.SysLog("Error getting README for " + packageName + ": " + err.Error())
		}

		// 尝试从README中提取MCP配置
		mcpConfig, _ := market.ExtractMCPConfig(details, readme)

		// 首先从MCP配置中提取环境变量
		if mcpConfig != nil {
			envVars = market.GetEnvVarsFromMCPConfig(mcpConfig)
		}

		// 如果MCP配置中没有找到环境变量，则从README中猜测
		if len(envVars) == 0 {
			envVars = market.GuessMCPEnvVarsFromReadme(readme)
		}

		// 如果包中声明了RequiresEnv字段
		if len(details.RequiresEnv) > 0 {
			for _, env := range details.RequiresEnv {
				if !contains(envVars, env) {
					envVars = append(envVars, env)
				}
			}
		}

	default:
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("unsupported_package_manager", lang))
		return
	}

	// 将猜测到的环境变量转换为EnvVarDefinition格式
	var envVarDefinitions []model.EnvVarDefinition
	for _, env := range envVars {
		definition := model.EnvVarDefinition{
			Name:        env,
			Description: "Auto discovered from package information",
			IsSecret:    strings.Contains(strings.ToLower(env), "token") || strings.Contains(strings.ToLower(env), "key") || strings.Contains(strings.ToLower(env), "secret"),
			Optional:    false,
		}
		envVarDefinitions = append(envVarDefinitions, definition)
	}

	response := map[string]interface{}{
		"env_vars": envVarDefinitions,
	}

	common.RespSuccess(c, response)
}

// InstallOrAddService godoc
// @Summary 安装或添加服务
// @Description 从市场安装服务或添加现有服务
// @Tags Market
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "请求体"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/install_or_add_service [post]
func InstallOrAddService(c *gin.Context) {
	lang := c.GetString("lang")
	var requestBody struct {
		SourceType          string                 `json:"source_type" binding:"required"`
		MCServiceID         int64                  `json:"mcp_service_id"`         // For predefined
		PackageName         string                 `json:"package_name"`           // For marketplace
		PackageManager      string                 `json:"package_manager"`        // For marketplace (npm, pypi, uv, pip)
		Version             string                 `json:"version"`                // For marketplace
		UserProvidedEnvVars map[string]interface{} `json:"user_provided_env_vars"` // Interface to handle potential type issues from UI, convert to string later.
		DisplayName         string                 `json:"display_name"`           // Optional: for creating MCPService
		ServiceDescription  string                 `json:"service_description"`    // Optional: for creating MCPService
		ServiceIconURL      string                 `json:"service_icon_url"`       // Optional: for creating MCPService
		Category            model.ServiceCategory  `json:"category"`               // Optional: for creating MCPService
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}

	userID := getUserIDFromContext(c)
	if userID == 0 && requestBody.SourceType != "predefined" { // Predefined might be admin setup
		common.RespErrorStr(c, http.StatusUnauthorized, i18n.Translate("user_not_authenticated", lang))
		return
	}

	envVarsForTask := convertEnvVarsMap(requestBody.UserProvidedEnvVars)

	if requestBody.SourceType == "predefined" {
		if requestBody.MCServiceID == 0 {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("mcp_service_id_required", lang))
			return
		}
		// For predefined, userID might be 0 if it's an admin setting up global defaults or if auth is handled differently.
		// The addServiceInstanceForUser function should be robust enough or this path needs specific logic for userID=0.
		// For now, we pass the userID obtained. If it's 0, addServiceInstanceForUser might need to handle it.
		if err := addServiceInstanceForUser(c, userID, requestBody.MCServiceID, requestBody.UserProvidedEnvVars); err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("add_service_instance_failed", lang), err)
			return
		}
		common.RespSuccessStr(c, i18n.Translate("service_added_successfully", lang))
		return
	} else if requestBody.SourceType == "marketplace" {
		if requestBody.PackageName == "" || requestBody.PackageManager == "" {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_name_and_manager_required", lang))
			return
		}

		// Check tool availability
		if requestBody.PackageManager == "npm" && !market.CheckNPXAvailable() {
			common.RespErrorStr(c, http.StatusInternalServerError, i18n.Translate("npx_not_available", lang))
			return
		}
		if (requestBody.PackageManager == "pypi" || requestBody.PackageManager == "uv" || requestBody.PackageManager == "pip") && !market.CheckUVXAvailable() {
			// Assuming "pip" also uses "uv" for now or this check is sufficient
			common.RespErrorStr(c, http.StatusInternalServerError, i18n.Translate("uv_not_available", lang))
			return
		}

		existingServices, err := model.GetServicesByPackageDetails(requestBody.PackageManager, requestBody.PackageName)
		if err == nil && len(existingServices) > 0 {
			mcpServiceID := existingServices[0].ID
			if err := addServiceInstanceForUser(c, userID, mcpServiceID, requestBody.UserProvidedEnvVars); err != nil {
				common.RespError(c, http.StatusInternalServerError, i18n.Translate("add_service_instance_failed", lang), err)
				return
			}
			common.RespSuccess(c, gin.H{
				"message":        i18n.Translate("service_instance_added_successfully", lang),
				"mcp_service_id": mcpServiceID,
				"status":         "already_installed_instance_added",
			})
			return
		}

		// New package, create MCPService, then submit installation task
		displayName := requestBody.DisplayName
		if displayName == "" {
			displayName = requestBody.PackageName
		}

		// 1. 检查必需环境变量（如 FIRECRAWL_API_KEY）是否齐全
		var requiredEnvVars []string
		switch requestBody.PackageManager {
		case "npm":
			details, err := market.GetNPMPackageDetails(c.Request.Context(), requestBody.PackageName)
			if err == nil {
				readme, _ := market.GetNPMPackageReadme(c.Request.Context(), requestBody.PackageName)
				mcpConfig, _ := market.ExtractMCPConfig(details, readme)
				if mcpConfig != nil {
					requiredEnvVars = market.GetEnvVarsFromMCPConfig(mcpConfig)
				}
				if len(requiredEnvVars) == 0 {
					requiredEnvVars = market.GuessMCPEnvVarsFromReadme(readme)
				}
				if len(details.RequiresEnv) > 0 {
					for _, env := range details.RequiresEnv {
						if !contains(requiredEnvVars, env) {
							requiredEnvVars = append(requiredEnvVars, env)
						}
					}
				}
			}
		case "pypi", "uv", "pip":
			// TODO: PyPI 包类似处理
		}
		// 检查 user_provided_env_vars 是否齐全
		var missingEnvVars []string
		for _, env := range requiredEnvVars {
			if env == "" {
				continue
			}
			if _, ok := envVarsForTask[env]; !ok {
				missingEnvVars = append(missingEnvVars, env)
			}
		}
		if len(missingEnvVars) > 0 {
			msg := "缺少必需环境变量: " + strings.Join(missingEnvVars, ", ")
			c.JSON(http.StatusOK, common.APIResponse{
				Success: true,
				Message: msg,
				Data: gin.H{
					"required_env_vars": missingEnvVars,
				},
			})
			return
		}

		newService := model.MCPService{
			Name:                  requestBody.PackageName,
			DisplayName:           displayName,
			Description:           requestBody.ServiceDescription,
			Category:              requestBody.Category,
			Icon:                  requestBody.ServiceIconURL,
			Type:                  model.ServiceTypeStdio,
			PackageManager:        requestBody.PackageManager,
			SourcePackageName:     requestBody.PackageName,
			ClientConfigTemplates: "{}",
			Enabled:               false,
			HealthStatus:          string(market.StatusPending),
		}
		if newService.Category == "" {
			newService.Category = model.CategoryAI
		}

		if err := model.CreateService(&newService); err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("create_mcp_service_failed", lang), err)
			return
		}

		for envName := range envVarsForTask {
			configServiceEntry := model.ConfigService{
				ServiceID:   newService.ID,
				Key:         envName,
				DisplayName: envName,
				Description: fmt.Sprintf("Environment variable %s for %s", envName, newService.DisplayName),
				Type:        model.ConfigTypeString,
				Required:    true,
			}
			if strings.Contains(strings.ToLower(envName), "token") || strings.Contains(strings.ToLower(envName), "key") || strings.Contains(strings.ToLower(envName), "secret") {
				configServiceEntry.Type = model.ConfigTypeSecret
			}
			if err := model.CreateConfigOption(&configServiceEntry); err != nil {
				log.Printf("Error creating ConfigService for %s (MCPService ID %d): %v", envName, newService.ID, err)
			}
		}

		installationTask := market.InstallationTask{
			ServiceID:      newService.ID,
			UserID:         userID,
			PackageName:    requestBody.PackageName,
			PackageManager: requestBody.PackageManager,
			Version:        requestBody.Version,
			EnvVars:        envVarsForTask,
		}

		market.GetInstallationManager().SubmitTask(installationTask)

		common.RespSuccess(c, gin.H{
			"message":        i18n.Translate("installation_submitted", lang),
			"mcp_service_id": newService.ID,
			"task_id":        newService.ID,
			"status":         market.StatusPending,
		})
	} else {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_source_type", lang))
	}
}

// GetInstallationStatus godoc
// @Summary 获取安装状态
// @Description 获取指定服务的安装状态
// @Tags Market
// @Accept json
// @Produce json
// @Param service_id query int true "服务ID"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/installation_status [get]
func GetInstallationStatus(c *gin.Context) {
	lang := c.GetString("lang")
	serviceIDStr := c.Query("service_id")

	serviceID, err := strconv.ParseInt(serviceIDStr, 10, 64)
	if err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang), err)
		return
	}

	// 获取安装管理器
	installationManager := market.GetInstallationManager()

	// 获取任务状态
	task, exists := installationManager.GetTaskStatus(serviceID)
	if !exists {
		// 如果任务不存在，尝试从服务状态获取信息
		service, err := model.GetServiceByID(serviceID)
		if err != nil {
			common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
			return
		}

		// 如果服务存在且已安装
		var status string
		if service.InstalledVersion == "installing" {
			status = "installing"
		} else if service.InstalledVersion != "" {
			status = "completed"
		} else {
			status = "unknown"
		}

		response := map[string]interface{}{
			"service_id":   service.ID,
			"service_name": service.Name,
			"status":       status,
		}

		common.RespSuccess(c, response)
		return
	}

	// 构建响应
	response := map[string]interface{}{
		"service_id":   task.ServiceID,
		"package_name": task.PackageName,
		"status":       task.Status,
		"start_time":   task.StartTime,
	}

	if task.Status == market.StatusCompleted || task.Status == market.StatusFailed {
		response["end_time"] = task.EndTime
		response["duration"] = task.EndTime.Sub(task.StartTime).Seconds()

		if task.Status == market.StatusFailed {
			response["error"] = task.Error
		}
	}

	common.RespSuccess(c, response)
}

// UninstallService godoc
// @Summary 卸载服务
// @Description 卸载指定的服务
// @Tags Market
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "请求体"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/uninstall [post]
func UninstallService(c *gin.Context) {
	lang := c.GetString("lang")
	var requestBody map[string]interface{}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}

	// 获取参数，支持通过configServiceID或packageName+packageManager卸载
	_, hasConfigServiceID := requestBody["config_service_id"].(float64)
	packageName, hasPackageName := requestBody["package_name"].(string)
	packageManager, hasPackageManager := requestBody["package_manager"].(string)

	if !hasConfigServiceID && (!hasPackageName || !hasPackageManager) {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_uninstall_params", lang))
		return
	}

	// 如果有 config_service_id，尝试获取相关服务信息
	var serviceID int64

	if hasConfigServiceID {
		// TODO: 当ConfigService模型完成后，这里需要从ConfigService获取MCPService信息
		// configServiceID := int64(configServiceIDFloat)
		// configService, err := model.GetConfigServiceByID(configServiceID)
		// if err != nil {...}
		// service, err := model.GetServiceByID(configService.ServiceID)
		// packageName = service.SourcePackageName
		// ...

		// 临时方案：暂时只处理packageName+packageManager
		common.RespErrorStr(c, http.StatusNotImplemented, "卸载通过config_service_id尚未实现，请使用package_name和package_manager")
		return
	} else {
		// 通过packageName和packageManager查找服务
		services, err := model.GetServicesByPackageDetails(packageManager, packageName)
		if err != nil || len(services) == 0 {
			common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
			return
		}

		serviceID = services[0].ID
	}

	// 卸载服务
	if packageManager == "npm" {
		if err := market.UninstallNPMPackage(packageName); err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("uninstall_failed", lang), err)
			return
		}
	} else {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("unsupported_package_manager", lang))
		return
	}

	// 标记服务为禁用
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		log.Printf("Warning: Could not get service with ID %d: %v", serviceID, err)
	} else {
		service.Enabled = false
		service.HealthStatus = "unknown"
		if err := model.UpdateService(service); err != nil {
			log.Printf("Warning: Could not update service status: %v", err)
		}
	}

	// 返回成功
	common.RespSuccessStr(c, i18n.Translate("service_uninstalled_successfully", lang))
}

// 辅助函数

// addServiceInstanceForUser adds or updates UserConfig entries for a given user and MCPService.
// It now also ensures that ConfigService entries exist for each provided environment variable.
func addServiceInstanceForUser(c *gin.Context, userID int64, serviceID int64, userProvidedEnvVars map[string]interface{}) error {
	lang := c.GetString("lang")
	if userID == 0 {
		// If userID is 0, it could be an admin setting up a predefined service without a specific user context,
		// or an unauthenticated call that shouldn't have reached here for marketplace type.
		// For now, if no user, we can't save UserConfig. This might need further role-based handling.
		// If serviceID is for a predefined service, maybe no UserConfig is needed, or it's a global setting.
		// This function's primary role is for user-specific instances. If userID is 0, perhaps it should skip UserConfig creation.
		// However, the call from "predefined" path passes userID which might be 0 for admin actions.
		// Let's assume for now that if userID is 0, we don't save UserConfigs.
		// A more robust solution would be to check roles or have separate functions.
		log.Printf("addServiceInstanceForUser called with userID 0 for serviceID %d. No UserConfig will be saved.", serviceID)
		// We still might want to ensure ConfigService entries exist if that's a general setup step.
		// For now, let's return nil if userID is 0, implying no user-specific action is taken.
		return nil // Or handle as an error if UserConfig is always expected.
	}

	mcpService, err := model.GetServiceByID(serviceID)
	if err != nil {
		return errors.New(i18n.Translate("service_not_found", lang))
	}

	convertedEnvVars := convertEnvVarsMap(userProvidedEnvVars)

	for key, value := range convertedEnvVars {
		configOption, err := model.GetConfigOptionByKey(serviceID, key)
		if err != nil {
			if err.Error() == model.ErrRecordNotFound.Error() || err.Error() == "config_service_not_found" || strings.Contains(err.Error(), "not found") {
				newConfigOption := model.ConfigService{
					ServiceID:   serviceID,
					Key:         key,
					DisplayName: key,
					Description: fmt.Sprintf("Environment variable %s for %s", key, mcpService.DisplayName),
					Type:        model.ConfigTypeString,
					Required:    true,
				}
				if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "secret") {
					newConfigOption.Type = model.ConfigTypeSecret
				}
				if errCreate := model.CreateConfigOption(&newConfigOption); errCreate != nil {
					log.Printf("Failed to create ConfigService for key %s, serviceID %d: %v", key, serviceID, errCreate)
					return fmt.Errorf(i18n.Translate("failed_to_create_config_option_for_env", lang)+": %s", key)
				}
				configOption = &newConfigOption
			} else {
				log.Printf("Error fetching ConfigService for key %s, serviceID %d: %v", key, serviceID, err)
				return fmt.Errorf(i18n.Translate("failed_to_get_config_option_for_env", lang)+": %s", key)
			}
		}

		userConfig := model.UserConfig{
			UserID:    userID,
			ServiceID: serviceID,
			ConfigID:  configOption.ID,
			Value:     value,
		}
		if err := model.SaveUserConfig(&userConfig); err != nil {
			log.Printf("Failed to save UserConfig for key %s, serviceID %d, userID %d: %v", key, serviceID, userID, err)
			return fmt.Errorf(i18n.Translate("failed_to_save_user_config_for_env", lang)+": %s", key)
		}
	}
	return nil
}

// convertEnvVarsMap converts map[string]interface{} to map[string]string
// This is a temporary helper. Ideally, types should align.
func convertEnvVarsMap(input map[string]interface{}) map[string]string {
	output := make(map[string]string)
	if input == nil {
		return output
	}
	for key, value := range input {
		if strValue, ok := value.(string); ok {
			output[key] = strValue
		} else {
			// Handle or log cases where conversion isn't straightforward if necessary
			log.Printf("Warning: Could not convert env var %s to string", key)
		}
	}
	return output
}

// getInstalledPackages 获取已安装的包列表
func getInstalledPackages() (map[string]bool, error) {
	// 获取所有服务
	services, err := model.GetAllServices()
	if err != nil {
		return nil, err
	}

	// 创建已安装包的映射
	installedPackages := make(map[string]bool)
	for _, service := range services {
		if service.PackageManager != "" && service.SourcePackageName != "" {
			installedPackages[service.SourcePackageName] = true
		}
	}

	return installedPackages, nil
}

// getUserIDFromContext 从上下文中获取用户ID
func getUserIDFromContext(c *gin.Context) int64 {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(int64)
}

// containsSource 检查数据源列表是否包含指定数据源
func containsSource(sources []string, source string) bool {
	for _, s := range sources {
		if s == source {
			return true
		}
	}
	return false
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// SearchMCPMarket godoc
// @Summary 搜索 MCP 市场服务
// @Description 支持从 npm、PyPI、推荐列表聚合搜索
// @Tags Market
// @Accept json
// @Produce json
// @Param query query string false "搜索关键词"
// @Param sources query string false "数据源, 逗号分隔 (npm,pypi,recommended)"
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/search [get]
func SearchMCPMarket(c *gin.Context) {
	ctx := c.Request.Context()
	originalQuery := c.Query("query") // Get original query
	sources := c.DefaultQuery("sources", "npm")
	pageStr := c.Query("page")
	sizeStr := c.Query("size")
	page := 1
	size := 20

	finalQuery := strings.TrimSpace(originalQuery)
	if finalQuery != "" { // Check if original query (after trim) is not empty
		finalQuery = finalQuery + " mcp"
	}

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
		size = s
	}

	var results []market.SearchPackageResult
	var err error

	// 目前仅实现 npm，后续可扩展 pypi/recommended
	if strings.Contains(sources, "npm") {
		// Use finalQuery for searching
		npmResult, e := market.SearchNPMPackages(ctx, finalQuery, size, page)
		if e != nil {
			err = e
		} else {
			// 查询已安装包
			installed, _ := market.GetInstalledMCPServersFromDB()
			installedMap := make(map[string]bool)
			for name := range installed {
				installedMap[name] = true
			}
			results = append(results, market.ConvertNPMToSearchResult(npmResult, installedMap)...)
		}
	}
	// TODO: 支持 pypi、recommended

	if err != nil {
		common.RespError(c, 500, "market_search_failed", err)
		return
	}
	common.RespSuccess(c, results)
}

// ListInstalledMCPServices godoc
// @Summary 列出已安装的 MCP 服务
// @Description 查询数据库中已安装的 MCP 服务
// @Tags Market
// @Accept json
// @Produce json
// @Success 200 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/installed [get]
func ListInstalledMCPServices(c *gin.Context) {
	// 获取所有已启用服务
	services, err := model.GetEnabledServices()
	if err != nil {
		common.RespError(c, 500, "list_installed_failed", err)
		return
	}

	userID := int64(0)
	if uid, exists := c.Get("user_id"); exists {
		userID, _ = uid.(int64)
	}

	var result []map[string]interface{}
	for _, svc := range services {
		// 获取所有环境变量定义
		configs, _ := model.GetConfigOptionsForService(svc.ID)
		// 获取用户配置（如有）
		userConfigs, _ := model.GetUserConfigsForService(userID, svc.ID)
		userConfigMap := map[int64]string{}
		for _, uc := range userConfigs {
			userConfigMap[uc.ConfigID] = uc.Value
		}
		// 组装 env_vars
		envVars := map[string]string{}
		for _, cfg := range configs {
			val := cfg.DefaultValue
			if v, ok := userConfigMap[cfg.ID]; ok && v != "" {
				val = v
			}
			envVars[cfg.Key] = val
		}
		// 转为 map[string]interface{}，并加上 env_vars 字段
		svcMap := map[string]interface{}{}
		b, _ := json.Marshal(svc)
		_ = json.Unmarshal(b, &svcMap)
		svcMap["env_vars"] = envVars
		result = append(result, svcMap)
	}
	common.RespSuccess(c, result)
}

// PatchEnvVar godoc
// @Summary 单独保存服务环境变量
// @Description 更新指定服务的单个环境变量
// @Tags Market
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "请求体"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/env_var [patch]
func PatchEnvVar(c *gin.Context) {
	lang := c.GetString("lang")
	var req struct {
		ServiceID int64  `json:"service_id" binding:"required"`
		VarName   string `json:"var_name" binding:"required"`
		VarValue  string `json:"var_value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}
	userID := getUserIDFromContext(c)
	// 查找变量定义
	configOpt, err := model.GetConfigOptionByKey(req.ServiceID, req.VarName)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("config_option_not_found", lang), err)
		return
	}
	// 查找/保存 UserConfig
	userConfig := &model.UserConfig{
		UserID:    userID,
		ServiceID: req.ServiceID,
		ConfigID:  configOpt.ID,
		Value:     req.VarValue,
	}
	if err := model.SaveUserConfig(userConfig); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("save_user_config_failed", lang), err)
		return
	}
	common.RespSuccessStr(c, i18n.Translate("env_var_saved_successfully", lang))
}
