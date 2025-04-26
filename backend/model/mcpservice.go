package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// MCPServiceType defines the type of MCP service connection.
// Matches the 'type' field in the technical architecture.
type MCPServiceType string

const (
	MCPServiceTypeStdio MCPServiceType = "stdio"
	MCPServiceTypeSSE   MCPServiceType = "sse"
	// Add other types here if needed
)

// JSONRawMessage is a wrapper for json.RawMessage to handle database storage.
// GORM might need help storing raw JSON depending on the driver (especially older ones).
// For SQLite and modern PostgreSQL with JSON/JSONB, this might be simplified,
// but this custom type provides explicit marshalling/unmarshalling.
type JSONRawMessage json.RawMessage

// Value implements the driver.Valuer interface, marshalling the json.RawMessage.
func (j JSONRawMessage) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	bytes, err := json.Marshal(j)
	return string(bytes), err
}

// Scan implements the sql.Scanner interface, unmarshalling the data from the database.
func (j *JSONRawMessage) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		// If it's a string, try to treat it as JSON bytes
		str, okStr := value.(string)
		if !okStr {
			return errors.New("Scan source is not []byte or string")
		}
		bytes = []byte(str)
	}
	if len(bytes) == 0 {
		*j = nil
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// MarshalJSON returns the *j as the JSON encoding of j.
func (j JSONRawMessage) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets *j to a copy of data.
func (j *JSONRawMessage) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONRawMessage: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// MCPService represents the configuration for an external MCP service.
// Based on the technical architecture document.
type MCPService struct {
	Id        int            `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"index;size:100;not null"`
	Type      MCPServiceType `json:"type" gorm:"size:10;not null"` // "stdio" or "sse"
	Config    JSONRawMessage `json:"config" gorm:"type:text"`       // Stored as JSON text in DB
	IsActive  bool           `json:"is_active" gorm:"index;default:true"`
	CallCount int            `json:"call_count" gorm:"default:0"`
	Status    string         `json:"status" gorm:"size:50;default:'UNKNOWN'"` // For health check status (Task 5.3)
	CreatedBy int            `json:"created_by" gorm:"index"`             // Foreign key to User.Id
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
} 