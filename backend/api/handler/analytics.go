package handler

import (
	"fmt"
	"net/http"
	"sort" // For sorting aggregated stats
	"strconv"

	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n" // Added back for Translate function
	"one-mcp/backend/model"       // Now using model for ProxyRequestStat

	// "one-mcp/backend/i18n" // Commented out as it's only used in placeholder error handling

	"github.com/gin-gonic/gin"
)

// GetServiceUtilization godoc
// @Summary 获取服务使用统计
// @Description 获取所有MCP服务的汇总使用统计数据，例如总请求数、成功率、平均延迟等。
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse{data=[]map[string]interface{}} "返回服务使用统计列表"
// @Failure 500 {object} common.APIResponse "服务器内部错误"
// @Router /api/analytics/services/utilization [get]
func GetServiceUtilization(c *gin.Context) {
	// lang := c.GetString("lang") // Commented out as it's only used in placeholder error handling

	statThing, err := model.GetProxyRequestStatThing() // Using the public getter
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error accessing statistics data store", err)
		return
	}

	// Fetch all stats - for a production system, this would need pagination or time-range filtering
	allStats, err := statThing.All()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error fetching statistics", err)
		return
	}

	// Aggregate stats by ServiceName
	type AggregatedStat struct {
		ServiceName    string
		TotalRequests  int64
		SuccessCount   int64
		TotalLatencyMs int64
	}

	aggregated := make(map[string]*AggregatedStat)

	for _, stat := range allStats {
		if _, ok := aggregated[stat.ServiceName]; !ok {
			aggregated[stat.ServiceName] = &AggregatedStat{ServiceName: stat.ServiceName}
		}
		aggregated[stat.ServiceName].TotalRequests++
		if stat.Success {
			aggregated[stat.ServiceName].SuccessCount++
		}
		aggregated[stat.ServiceName].TotalLatencyMs += stat.ResponseTimeMs
	}

	// Convert map to slice for response and calculate rates/averages
	resultStats := make([]map[string]interface{}, 0, len(aggregated))
	for _, agg := range aggregated {
		successRate := float64(0)
		if agg.TotalRequests > 0 {
			successRate = float64(agg.SuccessCount) / float64(agg.TotalRequests)
		}
		avgLatencyMs := float64(0)
		if agg.TotalRequests > 0 {
			avgLatencyMs = float64(agg.TotalLatencyMs) / float64(agg.TotalRequests)
		}

		resultStats = append(resultStats, map[string]interface{}{
			"service_name":   agg.ServiceName,
			"total_requests": agg.TotalRequests,
			"success_rate":   successRate,
			"avg_latency_ms": avgLatencyMs,
		})
	}

	// Sort by service name for consistent output
	sort.Slice(resultStats, func(i, j int) bool {
		return resultStats[i]["service_name"].(string) < resultStats[j]["service_name"].(string)
	})

	common.RespSuccess(c, resultStats)
}

// GetServiceMetrics godoc
// @Summary 获取单个服务的详细性能指标
// @Description 获取指定MCP服务的详细性能指标，例如随时间变化的请求数、延迟分布等。
// @Tags Analytics
// @Accept json
// @Produce json
// @Param service_id query string true "服务ID"
// @Param time_range query string false "时间范围 (e.g., last_24h, last_7d, last_30d)"
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse{data=map[string]interface{}} "返回服务的详细性能指标"
// @Failure 400 {object} common.APIResponse "无效的参数"
// @Failure 404 {object} common.APIResponse "服务未找到"
// @Failure 500 {object} common.APIResponse "服务器内部错误"
// @Router /api/analytics/services/metrics [get]
func GetServiceMetrics(c *gin.Context) {
	lang := c.GetString("lang") // lang is used here for error messages
	serviceIDStr := c.Query("service_id")
	// timeRange := c.Query("time_range") // Placeholder for time range filtering

	if serviceIDStr == "" {
		common.RespErrorStr(c, http.StatusBadRequest, fmt.Sprintf("%s: service_id is required", i18n.Translate("invalid_service_id", lang)))
		return
	}

	serviceID, err := strconv.ParseInt(serviceIDStr, 10, 64)
	if err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, fmt.Sprintf("%s: invalid service_id format", i18n.Translate("invalid_input", lang)))
		return
	}

	// Fetch service details to get the name
	mcpService, err := model.GetServiceByID(serviceID)
	if err != nil {
		// Handle error, e.g., service not found
		common.RespError(c, http.StatusNotFound, fmt.Sprintf("%s: %s", i18n.Translate("service_not_found", lang), serviceIDStr), err)
		return
	}

	statThing, err := model.GetProxyRequestStatThing()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error accessing statistics data store", err)
		return
	}

	// Fetch stats for the specific service
	// For production, consider time range filtering and ordering (e.g., by CreatedAt DESC)
	serviceStats, err := statThing.Where("service_id = ?", serviceID).All()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, fmt.Sprintf("Error fetching statistics for service %s", serviceIDStr), err)
		return
	}

	requestsOverTime := make([]map[string]interface{}, 0, len(serviceStats))
	var latencies []int64
	totalRequests := int64(0)
	successfulRequests := int64(0)

	for _, stat := range serviceStats {
		requestsOverTime = append(requestsOverTime, map[string]interface{}{
			"timestamp":  stat.CreatedAt, // Assuming CreatedAt from BaseModel is the request time
			"count":      1,              // Each record is one request for now; can be aggregated later
			"success":    stat.Success,
			"latency_ms": stat.ResponseTimeMs,
		})
		latencies = append(latencies, stat.ResponseTimeMs)
		totalRequests++
		if stat.Success {
			successfulRequests++
		}
	}

	// Sort latencies to calculate P95
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	latencyP95Ms := int64(0)
	if len(latencies) > 0 {
		indexP95 := int(float64(len(latencies)) * 0.95)
		if indexP95 >= len(latencies) {
			indexP95 = len(latencies) - 1
		}
		latencyP95Ms = latencies[indexP95]
	}

	errorRatePercentage := float64(0)
	if totalRequests > 0 {
		errorRatePercentage = (float64(totalRequests-successfulRequests) / float64(totalRequests)) * 100
	}

	metrics := map[string]interface{}{
		"service_id":            serviceIDStr,
		"service_name":          mcpService.DisplayName, // Using DisplayName from MCPService
		"requests_over_time":    requestsOverTime,       // This is a raw list of requests
		"latency_p95_ms":        latencyP95Ms,
		"error_rate_percentage": errorRatePercentage,
		"total_requests":        totalRequests,
		"successful_requests":   successfulRequests,
	}

	common.RespSuccess(c, metrics)
}

// GetSystemOverview godoc
// @Summary 获取系统分析概览
// @Description 获取整个MCP系统的分析概览数据，例如总服务数、总请求数、整体健康状况等。
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse{data=map[string]interface{}} "返回系统概览数据"
// @Failure 500 {object} common.APIResponse "服务器内部错误"
// @Router /api/analytics/system/overview [get]
func GetSystemOverview(c *gin.Context) {
	// lang := c.GetString("lang") // Placeholder for future i18n if needed

	// Get total and enabled services count
	mcpServiceThing, err := model.GetMCPServiceThing()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error accessing MCPService data store", err)
		return
	}
	allServices, err := mcpServiceThing.All()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error fetching all MCP services", err)
		return
	}
	totalServices := len(allServices)
	enabledServices := 0
	for _, srv := range allServices {
		if srv.Enabled {
			enabledServices++
		}
	}

	// Get overall request stats
	statThing, err := model.GetProxyRequestStatThing()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error accessing statistics data store", err)
		return
	}

	allStats, err := statThing.All() // For production, consider time-range and optimized aggregation
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error fetching all statistics", err)
		return
	}

	totalSystemRequests := int64(len(allStats))
	successfulSystemRequests := int64(0)
	for _, stat := range allStats {
		if stat.Success {
			successfulSystemRequests++
		}
	}

	overallSuccessRate := float64(0)
	if totalSystemRequests > 0 {
		overallSuccessRate = (float64(successfulSystemRequests) / float64(totalSystemRequests)) * 100
	}

	overview := map[string]interface{}{
		"total_services":                totalServices,
		"enabled_services":              enabledServices,
		"total_requests_all_time":       totalSystemRequests, // Consider renaming or adding time frame if filtered
		"overall_success_rate_all_time": overallSuccessRate,  // Consider renaming or adding time frame if filtered
	}

	common.RespSuccess(c, overview)
}
