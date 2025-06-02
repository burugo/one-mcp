package model

import (
	"errors"
	"fmt"
	"sync"

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
var proxyRequestStatThing *thing.Thing[ProxyRequestStat]
var initStatThingOnce sync.Once
var initStatThingErr error // To store initialization error

// GetProxyRequestStatThing initializes and returns the Thing ORM instance for ProxyRequestStat.
// This function is now public.
func GetProxyRequestStatThing() (*thing.Thing[ProxyRequestStat], error) {
	initStatThingOnce.Do(func() {
		// Use thing.Use, assuming thing.Configure was called at application startup.
		ormInstance, err := thing.Use[ProxyRequestStat]()
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
		// BaseModel fields (ID, CreatedAt, etc.) are handled by Thing ORM when passed by value to Save
		// if the ORM supports it, or are set to zero/defaults if not.
	}

	// Adhering to linter: passing stat by value.
	// This means the 'stat' variable in this function scope will not be updated with ID/timestamps post-save.
	if err := statThing.Save(stat); err != nil {
		common.SysError(fmt.Sprintf("Error saving ProxyRequestStat: %v", err))
	}
}

// TODO: Consider if a separate model for aggregated stats is needed, or if aggregation will be done via queries.
