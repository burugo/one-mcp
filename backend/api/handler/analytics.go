package handler

import (
	"context"
	"fmt"
	"net/http"
	"sort" // For sorting aggregated stats
	"strconv"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n" // Added back for Translate function
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model" // Now using model for ProxyRequestStat

	// "one-mcp/backend/i18n" // Commented out as it's only used in placeholder error handling

	"github.com/burugo/thing"
	"github.com/gin-gonic/gin"
)

// getTodayRequestCountFromCache retrieves today's request count for a specific service from cache
func getTodayRequestCountFromCache(serviceID int64) (int64, error) {
	cacheClient := thing.Cache()
	if cacheClient == nil {
		return 0, fmt.Errorf("cache client is nil")
	}

	today := time.Now().Format("2006-01-02")
	cacheKey := fmt.Sprintf("request:%s:%d:count", today, serviceID)

	ctx := context.Background()
	countStr, err := cacheClient.Get(ctx, cacheKey)
	if err != nil {
		// If key doesn't exist, return 0 (no requests today)
		return 0, nil
	}

	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cache count value: %v", err)
	}

	return count, nil
}

// getTodayTotalRequestCountFromCache calculates total requests across all services for today
func getTodayTotalRequestCountFromCache() (int64, error) {
	// Get all enabled services
	mcpServiceThing, err := model.GetMCPServiceThing()
	if err != nil {
		return 0, fmt.Errorf("error accessing MCPService data store: %v", err)
	}

	allServices, err := mcpServiceThing.Where("enabled = ?", true).All()
	if err != nil {
		return 0, fmt.Errorf("error fetching enabled services: %v", err)
	}

	var totalCount int64 = 0
	for _, service := range allServices {
		count, err := getTodayRequestCountFromCache(service.ID)
		if err != nil {
			// Log error but continue with other services
			common.SysError(fmt.Sprintf("Error getting today's request count for service %d: %v", service.ID, err))
			continue
		}
		totalCount += count
	}

	return totalCount, nil
}

// getTodayAverageLatencyFromDB calculates today's average latency from database records
func getTodayAverageLatencyFromDB() (float64, error) {
	statThing, err := model.GetProxyRequestStatThing()
	if err != nil {
		return 0, fmt.Errorf("error accessing statistics data store: %v", err)
	}

	// Get today's date range
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Fetch today's stats
	todayStats, err := statThing.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).All()
	if err != nil {
		return 0, fmt.Errorf("error fetching today's statistics: %v", err)
	}

	if len(todayStats) == 0 {
		return 0, nil
	}

	var totalLatency int64 = 0
	for _, stat := range todayStats {
		totalLatency += stat.ResponseTimeMs
	}

	return float64(totalLatency) / float64(len(todayStats)), nil
}

// GetServiceUtilization godoc
// @Summary 获取服务使用统计
// @Description 获取所有MCP服务的汇总使用统计数据，包括今日请求数和今日平均延迟等。
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse{data=[]map[string]interface{}} "返回服务使用统计列表"
// @Failure 500 {object} common.APIResponse "服务器内部错误"
// @Router /api/analytics/services/utilization [get]
func GetServiceUtilization(c *gin.Context) {
	// Get all services first
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

	// Get today's date range for DB queries
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	statThing, err := model.GetProxyRequestStatThing()
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "Error accessing statistics data store", err)
		return
	}

	resultStats := make([]map[string]interface{}, 0, len(allServices))

	for _, service := range allServices {
		// Get today's request count from cache
		todayRequestCount, err := getTodayRequestCountFromCache(service.ID)
		if err != nil {
			common.SysError(fmt.Sprintf("Error getting today's request count for service %s (ID: %d): %v", service.Name, service.ID, err))
			todayRequestCount = 0
		}

		// Get today's average latency from database
		todayStats, err := statThing.Where("service_id = ? AND created_at >= ? AND created_at < ?", service.ID, startOfDay, endOfDay).All()
		var todayAvgLatency float64 = 0
		if err != nil {
			common.SysError(fmt.Sprintf("Error fetching today's stats for service %s (ID: %d): %v", service.Name, service.ID, err))
		} else if len(todayStats) > 0 {
			var totalLatency int64 = 0
			for _, stat := range todayStats {
				totalLatency += stat.ResponseTimeMs
			}
			todayAvgLatency = float64(totalLatency) / float64(len(todayStats))
		}

		resultStats = append(resultStats, map[string]interface{}{
			"service_id":           service.ID,
			"service_name":         service.Name,
			"display_name":         service.DisplayName,
			"enabled":              service.Enabled,
			"today_request_count":  todayRequestCount,
			"today_avg_latency_ms": todayAvgLatency,
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
		common.RespErrorStr(c, http.StatusNotFound, i18n.Translate("service_not_found", lang))
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
// @Description 获取整个MCP系统的分析概览数据，包括服务统计、健康状态统计、今日请求数、响应时间等。
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} common.APIResponse{data=map[string]interface{}} "返回系统概览数据"
// @Failure 500 {object} common.APIResponse "服务器内部错误"
// @Router /api/analytics/system/overview [get]
func GetSystemOverview(c *gin.Context) {
	// Get all services and count enabled ones
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
	healthyServices := 0
	unhealthyServices := 0

	// Get health cache manager for service health status
	healthCacheManager := proxy.GetHealthCacheManager()

	for _, srv := range allServices {
		if srv.Enabled {
			enabledServices++

			// Check health status from cache
			if healthInfo, exists := healthCacheManager.GetServiceHealth(srv.ID); exists {
				if healthInfo.Status == proxy.StatusHealthy {
					healthyServices++
				} else {
					unhealthyServices++
				}
			} else {
				// If no health info available, consider it unhealthy
				unhealthyServices++
			}
		}
	}

	// Get today's total request count from cache
	todayTotalRequests, err := getTodayTotalRequestCountFromCache()
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting today's total request count: %v", err))
		// Continue with 0 count rather than failing the entire request
		todayTotalRequests = 0
	}

	// Get today's average response time from database
	todayAvgLatency, err := getTodayAverageLatencyFromDB()
	if err != nil {
		common.SysError(fmt.Sprintf("Error getting today's average latency: %v", err))
		// Continue with 0 latency rather than failing the entire request
		todayAvgLatency = 0
	}

	overview := map[string]interface{}{
		"total_services":             totalServices,
		"enabled_services":           enabledServices,
		"healthy_services":           healthyServices,
		"unhealthy_services":         unhealthyServices,
		"today_total_requests":       todayTotalRequests,
		"today_avg_response_time_ms": todayAvgLatency,
	}

	common.RespSuccess(c, overview)
}
