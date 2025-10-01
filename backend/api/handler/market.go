package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/library/market"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"one-mcp/backend/service"
	"strconv"
	"strings"
	"sync"
	"time"

	"log"

	"github.com/burugo/thing"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// sanitizeURLForDisplay removes sensitive query parameters and fragments from URL
// keeping only scheme, host, port and path for display purposes
func sanitizeURLForDisplay(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// If URL parsing fails, return the original URL
		return rawURL
	}

	// Reconstruct URL without query parameters and fragment
	sanitized := &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   parsedURL.Path,
	}

	return sanitized.String()
}

// sanitizeServiceName ensures the service name is URL-friendly and consistent.
// It trims spaces, replaces whitespace with dashes, collapses multiple dashes,
// converts to lower-case (affecting ASCII letters only) and removes leading/trailing dashes.
// Chinese and other Unicode letters are preserved untouched.
func sanitizeServiceName(raw string) string {
	if raw == "" {
		return ""
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "-", "\t", "-", "\n", "-", "\r", "-", "/", "-")
	name := replacer.Replace(raw)
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	name = strings.ToLower(name)
	return name
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

	if packageName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_name_required", lang))
		return
	}
	if packageManager == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("package_manager_required", lang))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second) // Increased timeout
	defer cancel()

	switch packageManager {
	case "npm":
		baseDetails, err := market.GetNPMPackageDetails(ctx, packageName)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("get_npm_package_details_failed", lang), err)
			return
		}

		readme, _ := market.GetNPMPackageReadme(ctx, packageName)

		// Reverted: Original logic, search with packageName directly for score enrichment
		npmSearchResult, searchErr := market.SearchNPMPackages(ctx, packageName, 1, 1)

		type EnhancedPackageDetails struct {
			Name            string            `json:"name"`
			Version         string            `json:"version"`
			Description     string            `json:"description"`
			Homepage        string            `json:"homepage"`
			RepositoryURL   string            `json:"repository_url"`
			Author          string            `json:"author"`
			Keywords        []string          `json:"keywords"`
			License         string            `json:"license"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
			Stars           int               `json:"stars"`
			Score           float64           `json:"score"`
			LastUpdated     string            `json:"last_updated"`
			Downloads       int               `json:"downloads,omitempty"`
		}

		enhancedDetails := EnhancedPackageDetails{
			Name:            baseDetails.Name,
			Version:         baseDetails.Version,
			Description:     baseDetails.Description,
			Homepage:        baseDetails.Homepage,
			Keywords:        baseDetails.Keywords,
			License:         baseDetails.License,
			Dependencies:    baseDetails.Dependencies,
			DevDependencies: baseDetails.DevDependencies,
		}
		if baseDetails.Repository.URL != "" {
			enhancedDetails.RepositoryURL = baseDetails.Repository.URL
		}

		if searchErr == nil && npmSearchResult != nil && len(npmSearchResult.Objects) > 0 {
			npmObject := npmSearchResult.Objects[0]
			npmPkg := npmObject.Package

			enhancedDetails.Name = npmPkg.Name
			if npmPkg.Version != "" {
				enhancedDetails.Version = npmPkg.Version
			}
			if npmPkg.Description != "" {
				enhancedDetails.Description = npmPkg.Description
			}
			if npmPkg.Links.Homepage != "" {
				enhancedDetails.Homepage = npmPkg.Links.Homepage
			}
			if npmPkg.Links.Repository != "" {
				enhancedDetails.RepositoryURL = npmPkg.Links.Repository
			}
			if npmPkg.Publisher.Username != "" {
				enhancedDetails.Author = npmPkg.Publisher.Username
			} else if len(npmPkg.Maintainers) > 0 {
				enhancedDetails.Author = npmPkg.Maintainers[0].Username
			}
			if npmPkg.Keywords != nil {
				enhancedDetails.Keywords = npmPkg.Keywords
			}
			enhancedDetails.Score = npmObject.Score.Final
			enhancedDetails.Downloads = npmObject.Downloads.Weekly
			enhancedDetails.LastUpdated = npmPkg.Date.Format(time.RFC3339)

			if strings.Contains(enhancedDetails.RepositoryURL, "github.com") {
				owner, repo := market.ParseGitHubRepo(enhancedDetails.RepositoryURL) // Public function
				if owner != "" && repo != "" {
					enhancedDetails.Stars = market.FetchGitHubStars(ctx, owner, repo) // Public function, pass ctx
				}
			}
		} else if searchErr != nil {
			common.SysLog("Error fetching search details for " + packageName + ": " + searchErr.Error())
		}

		isInstalled := false
		var installedServiceID int64
		services, err := model.GetServicesByPackageDetails(packageManager, packageName)
		if err == nil && len(services) > 0 {
			isInstalled = true
			installedServiceID = services[0].ID
		}

		mcpConfig, _ := market.ExtractMCPConfig(baseDetails, readme)

		if isInstalled && mcpConfig != nil {
			userID := getUserIDFromContext(c)
			installedService, serviceErr := model.GetServiceByID(installedServiceID)
			if serviceErr != nil {
				common.SysLog(fmt.Sprintf("Error fetching service details for ID %d: %v", installedServiceID, serviceErr))
			} else {
				// 1. 从 DefaultEnvsJSON 加载默认环境变量
				finalEnvValues := make(map[string]string)
				if installedService.DefaultEnvsJSON != "" {
					if err := json.Unmarshal([]byte(installedService.DefaultEnvsJSON), &finalEnvValues); err != nil {
						common.SysLog(fmt.Sprintf("Error unmarshaling DefaultEnvsJSON for service ID %d: %v", installedServiceID, err))
					}
				}

				// 2. 如果用户已登录，尝试加载并合并UserConfig（用户特定配置应覆盖默认配置）
				if userID != 0 {
					userConfigs, err_uc := model.GetUserConfigsForService(userID, installedServiceID)
					if err_uc == nil {
						serviceConfigOptions, _ := model.GetConfigOptionsForService(installedServiceID)
						configIDToNameMap := make(map[int64]string)
						for _, opt := range serviceConfigOptions {
							configIDToNameMap[opt.ID] = opt.Key
						}
						for _, uc := range userConfigs {
							if varName, ok := configIDToNameMap[uc.ConfigID]; ok {
								finalEnvValues[varName] = uc.Value // 用户特定配置覆盖默认配置
							}
						}
					} else {
						common.SysLog(fmt.Sprintf("Error fetching user configs for service ID %d, user ID %d: %v", installedServiceID, userID, err_uc))
					}
				}

				// 3. 使用 finalEnvValues 更新 mcpConfig
				for serverKey, serverConf := range mcpConfig.MCPServers {
					if serverConf.Env == nil {
						serverConf.Env = make(map[string]string)
					}
					// 首先用 mcp_config 本身的 env (来自 readme/package.json) 作为基础
					// 然后用 finalEnvValues (来自DB的 DefaultEnvsJSON + UserConfig) 覆盖
					for envNameInDB, envValueInDB := range finalEnvValues {
						serverConf.Env[envNameInDB] = envValueInDB
					}
					mcpConfig.MCPServers[serverKey] = serverConf
				}
			}
		}

		// Inline Env Var Discovery Logic
		var discoveredEnvVars []string
		if mcpConfig != nil { // Use the potentially updated mcpConfig
			discoveredEnvVars = market.GetEnvVarsFromMCPConfig(mcpConfig)
		}
		if len(discoveredEnvVars) == 0 && readme != "" {
			discoveredEnvVars = market.GuessMCPEnvVarsFromReadme(readme)
		}
		if baseDetails != nil && len(baseDetails.RequiresEnv) > 0 {
			for _, env := range baseDetails.RequiresEnv {
				found := false
				for _, existingEnv := range discoveredEnvVars {
					if existingEnv == env {
						found = true
						break
					}
				}
				if !found {
					discoveredEnvVars = append(discoveredEnvVars, env)
				}
			}
		}

		var envVarDefinitions []model.EnvVarDefinition
		for _, env := range discoveredEnvVars {
			definition := model.EnvVarDefinition{
				Name:        env,
				Description: "Discovered from package information",
				IsSecret:    strings.Contains(strings.ToLower(env), "token") || strings.Contains(strings.ToLower(env), "key") || strings.Contains(strings.ToLower(env), "secret"),
				Optional:    false,
			}
			envVarDefinitions = append(envVarDefinitions, definition)
		}
		// End Inline Env Var Discovery Logic

		response := map[string]interface{}{
			"details":        enhancedDetails,
			"env_vars":       envVarDefinitions,
			"is_installed":   isInstalled,
			"mcp_config":     mcpConfig,
			"readme":         readme,
			"author":         enhancedDetails.Author,
			"stars":          enhancedDetails.Stars,
			"repository_url": enhancedDetails.RepositoryURL,
			"version_info":   enhancedDetails.Version,
			"last_publish":   enhancedDetails.LastUpdated,
			"downloads":      enhancedDetails.Downloads,
		}

		if isInstalled && installedServiceID > 0 {
			response["installed_service_id"] = installedServiceID
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

// validateAndGetPyPIPackageInfo validates if PyPI package exists and retrieves description info
func validateAndGetPyPIPackageInfo(ctx context.Context, packageName string) (string, error) {
	// Build PyPI API URL
	reqURL := fmt.Sprintf("https://pypi.org/pypi/%s/json", packageName)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to check package: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("package not found in PyPI")
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PyPI API returned error: status code %d", resp.StatusCode)
	}

	// Read response content
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON to get package info
	var result struct {
		Info struct {
			Summary string `json:"summary"`
		} `json:"info"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Info.Summary, nil
}

// extractPackageNameWithoutVersion extracts the package name without version specifier
// Examples: "package@1.0.0" -> "package", "package@latest" -> "package", "package" -> "package"
func extractPackageNameWithoutVersion(packageNameWithVersion string) string {
	// Split by '@' and take the first part
	// Handle scoped packages like "@scope/package@version" -> "@scope/package"
	if strings.HasPrefix(packageNameWithVersion, "@") {
		// Scoped package: @scope/package@version
		parts := strings.SplitN(packageNameWithVersion, "@", 3) // Split into ["", "scope/package", "version"]
		if len(parts) >= 2 {
			return "@" + parts[1] // Return "@scope/package"
		}
		return packageNameWithVersion
	} else {
		// Regular package: package@version
		parts := strings.SplitN(packageNameWithVersion, "@", 2)
		return parts[0]
	}
}

type CustomServiceReq struct {
	Type    model.ServiceType `json:"type"`
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Envs    map[string]string `json:"envs"`
	Headers map[string]string `json:"headers"`
}

type BatchImportTask struct {
	ID          string
	Status      string // e.g., "processing", "completed", "failed"
	Progress    chan ProgressUpdate
	CreatedAt   time.Time
	ServiceData map[string]interface{}
}

type ProgressUpdate struct {
	Name    string              `json:"name"`
	Status  string              `json:"status"` // "success", "skipped", "failed", "done"
	Message string              `json:"message"`
	Summary *BatchImportSummary `json:"summary,omitempty"`
}

type BatchImportSummary struct {
	Success int `json:"success"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

var (
	batchImportTasks = make(map[string]*BatchImportTask)
	tasksMutex       = &sync.Mutex{}
)

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
		Headers             map[string]string      `json:"headers"`                // Optional: for SSE/HTTP services custom headers
		CustomArgs          []string               `json:"custom_args"`            // Optional: for stdio services custom arguments
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

		// Extract package name without version for API calls
		cleanPackageName := extractPackageNameWithoutVersion(requestBody.PackageName)

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

		// Check for existing services using clean package name, but also check exact match
		existingServices, err := model.GetServicesByPackageDetails(requestBody.PackageManager, cleanPackageName)
		if err == nil && len(existingServices) == 0 {
			// Also check for exact match in case the stored package name includes version
			existingServices, err = model.GetServicesByPackageDetails(requestBody.PackageManager, requestBody.PackageName)
		}

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

		// 1. Check if package exists and get required environment variables and description
		var requiredEnvVars []string
		var packageDescription string

		switch requestBody.PackageManager {
		case "npm":
			details, err := market.GetNPMPackageDetails(c.Request.Context(), cleanPackageName)
			if err != nil {
				// Package not found or unable to get package info, return error immediately
				common.RespError(c, http.StatusBadRequest,
					i18n.Translate("package_not_found", lang, requestBody.PackageName), err)
				return
			}
			// Get package description
			packageDescription = details.Description

			readme, _ := market.GetNPMPackageReadme(c.Request.Context(), cleanPackageName)
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
		case "pypi", "uv", "pip":
			// PyPI package validation and get description info
			description, err := validateAndGetPyPIPackageInfo(c.Request.Context(), cleanPackageName)
			if err != nil {
				common.RespError(c, http.StatusBadRequest,
					i18n.Translate("package_not_found", lang, requestBody.PackageName), err)
				return
			}
			packageDescription = description
			// TODO: Implement automatic environment variable discovery for PyPI packages
		}
		// Check if all required environment variables are provided
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
			// Use i18n for the error message
			msg := i18n.Translate("missing_required_env_vars", lang, strings.Join(missingEnvVars, ", "))
			c.JSON(http.StatusOK, common.APIResponse{ // use 200 OK, because this is not an error, but requires user input
				Success: false, // require next action from user
				Message: msg,
				Data: gin.H{
					"required_env_vars": missingEnvVars,
				},
			})
			return
		}

		// Use user-provided description, fallback to package description if empty
		serviceDescription := requestBody.ServiceDescription
		if serviceDescription == "" {
			serviceDescription = packageDescription
		}

		newService := model.MCPService{
			Name:                  sanitizeServiceName(requestBody.PackageName),
			DisplayName:           displayName,
			Description:           serviceDescription,
			Category:              requestBody.Category,
			Icon:                  requestBody.ServiceIconURL,
			Type:                  model.ServiceTypeStdio,
			PackageManager:        requestBody.PackageManager,
			SourcePackageName:     requestBody.PackageName,
			ClientConfigTemplates: "{}",
			Enabled:               true, // 安装时直接启用服务
			HealthStatus:          string(market.StatusPending),
			InstallerUserID:       userID, // 记录安装者
		}
		if newService.Category == "" {
			newService.Category = model.CategoryAI
		}

		// Check if the processed service name already exists
		existingServiceByName, errByName := model.GetServiceByName(newService.Name)
		if errByName == nil && existingServiceByName != nil {
			common.RespErrorStr(c, http.StatusConflict, i18n.Translate("service_name_already_exists", lang, newService.Name))
			return
		}

		// Set Command and ArgsJSON configuration based on package manager
		log.Printf("[InstallOrAddService] Setting Command and ArgsJSON for PackageManager: %s, PackageName: %s, CustomArgs: %v", requestBody.PackageManager, requestBody.PackageName, requestBody.CustomArgs)
		switch requestBody.PackageManager {
		case "npm":
			newService.Command = "npx"
			var args []string
			if len(requestBody.CustomArgs) > 0 {
				// Use custom arguments provided by user
				args = append(args, requestBody.CustomArgs...)
				// Ensure package name is included if not already present
				packageNameFound := false
				for _, arg := range requestBody.CustomArgs {
					if arg == requestBody.PackageName {
						packageNameFound = true
						break
					}
				}
				if !packageNameFound {
					args = append(args, requestBody.PackageName)
				}
			} else {
				// Use default arguments
				args = []string{"-y", requestBody.PackageName}
			}
			argsJSON, err := json.Marshal(args)
			if err != nil {
				log.Printf("[InstallOrAddService] Error marshaling args for npm package %s: %v", requestBody.PackageName, err)
			} else {
				newService.ArgsJSON = string(argsJSON)
				log.Printf("[InstallOrAddService] Set Command='%s' and ArgsJSON='%s' for npm package %s", newService.Command, newService.ArgsJSON, requestBody.PackageName)
			}
		case "pypi", "uv", "pip":
			newService.Command = "uvx"
			var args []string
			if len(requestBody.CustomArgs) > 0 {
				// Use custom arguments provided by user
				args = append(args, requestBody.CustomArgs...)
			} else {
				// Use default arguments
				args = []string{"--from", requestBody.PackageName, requestBody.PackageName}
			}
			argsJSON, err := json.Marshal(args)
			if err != nil {
				log.Printf("[InstallOrAddService] Error marshaling args for python package %s: %v", requestBody.PackageName, err)
			} else {
				newService.ArgsJSON = string(argsJSON)
				log.Printf("[InstallOrAddService] Set Command='%s' and ArgsJSON='%s' for python package %s", newService.Command, newService.ArgsJSON, requestBody.PackageName)
			}
		default:
			log.Printf("[InstallOrAddService] Warning: Unknown package manager %s for service %s, Command field will be empty", requestBody.PackageManager, requestBody.PackageName)
		}

		// Set DefaultEnvsJSON (environment variables during installation as default configuration)
		if len(envVarsForTask) > 0 {
			defaultEnvsJSON, err := json.Marshal(envVarsForTask)
			if err != nil {
				log.Printf("[InstallOrAddService] Error marshaling default envs for service %s: %v", requestBody.PackageName, err)
			} else {
				newService.DefaultEnvsJSON = string(defaultEnvsJSON)
				log.Printf("[InstallOrAddService] Set DefaultEnvsJSON for service %s: %s", requestBody.PackageName, newService.DefaultEnvsJSON)
			}
		}

		// Process Headers if provided
		if len(requestBody.Headers) > 0 {
			headersJSON, err := json.Marshal(requestBody.Headers)
			if err != nil {
				common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_headers", lang), err)
				return
			}
			newService.HeadersJSON = string(headersJSON)
		}

		log.Printf("[InstallOrAddService] About to create service with Command='%s', ArgsJSON='%s', PackageManager='%s'", newService.Command, newService.ArgsJSON, newService.PackageManager)
		if err := model.CreateService(&newService); err != nil {
			log.Printf("[InstallOrAddService] Failed to create service: %v", err)
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("create_mcp_service_failed", lang), err)
			return
		}
		log.Printf("[InstallOrAddService] Successfully created service with ID: %d, Command='%s', ArgsJSON='%s', DefaultEnvsJSON='%s'", newService.ID, newService.Command, newService.ArgsJSON, newService.DefaultEnvsJSON)

		// Note: No longer create ConfigService during installation, as installation environment variables are default configuration
		// ConfigService is only created dynamically when users need personal configuration

		// Parse ArgsJSON to get command arguments
		var args []string
		if newService.ArgsJSON != "" {
			if err := json.Unmarshal([]byte(newService.ArgsJSON), &args); err != nil {
				log.Printf("[InstallOrAddService] Failed to parse ArgsJSON for service %d: %v", newService.ID, err)
				args = []string{} // Use empty args on parse error
			}
		}

		installationTask := market.InstallationTask{
			ServiceID:      newService.ID,
			UserID:         userID,
			PackageName:    requestBody.PackageName,
			PackageManager: requestBody.PackageManager,
			Version:        requestBody.Version,
			Command:        newService.Command,
			Args:           args,
			EnvVars:        envVarsForTask,
		}

		log.Printf("[InstallOrAddService] About to submit installation task for ServiceID=%d, Package=%s, Manager=%s, Version=%s, EnvVars=%v",
			newService.ID, requestBody.PackageName, requestBody.PackageManager, requestBody.Version, envVarsForTask)

		market.GetInstallationManager().SubmitTask(installationTask)

		log.Printf("[InstallOrAddService] Installation task submitted successfully for ServiceID=%d", newService.ID)

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
	serviceIDStr := c.Param("id")
	if serviceIDStr == "" {
		serviceIDStr = c.Query("service_id")
	}
	if serviceIDStr == "" {
		common.RespErrorStr(c, http.StatusNotFound, "service_id required")
		return
	}
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
// @Param body body struct{ ServiceID int64 `json:"service_id" binding:"required"` } true "请求体，包含 service_id"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 404 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/uninstall [post]
func UninstallService(c *gin.Context) {
	lang := c.GetString("lang")
	var requestBody struct {
		ServiceID int64 `json:"service_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang)+": "+i18n.Translate("service_id_required", lang), err)
		return
	}

	if requestBody.ServiceID == 0 {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_service_id", lang))
		return
	}

	serviceID := requestBody.ServiceID

	// 获取服务详情
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
	}

	// 检查是否是处于安装中的服务
	isPendingOrInstalling := false
	if service.InstalledVersion == "" || service.InstalledVersion == "installing" {
		// 进一步检查安装任务状态
		installationManager := market.GetInstallationManager()
		if task, exists := installationManager.GetTaskStatus(service.ID); exists {
			if task.Status == market.StatusPending || task.Status == market.StatusInstalling {
				isPendingOrInstalling = true
				log.Printf("[UninstallService] Service ID %d is in %s state, will skip physical uninstall and proceed with soft delete only", service.ID, task.Status)
			}
		} else if service.InstalledVersion == "" {
			// 没有安装任务但也没有安装版本，可能是之前失败的安装遗留
			isPendingOrInstalling = true
			log.Printf("[UninstallService] Service ID %d has no installed version and no running task, treating as pending installation - will skip physical uninstall", service.ID)
		}
	}

	// 对于非安装中的服务，进行 ServiceManager 注销（适用于所有类型）
	if !isPendingOrInstalling {
		log.Printf("[UninstallService] Attempting to unregister service ID %d (Name: %s) from ServiceManager", service.ID, service.Name)
		serviceManager := proxy.GetServiceManager()

		// 创建一个专门的超时上下文，给予足够的时间进行清理
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := serviceManager.UnregisterService(ctx, service.ID); err != nil {
			// 非注册状态也不必视为致命错误
			if err == proxy.ErrServiceNotFound {
				log.Printf("[UninstallService] Service ID %d was not registered. Continuing.", service.ID)
			} else {
				log.Printf("[UninstallService] Error unregistering service ID %d from ServiceManager: %v.", service.ID, err)
				// 如果是超时错误，我们跳过物理卸载，直接进行软删除
				if errors.Is(err, context.DeadlineExceeded) {
					log.Printf("[UninstallService] Unregistration timed out. Skipping physical uninstall.")
					isPendingOrInstalling = true // Treat as pending to skip physical uninstall
				}
			}
		} else {
			log.Printf("[UninstallService] Successfully unregistered service ID %d from ServiceManager.", service.ID)
		}
		// 无论注销是否成功，都主动删除健康缓存，避免残留状态
		proxy.GetHealthCacheManager().DeleteServiceHealth(service.ID)
	} else {
		log.Printf("[UninstallService] Skipping ServiceManager unregistration for pending/installing service ID %d", service.ID)
	}

	// 对于安装中的服务，跳过物理卸载，直接进行软删除
	if isPendingOrInstalling {
		log.Printf("[UninstallService] Service ID %d is pending/installing, skipping physical package uninstall", service.ID)
	} else {
		// 卸载服务 - 根据 PackageManager 调用相应的卸载逻辑
		// 注意: 只有 stdio 类型且有 PackageManager 的服务才涉及物理卸载
		// SSE/HTTP 类型服务通常没有物理包卸载步骤，主要是DB记录的清理
		if service.Type == model.ServiceTypeStdio && service.PackageManager != "" && service.SourcePackageName != "" {
			switch service.PackageManager {
			case "npm":
				if err := market.UninstallNPMPackage(service.SourcePackageName); err != nil {
					// Log error but proceed to mark as uninstalled, as it might be partially uninstalled or FS issues
					log.Printf("Error during npm uninstall for service ID %d (%s): %v", serviceID, service.SourcePackageName, err)
					// common.RespError(c, http.StatusInternalServerError, i18n.Translate("uninstall_failed", lang), err)
					// return // Decide if this should be a hard stop or a soft failure for DB cleanup
				}
			case "uv", "pypi", "pip": // Assuming uv, pypi, pip might use a similar mechanism
				// Assuming UninstallPyPIPackage exists and works similarly for these
				if err := market.UninstallPyPIPackage(c.Request.Context(), service.SourcePackageName); err != nil {
					log.Printf("Error during pypi/uv/pip uninstall for service ID %d (%s): %v", serviceID, service.SourcePackageName, err)
				}
			default:
				log.Printf("Uninstall requested for service ID %d (%s) with unsupported package manager: %s. Skipping physical uninstall.", serviceID, service.SourcePackageName, service.PackageManager)
			}
		} else {
			log.Printf("Service ID %d is not a stdio type with a package manager, or SourcePackageName is empty. Skipping physical uninstall.", serviceID)
		}
	}

	// 标记服务为软删除 (or hard delete if preferred)
	// Current logic from GetServiceByID already fetched the service
	service.Enabled = false // Explicitly disable
	service.Deleted = true
	service.HealthStatus = "unknown"
	service.InstalledVersion = "" // Clear installed version
	if err := model.UpdateService(service); err != nil {
		log.Printf("Warning: Could not update service (ID: %d) status to deleted: %v", serviceID, err)
		// Even if DB update fails, if physical uninstall happened, it's a partial success.
		// However, for the user, the service might still appear.
		// Consider if a more robust transaction/rollback is needed if this is critical.
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("update_service_status_failed", lang), err)
		return
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
			// 查询已安装包的 numeric IDs
			installedServiceIDs, err_installed := market.GetInstalledMCPServersFromDB() // Returns map[string]int64 now
			if err_installed != nil {
				common.SysLog("SearchMCPMarket: Error fetching installed server IDs: " + err_installed.Error())
				// Continue without installed info if this fails, or handle error more strictly
			}
			results = append(results, market.ConvertNPMToSearchResult(ctx, npmResult, installedServiceIDs)...)
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
	// 获取所有已安装服务（不论启用状态）
	services, err := model.GetInstalledServices()
	if err != nil {
		common.RespError(c, 500, "list_installed_failed", err)
		return
	}

	userID := int64(0)
	if uid, exists := c.Get("user_id"); exists {
		userID, _ = uid.(int64)
	}

	// 获取缓存管理器
	cacheManager := proxy.GetHealthCacheManager()

	var result []map[string]interface{}
	for _, svc := range services {
		// 1. 从 DefaultEnvsJSON 加载默认环境变量
		finalEnvVars := make(map[string]string)
		if svc.DefaultEnvsJSON != "" {
			if err := json.Unmarshal([]byte(svc.DefaultEnvsJSON), &finalEnvVars); err != nil {
				common.SysLog(fmt.Sprintf("Error unmarshaling DefaultEnvsJSON for service ID %d: %v", svc.ID, err))
			}
		}

		// 2. 如果用户已登录，获取并合并 UserConfig
		if userID != 0 {
			userConfigs, err_uc := model.GetUserConfigsForService(userID, svc.ID)
			if err_uc == nil {
				serviceConfigOptions, _ := model.GetConfigOptionsForService(svc.ID)
				configIDToNameMap := make(map[int64]string)
				for _, opt := range serviceConfigOptions {
					configIDToNameMap[opt.ID] = opt.Key
				}
				for _, uc := range userConfigs {
					if varName, ok := configIDToNameMap[uc.ConfigID]; ok {
						finalEnvVars[varName] = uc.Value // 用户特定配置覆盖默认配置
					}
				}
			} else {
				common.SysLog(fmt.Sprintf("Error fetching user configs for service ID %d, user ID %d: %v", svc.ID, userID, err_uc))
			}
		}

		// 组装结果
		svcMap := make(map[string]interface{})
		b, _ := json.Marshal(svc)
		_ = json.Unmarshal(b, &svcMap)
		svcMap["env_vars"] = finalEnvVars // 使用合并后的环境变量

		// 添加用户今日请求统计
		if svc.RPDLimit > 0 && userID > 0 {
			// 获取用户今日请求数
			today := time.Now().Format("2006-01-02")
			userCacheKey := fmt.Sprintf("user_request:%s:%d:%d:count", today, svc.ID, userID)

			cacheClient := thing.Cache()
			if cacheClient != nil {
				ctx := context.Background()
				countStr, err := cacheClient.Get(ctx, userCacheKey)
				if err == nil {
					if userRequestCount, parseErr := strconv.ParseInt(countStr, 10, 64); parseErr == nil {
						svcMap["user_daily_request_count"] = userRequestCount
						svcMap["remaining_requests"] = int64(svc.RPDLimit) - userRequestCount
					} else {
						svcMap["user_daily_request_count"] = 0
						svcMap["remaining_requests"] = int64(svc.RPDLimit)
					}
				} else {
					// 缓存键不存在，说明今天还没有请求
					svcMap["user_daily_request_count"] = 0
					svcMap["remaining_requests"] = int64(svc.RPDLimit)
				}
			} else {
				svcMap["user_daily_request_count"] = 0
				svcMap["remaining_requests"] = int64(svc.RPDLimit)
			}
		} else {
			svcMap["user_daily_request_count"] = 0
			svcMap["remaining_requests"] = -1 // -1 表示无限制
		}

		// 尝试从缓存获取健康状态
		if cachedHealth, found := cacheManager.GetServiceHealth(svc.ID); found {
			// 使用缓存中的健康状态
			svcMap["health_status"] = string(cachedHealth.Status)
			if !cachedHealth.LastChecked.IsZero() {
				svcMap["last_health_check"] = cachedHealth.LastChecked.Format(time.RFC3339)
			} else {
				svcMap["last_health_check"] = nil
			}

			// 构造 health_details JSON
			healthDetailsMap := map[string]interface{}{
				"status":           string(cachedHealth.Status),
				"response_time_ms": cachedHealth.ResponseTime,
				"success_count":    cachedHealth.SuccessCount,
				"failure_count":    cachedHealth.FailureCount,
				"error_message":    cachedHealth.ErrorMessage,
				"start_time":       cachedHealth.StartTime,
				"up_time":          cachedHealth.UpTime,
				"warning_level":    cachedHealth.WarningLevel,
			}
			if !cachedHealth.LastChecked.IsZero() {
				healthDetailsMap["last_checked"] = cachedHealth.LastChecked.Format(time.RFC3339)
			} else {
				healthDetailsMap["last_checked"] = nil
			}

			if healthDetailsJSON, err_hd_marshal := json.Marshal(healthDetailsMap); err_hd_marshal == nil {
				svcMap["health_details"] = string(healthDetailsJSON)
			} else {
				log.Printf("Error marshalling health_details for service ID %d: %v", svc.ID, err_hd_marshal)
				svcMap["health_details"] = "{}" // Default to empty JSON object on error
			}
		} else {
			// 缓存未命中，返回 "unknown" 状态
			svcMap["health_status"] = string(proxy.StatusUnknown)
			svcMap["last_health_check"] = nil
			unknownHealthDetails := map[string]interface{}{
				"status":        string(proxy.StatusUnknown),
				"last_checked":  nil,
				"error_message": "Health status not found in cache.",
			}
			if healthDetailsJSON, err_unknown_marshal := json.Marshal(unknownHealthDetails); err_unknown_marshal == nil {
				svcMap["health_details"] = string(healthDetailsJSON)
			} else {
				log.Printf("Error marshalling unknown health_details for service ID %d: %v", svc.ID, err_unknown_marshal)
				svcMap["health_details"] = "{\"status\": \"unknown\"}" // Minimal fallback
			}
		}
		result = append(result, svcMap)
	}
	common.RespSuccess(c, result)
}

// PatchEnvVar godoc
// @Summary 单独保存服务环境变量
// @Description 更新指定服务的单个环境变量。管理员修改会更新服务默认配置，普通用户修改会保存为个人配置
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
	if userID == 0 {
		common.RespErrorStr(c, http.StatusUnauthorized, i18n.Translate("user_not_authenticated", lang))
		return
	}

	// 检查用户权限
	user, err := model.GetUserById(userID, false)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Failed to get user info", err)
		return
	}

	isAdmin := user.Role == common.RoleAdminUser

	if isAdmin {
		// 管理员：更新服务的默认环境变量配置
		service, err := model.GetServiceByID(req.ServiceID)
		if err != nil {
			common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
			return
		}

		// 解析现有的默认环境变量
		var defaultEnvs map[string]string
		if service.DefaultEnvsJSON != "" {
			if err := json.Unmarshal([]byte(service.DefaultEnvsJSON), &defaultEnvs); err != nil {
				log.Printf("[PatchEnvVar] Error unmarshaling existing DefaultEnvsJSON for service %d: %v", req.ServiceID, err)
				defaultEnvs = make(map[string]string)
			}
		} else {
			defaultEnvs = make(map[string]string)
		}

		// 更新指定的环境变量
		defaultEnvs[req.VarName] = req.VarValue

		// 重新序列化并保存
		defaultEnvsJSON, err := json.Marshal(defaultEnvs)
		if err != nil {
			common.RespError(c, http.StatusInternalServerError, "Failed to marshal default envs", err)
			return
		}

		service.DefaultEnvsJSON = string(defaultEnvsJSON)
		if err := model.UpdateService(service); err != nil {
			common.RespError(c, http.StatusInternalServerError, "Failed to update service", err)
			return
		}

		log.Printf("[PatchEnvVar] Admin user %d updated default env %s=%s for service %d (%s)", userID, req.VarName, req.VarValue, service.ID, service.Name)
		common.RespSuccessStr(c, "Default environment variable updated successfully")

	} else {
		// 普通用户：保存为个人配置
		// 查找或创建变量定义
		configOpt, err := model.GetConfigOptionByKey(req.ServiceID, req.VarName)
		if err != nil {
			if err.Error() == model.ErrRecordNotFound.Error() || err.Error() == "config_service_not_found" || strings.Contains(err.Error(), "not found") {
				// 如果ConfigService不存在，创建一个
				service, serviceErr := model.GetServiceByID(req.ServiceID)
				if serviceErr != nil {
					common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), serviceErr)
					return
				}

				newConfigOption := model.ConfigService{
					ServiceID:   req.ServiceID,
					Key:         req.VarName,
					DisplayName: req.VarName,
					Description: fmt.Sprintf("Environment variable %s for %s", req.VarName, service.DisplayName),
					Type:        model.ConfigTypeString,
					Required:    true,
				}
				if strings.Contains(strings.ToLower(req.VarName), "token") || strings.Contains(strings.ToLower(req.VarName), "key") || strings.Contains(strings.ToLower(req.VarName), "secret") {
					newConfigOption.Type = model.ConfigTypeSecret
				}
				if errCreate := model.CreateConfigOption(&newConfigOption); errCreate != nil {
					log.Printf("Failed to create ConfigService for key %s, serviceID %d: %v", req.VarName, req.ServiceID, errCreate)
					common.RespError(c, http.StatusInternalServerError, "Failed to create config option", errCreate)
					return
				}
				configOpt = &newConfigOption
			} else {
				common.RespError(c, http.StatusInternalServerError, "Failed to get config option", err)
				return
			}
		}

		// 保存用户配置
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

		log.Printf("[PatchEnvVar] User %d saved personal env %s=%s for service %d", userID, req.VarName, req.VarValue, req.ServiceID)
		common.RespSuccessStr(c, i18n.Translate("env_var_saved_successfully", lang))
	}
}

// CreateCustomService godoc
// @Summary 创建自定义服务
// @Description 创建一个自定义的MCP服务（支持stdio、sse、streamableHttp类型）
// @Tags Market
// @Accept json
// @Produce json
// @Param body body CustomServiceRequest true "自定义服务请求"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse
// @Failure 400 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_market/custom_service [post]
func CreateCustomService(c *gin.Context) {
	lang := c.GetString("lang")

	type CustomServiceRequest struct {
		Name         string `json:"name" binding:"required"`
		Type         string `json:"type" binding:"required"` // stdio, sse, streamableHttp
		Command      string `json:"command"`
		Arguments    string `json:"arguments"`
		Environments string `json:"environments"`
		URL          string `json:"url"`
		Headers      string `json:"headers"`
	}

	var requestBody CustomServiceRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}

	// 清理和验证服务名称
	sanitizedName := sanitizeServiceName(requestBody.Name)
	if sanitizedName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("service_name_cannot_be_empty", lang))
		return
	}

	// 检查服务名称唯一性
	existingService, err := model.GetServiceByName(sanitizedName)
	if err == nil && existingService != nil {
		common.RespErrorStr(c, http.StatusConflict, i18n.Translate("service_name_already_exists", lang, sanitizedName))
		return
	}

	// 验证服务类型
	var serviceType model.ServiceType
	switch requestBody.Type {
	case "stdio":
		serviceType = model.ServiceTypeStdio
	case "sse":
		serviceType = model.ServiceTypeSSE
	case "streamableHttp": // 前端发送的是 streamableHttp
		serviceType = model.ServiceTypeStreamableHTTP
	default:
		common.RespErrorStr(c, http.StatusBadRequest, "无效的服务类型")
		return
	}

	// 生成服务描述
	var description string
	serviceTypeForDisplay := strings.ToLower(requestBody.Type) // Use the raw string from request

	switch requestBody.Type { // Compare with raw string type from request
	case "stdio":
		cmdDisplay := requestBody.Command
		if len(cmdDisplay) > 50 {
			cmdDisplay = cmdDisplay[:47] + "..."
		}
		argsDisplay := requestBody.Arguments // This is a string
		if argsDisplay == "" {
			argsDisplay = "no args"
		} else if len(argsDisplay) > 30 {
			argsDisplay = argsDisplay[:27] + "..."
		}
		description = fmt.Sprintf("%s/%s (%s)", cmdDisplay, argsDisplay, serviceTypeForDisplay)
	case "sse", "streamableHttp":
		if requestBody.URL == "" {
			description = fmt.Sprintf("Custom proxy service - URL not set (%s)", serviceTypeForDisplay)
		} else {
			// Use sanitized URL to remove sensitive query parameters
			cleanURL := sanitizeURLForDisplay(requestBody.URL)
			if len(cleanURL) > 80 {
				cleanURL = cleanURL[:77] + "..."
			}
			description = fmt.Sprintf("Custom proxy service to %s (%s)", cleanURL, serviceTypeForDisplay)
		}
	default:
		// This case should ideally not be reached due to prior validation of requestBody.Type
		description = fmt.Sprintf("Custom service (%s)", serviceTypeForDisplay)
	}

	// 创建新服务
	newService := model.MCPService{
		Name:                  sanitizedName,    // 使用清理后的名称作为系统标识符
		DisplayName:           requestBody.Name, // 保留原始名称作为显示名称
		Description:           description,      // 使用新的动态描述
		Category:              model.CategoryUtil,
		Type:                  serviceType, // Use the model.ServiceType constant
		ClientConfigTemplates: "{}",
		Enabled:               true, // 自定义服务默认启用
		HealthStatus:          "unknown",
	}

	// 处理不同类型的配置
	if requestBody.Type == "stdio" {
		newService.Command = requestBody.Command

		// 处理参数
		if requestBody.Arguments != "" {
			args := strings.Split(strings.ReplaceAll(requestBody.Arguments, "\r\n", "\n"), "\n")
			var filteredArgs []string
			for _, arg := range args {
				if strings.TrimSpace(arg) != "" {
					filteredArgs = append(filteredArgs, strings.TrimSpace(arg))
				}
			}
			if len(filteredArgs) > 0 {
				argsJSON, err := json.Marshal(filteredArgs)
				if err != nil {
					common.RespError(c, http.StatusBadRequest, "参数格式错误", err)
					return
				}
				newService.ArgsJSON = string(argsJSON)
			}
		}

		// 处理环境变量
		if requestBody.Environments != "" {
			envMap := make(map[string]string)
			envLines := strings.Split(strings.ReplaceAll(requestBody.Environments, "\r\n", "\n"), "\n")
			for _, line := range envLines {
				line = strings.TrimSpace(line)
				if line != "" && strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						envMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
			}
			if len(envMap) > 0 {
				envJSON, err := json.Marshal(envMap)
				if err != nil {
					common.RespError(c, http.StatusBadRequest, "环境变量格式错误", err)
					return
				}
				newService.DefaultEnvsJSON = string(envJSON)
			}
		}
	} else {
		// 对于sse和streamableHttp类型，将URL存储在Command字段
		newService.Command = requestBody.URL

		// 处理Headers
		if requestBody.Headers != "" {
			headersMap := make(map[string]string)
			headerLines := strings.Split(strings.ReplaceAll(requestBody.Headers, "\r\n", "\n"), "\n")
			for _, line := range headerLines {
				line = strings.TrimSpace(line)
				if line != "" && strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						headersMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
			}
			if len(headersMap) > 0 {
				headersJSON, err := json.Marshal(headersMap)
				if err != nil {
					common.RespError(c, http.StatusBadRequest, "请求头格式错误", err)
					return
				}
				newService.HeadersJSON = string(headersJSON)
			}
		}
	}

	// 保存服务到数据库
	if err := model.CreateService(&newService); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("create_mcp_service_failed", lang), err)
		return
	}

	// 自动注册服务到 ServiceManager 以启用健康检查
	serviceManager := proxy.GetServiceManager()
	ctx := c.Request.Context()
	if err := serviceManager.RegisterService(ctx, &newService); err != nil {
		// 记录错误但不让API调用失败，因为服务已经成功创建
		log.Printf("Warning: Failed to register custom service %s (ID: %d) with ServiceManager: %v", newService.Name, newService.ID, err)
		// 在响应中包含警告信息
		common.RespSuccess(c, gin.H{
			"message":        "自定义服务创建成功，但服务注册出现警告",
			"mcp_service_id": newService.ID,
			"service":        newService,
			"warning":        fmt.Sprintf("服务健康检查可能无法正常工作: %v", err),
		})
		return
	}

	log.Printf("Successfully registered custom service %s (ID: %d) with ServiceManager", newService.Name, newService.ID)

	// 注册后立即主动健康检查并刷新数据库状态
	if _, err := serviceManager.ForceCheckServiceHealth(newService.ID); err != nil {
		log.Printf("Warning: Force health check failed for custom service %s (ID: %d): %v", newService.Name, newService.ID, err)
	} else {
		if err := serviceManager.UpdateMCPServiceHealth(newService.ID); err != nil {
			log.Printf("Warning: UpdateMCPServiceHealth failed for custom service %s (ID: %d): %v", newService.Name, newService.ID, err)
		}
	}

	common.RespSuccess(c, gin.H{
		"message":        "自定义服务创建成功",
		"mcp_service_id": newService.ID,
		"service":        newService,
	})
}

func StartBatchImport(c *gin.Context) {
	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format: " + err.Error()})
		return
	}

	// Support both direct format and wrapped format
	var services map[string]interface{}
	if mcpServers, exists := requestBody["mcpServers"]; exists {
		// New wrapped format: {"mcpServers": {...}}
		if mcpServersMap, ok := mcpServers.(map[string]interface{}); ok {
			services = mcpServersMap
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mcpServers field must be an object"})
			return
		}
	} else {
		// Legacy direct format: {"service1": {...}, "service2": {...}}
		services = requestBody
	}

	if len(services) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No services provided"})
		return
	}

	taskID := uuid.New().String()
	task := &BatchImportTask{
		ID:          taskID,
		Status:      "processing",
		Progress:    make(chan ProgressUpdate, 100), // Buffered channel
		CreatedAt:   time.Now(),
		ServiceData: services,
	}

	tasksMutex.Lock()
	batchImportTasks[taskID] = task
	tasksMutex.Unlock()

	go processBatchImport(task)

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
}

func processBatchImport(task *BatchImportTask) {
	defer func() {
		close(task.Progress)
		tasksMutex.Lock()
		delete(batchImportTasks, task.ID)
		tasksMutex.Unlock()
	}()

	summary := &BatchImportSummary{}
	ctx := context.Background()

	for name, serviceData := range task.ServiceData {
		serviceMap, ok := serviceData.(map[string]interface{})
		if !ok {
			summary.Failed++
			task.Progress <- ProgressUpdate{
				Name:    name,
				Status:  "failed",
				Message: "Invalid service data format.",
			}
			continue
		}

		// We will handle transactions inside the creation function
		err := createSingleServiceFromBatch(ctx, name, serviceMap)
		if err != nil {
			if errors.Is(err, ErrServiceExists) {
				summary.Skipped++
				task.Progress <- ProgressUpdate{
					Name:    name,
					Status:  "skipped",
					Message: "Service already exists",
				}
			} else {
				summary.Failed++
				task.Progress <- ProgressUpdate{
					Name:    name,
					Status:  "failed",
					Message: err.Error(),
				}
			}
		} else {
			summary.Success++
			task.Progress <- ProgressUpdate{
				Name:    name,
				Status:  "success",
				Message: "Service imported successfully.",
			}
		}
	}

	task.Status = "completed"
	task.Progress <- ProgressUpdate{
		Status:  "done",
		Summary: summary,
	}
}

// createSingleServiceFromBatch handles the creation of a single service.
// It manages its own transaction.
// Returns: nil for success, ErrServiceExists for skip, other errors for failure
var ErrServiceExists = errors.New("service already exists")

func createSingleServiceFromBatch(ctx context.Context, serviceName string, serviceData map[string]interface{}) error {
	common.SysLog(fmt.Sprintf("Starting batch import for service: %s", serviceName))

	// 1. Sanitize and check for existing service name using Thing ORM
	sanitizedName := sanitizeServiceName(serviceName)
	common.SysLog(fmt.Sprintf("Sanitized name for %s: %s", serviceName, sanitizedName))

	// Use Thing ORM to check for existing service
	mcpServiceThing, err := thing.Use[*model.MCPService]()
	if err != nil {
		err = fmt.Errorf("failed to get Thing ORM instance: %w", err)
		common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
		return err
	}

	// Check if service already exists
	existingServices, err := mcpServiceThing.Query(thing.QueryParams{
		Where: "name = ? AND deleted = ?",
		Args:  []interface{}{sanitizedName, false},
	}).Fetch(0, 1)

	if err != nil {
		err = fmt.Errorf("failed to check for existing service: %w", err)
		common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
		return err
	}

	if len(existingServices) > 0 {
		// Service exists, this should be treated as "skip", not failure
		common.SysLog(fmt.Sprintf("Service '%s' already exists (ID: %d), skipping", sanitizedName, existingServices[0].ID))
		return ErrServiceExists
	}

	common.SysLog(fmt.Sprintf("Service %s does not exist, proceeding with creation", sanitizedName))

	// 2. Map data and determine service type
	var req CustomServiceReq
	if err := mapToCustomServiceReq(serviceData, &req); err != nil {
		err = fmt.Errorf("error mapping service data: %w", err)
		common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
		return err
	}
	req.Name = sanitizedName

	common.SysLog(fmt.Sprintf("Mapped service data for %s: Type=%s, Command=%s, URL=%s", sanitizedName, req.Type, req.Command, req.URL))

	if req.URL != "" {
		// Parse URL to extract path without query parameters
		parsedURL, err := url.Parse(req.URL)
		if err != nil {
			err = fmt.Errorf("invalid URL format: %w", err)
			common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
			return err
		}

		urlPath := parsedURL.Path
		common.SysLog(fmt.Sprintf("Parsed URL path for %s: %s", sanitizedName, urlPath))

		if strings.HasSuffix(urlPath, "/sse") {
			req.Type = model.ServiceTypeSSE
		} else {
			req.Type = model.ServiceTypeStreamableHTTP
		}
		common.SysLog(fmt.Sprintf("Determined service type for %s: %s", sanitizedName, req.Type))
	} else if req.Command != "" {
		req.Type = model.ServiceTypeStdio
		common.SysLog(fmt.Sprintf("Service %s is stdio type with command: %s", sanitizedName, req.Command))
	} else {
		err = errors.New("invalid service definition: must contain 'url' or 'command'")
		common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
		return err
	}

	common.SysLog(fmt.Sprintf("Creating service %s with type %s", sanitizedName, req.Type))

	// Determine package manager based on command and args
	packageManager := "custom"
	sourcePackageName := req.Name
	if req.Command == "npx" && len(req.Args) > 0 {
		// Check if this looks like an npm package
		lastArg := req.Args[len(req.Args)-1]
		if strings.Contains(lastArg, "@") || strings.Contains(lastArg, "/") {
			packageManager = "npm"
			sourcePackageName = lastArg
		}
	} else if req.Command == "uvx" && len(req.Args) > 0 {
		// Check if this looks like a python package
		// Case 1: uvx --from package_name command (args: ["--from", "package_name", "command"])
		for i, arg := range req.Args {
			if arg == "--from" && i+1 < len(req.Args) {
				packageManager = "pypi"
				sourcePackageName = req.Args[i+1]
				break
			}
		}

		// Case 2: uvx package_name [args...] (args: ["package_name", "arg1", "arg2"])
		if packageManager == "custom" {
			// First arg is typically the package name if no --from is used
			firstArg := req.Args[0]
			if firstArg != "" && !strings.HasPrefix(firstArg, "-") {
				packageManager = "pypi"
				sourcePackageName = firstArg
			}
		}
	}

	common.SysLog(fmt.Sprintf("Detected package manager for %s: %s, source package: %s", sanitizedName, packageManager, sourcePackageName))

	// Get real package information if it's from npm or pypi
	var packageDescription string
	var packageVersion string
	switch packageManager {
	case "npm":
		// Extract package name without version for API calls
		cleanPackageName := extractPackageNameWithoutVersion(sourcePackageName)
		if details, err := market.GetNPMPackageDetails(ctx, cleanPackageName); err == nil {
			packageDescription = details.Description
			packageVersion = details.Version
			common.SysLog(fmt.Sprintf("Retrieved npm package info for %s: description=%s, version=%s", cleanPackageName, packageDescription, packageVersion))
		} else {
			common.SysLog(fmt.Sprintf("WARNING: Failed to get npm package details for %s: %v", cleanPackageName, err))
		}
	case "pypi":
		// Extract package name without version
		cleanPackageName := sourcePackageName
		if strings.Contains(cleanPackageName, "==") {
			parts := strings.Split(cleanPackageName, "==")
			cleanPackageName = parts[0]
		}
		if description, err := validateAndGetPyPIPackageInfo(ctx, cleanPackageName); err == nil {
			packageDescription = description
			// PyPI version might need separate call or parsing from sourcePackageName
			packageVersion = "latest"
			common.SysLog(fmt.Sprintf("Retrieved pypi package info for %s: description=%s", cleanPackageName, packageDescription))
		} else {
			common.SysLog(fmt.Sprintf("WARNING: Failed to get pypi package details for %s: %v", cleanPackageName, err))
		}
	}

	// Use retrieved package info or fallbacks
	description := packageDescription
	if description == "" {
		description = "Imported via batch import"
	}

	installedVersion := packageVersion
	if installedVersion == "" {
		installedVersion = "0.0.1"
	}

	common.SysLog(fmt.Sprintf("Using for service %s: description=%s, version=%s", sanitizedName, description, installedVersion))

	// 3. Create the service using Thing ORM instead of raw SQL
	var mcpService model.MCPService
	envsJSON, _ := json.Marshal(req.Envs)

	switch req.Type {
	case model.ServiceTypeStdio:
		if req.Command == "" {
			err = errors.New("missing 'command' for stdio service")
			common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
			return err
		}
		argsJSON, _ := json.Marshal(req.Args)
		common.SysLog(fmt.Sprintf("Creating stdio service %s: command=%s, args=%s", sanitizedName, req.Command, string(argsJSON)))

		mcpService = model.MCPService{
			Name:                  req.Name,
			DisplayName:           req.Name,
			Description:           description,
			Category:              model.CategoryUtil,
			Icon:                  "",
			DefaultOn:             true,
			AdminOnly:             false,
			OrderNum:              0,
			AllowUserOverride:     true,
			ClientConfigTemplates: "{}",
			RequiredEnvVarsJSON:   "[]",
			InstalledVersion:      installedVersion,
			Type:                  req.Type,
			Command:               req.Command,
			ArgsJSON:              string(argsJSON),
			DefaultEnvsJSON:       string(envsJSON),
			Enabled:               true,
			InstallerUserID:       0, // System user for batch import
			PackageManager:        packageManager,
			SourcePackageName:     sourcePackageName,
		}

	case model.ServiceTypeSSE, model.ServiceTypeStreamableHTTP:
		if req.URL == "" {
			err = errors.New("missing 'url' for web service")
			common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
			return err
		}
		headersJSON, _ := json.Marshal(req.Headers)
		common.SysLog(fmt.Sprintf("Creating web service %s: url=%s, headers=%s", sanitizedName, req.URL, string(headersJSON)))

		mcpService = model.MCPService{
			Name:                  req.Name,
			DisplayName:           req.Name,
			Description:           description,
			Category:              model.CategoryUtil,
			Icon:                  "",
			DefaultOn:             true,
			AdminOnly:             false,
			OrderNum:              0,
			AllowUserOverride:     true,
			ClientConfigTemplates: "{}",
			RequiredEnvVarsJSON:   "[]",
			InstalledVersion:      installedVersion,
			Type:                  req.Type,
			Command:               req.URL, // Store URL in Command field
			HeadersJSON:           string(headersJSON),
			DefaultEnvsJSON:       string(envsJSON),
			Enabled:               true,
			InstallerUserID:       0, // System user for batch import
			PackageManager:        packageManager,
			SourcePackageName:     sourcePackageName,
		}

	default:
		err = fmt.Errorf("unsupported service type: %s", req.Type)
		common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
		return err
	}

	// Use Thing ORM to create the service (this will set created_at, updated_at, etc.)
	common.SysLog(fmt.Sprintf("Creating service %s using Thing ORM", sanitizedName))
	if err := model.CreateService(&mcpService); err != nil {
		err = fmt.Errorf("failed to create service using Thing ORM: %w", err)
		common.SysLog(fmt.Sprintf("ERROR for service %s: %v", serviceName, err))
		return err
	}

	common.SysLog(fmt.Sprintf("Successfully created service %s with ID %d", sanitizedName, mcpService.ID))

	// For stdio services, submit installation task asynchronously for batch import
	if mcpService.Type == model.ServiceTypeStdio && mcpService.PackageManager != "custom" {
		// Parse ArgsJSON to get command arguments
		var args []string
		if mcpService.ArgsJSON != "" {
			if err := json.Unmarshal([]byte(mcpService.ArgsJSON), &args); err != nil {
				common.SysLog(fmt.Sprintf("WARNING: Failed to parse ArgsJSON for service %s: %v", sanitizedName, err))
				args = []string{} // Use empty args on parse error
			}
		}

		installationTask := market.InstallationTask{
			ServiceID:      mcpService.ID,
			UserID:         0, // System user for batch import
			PackageName:    mcpService.SourcePackageName,
			PackageManager: mcpService.PackageManager,
			Version:        mcpService.InstalledVersion,
			Command:        mcpService.Command,
			Args:           args,
			EnvVars:        req.Envs,
		}

		common.SysLog(fmt.Sprintf("Submitting installation task for service %s (ID: %d) - async mode for batch import", sanitizedName, mcpService.ID))
		market.GetInstallationManager().SubmitTask(installationTask)

		// For batch import, don't wait for installation completion
		// Installation will happen asynchronously in the background
		common.SysLog(fmt.Sprintf("Service %s (ID: %d) installation submitted, continuing with next service", sanitizedName, mcpService.ID))
	}

	return nil
}

func mapToCustomServiceReq(data map[string]interface{}, req *CustomServiceReq) error {
	if url, ok := data["url"].(string); ok {
		req.URL = url
	}
	if command, ok := data["command"].(string); ok {
		req.Command = command
	}
	if args, ok := data["args"].([]interface{}); ok {
		for _, arg := range args {
			if argStr, ok := arg.(string); ok {
				req.Args = append(req.Args, argStr)
			}
		}
	} else if data["args"] != nil {
		return errors.New("'args' field must be an array of strings")
	}

	if envs, ok := data["envs"].(map[string]interface{}); ok {
		req.Envs = make(map[string]string)
		for k, v := range envs {
			if vStr, ok := v.(string); ok {
				req.Envs[k] = vStr
			}
		}
	} else if data["envs"] != nil {
		return errors.New("'envs' field must be a map of strings")
	}

	// Also support 'env' field for compatibility with mcp.json format
	if env, ok := data["env"].(map[string]interface{}); ok {
		if req.Envs == nil {
			req.Envs = make(map[string]string)
		}
		for k, v := range env {
			if vStr, ok := v.(string); ok {
				req.Envs[k] = vStr
			}
		}
	} else if data["env"] != nil {
		return errors.New("'env' field must be a map of strings")
	}

	if headers, ok := data["headers"].(map[string]interface{}); ok {
		req.Headers = make(map[string]string)
		for k, v := range headers {
			if vStr, ok := v.(string); ok {
				req.Headers[k] = vStr
			}
		}
	} else if data["headers"] != nil {
		return errors.New("'headers' field must be a map of strings")
	}

	return nil
}

func StreamBatchImportProgress(c *gin.Context) {
	taskID := c.Param("task_id")

	// Check authentication via query parameter since SSE doesn't support custom headers
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
		return
	}

	// Validate the token (reuse existing JWT validation logic)
	claims, err := service.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Check if user is admin (similar to AdminAuth middleware)
	if claims.Role < common.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	tasksMutex.Lock()
	task, exists := batchImportTasks[taskID]
	tasksMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	// Listen to the progress channel and send updates to the client
	for update := range task.Progress {
		jsonData, err := json.Marshal(update)
		if err != nil {
			// Log the error but continue if possible
			common.SysLog(fmt.Sprintf("Error marshaling progress update: %v", err))
			continue
		}

		// SSE message format: "data: <json_string>\n\n"
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// When the loop finishes, it means task.Progress was closed, so the task is complete.
	common.SysLog(fmt.Sprintf("SSE stream for task %s finished.", taskID))
}
