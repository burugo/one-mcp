package model

import (
	"context"
	"fmt"
	"regexp"

	"github.com/burugo/thing"
)

// MCPLogPhase represents the phase of MCP operation
type MCPLogPhase string

const (
	MCPLogPhaseInstall MCPLogPhase = "install"
	MCPLogPhaseRun     MCPLogPhase = "run"
)

// MCPLogLevel represents the log level
type MCPLogLevel string

const (
	MCPLogLevelInfo  MCPLogLevel = "info"
	MCPLogLevelWarn  MCPLogLevel = "warn"
	MCPLogLevelError MCPLogLevel = "error"
)

// MCPLog represents a log entry for MCP service operations
type MCPLog struct {
	thing.BaseModel
	ServiceID   int64       `db:"service_id,index:idx_service_time" json:"service_id"`
	ServiceName string      `db:"service_name,index:idx_name_time" json:"service_name"`
	Phase       MCPLogPhase `db:"phase,index:idx_phase_time" json:"phase"`
	Level       MCPLogLevel `db:"level" json:"level"`
	Message     string      `db:"message" json:"message"`
	// BaseModel already includes: ID, CreatedAt, UpdatedAt, Deleted
}

// TableName sets the table name for the MCPLog model
func (l *MCPLog) TableName() string {
	return "mcp_logs"
}

var MCPLogDB *thing.Thing[*MCPLog]

// MCPLogInit initializes the MCPLogDB
func MCPLogInit() error {
	var err error
	MCPLogDB, err = thing.Use[*MCPLog]()
	if err != nil {
		return fmt.Errorf("failed to initialize MCPLogDB: %w", err)
	}
	return nil
}

// CreateMCPLog creates a new MCP log entry
func CreateMCPLog(log *MCPLog) error {
	return MCPLogDB.Save(log)
}

// GetMCPLogs retrieves MCP logs with filtering and pagination
func GetMCPLogs(ctx context.Context, serviceID *int64, serviceName, phase, level *string, page, pageSize int) ([]*MCPLog, int64, error) {
	query := MCPLogDB.Query(thing.QueryParams{})

	// Apply filters
	if serviceID != nil {
		query = query.Where("service_id = ?", *serviceID)
	}
	if serviceName != nil && *serviceName != "" {
		query = query.Where("service_name LIKE ?", "%"+*serviceName+"%")
	}
	if phase != nil && *phase != "" {
		query = query.Where("phase = ?", *phase)
	}
	if level != nil && *level != "" {
		query = query.Where("level = ?", *level)
	}

	// Get total count first
	total, err := query.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count MCP logs: %w", err)
	}

	// Get paginated results
	logs, err := query.Order("created_at DESC").Fetch((page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch MCP logs: %w", err)
	}

	return logs, total, nil
}

// GetMCPLogThing returns the initialized Thing ORM instance for MCPLog
func GetMCPLogThing() (*thing.Thing[*MCPLog], error) {
	if MCPLogDB == nil {
		if err := MCPLogInit(); err != nil {
			return nil, fmt.Errorf("failed to explicitly initialize MCPLogDB in GetMCPLogThing: %w", err)
		}
		if MCPLogDB == nil {
			return nil, fmt.Errorf("MCPLogDB is nil even after explicit re-initialization attempt in GetMCPLogThing")
		}
	}
	return MCPLogDB, nil
}

// SaveMCPLog is a utility function to save MCP logs with message length limit and sanitization
func SaveMCPLog(ctx context.Context, serviceID int64, serviceName string, phase MCPLogPhase, level MCPLogLevel, message string) error {
	// Limit message length to prevent database bloat
	const maxMessageLength = 8192
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength] + "... [truncated]"
	}

	// Simple sanitization to remove sensitive information
	message = sanitizeMessage(message)

	log := &MCPLog{
		ServiceID:   serviceID,
		ServiceName: serviceName,
		Phase:       phase,
		Level:       level,
		Message:     message,
	}

	return CreateMCPLog(log)
}

// sanitizeMessage removes potentially sensitive information from log messages
func sanitizeMessage(message string) string {
	// Simple regex-based sanitization for common sensitive patterns
	// This is a basic implementation - could be enhanced with more sophisticated patterns

	// Replace Bearer tokens
	re := `(?i)(bearer\s+)[a-zA-Z0-9\-_.]+`
	message = regexp.MustCompile(re).ReplaceAllString(message, "${1}***")

	// Replace API keys
	re = `(?i)(api[_-]?key[^=]*[=:]?\s*)[a-zA-Z0-9\-_.]+`
	message = regexp.MustCompile(re).ReplaceAllString(message, "${1}***")

	// Replace tokens
	re = `(?i)(token[^=]*[=:]?\s*)[a-zA-Z0-9\-_.]+`
	message = regexp.MustCompile(re).ReplaceAllString(message, "${1}***")

	// Replace passwords
	re = `(?i)(password[^=]*[=:]?\s*)[^\s]+`
	message = regexp.MustCompile(re).ReplaceAllString(message, "${1}***")

	return message
}
