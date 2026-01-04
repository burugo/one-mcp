package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"strings"

	"github.com/gin-gonic/gin"
	mcp_protocol "github.com/mark3labs/mcp-go/mcp"
)

type MCPRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"params"`
	ID any `json:"id"`
}

type MCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

type groupSearchArgs struct {
	MCPName string
}

type executeArgs struct {
	MCPName   string
	ToolName  string
	Arguments map[string]any
}

func GroupMCPHandler(c *gin.Context) {
	groupName := c.Param("name")
	userID := c.GetInt64("user_id")

	if userID == 0 {
		common.RespErrorStr(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	group, err := model.GetMCPServiceGroupByName(groupName, userID)
	if err != nil {
		common.RespError(c, http.StatusNotFound, "Group not found", err)
		return
	}

	if !group.Enabled {
		common.RespErrorStr(c, http.StatusServiceUnavailable, "Group disabled")
		return
	}

	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RespError(c, http.StatusBadRequest, "Invalid MCP request", err)
		return
	}

	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = handleGroupInitialize(group)
	case "tools/list":
		resp.Result = handleGroupToolsList(group)
	case "tools/call":
		toolName := req.Params.Name
		args := req.Params.Arguments
		result, err := handleGroupToolCall(c.Request.Context(), group, toolName, args)
		if err != nil {
			resp.Error = map[string]any{
				"code":    -32603,
				"message": err.Error(),
			}
		} else {
			resp.Result = result
		}
	default:
		resp.Error = map[string]any{
			"code":    -32601,
			"message": "Method not found",
		}
	}

	c.JSON(http.StatusOK, resp)
}

// getGroupServiceNames returns a list of service names in the group
func getGroupServiceNames(group *model.MCPServiceGroup) []string {
	ids := group.GetServiceIDs()
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		svc, err := model.GetServiceByID(id)
		if err == nil {
			names = append(names, svc.Name)
		}
	}
	return names
}

func handleGroupInitialize(group *model.MCPServiceGroup) map[string]any {
	serviceNames := getGroupServiceNames(group)
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"serverInfo": map[string]any{
			"name":        fmt.Sprintf("one-mcp-group-%s", group.Name),
			"version":     "1.0.0",
			"description": group.Description,
			"services":    serviceNames,
		},
	}
}

func handleGroupToolsList(group *model.MCPServiceGroup) map[string]any {
	serviceNames := getGroupServiceNames(group)

	return map[string]any{
		"tools": []map[string]any{
			{
				"name":        "search_tools",
				"description": "STEP 1: Discover available tools in a service. You MUST call this first before execute_tool.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mcp_name": map[string]any{
							"type":        "string",
							"enum":        serviceNames,
							"description": "MCP service name",
						},
					},
					"required": []string{"mcp_name"},
				},
			},
			{
				"name":        "execute_tool",
				"description": "STEP 2: Execute a tool found via search_tools. Pass arguments directly, do NOT nest.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mcp_name": map[string]any{
							"type":        "string",
							"enum":        serviceNames,
							"description": "MCP service name",
						},
						"tool_name": map[string]any{
							"type":        "string",
							"description": "Tool name from search_tools",
						},
						"arguments": map[string]any{
							"type":        "object",
							"description": "Tool arguments per inputSchema. e.g. {\"query\": \"...\"} not {\"arguments\": {...}}",
						},
					},
					"required": []string{"mcp_name", "tool_name", "arguments"},
				},
			},
		},
	}
}

func handleGroupToolCall(ctx context.Context, group *model.MCPServiceGroup, toolName string, args map[string]any) (any, error) {
	switch toolName {
	case "search_tools":
		parsed, err := parseGroupSearchArgs(args)
		if err != nil {
			return nil, err
		}
		return searchGroupTools(ctx, group, parsed)
	case "execute_tool":
		parsed, err := parseExecuteArgs(args)
		if err != nil {
			return nil, err
		}
		return executeGroupTool(ctx, group, parsed)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func parseGroupSearchArgs(args map[string]any) (*groupSearchArgs, error) {
	mcpName, _ := args["mcp_name"].(string)
	if strings.TrimSpace(mcpName) == "" {
		return nil, fmt.Errorf("mcp_name is required")
	}
	return &groupSearchArgs{
		MCPName: strings.TrimSpace(mcpName),
	}, nil
}

func parseExecuteArgs(args map[string]any) (*executeArgs, error) {
	mcpName, _ := args["mcp_name"].(string)
	toolName, _ := args["tool_name"].(string)
	if strings.TrimSpace(mcpName) == "" || strings.TrimSpace(toolName) == "" {
		return nil, fmt.Errorf("mcp_name and tool_name are required")
	}

	// Parse arguments - support both object and JSON string
	// Also supports "parameters" field name for client compatibility
	arguments, fieldFound := parseArgumentsValue(args)
	if !fieldFound {
		return nil, fmt.Errorf("arguments is required")
	}
	if arguments == nil {
		arguments = map[string]any{}
	}

	return &executeArgs{
		MCPName:   strings.TrimSpace(mcpName),
		ToolName:  strings.TrimSpace(toolName),
		Arguments: arguments,
	}, nil
}

// parseArgumentsValue parses arguments that could be either a map or a JSON string
// Supports field names: "arguments" or "parameters"
// Returns (parsed map, field was found)
func parseArgumentsValue(args map[string]any) (map[string]any, bool) {
	// Try "arguments" first (preferred)
	if v, ok := args["arguments"]; ok && v != nil {
		return parseAnyToMap(v), true
	}
	// Fallback to "parameters" for client compatibility
	if v, ok := args["parameters"]; ok && v != nil {
		return parseAnyToMap(v), true
	}
	return nil, false
}

// parseAnyToMap converts a value to map[string]any, supporting both object and JSON string
func parseAnyToMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	// Try as map first
	if m, ok := v.(map[string]any); ok {
		return m
	}
	// Try as JSON string
	if s, ok := v.(string); ok && s != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err == nil {
			return m
		}
	}
	return nil
}

func searchGroupTools(ctx context.Context, group *model.MCPServiceGroup, args *groupSearchArgs) (any, error) {
	svc, err := group.GetServiceByName(args.MCPName)
	if err != nil {
		return nil, fmt.Errorf("mcp_name not in group: %s", args.MCPName)
	}

	toolsCacheMgr := proxy.GetToolsCacheManager()
	entry, ok := toolsCacheMgr.GetServiceTools(svc.ID)

	// If cache is empty, fetch tools by connecting to the service
	if !ok || len(entry.Tools) == 0 {
		tools, fetchErr := fetchToolsFromService(ctx, svc)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch tools from %s: %v", svc.Name, fetchErr)
		}
		// Return fetched tools directly
		matched := convertTools(tools, svc.Name)
		return map[string]any{"tools": matched}, nil
	}

	matched := convertTools(entry.Tools, svc.Name)
	return map[string]any{"tools": matched}, nil
}

func fetchToolsFromService(ctx context.Context, svc *model.MCPService) ([]mcp_protocol.Tool, error) {
	sharedInst, err := proxy.GetOrCreateSharedMcpInstanceWithKey(ctx, svc, sharedCacheKey(svc.ID), sharedInstanceName(svc.ID), svc.DefaultEnvsJSON)
	if err != nil {
		return nil, err
	}

	toolsReq := mcp_protocol.ListToolsRequest{}
	result, err := sharedInst.Client.ListTools(ctx, toolsReq)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []mcp_protocol.Tool{}, nil
	}
	return result.Tools, nil
}

func convertTools(tools []mcp_protocol.Tool, mcpName string) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"mcp_name":    mcpName,
			"tool_name":   tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	return result
}

func executeGroupTool(ctx context.Context, group *model.MCPServiceGroup, args *executeArgs) (any, error) {
	svc, err := group.GetServiceByName(args.MCPName)
	if err != nil {
		return nil, fmt.Errorf("mcp_name not in group: %s", args.MCPName)
	}

	sharedInst, err := proxy.GetOrCreateSharedMcpInstanceWithKey(ctx, svc, sharedCacheKey(svc.ID), sharedInstanceName(svc.ID), svc.DefaultEnvsJSON)
	if err != nil {
		return nil, err
	}

	callReq := mcp_protocol.CallToolRequest{}
	callReq.Params.Name = args.ToolName
	callReq.Params.Arguments = args.Arguments

	result, err := sharedInst.Client.CallTool(ctx, callReq)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func sharedCacheKey(serviceID int64) string {
	return fmt.Sprintf("global-service-%d-shared", serviceID)
}

func sharedInstanceName(serviceID int64) string {
	return fmt.Sprintf("global-shared-svc-%d", serviceID)
}
