package handler

import (
	"net/http"
	"strconv"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
)

// GetMCPLogs godoc
// @Summary 获取MCP日志列表
// @Description 获取MCP服务的安装和运行日志，支持多维度筛选和分页
// @Tags MCP日志
// @Accept json
// @Produce json
// @Param service_id query int false "服务ID"
// @Param service_name query string false "服务名称（支持模糊搜索）"
// @Param phase query string false "阶段 (install/run)"
// @Param level query string false "日志级别 (info/warn/error)"
// @Param page query int false "页码，从1开始" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse{data=object{logs=[]model.MCPLog,total=int64,page=int,page_size=int}}
// @Failure 400 {object} common.APIResponse
// @Failure 401 {object} common.APIResponse
// @Failure 403 {object} common.APIResponse
// @Failure 500 {object} common.APIResponse
// @Router /api/mcp_logs [get]
func GetMCPLogs(c *gin.Context) {
	// Parse query parameters (admin auth already handled by middleware)
	var serviceID *int64
	if serviceIDStr := c.Query("service_id"); serviceIDStr != "" {
		if id, err := strconv.ParseInt(serviceIDStr, 10, 64); err == nil {
			serviceID = &id
		} else {
			common.RespErrorStr(c, http.StatusBadRequest, "Invalid service_id parameter")
			return
		}
	}

	serviceName := c.Query("service_name")
	phase := c.Query("phase")
	level := c.Query("level")

	// Validate phase parameter
	if phase != "" && phase != "install" && phase != "run" {
		common.RespErrorStr(c, http.StatusBadRequest, "Invalid phase parameter. Must be 'install' or 'run'")
		return
	}

	// Validate level parameter
	if level != "" && level != "info" && level != "warn" && level != "error" {
		common.RespErrorStr(c, http.StatusBadRequest, "Invalid level parameter. Must be 'info', 'warn', or 'error'")
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 10
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Convert string parameters to pointers
	var serviceNamePtr, phasePtr, levelPtr *string
	if serviceName != "" {
		serviceNamePtr = &serviceName
	}
	if phase != "" {
		phasePtr = &phase
	}
	if level != "" {
		levelPtr = &level
	}

	// Get logs from database (now returns both logs and total)
	logs, total, err := model.GetMCPLogs(c.Request.Context(), serviceID, serviceNamePtr, phasePtr, levelPtr, page, pageSize)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Failed to retrieve logs", err)
		return
	}

	// Return response
	common.RespSuccess(c, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
