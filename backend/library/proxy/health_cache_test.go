package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthCacheManager_DefaultExpireTime(t *testing.T) {
	hcm := NewHealthCacheManager(0)
	assert.NotNil(t, hcm)
	assert.Equal(t, 1*time.Hour, hcm.expireTime)
}

func TestNewHealthCacheManager_CustomExpireTime(t *testing.T) {
	hcm := NewHealthCacheManager(30 * time.Minute)
	assert.NotNil(t, hcm)
	assert.Equal(t, 30*time.Minute, hcm.expireTime)
}

func TestHealthCacheManager_GenerateCacheKey(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)

	key := hcm.generateCacheKey(123)
	assert.Equal(t, "health:service:123", key)

	key2 := hcm.generateCacheKey(456)
	assert.Equal(t, "health:service:456", key2)
}

func TestHealthCacheManager_SetAndGetServiceHealth(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)
	serviceID := int64(100001)

	health := &ServiceHealth{
		Status:       StatusHealthy,
		LastChecked:  time.Now(),
		ErrorMessage: "",
		WarningLevel: 0,
		ToolCount:    5,
		ToolsFetched: true,
	}

	hcm.SetServiceHealth(serviceID, health)

	retrieved, found := hcm.GetServiceHealth(serviceID)
	assert.True(t, found)
	assert.NotNil(t, retrieved)
	assert.Equal(t, StatusHealthy, retrieved.Status)
	assert.Equal(t, 5, retrieved.ToolCount)
	assert.True(t, retrieved.ToolsFetched)
}

func TestHealthCacheManager_SetNilHealth(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)
	serviceID := int64(100002)

	// Setting nil should not panic
	hcm.SetServiceHealth(serviceID, nil)

	retrieved, found := hcm.GetServiceHealth(serviceID)
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestHealthCacheManager_GetNonExistentService(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)
	serviceID := int64(999999)

	retrieved, found := hcm.GetServiceHealth(serviceID)
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestHealthCacheManager_DeleteServiceHealth(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)
	serviceID := int64(100003)

	health := &ServiceHealth{
		Status:      StatusHealthy,
		LastChecked: time.Now(),
	}

	hcm.SetServiceHealth(serviceID, health)

	// Verify it's stored
	retrieved, found := hcm.GetServiceHealth(serviceID)
	assert.True(t, found)
	assert.NotNil(t, retrieved)

	// Delete it
	hcm.DeleteServiceHealth(serviceID)

	// Verify it's gone
	retrieved, found = hcm.GetServiceHealth(serviceID)
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestHealthCacheManager_UpdateExistingHealth(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)
	serviceID := int64(100004)

	health1 := &ServiceHealth{
		Status:       StatusHealthy,
		LastChecked:  time.Now(),
		ErrorMessage: "",
		ToolCount:    3,
	}
	hcm.SetServiceHealth(serviceID, health1)

	health2 := &ServiceHealth{
		Status:       StatusUnhealthy,
		LastChecked:  time.Now(),
		ErrorMessage: "Service crashed",
		ToolCount:    0,
		WarningLevel: 3,
	}
	hcm.SetServiceHealth(serviceID, health2)

	retrieved, found := hcm.GetServiceHealth(serviceID)
	assert.True(t, found)
	assert.Equal(t, StatusUnhealthy, retrieved.Status)
	assert.Equal(t, "Service crashed", retrieved.ErrorMessage)
	assert.Equal(t, 3, retrieved.WarningLevel)
}

func TestHealthCacheManager_MultipleServices(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)

	services := []struct {
		id     int64
		status ServiceStatus
	}{
		{100005, StatusHealthy},
		{100006, StatusUnhealthy},
		{100007, StatusStarting},
		{100008, StatusStopped},
	}

	// Set health for multiple services
	for _, svc := range services {
		health := &ServiceHealth{
			Status:      svc.status,
			LastChecked: time.Now(),
		}
		hcm.SetServiceHealth(svc.id, health)
	}

	// Verify each service
	for _, svc := range services {
		retrieved, found := hcm.GetServiceHealth(svc.id)
		assert.True(t, found, "Service %d should be found", svc.id)
		assert.Equal(t, svc.status, retrieved.Status, "Service %d status mismatch", svc.id)
	}
}

func TestHealthCacheManager_GetCacheStats(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)

	stats := hcm.GetCacheStats()
	assert.NotNil(t, stats)
	assert.Equal(t, "thing_orm_cache", stats["cache_type"])
	assert.Contains(t, stats, "expire_time")
	assert.Contains(t, stats, "thing_cache_info")
}

func TestHealthCacheManager_CleanExpiredEntries(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)

	// This should not panic
	hcm.CleanExpiredEntries()
}

func TestHealthCacheManager_Shutdown(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)

	// This should not panic
	hcm.Shutdown()
}

func TestHealthCacheManager_LocalCacheExpiration(t *testing.T) {
	// Create a manager with very short expiration for local cache testing
	hcm := &HealthCacheManager{
		cacheClient: nil, // Force local cache usage
		expireTime:  50 * time.Millisecond,
		local:       make(map[string]healthLocalCacheItem),
	}

	serviceID := int64(100009)
	health := &ServiceHealth{
		Status:      StatusHealthy,
		LastChecked: time.Now(),
	}

	hcm.SetServiceHealth(serviceID, health)

	// Should be found immediately
	retrieved, found := hcm.GetServiceHealth(serviceID)
	assert.True(t, found)
	assert.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not be found after expiration
	retrieved, found = hcm.GetServiceHealth(serviceID)
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestGetHealthCacheManager_Singleton(t *testing.T) {
	hcm1 := GetHealthCacheManager()
	hcm2 := GetHealthCacheManager()

	assert.Same(t, hcm1, hcm2, "GetHealthCacheManager should return the same instance")
}

func TestHealthCacheManager_ConcurrentAccess(t *testing.T) {
	hcm := NewHealthCacheManager(1 * time.Hour)
	serviceID := int64(100010)

	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			health := &ServiceHealth{
				Status:      StatusHealthy,
				LastChecked: time.Now(),
				ToolCount:   idx,
			}
			hcm.SetServiceHealth(serviceID, health)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			hcm.GetServiceHealth(serviceID)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have some value (no panic from concurrent access)
	_, found := hcm.GetServiceHealth(serviceID)
	assert.True(t, found)
}
