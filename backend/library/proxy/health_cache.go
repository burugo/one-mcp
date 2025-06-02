package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/burugo/thing"
)

// HealthCacheManager 健康状态缓存管理器 - 基于 Thing ORM 缓存
type HealthCacheManager struct {
	cacheClient thing.CacheClient
	expireTime  time.Duration
	mutex       sync.RWMutex // 用于保护并发访问
}

// NewHealthCacheManager 创建新的健康状态缓存管理器
func NewHealthCacheManager(expireTime time.Duration) *HealthCacheManager {
	if expireTime <= 0 {
		expireTime = 5 * time.Minute // 默认5分钟过期
	}

	return &HealthCacheManager{
		cacheClient: thing.Cache(), // 使用 Thing ORM v0.1.17 的全局缓存
		expireTime:  expireTime,
	}
}

// generateCacheKey 生成服务健康状态的缓存键
func (hcm *HealthCacheManager) generateCacheKey(serviceID int64) string {
	return fmt.Sprintf("health:service:%d", serviceID)
}

// SetServiceHealth 设置服务健康状态到缓存
func (hcm *HealthCacheManager) SetServiceHealth(serviceID int64, health *ServiceHealth) {
	if health == nil {
		return
	}

	hcm.mutex.Lock()
	defer hcm.mutex.Unlock()

	ctx := context.Background()
	cacheKey := hcm.generateCacheKey(serviceID)

	// 创建健康状态的副本以避免并发修改
	healthCopy := *health

	// 将 ServiceHealth 序列化为 JSON 存储到缓存
	healthJSON, err := json.Marshal(&healthCopy)
	if err != nil {
		log.Printf("Error marshaling health status for service %d: %v", serviceID, err)
		return
	}

	// 使用 Thing ORM 缓存设置值
	err = hcm.cacheClient.Set(ctx, cacheKey, string(healthJSON), hcm.expireTime)
	if err != nil {
		log.Printf("Error setting health status cache for service %d: %v", serviceID, err)
		return
	}

	log.Printf("Successfully cached health status for service %d (key: %s)", serviceID, cacheKey)
}

// GetServiceHealth 从缓存获取服务健康状态
func (hcm *HealthCacheManager) GetServiceHealth(serviceID int64) (*ServiceHealth, bool) {
	hcm.mutex.RLock()
	defer hcm.mutex.RUnlock()

	ctx := context.Background()
	cacheKey := hcm.generateCacheKey(serviceID)

	// 从 Thing ORM 缓存获取值
	healthJSON, err := hcm.cacheClient.Get(ctx, cacheKey)
	if err != nil {
		// 缓存中不存在或其他错误
		return nil, false
	}

	// 反序列化 JSON 为 ServiceHealth 结构
	var health ServiceHealth
	err = json.Unmarshal([]byte(healthJSON), &health)
	if err != nil {
		log.Printf("Error unmarshaling health status for service %d: %v", serviceID, err)
		// 如果反序列化失败，删除无效的缓存条目
		go hcm.DeleteServiceHealth(serviceID)
		return nil, false
	}

	// 返回健康状态的副本
	return &health, true
}

// DeleteServiceHealth 从缓存删除服务健康状态
func (hcm *HealthCacheManager) DeleteServiceHealth(serviceID int64) {
	hcm.mutex.Lock()
	defer hcm.mutex.Unlock()

	ctx := context.Background()
	cacheKey := hcm.generateCacheKey(serviceID)

	err := hcm.cacheClient.Delete(ctx, cacheKey)
	if err != nil {
		log.Printf("Error deleting health status cache for service %d: %v", serviceID, err)
	} else {
		log.Printf("Successfully deleted health status cache for service %d (key: %s)", serviceID, cacheKey)
	}
}

// CleanExpiredEntries Thing ORM 缓存自动处理过期，此方法保留兼容性但不执行操作
func (hcm *HealthCacheManager) CleanExpiredEntries() {
	// Thing ORM 缓存会自动处理过期条目
	// 此方法保留以维持接口兼容性
	log.Printf("CleanExpiredEntries called - Thing ORM cache handles expiration automatically")
}

// GetCacheStats 获取缓存统计信息
func (hcm *HealthCacheManager) GetCacheStats() map[string]interface{} {
	ctx := context.Background()

	// 获取 Thing ORM 缓存统计信息
	var thingCacheStats map[string]interface{}
	if hcm.cacheClient != nil {
		stats := hcm.cacheClient.GetCacheStats(ctx)
		thingCacheStats = map[string]interface{}{
			"thing_cache_counters": stats.Counters,
		}
	}

	// 组合我们自己的统计信息
	combinedStats := map[string]interface{}{
		"expire_time":      hcm.expireTime.String(),
		"cache_type":       "thing_orm_cache",
		"thing_cache_info": thingCacheStats,
	}

	return combinedStats
}

// Shutdown 关闭缓存管理器
func (hcm *HealthCacheManager) Shutdown() {
	// Thing ORM 缓存是全局的，不需要显式关闭
	// 此方法保留以维持接口兼容性
	log.Printf("HealthCacheManager shutdown called - Thing ORM cache is global and managed separately")
}

// 全局健康状态缓存管理器实例
var globalHealthCacheManager *HealthCacheManager
var healthCacheOnce sync.Once

// GetHealthCacheManager 获取全局健康状态缓存管理器实例
func GetHealthCacheManager() *HealthCacheManager {
	healthCacheOnce.Do(func() {
		globalHealthCacheManager = NewHealthCacheManager(5 * time.Minute)
	})
	return globalHealthCacheManager
}
