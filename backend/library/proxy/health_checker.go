package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"one-mcp/backend/model"
)

// HealthChecker 负责定期检查服务的健康状态
type HealthChecker struct {
	services        map[int64]Service
	servicesMu      sync.RWMutex
	checkInterval   time.Duration
	stopChan        chan struct{}
	running         bool
	lastUpdateTimes map[int64]time.Time
}

// NewHealthChecker 创建一个新的健康检查管理器
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	if checkInterval <= 0 {
		checkInterval = 1 * time.Minute // 默认检查间隔为1分钟
	}

	return &HealthChecker{
		services:        make(map[int64]Service),
		checkInterval:   checkInterval,
		stopChan:        make(chan struct{}),
		running:         false,
		lastUpdateTimes: make(map[int64]time.Time),
	}
}

// RegisterService 注册一个服务到健康检查管理器
func (hc *HealthChecker) RegisterService(service Service) {
	hc.servicesMu.Lock()
	defer hc.servicesMu.Unlock()

	hc.services[service.ID()] = service
}

// UnregisterService 从健康检查管理器移除一个服务
func (hc *HealthChecker) UnregisterService(serviceID int64) {
	hc.servicesMu.Lock()
	defer hc.servicesMu.Unlock()

	delete(hc.services, serviceID)
	delete(hc.lastUpdateTimes, serviceID)
}

// Start 启动健康检查任务
func (hc *HealthChecker) Start() {
	if hc.running {
		return
	}

	hc.running = true
	go hc.runChecks()
}

// Stop 停止健康检查任务
func (hc *HealthChecker) Stop() {
	if !hc.running {
		return
	}

	hc.stopChan <- struct{}{}
	hc.running = false
}

// runChecks 运行定期健康检查任务
func (hc *HealthChecker) runChecks() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// 立即进行一次检查
	hc.checkAllServices()

	for {
		select {
		case <-ticker.C:
			hc.checkAllServices()
		case <-hc.stopChan:
			return
		}
	}
}

// checkAllServices 检查所有注册的服务
func (hc *HealthChecker) checkAllServices() {
	hc.servicesMu.RLock()
	services := make([]Service, 0, len(hc.services))
	for _, service := range hc.services {
		services = append(services, service)
	}
	hc.servicesMu.RUnlock()

	for _, service := range services {
		go hc.checkService(service)
	}
}

// checkService 检查单个服务的健康状态
func (hc *HealthChecker) checkService(service Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := service.CheckHealth(ctx)
	if err != nil {
		log.Printf("Error checking health for service %s (ID: %d): %v", service.Name(), service.ID(), err)
		// 错误情况下仍然更新健康状态为异常
		health = &ServiceHealth{
			Status:       StatusUnhealthy,
			LastChecked:  time.Now(),
			ErrorMessage: err.Error(),
		}
	}

	// 更新数据库中的健康状态
	hc.updateDatabaseHealthStatus(service.ID(), health)
}

// updateDatabaseHealthStatus 更新数据库中的服务健康状态
func (hc *HealthChecker) updateDatabaseHealthStatus(serviceID int64, health *ServiceHealth) {
	hc.servicesMu.Lock()
	lastUpdate := hc.lastUpdateTimes[serviceID]
	hc.servicesMu.Unlock()

	// 如果上次更新时间距现在不到5秒，则跳过更新以减少数据库负担
	if time.Since(lastUpdate) < 5*time.Second {
		return
	}

	// 获取服务实例
	service, err := model.GetServiceByID(serviceID)
	if err != nil {
		log.Printf("Failed to get service (ID: %d) from database: %v", serviceID, err)
		return
	}

	// 将健康详情序列化为JSON
	healthDetails, err := json.Marshal(health)
	if err != nil {
		log.Printf("Failed to marshal health details for service (ID: %d): %v", serviceID, err)
		return
	}

	// 更新服务的健康状态
	service.HealthStatus = string(health.Status)
	service.LastHealthCheck = health.LastChecked
	service.HealthDetails = string(healthDetails)

	// 保存到数据库
	if err := model.UpdateService(service); err != nil {
		log.Printf("Failed to update health status for service (ID: %d): %v", serviceID, err)
		return
	}

	// 更新最后更新时间
	hc.servicesMu.Lock()
	hc.lastUpdateTimes[serviceID] = time.Now()
	hc.servicesMu.Unlock()
}

// ForceCheckService 强制立即检查指定服务的健康状态
func (hc *HealthChecker) ForceCheckService(serviceID int64) (*ServiceHealth, error) {
	hc.servicesMu.RLock()
	service, exists := hc.services[serviceID]
	hc.servicesMu.RUnlock()

	if !exists {
		return nil, ErrServiceNotRegistered
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := service.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}

	// 更新数据库中的健康状态
	hc.updateDatabaseHealthStatus(serviceID, health)

	return health, nil
}

// GetServiceHealth 获取指定服务的最新健康状态
func (hc *HealthChecker) GetServiceHealth(serviceID int64) (*ServiceHealth, error) {
	hc.servicesMu.RLock()
	service, exists := hc.services[serviceID]
	hc.servicesMu.RUnlock()

	if !exists {
		return nil, ErrServiceNotRegistered
	}

	return service.GetHealth(), nil
}

// ErrServiceNotRegistered 表示服务未注册到健康检查管理器
var ErrServiceNotRegistered = errors.New("service not registered to health checker")
