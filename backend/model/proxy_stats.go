package model

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"one-mcp/backend/common" // For SysError logging, if available and configured

	"github.com/burugo/thing"
)

// ProxyRequestType defines the type of proxied request for statistics.
type ProxyRequestType string

const (
	ProxyRequestTypeSSE  ProxyRequestType = "sse"
	ProxyRequestTypeHTTP ProxyRequestType = "http"
)

// ProxyRequestStat represents a single recorded statistic for a proxied request.
// This model will be used with the Thing ORM for database operations.
type ProxyRequestStat struct {
	thing.BaseModel                  // Includes ID, CreatedAt, UpdatedAt, DeletedAt
	ServiceID       int64            `db:"service_id,index"`
	ServiceName     string           `db:"service_name"` // Denormalized for easier querying, but can be joined from MCPService
	UserID          int64            `db:"user_id,index"`
	RequestType     ProxyRequestType `db:"request_type,index"` // "sse" or "http"
	Method          string           `db:"method"`             // e.g., "tools/call" for http, "message" for sse
	RequestPath     string           `db:"request_path"`
	ResponseTimeMs  int64            `db:"response_time_ms"`
	StatusCode      int              `db:"status_code"`
	Success         bool             `db:"success,index"`
	// CreatedAt from BaseModel will be used for the timestamp of the request
}

// TableName specifies the database table name for ProxyRequestStat.
func (prs *ProxyRequestStat) TableName() string {
	return "proxy_request_stats"
}

// proxyRequestStatThing is a global Thing ORM instance for ProxyRequestStat.
// It's initialized once to be reused.
var proxyRequestStatThing *thing.Thing[*ProxyRequestStat]
var initStatThingOnce sync.Once
var initStatThingErr error // To store initialization error

// GetProxyRequestStatThing initializes and returns the Thing ORM instance for ProxyRequestStat.
// This function is now public.
func GetProxyRequestStatThing() (*thing.Thing[*ProxyRequestStat], error) {
	initStatThingOnce.Do(func() {
		// Use thing.Use, assuming thing.Configure was called at application startup.
		ormInstance, err := thing.Use[*ProxyRequestStat]()
		if err != nil {
			msg := fmt.Sprintf("Error initializing ProxyRequestStatThing with thing.Use: %v. DB might not be configured globally for Thing ORM.", err)
			common.SysError(msg)               // Using common.SysError for consistent logging
			initStatThingErr = errors.New(msg) // Store the error
			proxyRequestStatThing = nil
			return
		}
		proxyRequestStatThing = ormInstance
	})

	if initStatThingErr != nil {
		return nil, initStatThingErr
	}
	if proxyRequestStatThing == nil && initStatThingErr == nil {
		// This case should ideally not be reached if initStatThingOnce.Do completed without error
		// but a race condition occurred or Do didn't run. Or if common.SysError panics and is recovered outside.
		return nil, errors.New("ProxyRequestStatThing is nil after initialization attempt without a specific error")
	}
	return proxyRequestStatThing, nil
}

// RecordRequestStat creates and saves a ProxyRequestStat entry.
// It will degrade gracefully (log and not save) if the ORM instance is not initialized.
func RecordRequestStat(serviceID int64, serviceName string, userID int64, reqType ProxyRequestType, method string, requestPath string, responseTimeMs int64, statusCode int, success bool) {
	statThing, err := GetProxyRequestStatThing()
	if err != nil {
		common.SysError(fmt.Sprintf("Failed to get ProxyRequestStatThing, cannot record stat: %v", err))
		return
	}

	stat := ProxyRequestStat{
		ServiceID:      serviceID,
		ServiceName:    serviceName,
		UserID:         userID,
		RequestType:    reqType,
		Method:         method,
		RequestPath:    requestPath,
		ResponseTimeMs: responseTimeMs,
		StatusCode:     statusCode,
		Success:        success,
	}

	if err := statThing.Save(&stat); err != nil {
		common.SysError(fmt.Sprintf("Error saving ProxyRequestStat: %v", err))
		// Do not return here, try to update cache even if DB save fails for some reason?
		// Or should we return? For now, let's try to update cache.
		// User's original code proceeded with cache update irrespective of this specific error.
	}

	// Record daily request count to cache only if status is 200 or 202
	if statusCode == http.StatusOK || statusCode == http.StatusAccepted {
		cacheClient := thing.Cache()
		if cacheClient == nil {
			common.SysError(fmt.Sprintf("[RecordRequestStat-CACHE] Cache client is nil for service %s (ID: %d)", serviceName, serviceID))
			return
		}

		today := time.Now().Format("2006-01-02")
		ctx := context.Background() // Using background context as in original handler

		// Increment global service request count
		globalCacheKey := fmt.Sprintf("request:%s:%d:count", today, serviceID)
		globalNewCount, err := cacheClient.Incr(ctx, globalCacheKey)
		if err != nil {
			common.SysError(fmt.Sprintf("[RecordRequestStat-CACHE] Error incrementing daily count for service %s (ID: %d): %v", serviceName, serviceID, err))
		} else {
			if globalNewCount == 1 {
				// Key was newly created by Incr, set expiration
				err = cacheClient.Expire(ctx, globalCacheKey, 24*time.Hour)
				if err != nil {
					// Log error for expiry failure, but the count was successfully incremented.
					common.SysError(fmt.Sprintf("[RecordRequestStat-CACHE] Error setting expiration for new daily count key %s (service %s, ID: %d): %v", globalCacheKey, serviceName, serviceID, err))
				}
			}
			common.SysLog(fmt.Sprintf("[RecordRequestStat-CACHE] Daily count for service %s (ID: %d): %d", serviceName, serviceID, globalNewCount))
		}

		// Increment user-specific request count if user is authenticated
		if userID > 0 {
			userCacheKey := fmt.Sprintf("user_request:%s:%d:%d:count", today, serviceID, userID)
			userNewCount, userErr := cacheClient.Incr(ctx, userCacheKey)
			if userErr != nil {
				common.SysError(fmt.Sprintf("[RecordRequestStat-CACHE] Error incrementing user daily count for service %d, user %d: %v", serviceID, userID, userErr))
			} else {
				if userNewCount == 1 {
					// Key was newly created by Incr, set expiration
					err = cacheClient.Expire(ctx, userCacheKey, 24*time.Hour)
					if err != nil {
						common.SysError(fmt.Sprintf("[RecordRequestStat-CACHE] Error setting expiration for user daily count key %s: %v", userCacheKey, err))
					}
				}
				common.SysLog(fmt.Sprintf("[RecordRequestStat-CACHE] User %d daily count for service %d: %d", userID, serviceID, userNewCount))
			}
		}
	} else {
		common.SysLog(fmt.Sprintf("[RecordRequestStat-CACHE] Daily count for service %s (ID: %d) not incremented due to status code: %d", serviceName, serviceID, statusCode))
	}
}

// TODO: Consider if a separate model for aggregated stats is needed, or if aggregation will be done via queries.
