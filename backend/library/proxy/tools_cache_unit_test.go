package proxy

import (
	"context"
	"testing"
	"time"

	"one-mcp/backend/model"

	"github.com/burugo/thing"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
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
