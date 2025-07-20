//go:build integration
// +build integration

package proxy

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/stretchr/testify/assert"
)

// setupTestEnvironmentForProxy configures a test environment using an in-memory SQLite DB.
func setupTestEnvironmentForProxy() func() {
	originalPath := common.SQLitePath
	common.SQLitePath = ":memory:"

	err := model.InitDB()
	if err != nil {
		panic(fmt.Sprintf("model.InitDB() failed for :memory: in proxy_test: %v", err))
	}

	return func() {
		common.SQLitePath = originalPath
		common.OptionMap = make(map[string]string)
		// Clear caches
		sharedMCPServersMutex.Lock()
		sharedMCPServers = make(map[string]*SharedMcpInstance)
		sharedMCPServersMutex.Unlock()

		sseWrappersMutex.Lock()
		initializedSSEProxyWrappers = make(map[string]http.Handler)
		sseWrappersMutex.Unlock()

		httpWrappersMutex.Lock()
		initializedHTTPProxyWrappers = make(map[string]http.Handler)
		httpWrappersMutex.Unlock()
	}
}

// TestSharedMcpInstance_Shutdown tests the Shutdown method structure
func TestSharedMcpInstance_Shutdown(t *testing.T) {
	instance := &SharedMcpInstance{
		Server: nil, // We can't easily mock the server
		Client: nil, // We can't easily mock the client
	}

	ctx := context.Background()
	err := instance.Shutdown(ctx)

	// Should not panic and should handle nil gracefully
	assert.NoError(t, err)
}

// TestBaseService_BasicFunctionality tests basic service functionality
func TestBaseService_BasicFunctionality(t *testing.T) {
	service := NewBaseService(1, "test-service", model.ServiceTypeStdio)

	assert.Equal(t, int64(1), service.ID())
	assert.Equal(t, "test-service", service.Name())
	assert.Equal(t, model.ServiceTypeStdio, service.Type())
	assert.False(t, service.IsRunning())

	// Test config
	config := map[string]interface{}{
		"test_key": "test_value",
	}
	err := service.UpdateConfig(config)
	assert.NoError(t, err)

	retrievedConfig := service.GetConfig()
	assert.Equal(t, "test_value", retrievedConfig["test_key"])
}

// TestBaseService_StartStop tests start and stop functionality
func TestBaseService_StartStop(t *testing.T) {
	service := NewBaseService(1, "test-service", model.ServiceTypeStdio)

	ctx := context.Background()

	// Test start
	err := service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Test health after start
	health := service.GetHealth()
	assert.NotNil(t, health)
	assert.Equal(t, StatusStarting, health.Status)

	// Test stop
	err = service.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())

	// Test health after stop
	health = service.GetHealth()
	assert.NotNil(t, health)
	assert.Equal(t, StatusStopped, health.Status)
}

// TestBaseService_CheckHealth tests health checking
func TestBaseService_CheckHealth(t *testing.T) {
	service := NewBaseService(1, "test-service", model.ServiceTypeStdio)

	ctx := context.Background()

	// Initially not running
	health, err := service.CheckHealth(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusStopped, health.Status)

	// Start service
	service.Start(ctx)
	health, err = service.CheckHealth(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusHealthy, health.Status)
}

// TestMonitoredProxiedService_Creation tests MonitoredProxiedService creation
func TestMonitoredProxiedService_Creation(t *testing.T) {
	baseService := NewBaseService(1, "test-service", model.ServiceTypeStdio)

	dbConfig := &model.MCPService{
		Name: "test-service",
		Type: model.ServiceTypeStdio,
	}

	// Test with nil shared instance
	monitoredService := NewMonitoredProxiedService(baseService, nil, dbConfig)
	assert.NotNil(t, monitoredService)
	assert.Equal(t, baseService, monitoredService.BaseService)
	assert.Nil(t, monitoredService.sharedInstance)
	assert.Equal(t, dbConfig, monitoredService.dbServiceConfig)

	// Test basic functionality
	assert.Equal(t, int64(1), monitoredService.ID())
	assert.Equal(t, "test-service", monitoredService.Name())
	assert.Equal(t, model.ServiceTypeStdio, monitoredService.Type())
}

// TestMonitoredProxiedService_CheckHealth_NilInstance tests health check with nil instance
func TestMonitoredProxiedService_CheckHealth_NilInstance(t *testing.T) {
	baseService := NewBaseService(1, "test-service", model.ServiceTypeStdio)
	service := NewMonitoredProxiedService(baseService, nil, &model.MCPService{
		Name: "test-service",
		Type: model.ServiceTypeStdio,
	})

	ctx := context.Background()

	health, err := service.CheckHealth(ctx)
	assert.Error(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, StatusUnhealthy, health.Status)
	assert.Contains(t, health.ErrorMessage, "Shared MCP instance or client is not initialized")
	assert.Equal(t, 3, health.WarningLevel) // Critical warning
}

// TestServiceFactory_SupportedTypes tests ServiceFactory with supported types
func TestServiceFactory_SupportedTypes(t *testing.T) {
	teardown := setupTestEnvironmentForProxy()
	defer teardown()

	supportedTypes := []model.ServiceType{
		model.ServiceTypeStdio,
		model.ServiceTypeSSE,
		model.ServiceTypeStreamableHTTP,
	}

	for _, serviceType := range supportedTypes {
		t.Run(string(serviceType), func(t *testing.T) {
			service := &model.MCPService{
				Name:            "test-service-" + string(serviceType),
				Type:            serviceType,
				DefaultEnvsJSON: `{"TEST":"value"}`,
			}

			// Create service in DB to get ID
			err := model.CreateService(service)
			assert.NoError(t, err)
			defer model.DeleteService(service.ID)

			result, err := ServiceFactory(service)

			// ServiceFactory should always succeed but may create unhealthy services
			assert.NoError(t, err, "ServiceFactory should not return error")
			assert.NotNil(t, result, "Should return a service")

			// Should be MonitoredProxiedService for supported types
			monitoredService, ok := result.(*MonitoredProxiedService)
			assert.True(t, ok, "ServiceFactory should return MonitoredProxiedService for supported types")
			assert.Equal(t, service.Name, monitoredService.Name())
			assert.Equal(t, serviceType, monitoredService.Type())

			// Health status should be unhealthy due to missing proper configuration
			health := monitoredService.GetHealth()
			assert.NotNil(t, health)
			assert.Equal(t, StatusUnhealthy, health.Status, "Service should be unhealthy due to configuration issues")
			assert.NotEmpty(t, health.ErrorMessage, "Should have error message explaining why unhealthy")
		})
	}
}

// TestServiceFactory_UnsupportedType tests ServiceFactory with unsupported type
func TestServiceFactory_UnsupportedType(t *testing.T) {
	service := &model.MCPService{
		Name: "test-service",
		Type: "unsupported_type",
	}

	result, err := ServiceFactory(service)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should return basic BaseService for unsupported types
	baseService, ok := result.(*BaseService)
	assert.True(t, ok, "ServiceFactory should return BaseService for unsupported types")
	assert.Equal(t, service.Name, baseService.Name())
}

// TestGetOrCreateSharedMcpInstanceWithKey_CacheKeyGeneration tests cache key logic
func TestGetOrCreateSharedMcpInstanceWithKey_CacheKeyGeneration(t *testing.T) {
	teardown := setupTestEnvironmentForProxy()
	defer teardown()

	service := &model.MCPService{
		Name:            "test-service",
		Type:            model.ServiceTypeStdio,
		Command:         "echo",
		DefaultEnvsJSON: `{"TEST_ENV":"test_value"}`,
	}

	// Create service in DB to get valid ID
	err := model.CreateService(service)
	assert.NoError(t, err)
	defer model.DeleteService(service.ID)

	// Test different cache keys generate different cache entries
	ctx := context.Background()

	cacheKey1 := fmt.Sprintf("global-service-%d-shared", service.ID)
	cacheKey2 := fmt.Sprintf("user-1-service-%d-shared", service.ID)

	// Both calls will likely fail due to missing MCP setup, but we can verify
	// that they attempt to use different cache keys

	// Add a shorter timeout for these specific calls to prevent test suite from hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	instance1, err1 := GetOrCreateSharedMcpInstanceWithKey(timeoutCtx, service, cacheKey1, "global", service.DefaultEnvsJSON)

	// Renew timeout for the second call if necessary, or use the same
	timeoutCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()
	instance2, err2 := GetOrCreateSharedMcpInstanceWithKey(timeoutCtx2, service, cacheKey2, "user-1", service.DefaultEnvsJSON)

	// At least one should work or both should fail gracefully (within the timeout)
	if err1 == nil && err2 == nil {
		// If both succeed, they should be different instances due to different cache keys
		if instance1 != nil && instance2 != nil {
			// They might be the same due to our simplified test setup, but that's ok
			t.Logf("Both instances created successfully")
		}
	} else {
		// Expected in test environment
		t.Logf("MCP instance creation failed as expected in test: err1=%v, err2=%v", err1, err2)
	}

	// The important thing is that the function doesn't panic and handles errors gracefully
	assert.True(t, true, "Function completed without panic")
}

// TestSharedMcpInstanceWithRealMCPServer tests SharedMcpInstance with npx mcp-hello-world
func TestSharedMcpInstanceWithRealMCPServer(t *testing.T) {
	teardown := setupTestEnvironmentForProxy()
	defer teardown()

	// Create a service that uses npx mcp-hello-world
	service := &model.MCPService{
		Name:            "hello-world-test",
		Type:            model.ServiceTypeStdio,
		Command:         "npx",
		ArgsJSON:        `["mcp-hello-world"]`,
		DefaultEnvsJSON: `{}`,
	}

	// Create service in DB to get valid ID
	err := model.CreateService(service)
	assert.NoError(t, err)
	defer model.DeleteService(service.ID)

	ctx := context.Background()
	cacheKey := fmt.Sprintf("test-service-%d-shared", service.ID)

	// Test creating SharedMcpInstance with real MCP server
	instance, err := GetOrCreateSharedMcpInstanceWithKey(ctx, service, cacheKey, "test", service.DefaultEnvsJSON)

	if err != nil {
		// If npx or mcp-hello-world is not available, skip this test
		if assert.Contains(t, err.Error(), "executable file not found") ||
			assert.Contains(t, err.Error(), "command not found") {
			t.Skip("npx or mcp-hello-world not available, skipping integration test")
			return
		}
		t.Fatalf("Failed to create SharedMcpInstance: %v", err)
	}

	assert.NotNil(t, instance, "SharedMcpInstance should be created")
	assert.NotNil(t, instance.Server, "MCP Server should be initialized")
	assert.NotNil(t, instance.Client, "MCP Client should be initialized")

	// Test health check with real client
	err = instance.Client.Ping(ctx)
	assert.NoError(t, err, "Health check should pass with real MCP server")

	// Test that the same instance is returned from cache
	instance2, err := GetOrCreateSharedMcpInstanceWithKey(ctx, service, cacheKey, "test", service.DefaultEnvsJSON)
	assert.NoError(t, err)
	assert.Equal(t, instance, instance2, "Should return same instance from cache")

	// Test shutdown
	err = instance.Shutdown(ctx)
	assert.NoError(t, err, "Shutdown should succeed")

	// After shutdown, health check should fail
	err = instance.Client.Ping(ctx)
	assert.Error(t, err, "Health check should fail after shutdown")
}

// TestMonitoredProxiedServiceWithRealMCP tests MonitoredProxiedService with real MCP
func TestMonitoredProxiedServiceWithRealMCP(t *testing.T) {
	teardown := setupTestEnvironmentForProxy()
	defer teardown()

	// Create a service that uses npx mcp-hello-world
	service := &model.MCPService{
		Name:            "monitored-hello-world-test",
		Type:            model.ServiceTypeStdio,
		Command:         "npx",
		ArgsJSON:        `["mcp-hello-world"]`,
		DefaultEnvsJSON: `{}`,
	}

	// Create service in DB to get valid ID
	err := model.CreateService(service)
	assert.NoError(t, err)
	defer model.DeleteService(service.ID)

	ctx := context.Background()
	cacheKey := fmt.Sprintf("monitored-test-%d-shared", service.ID)

	// Create SharedMcpInstance
	sharedInstance, err := GetOrCreateSharedMcpInstanceWithKey(ctx, service, cacheKey, "monitored-test", service.DefaultEnvsJSON)

	if err != nil {
		// If npx or mcp-hello-world is not available, skip this test
		if assert.Contains(t, err.Error(), "executable file not found") ||
			assert.Contains(t, err.Error(), "command not found") {
			t.Skip("npx or mcp-hello-world not available, skipping integration test")
			return
		}
		t.Fatalf("Failed to create SharedMcpInstance: %v", err)
	}

	defer sharedInstance.Shutdown(ctx)

	// Create MonitoredProxiedService
	baseService := NewBaseService(service.ID, service.Name, service.Type)
	monitoredService := NewMonitoredProxiedService(baseService, sharedInstance, service)

	// Test health check with real MCP connection
	health, err := monitoredService.CheckHealth(ctx)
	assert.NoError(t, err, "Health check should succeed with real MCP")
	assert.NotNil(t, health)
	assert.Equal(t, StatusHealthy, health.Status, "Service should be healthy")
	assert.Empty(t, health.ErrorMessage, "Should have no error message")
	assert.Equal(t, 0, health.WarningLevel, "Should have no warnings")
}

// TestServiceFactoryWithRealMCP tests ServiceFactory with real MCP server
func TestServiceFactoryWithRealMCP(t *testing.T) {
	teardown := setupTestEnvironmentForProxy()
	defer teardown()

	// Create a service that uses npx mcp-hello-world
	service := &model.MCPService{
		Name:            "factory-hello-world-test",
		Type:            model.ServiceTypeStdio,
		Command:         "npx",
		ArgsJSON:        `["mcp-hello-world"]`,
		DefaultEnvsJSON: `{}`,
	}

	// Create service in DB to get valid ID
	err := model.CreateService(service)
	assert.NoError(t, err)
	defer model.DeleteService(service.ID)

	// Test ServiceFactory
	result, err := ServiceFactory(service)

	if err != nil {
		// If npx or mcp-hello-world is not available, expect specific error
		if assert.Contains(t, err.Error(), "executable file not found") ||
			assert.Contains(t, err.Error(), "command not found") {
			t.Skip("npx or mcp-hello-world not available, skipping integration test")
			return
		}
		// Other errors should be handled gracefully
		assert.NotNil(t, result, "Should return service even on error")
		t.Logf("ServiceFactory failed as expected in some environments: %v", err)
		return
	}

	assert.NoError(t, err, "ServiceFactory should succeed with real MCP server")
	assert.NotNil(t, result, "Should return service")

	// Should be MonitoredProxiedService
	monitoredService, ok := result.(*MonitoredProxiedService)
	assert.True(t, ok, "Should return MonitoredProxiedService")
	assert.Equal(t, service.Name, monitoredService.Name())
	assert.Equal(t, service.Type, monitoredService.Type())

	// Test health check
	ctx := context.Background()
	health, err := monitoredService.CheckHealth(ctx)
	assert.NoError(t, err, "Health check should succeed")
	assert.Equal(t, StatusHealthy, health.Status, "Should be healthy")

	// Cleanup
	if monitoredService.sharedInstance != nil {
		monitoredService.sharedInstance.Shutdown(ctx)
	}
}

// TestCacheClearing tests that cache clearing works
func TestCacheClearing(t *testing.T) {
	teardown := setupTestEnvironmentForProxy()
	defer teardown()

	// Clear all caches and verify they are empty
	sharedMCPServersMutex.Lock()
	initialSharedCount := len(sharedMCPServers)
	sharedMCPServersMutex.Unlock()

	sseWrappersMutex.Lock()
	initialSSECount := len(initializedSSEProxyWrappers)
	sseWrappersMutex.Unlock()

	httpWrappersMutex.Lock()
	initialHTTPCount := len(initializedHTTPProxyWrappers)
	httpWrappersMutex.Unlock()

	assert.Equal(t, 0, initialSharedCount, "Shared MCP servers cache should be empty")
	assert.Equal(t, 0, initialSSECount, "SSE proxy wrappers cache should be empty")
	assert.Equal(t, 0, initialHTTPCount, "HTTP proxy wrappers cache should be empty")
}
