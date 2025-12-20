package proxy

import (
	"context"
	"testing"
	"time"

	"one-mcp/backend/model"

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
func (s *fakeHealthyService) HealthCheckTimeout() time.Duration   { return 0 }
func (s *fakeHealthyService) GetTools() []mcp.Tool                { return s.tools }
func (s *fakeHealthyService) GetServerInfo() *mcp.Implementation  { return nil }

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
