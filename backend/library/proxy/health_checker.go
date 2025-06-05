package proxy

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
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
	_, exists := hc.services[service.ID()]
	hc.services[service.ID()] = service
	// Read hc.running while under lock to ensure consistency with a potential Stop() call.
	// This determines if an immediate check should be scheduled for a new service.
	shouldCheckImmediately := !exists && hc.running
	hc.servicesMu.Unlock() // Unlock before logging or spawning a goroutine.

	if shouldCheckImmediately {
		// Log that an immediate check is being scheduled for the new service.
		log.Printf("HealthChecker: New service %s (ID: %d) registered, scheduling immediate check.", service.Name(), service.ID())
		// Perform the check in a new goroutine to avoid blocking the registration process.
		go hc.checkService(service)
	}
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
	timeout := service.HealthCheckTimeout()
	if timeout <= 0 {
		timeout = 10 * time.Second // 如果服务未指定或指定无效值，则使用默认超时10秒
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	health, err := service.CheckHealth(ctx)
	if err != nil {
		log.Printf("Error checking health for service %s (ID: %d) with timeout %v: %v", service.Name(), service.ID(), timeout, err)
		// 错误情况下仍然更新健康状态为异常
		health = &ServiceHealth{
			Status:       StatusUnhealthy,
			LastChecked:  time.Now(),
			ErrorMessage: err.Error(),
		}
	}

	// 更新缓存中的健康状态
	hc.updateCacheHealthStatus(service.ID(), health)
}

// updateCacheHealthStatus 更新缓存中的服务健康状态
func (hc *HealthChecker) updateCacheHealthStatus(serviceID int64, health *ServiceHealth) {
	hc.servicesMu.Lock()
	lastUpdate := hc.lastUpdateTimes[serviceID]
	hc.servicesMu.Unlock()

	// 如果上次更新时间距现在不到5秒，则跳过更新以减少频繁操作
	if time.Since(lastUpdate) < 5*time.Second {
		return
	}

	// 获取全局健康状态缓存管理器
	cacheManager := GetHealthCacheManager()

	// 将健康状态存储到缓存中
	cacheManager.SetServiceHealth(serviceID, health)

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

	startTimeForCheckAttempt := time.Now() // Record start time for the CheckHealth attempt

	returnedHealthFromService, returnedErrFromService := service.CheckHealth(ctx)

	if returnedErrFromService != nil {
		log.Printf("Health check for service ID %d (%s) resulted in an error: %v", serviceID, service.Name(), returnedErrFromService)

		healthForCache := returnedHealthFromService

		if healthForCache == nil { // If CheckHealth returned (nil, error)
			healthForCache = &ServiceHealth{}
		}

		// Ensure standard fields are set for an error scenario
		healthForCache.Status = StatusUnhealthy
		healthForCache.LastChecked = time.Now() // Always update to current time for this check event
		if healthForCache.ErrorMessage == "" {  // If not already set by service.CheckHealth
			healthForCache.ErrorMessage = returnedErrFromService.Error()
		}
		// If the specific service's CheckHealth didn't set a ResponseTime (or returned nil health object),
		// set it to the duration of this attempt.
		if healthForCache.ResponseTime == 0 {
			healthForCache.ResponseTime = time.Since(startTimeForCheckAttempt).Milliseconds()
		}
		// Note: Fields like SuccessCount, FailureCount are expected to be handled by the service.CheckHealth() impl.

		// Directly update the cache and the HealthChecker's last update time for this service
		cacheManagerAfterError := GetHealthCacheManager()
		cacheManagerAfterError.SetServiceHealth(serviceID, healthForCache)
		hc.servicesMu.Lock()
		hc.lastUpdateTimes[serviceID] = healthForCache.LastChecked // Ensure consistency for background checker
		hc.servicesMu.Unlock()

		// Return the unhealthy status object and a nil error to the caller
		// This indicates the error was handled by creating a valid (unhealthy) health status
		return healthForCache, nil // Return the (unhealthy) health status and nil error to indicate handling
	}

	// If returnedErrFromService == nil, then 'returnedHealthFromService' is the valid, successful health status.
	// The service.CheckHealth implementation should have set LastChecked and ResponseTime appropriately.

	// For a forced check, ensure LastChecked accurately reflects the current time.
	// service.CheckHealth() provides the status (Healthy/Unhealthy) and other details like ResponseTime.
	if returnedHealthFromService != nil { // Guard against nil if service.CheckHealth() could return (nil, nil)
		returnedHealthFromService.LastChecked = time.Now()
	} else {
		// This case should ideally not happen if CheckHealth guarantees non-nil health on nil error.
		// However, defensively create a basic healthy status if it does.
		log.Printf("Warning: service.CheckHealth for service ID %d returned (nil, nil). Assuming healthy.", serviceID)
		returnedHealthFromService = &ServiceHealth{
			Status:      StatusHealthy,
			LastChecked: time.Now(),
		}
	}

	// Directly update the cache and the HealthChecker's last update time for this service
	cacheManagerSuccess := GetHealthCacheManager()
	cacheManagerSuccess.SetServiceHealth(serviceID, returnedHealthFromService)
	hc.servicesMu.Lock()
	hc.lastUpdateTimes[serviceID] = returnedHealthFromService.LastChecked // Ensure consistency for background checker
	hc.servicesMu.Unlock()

	return returnedHealthFromService, nil
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
