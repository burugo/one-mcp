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

// mockService implements the Service interface for testing
type mockService struct {
	id           int64
	name         string
	serviceType  model.ServiceType
	running      bool
	health       *ServiceHealth
	config       map[string]interface{}
	startErr     error
	stopErr      error
	checkErr     error
	tools        []mcp.Tool
	mu           sync.RWMutex
}

func newMockService(id int64, name string, serviceType model.ServiceType) *mockService {
	return &mockService{
		id:          id,
		name:        name,
		serviceType: serviceType,
		running:     false,
		health: &ServiceHealth{
			Status:      StatusStopped,
			LastChecked: time.Now(),
		},
		config: make(map[string]interface{}),
		tools:  []mcp.Tool{},
	}
}

func (m *mockService) ID() int64            { return m.id }
func (m *mockService) Name() string         { return m.name }
func (m *mockService) Type() model.ServiceType { return m.serviceType }

func (m *mockService) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startErr != nil {
		return m.startErr
	}
	m.running = true
	m.health.Status = StatusHealthy
	return nil
}

func (m *mockService) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return m.stopErr
	}
	m.running = false
	m.health.Status = StatusStopped
	return nil
}

func (m *mockService) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *mockService) CheckHealth(ctx context.Context) (*ServiceHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.checkErr != nil {
		return nil, m.checkErr
	}
	m.health.LastChecked = time.Now()
	return m.health, nil
}

func (m *mockService) GetHealth() *ServiceHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health
}

func (m *mockService) GetConfig() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *mockService) UpdateConfig(config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
	return nil
}

func (m *mockService) HealthCheckTimeout() time.Duration {
	return 10 * time.Second
}

func (m *mockService) GetTools() []mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools
}

func (m *mockService) GetServerInfo() *mcp.Implementation {
	return nil
}

// createTestServiceManager creates a fresh ServiceManager for testing
func createTestServiceManager() *ServiceManager {
	return &ServiceManager{
		services:                 make(map[int64]Service),
		healthChecker:            NewHealthChecker(10 * time.Minute),
		initialized:              false,
		lastAccessed:             make(map[int64]time.Time),
		stdioOnDemandIdleTimeout: 10 * time.Minute,
	}
}

func TestServiceManager_SetAndGetService(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)

	manager.SetService(1, mockSvc)

	svc, err := manager.GetService(1)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Equal(t, int64(1), svc.ID())
	assert.Equal(t, "test-service", svc.Name())
}

func TestServiceManager_GetService_NotFound(t *testing.T) {
	manager := createTestServiceManager()

	svc, err := manager.GetService(999)
	assert.Error(t, err)
	assert.Equal(t, ErrServiceNotFound, err)
	assert.Nil(t, svc)
}

func TestServiceManager_StartService(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.StartService(ctx, 1)
	assert.NoError(t, err)
	assert.True(t, mockSvc.IsRunning())
}

func TestServiceManager_StartService_AlreadyRunning(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = true
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.StartService(ctx, 1)
	assert.NoError(t, err) // Should not error, just skip
}

func TestServiceManager_StartService_NotFound(t *testing.T) {
	manager := createTestServiceManager()

	ctx := context.Background()
	err := manager.StartService(ctx, 999)
	assert.Error(t, err)
	assert.Equal(t, ErrServiceNotFound, err)
}

func TestServiceManager_StopService(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = true
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.StopService(ctx, 1)
	assert.NoError(t, err)
	assert.False(t, mockSvc.IsRunning())
}

func TestServiceManager_StopService_AlreadyStopped(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = false
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.StopService(ctx, 1)
	assert.NoError(t, err) // Should not error, just skip
}

func TestServiceManager_StopService_NotFound(t *testing.T) {
	manager := createTestServiceManager()

	ctx := context.Background()
	err := manager.StopService(ctx, 999)
	assert.Error(t, err)
	assert.Equal(t, ErrServiceNotFound, err)
}

func TestServiceManager_RestartService(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = true
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.RestartService(ctx, 1)
	assert.NoError(t, err)
	assert.True(t, mockSvc.IsRunning())
}

func TestServiceManager_RestartService_FromStopped(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = false
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.RestartService(ctx, 1)
	assert.NoError(t, err)
	assert.True(t, mockSvc.IsRunning())
}

func TestServiceManager_RestartService_StopFails(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = true
	mockSvc.stopErr = errors.New("stop failed")
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.RestartService(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop failed")
}

func TestServiceManager_RestartService_StartFails(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.running = false
	mockSvc.startErr = errors.New("start failed")
	manager.SetService(1, mockSvc)

	ctx := context.Background()
	err := manager.RestartService(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start failed")
}

func TestServiceManager_GetAllServices(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc1 := newMockService(1, "service-1", model.ServiceTypeStdio)
	mockSvc2 := newMockService(2, "service-2", model.ServiceTypeSSE)
	mockSvc3 := newMockService(3, "service-3", model.ServiceTypeStreamableHTTP)

	manager.SetService(1, mockSvc1)
	manager.SetService(2, mockSvc2)
	manager.SetService(3, mockSvc3)

	services := manager.GetAllServices()
	assert.Len(t, services, 3)
}

func TestServiceManager_GetAllServices_Empty(t *testing.T) {
	manager := createTestServiceManager()

	services := manager.GetAllServices()
	assert.Len(t, services, 0)
}

func TestServiceManager_UpdateServiceAccessTime(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	manager.SetService(1, mockSvc)

	before := time.Now()
	manager.UpdateServiceAccessTime(1)
	after := time.Now()

	manager.mutex.RLock()
	accessTime := manager.lastAccessed[1]
	manager.mutex.RUnlock()

	assert.True(t, accessTime.After(before) || accessTime.Equal(before))
	assert.True(t, accessTime.Before(after) || accessTime.Equal(after))
}

func TestServiceManager_UpdateServiceConfig(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	manager.SetService(1, mockSvc)

	config := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	err := manager.UpdateServiceConfig(1, config)
	assert.NoError(t, err)

	retrievedConfig := mockSvc.GetConfig()
	assert.Equal(t, "value1", retrievedConfig["key1"])
	assert.Equal(t, 42, retrievedConfig["key2"])
}

func TestServiceManager_UpdateServiceConfig_NotFound(t *testing.T) {
	manager := createTestServiceManager()

	err := manager.UpdateServiceConfig(999, map[string]interface{}{})
	assert.Error(t, err)
	assert.Equal(t, ErrServiceNotFound, err)
}

func TestServiceManager_GetServiceHealth(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.health = &ServiceHealth{
		Status:       StatusHealthy,
		LastChecked:  time.Now(),
		ResponseTime: 100,
	}
	manager.SetService(1, mockSvc)
	manager.healthChecker.RegisterService(mockSvc)

	health, err := manager.GetServiceHealth(1)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusHealthy, health.Status)
}

func TestServiceManager_GetServiceHealthJSON(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc := newMockService(1, "test-service", model.ServiceTypeStdio)
	mockSvc.health = &ServiceHealth{
		Status:      StatusHealthy,
		LastChecked: time.Now(),
	}
	manager.SetService(1, mockSvc)
	manager.healthChecker.RegisterService(mockSvc)

	jsonStr, err := manager.GetServiceHealthJSON(1)
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "healthy")
}

func TestServiceManager_Shutdown(t *testing.T) {
	manager := createTestServiceManager()
	mockSvc1 := newMockService(1, "service-1", model.ServiceTypeStdio)
	mockSvc1.running = true
	mockSvc2 := newMockService(2, "service-2", model.ServiceTypeSSE)
	mockSvc2.running = true

	manager.SetService(1, mockSvc1)
	manager.SetService(2, mockSvc2)
	manager.initialized = true

	ctx := context.Background()
	err := manager.Shutdown(ctx)
	assert.NoError(t, err)
	assert.False(t, manager.initialized)
	assert.Len(t, manager.services, 0)
}

func TestServiceManager_ConcurrentAccess(t *testing.T) {
	manager := createTestServiceManager()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent service registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mockSvc := newMockService(int64(id), "service-"+string(rune('a'+id)), model.ServiceTypeStdio)
			manager.SetService(int64(id), mockSvc)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.GetAllServices()
		}()
	}

	wg.Wait()

	// Verify all services were registered
	services := manager.GetAllServices()
	assert.Equal(t, numGoroutines, len(services))
}

func TestServiceManagerErrors(t *testing.T) {
	assert.Equal(t, "service already exists", ErrServiceAlreadyExists.Error())
	assert.Equal(t, "service not found", ErrServiceNotFound.Error())
	assert.Equal(t, "service start failed", ErrServiceStartFailed.Error())
	assert.Equal(t, "service stop failed", ErrServiceStopFailed.Error())
}
