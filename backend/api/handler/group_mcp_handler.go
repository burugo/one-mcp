package handler

import (
	"context"
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
	ToolKey string
	Limit   int
}

type executeArgs struct {
	MCPName  string
	ToolName string
	Params   map[string]any
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
		resp.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"serverInfo": map[string]string{
				"name":    "one-mcp-group",
				"version": "1.0.0",
			},
		}
	case "tools/list":
		resp.Result = handleGroupToolsList()
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

func handleGroupToolsList() map[string]any {
	return map[string]any{
		"tools": []map[string]any{
			{
				"name":        "search_tools",
				"description": "Search tools within a specific MCP service in this group",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mcp_name": map[string]any{
							"type":        "string",
							"description": "Required MCP service name (must be one of the group's services)",
						},
						"tool_name": map[string]any{
							"type":        "string",
							"description": "Optional tool name keywords; comma or space separated",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "Maximum number of tools to return (default 10)",
							"default":     10,
						},
					},
					"required": []string{"mcp_name"},
				},
			},
			{
				"name":        "execute_tool",
				"description": "Execute a tool from a specific MCP service",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"mcp_name": map[string]any{
							"type":        "string",
							"description": "The MCP service name",
						},
						"tool_name": map[string]any{
							"type":        "string",
							"description": "The tool name to execute",
						},
						"params": map[string]any{
							"type":        "object",
							"description": "Tool parameters as returned by search_tools",
						},
					},
					"required": []string{"mcp_name", "tool_name", "params"},
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
		return searchGroupTools(group, parsed)
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
	toolKey, _ := args["tool_name"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	return &groupSearchArgs{
		MCPName: strings.TrimSpace(mcpName),
		ToolKey: strings.TrimSpace(toolKey),
		Limit:   limit,
	}, nil
}

func parseExecuteArgs(args map[string]any) (*executeArgs, error) {
	mcpName, _ := args["mcp_name"].(string)
	toolName, _ := args["tool_name"].(string)
	if strings.TrimSpace(mcpName) == "" || strings.TrimSpace(toolName) == "" {
		return nil, fmt.Errorf("mcp_name and tool_name are required")
	}

	params, _ := args["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}

	return &executeArgs{
		MCPName:  strings.TrimSpace(mcpName),
		ToolName: strings.TrimSpace(toolName),
		Params:   params,
	}, nil
}

func searchGroupTools(group *model.MCPServiceGroup, args *groupSearchArgs) (any, error) {
	svc, err := group.GetServiceByName(args.MCPName)
	if err != nil {
		return nil, fmt.Errorf("mcp_name not in group: %s", args.MCPName)
	}

	entry, ok := proxy.GetToolsCacheManager().GetServiceTools(svc.ID)
	if !ok {
		return map[string]any{"tools": []map[string]any{}}, nil
	}

	keywords := splitKeywords(args.ToolKey)
	matched := make([]map[string]any, 0, len(entry.Tools))
	for _, tool := range entry.Tools {
		if matchesTool(tool.Name, tool.Description, keywords) {
			matched = append(matched, map[string]any{
				"mcp_name":    svc.Name,
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			})
		}
		if args.Limit > 0 && len(matched) >= args.Limit {
			break
		}
	}

	return map[string]any{"tools": matched}, nil
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
	callReq.Params.Arguments = args.Params

	result, err := sharedInst.Client.CallTool(ctx, callReq)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func splitKeywords(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Fields(strings.ReplaceAll(raw, ",", " "))
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			keywords = append(keywords, strings.ToLower(trimmed))
		}
	}
	return keywords
}

func matchesTool(name string, desc string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	combined := strings.ToLower(name + " " + desc)
	for _, kw := range keywords {
		if !strings.Contains(combined, kw) {
			return false
		}
	}
	return true
}

func sharedCacheKey(serviceID int64) string {
	return fmt.Sprintf("global-service-%d-shared", serviceID)
}

func sharedInstanceName(serviceID int64) string {
	return fmt.Sprintf("global-shared-svc-%d", serviceID)
}
