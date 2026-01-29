package handler

import (
	"context"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	mcp "github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

type groupSearchArgs struct {
	MCPName string
}

type executeArgs struct {
	MCPName   string
	ToolName  string
	Arguments map[string]any
}

type contextKey string

const (
	clientNameKey contextKey = "client_name"
	userIDKey     contextKey = "user_id"
)

func GroupMCPHandler(c *gin.Context) {
	groupName := c.Param("name")
	userID := c.GetInt64("user_id")

	if userID == 0 {
		common.RespJSONRPCError(c, http.StatusUnauthorized, common.JSONRPCErrorCodeInvalidRequest,
			"Authentication failed: Invalid or expired API key. Please check your API key in Profile settings or refresh it if recently changed.")
		return
	}

	group, err := model.GetMCPServiceGroupByName(groupName, userID)
	if err != nil {
		common.RespJSONRPCError(c, http.StatusNotFound, common.JSONRPCErrorCodeInvalidRequest,
			"Group not found: "+err.Error())
		return
	}

	if !group.Enabled {
		common.RespJSONRPCError(c, http.StatusServiceUnavailable, common.JSONRPCErrorCodeInvalidRequest,
			"Group is disabled")
		return
	}

	handler, err := getOrCreateGroupMCPHandler(group, userID)
	if err != nil {
		common.RespJSONRPCError(c, http.StatusInternalServerError, common.JSONRPCErrorCodeInvalidRequest,
			"Failed to create MCP handler: "+err.Error())
		return
	}

	// Store client name and userID in request context for logging and RPD check
	clientName := c.Request.Header.Get("User-Agent")
	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, clientNameKey, clientName)
	ctx = context.WithValue(ctx, userIDKey, userID)
	c.Request = c.Request.WithContext(ctx)

	handler.ServeHTTP(c.Writer, c.Request)
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
			return common.ParseAnyToMap(v), true
		}
	}
	return nil, false
}

func searchGroupTools(ctx context.Context, group *model.MCPServiceGroup, args *groupSearchArgs) (any, error) {
	svc, err := group.GetServiceByName(args.MCPName)
	if err != nil {
		available := getGroupServiceNames(group)
		return nil, fmt.Errorf("mcp_name '%s' not in group, available: %v", args.MCPName, available)
	}

	currentTime := time.Now().Format("2006-01-02 15:04")

	toolsCacheMgr := proxy.GetToolsCacheManager()
	entry, ok := toolsCacheMgr.GetServiceTools(svc.ID)

	var tools []mcp.Tool
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

	// Return in content for Cursor compatibility (Cursor doesn't read structuredContent)
	// Prepend current_time as a comment in YAML
	toolsSummaryWithTime := fmt.Sprintf("# current_time: %s\n%s", currentTime, toolsSummary)
	return map[string]any{
		"content": []map[string]any{
			{
				"type": mcp.ContentTypeText,
				"text": toolsSummaryWithTime,
			},
		},
	}, nil
}

func fetchToolsFromService(ctx context.Context, svc *model.MCPService) ([]mcp.Tool, error) {
	sharedInst, err := proxy.GetOrCreateSharedMcpInstanceWithKey(ctx, svc, proxy.SharedServiceCacheKey(svc.ID), proxy.SharedServiceInstanceName(svc.ID), svc.DefaultEnvsJSON)
	if err != nil {
		return nil, err
	}

	toolsReq := mcp.ListToolsRequest{}
	result, err := sharedInst.Client.ListTools(ctx, toolsReq)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []mcp.Tool{}, nil
	}
	return result.Tools, nil
}

// yamlTool is a compact YAML-friendly tool representation
type yamlTool struct {
	Name   string         `yaml:"name"`
	Desc   string         `yaml:"desc,omitempty"`
	Params map[string]any `yaml:"params,omitempty"`
}

func convertToolsToYAML(tools []mcp.Tool, mcpName string) []yamlTool {
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
		available := getGroupServiceNames(group)
		return nil, fmt.Errorf("mcp_name '%s' not in group, available: %v", args.MCPName, available)
	}

	// Get userID from context for RPD check and stats
	var userID int64
	if uid, ok := ctx.Value(userIDKey).(int64); ok {
		userID = uid
	}

	// Check daily request limit (RPD) if limit is set
	if userID > 0 && svc.RPDLimit > 0 {
		if rpdErr := checkDailyRequestLimit(svc.ID, userID, svc.RPDLimit); rpdErr != nil {
			return nil, rpdErr
		}
	}

	sharedInst, err := proxy.GetOrCreateSharedMcpInstanceWithKey(ctx, svc, proxy.SharedServiceCacheKey(svc.ID), proxy.SharedServiceInstanceName(svc.ID), svc.DefaultEnvsJSON)
	if err != nil {
		return nil, err
	}

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = args.ToolName
	callReq.Params.Arguments = args.Arguments

	result, err := sharedInst.Client.CallTool(ctx, callReq)
	duration := time.Since(start)

	// Get client name from context
	clientName := ""
	if cn, ok := ctx.Value(clientNameKey).(string); ok {
		clientName = cn
	}

	// Determine success: no error AND result.IsError is false
	success := err == nil && (result == nil || !result.IsError)

	// Only record stats for successful calls (not errors or isError responses)
	if success {
		go model.RecordRequestStat(
			svc.ID,
			svc.Name,
			userID,
			model.ProxyRequestTypeHTTP,
			"tools/call",
			fmt.Sprintf("/group/%s/mcp", group.Name),
			duration.Milliseconds(),
			200,
			true,
		)
	}

	// Log the execution
	logLevel := model.MCPLogLevelInfo
	logMsg := fmt.Sprintf("Group execute_tool OK | group=%s | mcp=%s | tool=%s | duration=%dms | client=%s",
		group.Name, svc.Name, args.ToolName, duration.Milliseconds(), clientName)
	if err != nil {
		logLevel = model.MCPLogLevelError
		logMsg = fmt.Sprintf("Group execute_tool FAILED | group=%s | mcp=%s | tool=%s | duration=%dms | client=%s | error=%v",
			group.Name, svc.Name, args.ToolName, duration.Milliseconds(), clientName, err)
	} else if result != nil && result.IsError {
		logLevel = model.MCPLogLevelError
		logMsg = fmt.Sprintf("Group execute_tool ERROR | group=%s | mcp=%s | tool=%s | duration=%dms | client=%s | isError=true",
			group.Name, svc.Name, args.ToolName, duration.Milliseconds(), clientName)
	}
	if saveErr := model.SaveMCPLog(ctx, svc.ID, svc.Name, model.MCPLogPhaseRun, logLevel, logMsg); saveErr != nil {
		common.SysError(fmt.Sprintf("Failed to save MCP log for %s: %v", svc.Name, saveErr))
	}

	if err != nil {
		return nil, err
	}

	// Faithfully return upstream response structure
	resp := map[string]any{}

	if result != nil && len(result.Content) > 0 {
		resp["content"] = result.Content
	}

	if result != nil && result.StructuredContent != nil {
		resp["structuredContent"] = result.StructuredContent
	}

	if result != nil && result.IsError {
		resp["isError"] = true
	}

	return resp, nil
}
