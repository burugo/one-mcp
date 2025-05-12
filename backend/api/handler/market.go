package handler

import (
	"context"
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

// SearchMarketPackages godoc
// @Summary 搜索市场包
// @Description 在npm等包管理器中搜索服务包
// @Tags Market
// @Accept json
// @Produce json
// @Param query query string false "搜索关键词"
// @Param sources query string false "数据源，多个值用逗号分隔，例如：npm,pypi"
// @Param limit query int false "每页结果数量"
// @Param page query int false "页码"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/search [get]
func SearchMarketPackages(c *gin.Context) {
	lang := c.GetString("lang")
	query := c.Query("query")
	sourcesStr := c.Query("sources")
	limitStr := c.Query("limit")
	pageStr := c.Query("page")

	// 解析参数
	limit := 20
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100 // 限制最大返回数量
		}
	}

	page := 1
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			page = 1
		}
	}

	// 处理data sources
	sources := []string{"npm"} // 目前只支持npm
	if sourcesStr != "" {
		// TODO: 解析并过滤sources
	}

	// 获取已安装的包列表，用于标记搜索结果
	installedPackages, err := getInstalledPackages()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_installed_packages_failed", lang), err)
		return
	}

	// 结果集
	var combinedResults []market.SearchPackageResult

	// 添加一个超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 搜索NPM包
	if containsSource(sources, "npm") {
		npmResults, err := market.SearchNPMPackages(ctx, query, limit, page)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("search_npm_packages_failed", lang), err)
			return
		}

		// 转换为统一格式
		results := market.ConvertNPMToSearchResult(npmResults, installedPackages)
		combinedResults = append(combinedResults, results...)

		// 构建响应
		response := map[string]interface{}{
			"results":      combinedResults,
			"total":        npmResults.Total,
			"per_page":     npmResults.PerPage,
			"current_page": npmResults.CurrentPage,
			"total_pages":  npmResults.TotalPages,
		}

		common.RespSuccess(c, response)
		return
	}

	// 如果没有搜索任何支持的源
	common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("no_supported_package_source", lang))
}

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
	var requestBody map[string]interface{}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}

	// 获取参数
	sourceType, _ := requestBody["source_type"].(string)

	if sourceType == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("source_type_required", lang))
		return
	}

	// 如果是marketplace类型，先检查npx是否可用
	if sourceType == "marketplace" {
		if !market.CheckNPXAvailable() {
			common.RespErrorStr(c, http.StatusInternalServerError, i18n.Translate("npx_not_available", lang))
			return
		}
	}

	// 获取安装管理器
	installationManager := market.GetInstallationManager()

	// 处理预定义服务
	if sourceType == "predefined" {
		// 获取服务ID
		mcpServiceIDFloat, ok := requestBody["mcp_service_id"].(float64)
		if !ok {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("mcp_service_id_required", lang))
			return
		}
		mcpServiceID := int64(mcpServiceIDFloat)

		// 获取用户提供的环境变量
		userProvidedEnvVars, _ := requestBody["user_provided_env_vars"].(map[string]interface{})

		// 为用户添加服务实例
		err := addServiceInstanceForUser(c, mcpServiceID, userProvidedEnvVars)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("add_service_instance_failed", lang), err)
			return
		}

		common.RespSuccessStr(c, i18n.Translate("service_added_successfully", lang))
		return
	}

	// 处理市场服务
	if sourceType == "marketplace" {
		// 获取包信息
		packageName, _ := requestBody["package_name"].(string)
		packageManager, _ := requestBody["package_manager"].(string)
		version, _ := requestBody["version"].(string)
		userProvidedEnvVars, _ := requestBody["user_provided_env_vars"].(map[string]interface{})

		if packageName == "" {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_name_required", lang))
			return
		}

		if packageManager == "" {
			common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_manager_required", lang))
			return
		}

		// 检查包是否已安装
		services, err := model.GetServicesByPackageDetails(packageManager, packageName)
		if err == nil && len(services) > 0 {
			// 包已安装，直接为用户添加服务实例
			err = addServiceInstanceForUser(c, services[0].ID, userProvidedEnvVars)
			if err != nil {
				common.RespError(c, http.StatusInternalServerError, i18n.Translate("add_service_instance_failed", lang), err)
				return
			}

			common.RespSuccessStr(c, i18n.Translate("service_added_successfully", lang))
			return
		}

		// 包未安装，创建MCPService记录
		ctx := c.Request.Context()

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

		// 发现环境变量
		var envVars []string

		// 首先从MCP配置中提取环境变量
		if mcpConfig != nil {
			envVars = market.GetEnvVarsFromMCPConfig(mcpConfig)
		}

		// 如果MCP配置中没有找到环境变量，则从README中猜测
		if len(envVars) == 0 {
			envVars = market.GuessMCPEnvVarsFromReadme(readme)
		}

		// 创建环境变量定义
		var envVarDefinitions []model.EnvVarDefinition
		for _, env := range envVars {
			definition := model.EnvVarDefinition{
				Name:        env,
				Description: "Auto discovered from package",
				IsSecret:    strings.Contains(strings.ToLower(env), "token") || strings.Contains(strings.ToLower(env), "key") || strings.Contains(strings.ToLower(env), "secret"),
				Optional:    false,
			}
			envVarDefinitions = append(envVarDefinitions, definition)
		}

		// 创建MCPService
		service := &model.MCPService{
			Name:              packageName,
			DisplayName:       details.Name,
			Description:       details.Description,
			Category:          model.CategoryUtil, // 默认分类
			Type:              model.ServiceTypeStdio,
			Enabled:           false, // 初始状态为禁用，安装成功后再启用
			DefaultOn:         true,
			PackageManager:    packageManager,
			SourcePackageName: packageName,
			InstalledVersion:  "installing", // 标记为正在安装
			HealthStatus:      "unknown",    // 使用字符串字面量
		}

		// 设置环境变量
		if len(envVarDefinitions) > 0 {
			if err := service.SetRequiredEnvVars(envVarDefinitions); err != nil {
				common.RespError(c, http.StatusInternalServerError, i18n.Translate("set_env_vars_failed", lang), err)
				return
			}
		}

		// 保存服务
		if err := model.CreateService(service); err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("create_service_failed", lang), err)
			return
		}

		// 为用户添加服务实例
		err = addServiceInstanceForUser(c, service.ID, userProvidedEnvVars)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("add_service_instance_failed", lang), err)
			return
		}

		// 收集用户提供的环境变量
		envVarsForInstall := make(map[string]string)
		if userProvidedEnvVars != nil {
			for key, value := range userProvidedEnvVars {
				if strValue, ok := value.(string); ok {
					envVarsForInstall[key] = strValue
				}
			}
		}

		// 创建安装任务
		task := market.InstallationTask{
			ServiceID:      service.ID,
			PackageName:    packageName,
			PackageManager: packageManager,
			Version:        version,
			EnvVars:        envVarsForInstall,
		}

		// 提交安装任务
		installationManager.SubmitTask(task)

		// 构建响应
		response := map[string]interface{}{
			"message":      i18n.Translate("service_installation_started", lang),
			"service_id":   service.ID,
			"service_name": service.Name,
			"status":       "installing",
		}

		common.RespSuccess(c, response)
		return
	}

	common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_source_type", lang))
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

// GetInstalledServices godoc
// @Summary 获取已安装的服务
// @Description 获取用户已安装的服务列表
// @Tags Market
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/installed [get]
func GetInstalledServices(c *gin.Context) {
	lang := c.GetString("lang")
	// userID未使用，先注释掉
	// userID := getUserIDFromContext(c)

	// TODO: 获取用户已安装的服务
	// 这里需要等待UserConfig和ConfigService模型实现

	// 暂时返回所有已安装的服务
	services, err := model.GetAllServices()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_service_list_failed", lang), err)
		return
	}

	// 筛选出已安装的服务（具有PackageManager和SourcePackageName）
	var installedServices []*model.MCPService
	for _, service := range services {
		if service.PackageManager != "" && service.SourcePackageName != "" {
			installedServices = append(installedServices, service)
		}
	}

	// 使用Thing ORM的ToJSON进行序列化
	jsonBytes, err := model.MCPServiceDB.ToJSON(installedServices)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("serialize_service_failed", lang), err)
		return
	}

	c.Data(http.StatusOK, "application/json", jsonBytes)
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

// addServiceInstanceForUser 为用户添加服务实例
func addServiceInstanceForUser(c *gin.Context, serviceID int64, userProvidedEnvVars map[string]interface{}) error {
	// TODO: 实现为用户添加服务实例的逻辑
	// 这里需要等待UserConfig和ConfigService模型实现
	return nil
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
	userID, exists := c.Get("userID")
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
