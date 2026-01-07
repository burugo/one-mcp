package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	mcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

type groupMCPHandlerEntry struct {
	handler     http.Handler
	fingerprint string
}

var (
	groupMCPHandlers   = map[string]*groupMCPHandlerEntry{}
	groupMCPHandlersMu sync.RWMutex
)

func getOrCreateGroupMCPHandler(group *model.MCPServiceGroup, userID int64) (http.Handler, error) {
	cacheKey := groupHandlerCacheKey(group.ID, userID)
	fingerprint := groupHandlerFingerprint(group)

	groupMCPHandlersMu.RLock()
	if entry, ok := groupMCPHandlers[cacheKey]; ok && entry.fingerprint == fingerprint {
		groupMCPHandlersMu.RUnlock()
		return entry.handler, nil
	}
	groupMCPHandlersMu.RUnlock()

	handler, err := buildGroupMCPHandler(group)
	if err != nil {
		return nil, err
	}

	groupMCPHandlersMu.Lock()
	groupMCPHandlers[cacheKey] = &groupMCPHandlerEntry{
		handler:     handler,
		fingerprint: fingerprint,
	}
	groupMCPHandlersMu.Unlock()

	return handler, nil
}

func groupHandlerCacheKey(groupID int64, userID int64) string {
	return fmt.Sprintf("group-%d-user-%d", groupID, userID)
}

func groupHandlerFingerprint(group *model.MCPServiceGroup) string {
	return fmt.Sprintf("%s|%s|%s", group.Name, group.Description, group.ServiceIDsJSON)
}

func buildGroupMCPHandler(group *model.MCPServiceGroup) (http.Handler, error) {
	server, err := buildGroupMCPServer(group)
	if err != nil {
		return nil, err
	}

	streamable := mcpserver.NewStreamableHTTPServer(server,
		mcpserver.WithHeartbeatInterval(30*time.Second),
	)

	return proxy.WrapSessionErrorFixingHandler(streamable), nil
}

func buildGroupMCPServer(group *model.MCPServiceGroup) (*mcpserver.MCPServer, error) {
	serverName := fmt.Sprintf("one-mcp-group-%s", group.Name)
	serverOptions := []mcpserver.ServerOption{}
	if strings.TrimSpace(group.Description) != "" {
		serverOptions = append(serverOptions, mcpserver.WithInstructions(group.Description))
	}

	server := mcpserver.NewMCPServer(serverName, "1.0.0", serverOptions...)
	if err := addGroupTools(server, group); err != nil {
		return nil, err
	}
	if err := addGroupResources(server, group); err != nil {
		return nil, err
	}
	return server, nil
}

func addGroupTools(server *mcpserver.MCPServer, group *model.MCPServiceGroup) error {
	if server == nil {
		return errors.New("mcp server is nil")
	}

	serviceNames := getGroupServiceNames(group)

	searchTool := mcp.Tool{
		Name:        "search_tools",
		Description: "STEP 1: Discover available tools in a service. You MUST call this first before execute_tool.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"mcp_name": map[string]any{
					"type":        "string",
					"enum":        serviceNames,
					"description": "MCP service name",
				},
			},
			Required: []string{"mcp_name"},
		},
	}

	executeTool := mcp.Tool{
		Name:        "execute_tool",
		Description: "STEP 2: Execute a tool found via search_tools. Pass arguments directly, do NOT nest.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
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
			Required: []string{"mcp_name", "tool_name", "arguments"},
		},
	}

	server.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := common.ParseAnyToMap(request.Params.Arguments)
		if args == nil {
			args = map[string]any{}
		}
		parsed, err := parseGroupSearchArgs(args)
		if err != nil {
			return toolErrorResult(err), nil
		}
		result, err := searchGroupTools(ctx, group, parsed)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return toolResultFromStructured(result), nil
	})

	server.AddTool(executeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := common.ParseAnyToMap(request.Params.Arguments)
		if args == nil {
			args = map[string]any{}
		}
		parsed, err := parseExecuteArgs(args)
		if err != nil {
			return toolErrorResult(err), nil
		}
		result, err := executeGroupTool(ctx, group, parsed)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return toolResultFromStructured(result), nil
	})

	return nil
}

func addGroupResources(server *mcpserver.MCPServer, group *model.MCPServiceGroup) error {
	if server == nil {
		return errors.New("mcp server is nil")
	}

	ids := group.GetServiceIDs()
	for _, id := range ids {
		svc, err := model.GetServiceByID(id)
		if err != nil {
			// Skip invalid services or handle error
			continue
		}

		// Create a unique URI for the resource
		resourceURI := fmt.Sprintf("mcp://%s/%s", group.Name, svc.Name)

		resource := mcp.Resource{
			URI:         resourceURI,
			Name:        svc.Name,
			Description: svc.Description,
			MIMEType:    "application/yaml",
		}

		// Capture svc for the closure
		currentSvc := svc

		server.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			args := &groupSearchArgs{
				MCPName: currentSvc.Name,
			}

			// Reuse searchGroupTools logic to get tools list
			result, err := searchGroupTools(ctx, group, args)
			if err != nil {
				return nil, err
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("internal error: unexpected result type")
			}

			// Extract tool list from content[0].text
			var contentStr string
			if rawContent, ok := resultMap["content"].([]map[string]any); ok && len(rawContent) > 0 {
				contentStr, _ = rawContent[0]["text"].(string)
			}

			if contentStr == "" {
				contentStr = "# No tools available or failed to fetch"
			}

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/yaml",
					Text:     contentStr,
				},
			}, nil
		})
	}

	return nil
}

func toolErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{
				Type: mcp.ContentTypeText,
				Text: err.Error(),
			},
		},
	}
}

func toolResultFromStructured(result any) *mcp.CallToolResult {
	resultMap, _ := result.(map[string]any)

	// Extract content
	contents := extractContent(resultMap)

	callResult := &mcp.CallToolResult{
		Content: contents,
	}
	// Only set StructuredContent if the key exists in the result map
	if resultMap != nil {
		if sc, exists := resultMap["structuredContent"]; exists {
			if scMap, ok := sc.(map[string]any); ok {
				callResult.StructuredContent = scMap
			} else {
				callResult.StructuredContent = map[string]any{}
			}
		}
	}
	return callResult
}

func extractContent(result map[string]any) []mcp.Content {
	if result == nil {
		return nil
	}
	rawContent, ok := result["content"]
	if !ok || rawContent == nil {
		return nil
	}
	switch content := rawContent.(type) {
	case []mcp.Content:
		return content
	case []map[string]any:
		contents := make([]mcp.Content, 0, len(content))
		for _, item := range content {
			itemBytes, err := json.Marshal(item)
			if err != nil {
				continue
			}
			parsed, err := mcp.UnmarshalContent(itemBytes)
			if err != nil {
				continue
			}
			contents = append(contents, parsed)
		}
		return contents
	case []any:
		contents := make([]mcp.Content, 0, len(content))
		for _, item := range content {
			itemBytes, err := json.Marshal(item)
			if err != nil {
				continue
			}
			parsed, err := mcp.UnmarshalContent(itemBytes)
			if err != nil {
				continue
			}
			contents = append(contents, parsed)
		}
		return contents
	default:
		return nil
	}
}
