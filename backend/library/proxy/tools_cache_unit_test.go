package proxy

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/burugo/thing"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHealthyService struct {
	id    int64
	name  string
	tools []mcp.Tool

	running bool
	health  ServiceHealth
}

func (s *fakeHealthyService) ID() int64 { return s.id }
func (s *fakeHealthyService) Name() string {
	return s.name
}
func (s *fakeHealthyService) Type() model.ServiceType { return model.ServiceTypeStdio }
func (s *fakeHealthyService) Start(ctx context.Context) error {
	s.running = true
	return nil
}
func (s *fakeHealthyService) Stop(ctx context.Context) error {
	s.running = false
	return nil
}
func (s *fakeHealthyService) IsRunning() bool { return s.running }
func (s *fakeHealthyService) CheckHealth(ctx context.Context) (*ServiceHealth, error) {
	s.health.Status = StatusHealthy
	s.health.LastChecked = time.Now()
	return &s.health, nil
}
func (s *fakeHealthyService) GetHealth() *ServiceHealth { return &s.health }
func (s *fakeHealthyService) GetConfig() map[string]interface{} {
	return map[string]interface{}{}
}
func (s *fakeHealthyService) UpdateConfig(config map[string]interface{}) error { return nil }
func (s *fakeHealthyService) HealthCheckTimeout() time.Duration                { return 0 }
func (s *fakeHealthyService) GetTools() []mcp.Tool                             { return s.tools }
func (s *fakeHealthyService) GetServerInfo() *mcp.Implementation               { return nil }

func TestToolsCache_EmptyListIsHit(t *testing.T) {
	serviceID := int64(991001)
	toolsCache := GetToolsCacheManager()
	toolsCache.DeleteServiceTools(serviceID)

	toolsCache.SetServiceTools(serviceID, &ToolsCacheEntry{Tools: []mcp.Tool{}, FetchedAt: time.Now()})
	entry, found := toolsCache.GetServiceTools(serviceID)
	assert.True(t, found)
	assert.NotNil(t, entry)
	assert.Equal(t, 0, len(entry.Tools))
}

func TestHealthChecker_PopulatesToolsCacheAndToolCountWhenHealthy(t *testing.T) {
	serviceID := int64(991002)
	GetToolsCacheManager().DeleteServiceTools(serviceID)
	GetHealthCacheManager().DeleteServiceHealth(serviceID)

	hc := NewHealthChecker(1 * time.Hour)
	svc := &fakeHealthyService{
		id:   serviceID,
		name: "fake-healthy",
		tools: []mcp.Tool{
			{Name: "tool-a", Description: "desc"},
		},
		running: true,
	}

	hc.RegisterService(svc)
	hc.checkService(svc)

	entry, found := GetToolsCacheManager().GetServiceTools(serviceID)
	assert.True(t, found)
	assert.Equal(t, 1, len(entry.Tools))

	health, ok := GetHealthCacheManager().GetServiceHealth(serviceID)
	assert.True(t, ok)
	assert.NotNil(t, health)
	assert.Equal(t, 1, health.ToolCount)
	assert.True(t, health.ToolsFetched)
}

func TestServiceManagerUnregisterServiceClearsHealthRegistrationAndCaches(t *testing.T) {
	serviceID := int64(991003)
	manager := &ServiceManager{
		services:      make(map[int64]Service),
		healthChecker: NewHealthChecker(1 * time.Hour),
		lastAccessed:  make(map[int64]time.Time),
	}
	service := &fakeHealthyService{
		id:      serviceID,
		name:    "fake-uninstall",
		running: true,
	}
	manager.services[serviceID] = service
	manager.healthChecker.RegisterService(service)
	manager.lastAccessed[serviceID] = time.Now()
	GetHealthCacheManager().SetServiceHealth(serviceID, &ServiceHealth{Status: StatusHealthy})
	GetToolsCacheManager().SetServiceTools(serviceID, &ToolsCacheEntry{Tools: []mcp.Tool{{Name: "stale"}}})

	assert.NoError(t, manager.UnregisterService(context.Background(), serviceID))
	_, err := manager.GetService(serviceID)
	assert.ErrorIs(t, err, ErrServiceNotFound)
	_, err = manager.ForceCheckServiceHealth(serviceID)
	assert.ErrorIs(t, err, ErrServiceNotRegistered)
	_, healthFound := GetHealthCacheManager().GetServiceHealth(serviceID)
	assert.False(t, healthFound)
	_, toolsFound := GetToolsCacheManager().GetServiceTools(serviceID)
	assert.False(t, toolsFound)
	_, accessFound := manager.lastAccessed[serviceID]
	assert.False(t, accessFound)
}

func TestServiceManagerUnregisterMissingServiceClearsStaleHealthRegistrationAndCaches(t *testing.T) {
	serviceID := int64(991004)
	manager := &ServiceManager{
		services:      make(map[int64]Service),
		healthChecker: NewHealthChecker(1 * time.Hour),
		lastAccessed:  map[int64]time.Time{serviceID: time.Now()},
	}
	service := &fakeHealthyService{id: serviceID, name: "stale-health-registration"}
	manager.healthChecker.RegisterService(service)
	GetHealthCacheManager().SetServiceHealth(serviceID, &ServiceHealth{Status: StatusHealthy})
	GetToolsCacheManager().SetServiceTools(serviceID, &ToolsCacheEntry{Tools: []mcp.Tool{{Name: "stale"}}})

	assert.ErrorIs(t, manager.UnregisterService(context.Background(), serviceID), ErrServiceNotFound)
	_, err := manager.ForceCheckServiceHealth(serviceID)
	assert.ErrorIs(t, err, ErrServiceNotRegistered)
	_, healthFound := GetHealthCacheManager().GetServiceHealth(serviceID)
	assert.False(t, healthFound)
	_, toolsFound := GetToolsCacheManager().GetServiceTools(serviceID)
	assert.False(t, toolsFound)
	_, accessFound := manager.lastAccessed[serviceID]
	assert.False(t, accessFound)
}

func TestServiceManagerRejectsDisabledOrUninstalledServices(t *testing.T) {
	tests := []struct {
		name    string
		service *model.MCPService
	}{
		{
			name: "disabled",
			service: &model.MCPService{
				Name:    "disabled-service",
				Type:    model.ServiceType("test"),
				Enabled: false,
			},
		},
		{
			name: "uninstalled",
			service: &model.MCPService{
				BaseModel: thing.BaseModel{Deleted: true},
				Name:      "uninstalled-service",
				Type:      model.ServiceType("test"),
				Enabled:   true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &ServiceManager{
				services:      make(map[int64]Service),
				healthChecker: NewHealthChecker(1 * time.Hour),
				lastAccessed:  make(map[int64]time.Time),
			}
			assert.Error(t, manager.RegisterService(context.Background(), test.service))
			assert.Empty(t, manager.GetAllServices())
		})
	}
}

func TestServiceManagerEnsureServiceReadyRegistersAndStartsMissingService(t *testing.T) {
	originalSQLitePath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "ensure-service-ready.db")
	require.NoError(t, model.InitDB())
	defer func() { common.SQLitePath = originalSQLitePath }()

	manager := &ServiceManager{
		services:      make(map[int64]Service),
		healthChecker: NewHealthChecker(1 * time.Hour),
		lastAccessed:  make(map[int64]time.Time),
	}
	serviceConfig := &model.MCPService{
		Name:    "missing-on-first-request",
		Type:    model.ServiceType("test"),
		Enabled: true,
	}
	require.NoError(t, model.CreateService(serviceConfig))

	const callers = 8
	services := make([]Service, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			services[index], errs[index] = manager.EnsureServiceReady(context.Background(), serviceConfig)
		}(i)
	}
	wg.Wait()

	service := services[0]
	for i := 0; i < callers; i++ {
		assert.NoError(t, errs[i])
		assert.Same(t, service, services[i])
	}
	assert.True(t, service.IsRunning())
	assert.Len(t, manager.GetAllServices(), 1)
	_, accessed := manager.lastAccessed[serviceConfig.ID]
	assert.True(t, accessed)

	secondService, err := manager.EnsureServiceReady(context.Background(), serviceConfig)
	assert.NoError(t, err)
	assert.Same(t, service, secondService)
	assert.Len(t, manager.GetAllServices(), 1)
}

func TestServiceManagerUninstallLifecycleBlocksStaleFirstRequestRecovery(t *testing.T) {
	originalSQLitePath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "uninstall-lifecycle.db")
	require.NoError(t, model.InitDB())
	defer func() { common.SQLitePath = originalSQLitePath }()

	manager := &ServiceManager{
		services:      make(map[int64]Service),
		healthChecker: NewHealthChecker(1 * time.Hour),
		lastAccessed:  make(map[int64]time.Time),
	}
	serviceConfig := &model.MCPService{
		Name:    "uninstall-race-service",
		Type:    model.ServiceType("test"),
		Enabled: true,
	}
	require.NoError(t, model.CreateService(serviceConfig))
	require.NoError(t, manager.RegisterService(context.Background(), serviceConfig))

	staleConfig := *serviceConfig
	finalizerStarted := make(chan struct{})
	allowFinalizer := make(chan struct{})
	uninstallDone := make(chan error, 1)
	go func() {
		uninstallDone <- manager.UnregisterServiceAndFinalize(context.Background(), serviceConfig.ID, func() error {
			close(finalizerStarted)
			<-allowFinalizer
			serviceConfig.Enabled = false
			serviceConfig.Deleted = true
			return model.UpdateService(serviceConfig)
		})
	}()
	<-finalizerStarted

	ensureDone := make(chan error, 1)
	go func() {
		_, err := manager.EnsureServiceReady(context.Background(), &staleConfig)
		ensureDone <- err
	}()

	select {
	case err := <-ensureDone:
		t.Fatalf("stale recovery completed before uninstall finalizer: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(allowFinalizer)
	require.NoError(t, <-uninstallDone)
	assert.Error(t, <-ensureDone)
	_, err := manager.GetService(serviceConfig.ID)
	assert.ErrorIs(t, err, ErrServiceNotFound)
}
