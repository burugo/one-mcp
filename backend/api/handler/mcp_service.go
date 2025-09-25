package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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
	oldCommand := service.Command                 // For SSE/HTTP services, this is the URL
	oldDefaultEnvsJSON := service.DefaultEnvsJSON // For stdio services, check env changes
	// Preserve original Command and ArgsJSON before binding, so we can see if user explicitly changed them
	// or if our PackageManager logic should take precedence if they become empty after binding.
	// However, the current logic is that PackageManager dictates Command/ArgsJSON if they are empty.

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
			// If PackageManager or SourcePackageName changes, ArgsJSON might need to be re-evaluated
			// or cleared if it was auto-generated. For now, we rely on the logic below to set it.
		}
	}

	// Set Command and potentially ArgsJSON based on PackageManager
	// This logic applies on update as well, ensuring Command/ArgsJSON are consistent with PackageManager
	if service.PackageManager == "npm" {
		service.Command = "npx"
		if service.ArgsJSON == "" && service.SourcePackageName != "" {
			service.ArgsJSON = fmt.Sprintf(`["-y", "%s"]`, service.SourcePackageName)
		}
	} else if service.PackageManager == "pypi" {
		service.Command = "uvx"
		if service.ArgsJSON == "" && service.SourcePackageName != "" {
			service.ArgsJSON = fmt.Sprintf(`["-y", "%s"]`, service.SourcePackageName)
		}
	} // Add else if for other package managers or if service.PackageManager == "" to potentially clear Command/ArgsJSON if they were auto-set.
	// For now, if PackageManager is not npm or pypi, Command and ArgsJSON remain as bound from request.

	// Check if URL (Command) changed for SSE/HTTP services - need to restart the service
	needsRestart := false
	if (service.Type == model.ServiceTypeSSE || service.Type == model.ServiceTypeStreamableHTTP) &&
		oldCommand != service.Command {
		needsRestart = true
		common.SysLog(fmt.Sprintf("URL changed for %s service %s (ID: %d) from '%s' to '%s', will restart instance",
			service.Type, service.Name, service.ID, oldCommand, service.Command))
	}

	// Check if environment variables changed for stdio services - need to restart the service
	if service.Type == model.ServiceTypeStdio && oldDefaultEnvsJSON != service.DefaultEnvsJSON {
		needsRestart = true
		common.SysLog(fmt.Sprintf("Environment variables changed for stdio service %s (ID: %d), will restart instance. Old: %s, New: %s",
			service.Name, service.ID, oldDefaultEnvsJSON, service.DefaultEnvsJSON))
	}

	// Skip immediate restart preparation - we'll handle everything in background after DB update
	// This avoids blocking the HTTP response
	var needsRestartAfterUpdate = needsRestart

	common.SysLog(fmt.Sprintf("Updating service %s (ID: %d) in database", service.Name, service.ID))
	if err := model.UpdateService(service); err != nil {
		common.SysError(fmt.Sprintf("Failed to update service %s (ID: %d) in database: %v", service.Name, service.ID, err))
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("update_service_failed", lang), err)
		return
	}
	common.SysLog(fmt.Sprintf("Successfully updated service %s (ID: %d) in database", service.Name, service.ID))

	// Restart the service if configuration changed - do everything in background to avoid blocking
	if needsRestartAfterUpdate {
		common.SysLog(fmt.Sprintf("Configuration changed for service %s (ID: %d), starting background restart process", service.Name, service.ID))

		// Handle everything in background to avoid blocking the HTTP response
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			serviceManager := proxy.GetServiceManager()

			// Step 1: Re-fetch fresh configuration from database to ensure we have the latest settings
			freshService, err := model.GetServiceByID(service.ID)
			if err != nil {
				common.SysError(fmt.Sprintf("Failed to re-fetch service %s (ID: %d) from database after configuration change: %v. Restart aborted.", service.Name, service.ID, err))
				return
			}
			common.SysLog(fmt.Sprintf("Re-fetched fresh configuration for service %s (ID: %d) from database. New DefaultEnvsJSON: %s", freshService.Name, freshService.ID, freshService.DefaultEnvsJSON))

			// Step 2: Check if service exists in manager and unregister it to clean up old configuration
			if currentService, err := serviceManager.GetService(service.ID); err == nil && currentService != nil {
				common.SysLog(fmt.Sprintf("Found service %s (ID: %d) in manager, unregistering to clean up old configuration", freshService.Name, freshService.ID))

				// Unregister the old service completely (this stops it and cleans up all caches)
				if err := serviceManager.UnregisterService(ctx, service.ID); err != nil {
					common.SysError(fmt.Sprintf("Failed to unregister service %s (ID: %d) after configuration change: %v. Restart aborted.", freshService.Name, freshService.ID, err))
					return
				}
				common.SysLog(fmt.Sprintf("Successfully unregistered service %s (ID: %d)", freshService.Name, freshService.ID))

				// Step 3: Register the service again with fresh configuration
				// RegisterService will create a new instance with the updated config and start it if enabled
				if err := serviceManager.RegisterService(ctx, freshService); err != nil {
					common.SysError(fmt.Sprintf("Failed to register service %s (ID: %d) with new configuration: %v. Please check system logs for details.", freshService.Name, freshService.ID, err))
				} else {
					common.SysLog(fmt.Sprintf("Successfully registered service %s (ID: %d) with updated configuration", freshService.Name, freshService.ID))
				}
			} else {
				common.SysLog(fmt.Sprintf("Service %s (ID: %d) not found in manager, no restart needed", freshService.Name, freshService.ID))
			}
		}()
	}

	common.RespSuccess(c, service)
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

	wasEnabled := service.Enabled
	if err := model.ToggleServiceEnabled(id); err != nil {
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("toggle_service_status_failed", lang), err)
		return
	}

	updatedService, err := model.GetServiceByID(id)
	if err != nil {
		// Attempt to revert to original state for consistency
		if revertErr := model.ToggleServiceEnabled(id); revertErr != nil {
			common.SysError(fmt.Sprintf("failed to revert service %d enabled state after reload failure: %v", id, revertErr))
		}
		common.RespError(c, http.StatusInternalServerError, i18n.Translate("toggle_service_status_failed", lang), err)
		return
	}

	serviceManager := proxy.GetServiceManager()
	ctx := c.Request.Context()

	if wasEnabled {
		if err := serviceManager.UnregisterService(ctx, id); err != nil && err != proxy.ErrServiceNotFound {
			common.SysError(fmt.Sprintf("failed to unregister disabled service %d: %v", id, err))
			if revertErr := model.ToggleServiceEnabled(id); revertErr != nil {
				common.SysError(fmt.Sprintf("failed to revert service %d enabled state after unregister failure: %v", id, revertErr))
			}
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("toggle_service_status_failed", lang), err)
			return
		}
	} else {
		if err := serviceManager.RegisterService(ctx, updatedService); err != nil && err != proxy.ErrServiceAlreadyExists {
			common.SysError(fmt.Sprintf("failed to register enabled service %d: %v", id, err))
			if revertErr := model.ToggleServiceEnabled(id); revertErr != nil {
				common.SysError(fmt.Sprintf("failed to revert service %d enabled state after register failure: %v", id, revertErr))
			}
			common.RespError(c, http.StatusInternalServerError, i18n.Translate("toggle_service_status_failed", lang), err)
			return
		}
	}

	status := i18n.Translate("disabled", lang)
	if updatedService.Enabled {
		status = i18n.Translate("enabled", lang)
	}

	common.RespSuccessStr(c, i18n.Translate("service_toggle_success", lang)+status)
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
