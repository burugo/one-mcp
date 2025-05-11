package handler

import (
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetAllMCPServices godoc
// @Summary 获取所有MCP服务
// @Description 返回所有MCP服务的列表
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
// @Description 根据ID返回一个MCP服务的详情
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
// @Description 创建一个新的MCP服务
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
	if service.Name == "" || service.DisplayName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("name_and_display_name_required", lang))
		return
	}
	if service.Type != model.ServiceTypeStdio && service.Type != model.ServiceTypeSSE && service.Type != model.ServiceTypeStreamableHTTP {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_service_type", lang))
		return
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
// @Description 更新现有的MCP服务
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
	if err := c.ShouldBindJSON(service); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_request_data", lang), err)
		return
	}
	if service.Name == "" || service.DisplayName == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("name_and_display_name_required", lang))
		return
	}
	if service.Type != model.ServiceTypeStdio && service.Type != model.ServiceTypeSSE && service.Type != model.ServiceTypeStreamableHTTP {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_service_type", lang))
		return
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

	// 尝试获取服务，确认它存在
	_, err = model.GetServiceByID(id)
	if err != nil {
		common.RespError(c, http.StatusNotFound, i18n.Translate("service_not_found", lang), err)
		return
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
