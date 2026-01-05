package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"one-mcp/backend/templates"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"
)

// ExportGroupSkill exports a group as an Anthropic Skill zip package
// GET /api/groups/:id/export
func ExportGroupSkill(c *gin.Context) {
	lang := c.GetString("lang")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_param", lang))
		return
	}

	userID := c.GetInt64("user_id")
	group, err := model.GetMCPServiceGroupByID(id, userID)
	if err != nil {
		common.RespError(c, http.StatusNotFound, "group not found", err)
		return
	}

	// Get user token for MCP config
	user, err := model.GetUserById(userID, false)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to get user", err)
		return
	}

	// Get server address from config or use default
	serverAddress := common.OptionMap["ServerAddress"]
	if serverAddress == "" {
		serverAddress = c.Request.Host
		scheme := "https"
		if c.Request.TLS == nil && !strings.HasPrefix(c.Request.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "http"
		}
		serverAddress = scheme + "://" + serverAddress
	}

	// Build the skill zip
	zipBuffer, err := buildSkillZip(c.Request.Context(), group, user, serverAddress)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to generate skill zip", err)
		return
	}

	// Set response headers for file download
	// Normalize name: replace underscores with hyphens, add one-mcp prefix
	skillName := "one-mcp-" + normalizeSkillName(group.Name)
	filename := fmt.Sprintf("%s.zip", skillName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Length", strconv.Itoa(zipBuffer.Len()))
	c.Data(http.StatusOK, "application/zip", zipBuffer.Bytes())
}

// normalizeSkillName replaces underscores with hyphens for consistent naming
func normalizeSkillName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func buildSkillZip(ctx context.Context, group *model.MCPServiceGroup, user *model.User, serverAddress string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	defer zipWriter.Close()

	serviceIDs := group.GetServiceIDs()
	services := make([]*model.MCPService, 0, len(serviceIDs))
	toolsCache := proxy.GetToolsCacheManager()

	// Collect services and their tools
	servicesWithTools := make([]skillServiceWithTools, 0, len(serviceIDs))

	for _, svcID := range serviceIDs {
		svc, err := model.GetServiceByID(svcID)
		if err != nil {
			continue
		}
		services = append(services, svc)

		var tools []mcp.Tool
		// Try cache first
		if entry, ok := toolsCache.GetServiceTools(svcID); ok && len(entry.Tools) > 0 {
			tools = entry.Tools
		} else {
			// Fetch tools from service if cache is empty
			fetchedTools, fetchErr := fetchToolsFromService(ctx, svc)
			if fetchErr == nil {
				tools = fetchedTools
			}
			// If fetch fails, tools remains empty - continue anyway
		}
		servicesWithTools = append(servicesWithTools, skillServiceWithTools{service: svc, tools: tools})
	}

	// 1. Generate SKILL.md
	skillMD := generateSkillMD(group, servicesWithTools)
	if err := addFileToZip(zipWriter, "SKILL.md", skillMD); err != nil {
		return nil, err
	}

	// 2. Generate tools/*.md for each service
	for _, swt := range servicesWithTools {
		toolsMD := generateToolsMD(swt.service, swt.tools)
		filename := fmt.Sprintf("tools/%s.md", swt.service.Name)
		if err := addFileToZip(zipWriter, filename, toolsMD); err != nil {
			return nil, err
		}
	}

	// 3. Generate mcp-config.json
	mcpConfig := generateMCPConfig(services, user, serverAddress)
	if err := addFileToZip(zipWriter, "mcp-config.json", mcpConfig); err != nil {
		return nil, err
	}

	// 4. Copy executor.py from embedded templates
	executorPy, err := templates.SkillTemplates.ReadFile("skill/executor.py")
	if err != nil {
		return nil, fmt.Errorf("failed to read executor.py template: %w", err)
	}
	if err := addFileToZip(zipWriter, "executor.py", string(executorPy)); err != nil {
		return nil, err
	}

	// 5. Copy refresh_tool_docs.py from embedded templates
	refreshToolDocsPy, err := templates.SkillTemplates.ReadFile("skill/refresh_tool_docs.py")
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh_tool_docs.py template: %w", err)
	}
	if err := addFileToZip(zipWriter, "refresh_tool_docs.py", string(refreshToolDocsPy)); err != nil {
		return nil, err
	}

	// 6. Generate requirements.txt (pyyaml is optional, for YAML output in refresh_tool_docs.py)
	if err := addFileToZip(zipWriter, "requirements.txt", "# Optional: for YAML output in refresh_tool_docs.py\n# pyyaml>=6.0\n"); err != nil {
		return nil, err
	}

	return buf, nil
}

func addFileToZip(zipWriter *zip.Writer, filename string, content string) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

// truncateString truncates a string to maxRunes characters, handling UTF-8 properly
func truncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

type skillServiceWithTools struct {
	service *model.MCPService
	tools   []mcp.Tool
}

func generateSkillMD(group *model.MCPServiceGroup, services []skillServiceWithTools) string {
	var sb strings.Builder

	// Collect stats and build service summaries for description
	totalTools := 0
	serviceNames := make([]string, 0, len(services))
	serviceSummaries := make([]string, 0, len(services))
	for _, swt := range services {
		totalTools += len(swt.tools)
		serviceNames = append(serviceNames, swt.service.Name)
		// Build summary: "name (short desc)"
		shortDesc := swt.service.Description
		if shortDesc == "" {
			shortDesc = swt.service.DisplayName
		}
		shortDesc = truncateString(shortDesc, 80)
		serviceSummaries = append(serviceSummaries, fmt.Sprintf("%s (%s)", swt.service.Name, shortDesc))
	}

	// YAML frontmatter with enhanced metadata
	// Use normalized name with one-mcp prefix (underscores -> hyphens) for consistency with zip filename
	skillName := "one-mcp-" + normalizeSkillName(group.Name)

	// Generate description from service summaries, max 500 chars total
	descLine := "External tools: " + strings.Join(serviceSummaries, ", ")
	descLine = truncateString(descLine, 500)

	// Use YAML library for proper escaping of frontmatter
	frontmatter := map[string]interface{}{
		"name":         skillName,
		"display_name": group.DisplayName,
		"description":  descLine,
		"mcp_count":    len(services),
		"tool_count":   totalTools,
		"services":     serviceNames,
	}
	frontmatterBytes, _ := yaml.Marshal(frontmatter)
	sb.WriteString("---\n")
	sb.WriteString(string(frontmatterBytes))
	sb.WriteString("---\n\n")

	// Title
	sb.WriteString(fmt.Sprintf("# %s\n\n", group.DisplayName))

	// Quick Reference - tool lookup table
	sb.WriteString("## Quick Reference\n\n")
	sb.WriteString("| Service | Tool | Description |\n")
	sb.WriteString("|---------|------|-------------|\n")
	toolsInTable := 0
	for _, swt := range services {
		count := 0
		for _, tool := range swt.tools {
			if count >= 5 {
				break // Max 5 tools per service in Quick Reference
			}
			// Truncate description for table (use runes to handle UTF-8)
			desc := truncateString(tool.Description, 60)
			// Escape pipe characters in description
			desc = strings.ReplaceAll(desc, "|", "\\|")
			sb.WriteString(fmt.Sprintf("| %s | `%s` | %s |\n", swt.service.Name, tool.Name, desc))
			count++
			toolsInTable++
		}
		// If there are more tools, add a hint row
		if len(swt.tools) > 5 {
			sb.WriteString(fmt.Sprintf("| %s | ... | +%d more tools, see [tools/%s.md](tools/%s.md) |\n",
				swt.service.Name, len(swt.tools)-5, swt.service.Name, swt.service.Name))
		}
	}
	sb.WriteString("\n")

	// Available Services with full description
	sb.WriteString("## Available Services\n\n")
	for _, swt := range services {
		toolCount := len(swt.tools)
		desc := swt.service.Description
		if desc == "" {
			desc = swt.service.DisplayName
		}
		sb.WriteString(fmt.Sprintf("### %s (%d tools)\n\n", swt.service.Name, toolCount))
		sb.WriteString(fmt.Sprintf("%s\n\n", desc))
		sb.WriteString(fmt.Sprintf("- [View all tools](tools/%s.md)\n", swt.service.Name))

		// List all tool names
		if toolCount > 0 {
			toolNames := make([]string, 0, toolCount)
			for _, t := range swt.tools {
				toolNames = append(toolNames, fmt.Sprintf("`%s`", t.Name))
			}
			sb.WriteString(fmt.Sprintf("- Tools: %s\n", strings.Join(toolNames, ", ")))
		}
		sb.WriteString("\n")
	}

	// How to Use section
	sb.WriteString("## How to Use\n\n")
	sb.WriteString("1. Find the tool you need in the Quick Reference table above\n")
	sb.WriteString("2. Read detailed documentation from `tools/{service-name}.md`\n")
	sb.WriteString("3. Execute using the syntax below\n\n")

	// Execution Syntax with real examples
	sb.WriteString("## Execution Syntax\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("python executor.py <service-name> <tool-name> '<json-params>'\n")
	sb.WriteString("```\n\n")

	// Generate one real example from first tool with params
	sb.WriteString("### Example\n\n")
	for _, swt := range services {
		for _, tool := range swt.tools {
			if len(tool.InputSchema.Properties) > 0 {
				exampleParams := generateExampleParams(tool.InputSchema)
				sb.WriteString(fmt.Sprintf("```bash\npython executor.py %s %s '%s'\n```\n\n",
					swt.service.Name, tool.Name, exampleParams))
				goto doneExample
			}
		}
	}
doneExample:

	// Refresh Tool Docs
	sb.WriteString("## Refresh Tool Docs\n\n")
	sb.WriteString("If the MCP tools change, refresh the docs:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("python refresh_tool_docs.py\n")
	sb.WriteString("```\n")

	return sb.String()
}

// generateExampleParams creates example JSON params from inputSchema
func generateExampleParams(schema mcp.ToolInputSchema) string {
	if len(schema.Properties) == 0 {
		return "{}"
	}

	example := make(map[string]any)
	for name, prop := range schema.Properties {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		propType, _ := propMap["type"].(string)
		desc, _ := propMap["description"].(string)

		// Generate example value based on type and name
		switch propType {
		case "string":
			if enum, ok := propMap["enum"].([]any); ok && len(enum) > 0 {
				example[name] = enum[0]
			} else if strings.Contains(strings.ToLower(name), "query") {
				example[name] = "example search query"
			} else if strings.Contains(strings.ToLower(name), "url") {
				example[name] = "https://example.com"
			} else if strings.Contains(strings.ToLower(name), "repo") {
				example[name] = "owner/repo"
			} else if strings.Contains(strings.ToLower(desc), "message") {
				example[name] = "Hello, world!"
			} else {
				example[name] = "..."
			}
		case "number", "integer":
			if def, ok := propMap["default"]; ok {
				example[name] = def
			} else {
				example[name] = 10
			}
		case "boolean":
			example[name] = true
		case "object":
			example[name] = map[string]any{}
		case "array":
			example[name] = []any{}
		}
	}

	jsonBytes, _ := json.Marshal(example)
	return string(jsonBytes)
}

func generateToolsMD(service *model.MCPService, tools []mcp.Tool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s Tools\n\n", service.DisplayName))

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("## %s\n\n", tool.Name))
		if tool.Description != "" {
			sb.WriteString(tool.Description + "\n\n")
		}

		// Use YAML format for params (more compact than JSON)
		if len(tool.InputSchema.Properties) > 0 {
			sb.WriteString("**Params:**\n")
			sb.WriteString("```yaml\n")
			paramsYAML := convertInputSchemaToYAML(tool.InputSchema)
			sb.WriteString(paramsYAML)
			sb.WriteString("```\n\n")
		}
	}

	return sb.String()
}

// convertInputSchemaToYAML converts inputSchema to compact YAML format
func convertInputSchemaToYAML(schema mcp.ToolInputSchema) string {
	params := make(map[string]map[string]any)

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	for name, prop := range schema.Properties {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		param := make(map[string]any)

		// Type
		if t, ok := propMap["type"].(string); ok {
			param["type"] = t
		}

		// Description (shortened key)
		if d, ok := propMap["description"].(string); ok {
			param["desc"] = d
		}

		// Enum
		if e, ok := propMap["enum"]; ok {
			param["enum"] = e
		}

		// Default
		if def, ok := propMap["default"]; ok {
			param["default"] = def
		}

		// Required
		if requiredSet[name] {
			param["required"] = true
		}

		params[name] = param
	}

	yamlBytes, _ := yaml.Marshal(params)
	return string(yamlBytes)
}

func generateMCPConfig(services []*model.MCPService, user *model.User, serverAddress string) string {
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}

	mcpServers := config["mcpServers"].(map[string]interface{})
	for _, svc := range services {
		url := fmt.Sprintf("%s/proxy/%s/mcp?key=%s", serverAddress, svc.Name, user.Token)
		mcpServers[svc.Name] = map[string]string{
			"url": url,
		}
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")
	return string(jsonBytes)
}
