package proxy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

// healthCheckMockService implements Service for health checker tests
type healthCheckMockService struct {
	id              int64
	name            string
	serviceType     model.ServiceType
	running         bool
	health          *ServiceHealth
	checkHealthErr  error
	checkHealthFunc func(ctx context.Context) (*ServiceHealth, error)
	tools           []mcp.Tool
	timeout         time.Duration
	mu              sync.RWMutex
}

func newHealthCheckMockService(id int64, name string) *healthCheckMockService {
	return &healthCheckMockService{
		id:          id,
		name:        name,
		serviceType: model.ServiceTypeStdio,
		running:     true,
		health: &ServiceHealth{
			Status:      StatusHealthy,
			LastChecked: time.Now(),
		},
		tools:   []mcp.Tool{},
		timeout: 10 * time.Second,
	}
}

func (m *healthCheckMockService) ID() int64               { return m.id }
func (m *healthCheckMockService) Name() string            { return m.name }
func (m *healthCheckMockService) Type() model.ServiceType { return m.serviceType }

func (m *healthCheckMockService) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = true
	return nil
}

func (m *healthCheckMockService) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	return nil
}

func (m *healthCheckMockService) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *healthCheckMockService) CheckHealth(ctx context.Context) (*ServiceHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(ctx)
	}
	if m.checkHealthErr != nil {
		return nil, m.checkHealthErr
	}
	m.health.LastChecked = time.Now()
	return m.health, nil
}

func (m *healthCheckMockService) GetHealth() *ServiceHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health
}

func (m *healthCheckMockService) GetConfig() map[string]interface{} {
	return map[string]interface{}{}
}

func (m *healthCheckMockService) UpdateConfig(config map[string]interface{}) error {
	return nil
}

func (m *healthCheckMockService) HealthCheckTimeout() time.Duration {
	return m.timeout
}

func (m *healthCheckMockService) GetTools() []mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools
}

func (m *healthCheckMockService) GetServerInfo() *mcp.Implementation {
	return nil
}

func TestNewHealthChecker_DefaultInterval(t *testing.T) {
	hc := NewHealthChecker(0)
	assert.NotNil(t, hc)
	assert.Equal(t, 1*time.Minute, hc.checkInterval)
}

func TestNewHealthChecker_CustomInterval(t *testing.T) {
	hc := NewHealthChecker(5 * time.Minute)
	assert.NotNil(t, hc)
	assert.Equal(t, 5*time.Minute, hc.checkInterval)
}

func TestHealthChecker_RegisterService(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc := newHealthCheckMockService(1, "test-service")

	hc.RegisterService(mockSvc)

	hc.servicesMu.RLock()
	_, exists := hc.services[1]
	hc.servicesMu.RUnlock()

	assert.True(t, exists)
}

func TestHealthChecker_RegisterService_Duplicate(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc1 := newHealthCheckMockService(1, "test-service-1")
	mockSvc2 := newHealthCheckMockService(1, "test-service-2")

	hc.RegisterService(mockSvc1)
	hc.RegisterService(mockSvc2) // Same ID, should replace

	hc.servicesMu.RLock()
	svc := hc.services[1]
	hc.servicesMu.RUnlock()

	assert.Equal(t, "test-service-2", svc.Name())
}

func TestHealthChecker_UnregisterService(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc := newHealthCheckMockService(1, "test-service")

	hc.RegisterService(mockSvc)
	hc.UnregisterService(1)

	hc.servicesMu.RLock()
	_, exists := hc.services[1]
	hc.servicesMu.RUnlock()

	assert.False(t, exists)
}

func TestHealthChecker_UnregisterService_NonExistent(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)

	// Should not panic
	hc.UnregisterService(999)
}

func TestHealthChecker_GetServiceHealth(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc := newHealthCheckMockService(1, "test-service")
	mockSvc.health = &ServiceHealth{
		Status:       StatusHealthy,
		LastChecked:  time.Now(),
		ResponseTime: 50,
	}

	hc.RegisterService(mockSvc)

	health, err := hc.GetServiceHealth(1)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusHealthy, health.Status)
}

func TestHealthChecker_GetServiceHealth_NotRegistered(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)

	health, err := hc.GetServiceHealth(999)
	assert.Error(t, err)
	assert.Equal(t, ErrServiceNotRegistered, err)
	assert.Nil(t, health)
}

func TestHealthChecker_ForceCheckService(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc := newHealthCheckMockService(1, "test-service")
	mockSvc.health = &ServiceHealth{
		Status:       StatusHealthy,
		LastChecked:  time.Now(),
		ResponseTime: 25,
	}

	hc.RegisterService(mockSvc)

	health, err := hc.ForceCheckService(1)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusHealthy, health.Status)
}

func TestHealthChecker_ForceCheckService_NotRegistered(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)

	health, err := hc.ForceCheckService(999)
	assert.Error(t, err)
	assert.Equal(t, ErrServiceNotRegistered, err)
	assert.Nil(t, health)
}

func TestHealthChecker_ForceCheckService_Error(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc := newHealthCheckMockService(1, "test-service")
	mockSvc.checkHealthErr = errors.New("health check failed")

	hc.RegisterService(mockSvc)

	health, err := hc.ForceCheckService(1)
	// ForceCheckService handles error by returning unhealthy status
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Contains(t, health.ErrorMessage, "health check failed")
}

func TestHealthChecker_StartStop(t *testing.T) {
	hc := NewHealthChecker(100 * time.Millisecond)
	mockSvc := newHealthCheckMockService(1, "test-service")

	hc.RegisterService(mockSvc)

	assert.False(t, hc.running)
	hc.Start()
	assert.True(t, hc.running)

	// Wait for at least one check cycle
	time.Sleep(150 * time.Millisecond)

	hc.Stop()
	assert.False(t, hc.running)
}

func TestHealthChecker_StartTwice(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)

	hc.Start()
	assert.True(t, hc.running)

	hc.Start() // Should not panic or create duplicate goroutines
	assert.True(t, hc.running)

	hc.Stop()
}

func TestHealthChecker_StopTwice(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)

	hc.Start()
	hc.Stop()
	assert.False(t, hc.running)

	hc.Stop() // Should not panic
	assert.False(t, hc.running)
}

func TestHealthChecker_CheckService_WithTools(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	mockSvc := newHealthCheckMockService(1, "test-service")
	mockSvc.tools = []mcp.Tool{
		{Name: "tool-1", Description: "First tool"},
		{Name: "tool-2", Description: "Second tool"},
	}
	mockSvc.health = &ServiceHealth{
		Status:      StatusHealthy,
		LastChecked: time.Now(),
	}

	hc.RegisterService(mockSvc)

	// Clear any existing cache
	GetToolsCacheManager().DeleteServiceTools(1)

	// Force check to populate tools cache
	health, err := hc.ForceCheckService(1)
	assert.NoError(t, err)
	assert.NotNil(t, health)

	// Verify tools were cached
	entry, found := GetToolsCacheManager().GetServiceTools(1)
	assert.True(t, found)
	assert.Equal(t, 2, len(entry.Tools))
}

func TestHealthChecker_UpdateCacheHealthStatus_Debounce(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)
	serviceID := int64(100100)

	// Set initial update time
	hc.servicesMu.Lock()
	hc.lastUpdateTimes[serviceID] = time.Now()
	hc.servicesMu.Unlock()

	health := &ServiceHealth{
		Status:      StatusHealthy,
		LastChecked: time.Now(),
	}

	// This should be debounced (skipped)
	hc.updateCacheHealthStatus(serviceID, health)

	// Wait for debounce period to pass
	time.Sleep(6 * time.Second)

	// This should go through
	hc.updateCacheHealthStatus(serviceID, health)
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	hc := NewHealthChecker(1 * time.Minute)

	var wg sync.WaitGroup
	numServices := 10

	// Register services concurrently
	for i := 0; i < numServices; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mockSvc := newHealthCheckMockService(int64(id), "service-"+string(rune('a'+id)))
			hc.RegisterService(mockSvc)
		}(i)
	}

	// Read services concurrently
	for i := 0; i < numServices; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			hc.GetServiceHealth(int64(id))
		}(i)
	}

	wg.Wait()

	hc.servicesMu.RLock()
	count := len(hc.services)
	hc.servicesMu.RUnlock()

	assert.Equal(t, numServices, count)
}

func TestErrServiceNotRegistered(t *testing.T) {
	assert.Equal(t, "service not registered to health checker", ErrServiceNotRegistered.Error())
}
