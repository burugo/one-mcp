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
	"time"

	"github.com/gin-gonic/gin"
	mcp_protocol "github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
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
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":     fmt.Sprintf("one-mcp-group-%s", group.Name),
			"version":  "1.0.0",
			"services": serviceNames,
		},
		"instructions": group.Description,
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
							"description": "Tool arguments. Example: {\"message\": \"hello\"} for a tool with message param",
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
		// Fallback: collect all other fields as arguments (for dumb LLMs)
		arguments = extractRemainingAsArguments(args)
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

// extractRemainingAsArguments collects all fields except mcp_name/tool_name as arguments
// This handles cases where LLM puts tool params at top level instead of in arguments
func extractRemainingAsArguments(args map[string]any) map[string]any {
	reserved := map[string]bool{"mcp_name": true, "tool_name": true, "arguments": true, "parameters": true}
	result := make(map[string]any)
	for k, v := range args {
		if !reserved[k] {
			result[k] = v
		}
	}
	return result
}

// parseArgumentsValue parses arguments that could be either a map or a JSON string
// Supports field names: "arguments" or "parameters"
// Returns (parsed map, field was found)
func parseArgumentsValue(args map[string]any) (map[string]any, bool) {
	for _, fieldName := range []string{"arguments", "parameters"} {
		if v, ok := args[fieldName]; ok && v != nil {
			return parseAnyToMap(v), true
		}
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

	currentTime := time.Now().Format("2006-01-02 15:04")

	toolsCacheMgr := proxy.GetToolsCacheManager()
	entry, ok := toolsCacheMgr.GetServiceTools(svc.ID)

	var tools []mcp_protocol.Tool
	// If cache is empty, fetch tools by connecting to the service
	if !ok || len(entry.Tools) == 0 {
		fetchedTools, fetchErr := fetchToolsFromService(ctx, svc)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch tools from %s: %v", svc.Name, fetchErr)
		}
		tools = fetchedTools
	} else {
		tools = entry.Tools
	}

	// Convert to YAML for compact response
	yamlTools := convertToolsToYAML(tools, svc.Name)
	yamlBytes, err := yaml.Marshal(yamlTools)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize tools: %v", err)
	}

	toolsSummary := string(yamlBytes)

	return map[string]any{
		"tools_yaml":   toolsSummary,
		"current_time": currentTime,
		"tool_count":   len(tools),
		"content": []map[string]any{
			{
				"type": "text",
				"text": toolsSummary,
			},
		},
	}, nil
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

// yamlTool is a compact YAML-friendly tool representation
type yamlTool struct {
	Name   string         `yaml:"name"`
	Desc   string         `yaml:"desc,omitempty"`
	Params map[string]any `yaml:"params,omitempty"`
}

func convertToolsToYAML(tools []mcp_protocol.Tool, mcpName string) []yamlTool {
	result := make([]yamlTool, 0, len(tools))
	for _, tool := range tools {
		yt := yamlTool{
			Name: tool.Name,
			Desc: tool.Description,
		}
		// Extract just the properties from inputSchema for compactness
		if len(tool.InputSchema.Properties) > 0 {
			yt.Params = tool.InputSchema.Properties
		}
		result = append(result, yt)
	}
	return result
}

func executeGroupTool(ctx context.Context, group *model.MCPServiceGroup, args *executeArgs) (any, error) {
	start := time.Now()

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

	executionSeconds := time.Since(start).Seconds()

	var content any = result
	if result != nil && len(result.Content) > 0 {
		content = result.Content
	} else if result != nil {
		content = []map[string]any{
			{
				"type": "text",
				"text": fmt.Sprintf("%v", result),
			},
		}
	}

	// Wrap result with execution time
	return map[string]any{
		"execution_seconds": fmt.Sprintf("%.2f", executionSeconds),
		"content":           content,
	}, nil
}

func sharedCacheKey(serviceID int64) string {
	return fmt.Sprintf("global-service-%d-shared", serviceID)
}

func sharedInstanceName(serviceID int64) string {
	return fmt.Sprintf("global-shared-svc-%d", serviceID)
}
