package proxy

import "fmt"

// SharedServiceCacheKey generates the cache key for a shared MCP service instance.
func SharedServiceCacheKey(serviceID int64) string {
	return fmt.Sprintf("global-service-%d-shared", serviceID)
}

// SharedServiceInstanceName generates the instance name for a shared MCP service.
func SharedServiceInstanceName(serviceID int64) string {
	return fmt.Sprintf("global-shared-svc-%d", serviceID)
}
