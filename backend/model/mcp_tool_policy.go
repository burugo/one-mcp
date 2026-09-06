package model

import (
	"errors"
	"fmt"
	"sync"

	"github.com/burugo/thing"
)

var ErrMCPToolPolicyNotFound = errors.New("mcp_tool_policy_not_found")

// MCPToolPolicy stores the administrator-controlled state for one discovered tool.
// Missing rows intentionally mean enabled so newly discovered tools remain available.
type MCPToolPolicy struct {
	thing.BaseModel
	ServiceID int64  `db:"service_id,unique:uq_mcp_tool_policy" json:"service_id"`
	ToolName  string `db:"tool_name,unique:uq_mcp_tool_policy" json:"tool_name"`
	Enabled   bool   `db:"enabled" json:"enabled"`
	UpdatedBy int64  `db:"updated_by" json:"updated_by"`
}

func (p *MCPToolPolicy) TableName() string {
	return "mcp_tool_policies"
}

var MCPToolPolicyDB *thing.Thing[*MCPToolPolicy]
var mcpToolPolicyWriteMu sync.Mutex

func MCPToolPolicyInit() error {
	var err error
	MCPToolPolicyDB, err = thing.Use[*MCPToolPolicy]()
	if err != nil {
		return fmt.Errorf("initialize MCP tool policies: %w", err)
	}
	return nil
}

func GetMCPToolPolicy(serviceID int64, toolName string) (*MCPToolPolicy, error) {
	policies, err := MCPToolPolicyDB.Where("service_id = ? AND tool_name = ?", serviceID, toolName).Fetch(0, 1)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, ErrMCPToolPolicyNotFound
	}
	return policies[0], nil
}

func SetMCPToolPolicy(serviceID int64, toolName string, enabled bool, updatedBy int64) (*MCPToolPolicy, error) {
	mcpToolPolicyWriteMu.Lock()
	defer mcpToolPolicyWriteMu.Unlock()

	policy, err := GetMCPToolPolicy(serviceID, toolName)
	if err != nil {
		if !errors.Is(err, ErrMCPToolPolicyNotFound) {
			return nil, err
		}
		policy = &MCPToolPolicy{ServiceID: serviceID, ToolName: toolName}
	}
	policy.Enabled = enabled
	policy.UpdatedBy = updatedBy
	if err := MCPToolPolicyDB.Save(policy); err != nil {
		return nil, err
	}
	return policy, nil
}

func GetMCPToolPoliciesForService(serviceID int64) ([]*MCPToolPolicy, error) {
	return MCPToolPolicyDB.Where("service_id = ?", serviceID).All()
}

func GetAllMCPToolPolicies() ([]*MCPToolPolicy, error) {
	return MCPToolPolicyDB.All()
}

func DeleteMCPToolPoliciesForService(serviceID int64) error {
	policies, err := GetMCPToolPoliciesForService(serviceID)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if err := MCPToolPolicyDB.Delete(policy); err != nil {
			return err
		}
	}
	return nil
}
