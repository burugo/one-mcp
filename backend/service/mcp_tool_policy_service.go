package service

import (
	"errors"
	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/mcp"
)

// IsMCPToolEnabled returns true when no explicit administrator policy exists.
func IsMCPToolEnabled(serviceID int64, toolName string) (bool, error) {
	policy, err := model.GetMCPToolPolicy(serviceID, toolName)
	if err != nil {
		if errors.Is(err, model.ErrMCPToolPolicyNotFound) {
			return true, nil
		}
		return false, err
	}
	return policy.Enabled, nil
}

// FilterEnabledMCPTools applies global administrator policy without modifying
// the upstream inventory slice.
func FilterEnabledMCPTools(serviceID int64, tools []mcp.Tool) ([]mcp.Tool, error) {
	policies, err := model.GetMCPToolPoliciesForService(serviceID)
	if err != nil {
		return nil, err
	}
	disabled := make(map[string]struct{}, len(policies))
	for _, policy := range policies {
		if !policy.Enabled {
			disabled[policy.ToolName] = struct{}{}
		}
	}
	filtered := make([]mcp.Tool, 0, len(tools))
	for _, tool := range tools {
		if _, blocked := disabled[tool.Name]; blocked {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered, nil
}
