package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
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
	zipBuffer, err := buildSkillZip(group, user, serverAddress)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to generate skill zip", err)
		return
	}

	// Set response headers for file download
	filename := fmt.Sprintf("one-mcp-skill-%s.zip", group.Name)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Length", strconv.Itoa(zipBuffer.Len()))
	c.Data(http.StatusOK, "application/zip", zipBuffer.Bytes())
}

func buildSkillZip(group *model.MCPServiceGroup, user *model.User, serverAddress string) (*bytes.Buffer, error) {
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
		if entry, ok := toolsCache.GetServiceTools(svcID); ok {
			tools = entry.Tools
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

	// 4. Generate executor.py
	if err := addFileToZip(zipWriter, "executor.py", executorPy); err != nil {
		return nil, err
	}

	// 5. Generate refresh_tool_docs.py
	if err := addFileToZip(zipWriter, "refresh_tool_docs.py", refreshToolDocsPy); err != nil {
		return nil, err
	}

	// 6. Generate requirements.txt
	if err := addFileToZip(zipWriter, "requirements.txt", "requests>=2.28.0\n"); err != nil {
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

type skillServiceWithTools struct {
	service *model.MCPService
	tools   []mcp.Tool
}

func generateSkillMD(group *model.MCPServiceGroup, services []skillServiceWithTools) string {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", group.Name))

	totalTools := 0
	serviceNames := make([]string, 0, len(services))
	for _, swt := range services {
		totalTools += len(swt.tools)
		serviceNames = append(serviceNames, swt.service.Name)
	}
	sb.WriteString(fmt.Sprintf("description: Access %d tools from %d MCP services. Services: %s\n",
		totalTools, len(services), strings.Join(serviceNames, ", ")))
	sb.WriteString("---\n\n")

	// Title and description
	sb.WriteString(fmt.Sprintf("# %s\n\n", group.DisplayName))
	sb.WriteString("This skill provides access to the following MCP services through one-mcp:\n\n")

	// Available Services
	sb.WriteString("## Available Services\n\n")
	for _, swt := range services {
		toolCount := len(swt.tools)
		desc := swt.service.Description
		if desc == "" {
			desc = swt.service.DisplayName
		}
		sb.WriteString(fmt.Sprintf("- **%s** (%d tools) - %s\n", swt.service.Name, toolCount, desc))
		sb.WriteString(fmt.Sprintf("  - [View all tools](tools/%s.md)\n", swt.service.Name))

		// Show up to 3 popular tools
		if toolCount > 0 {
			popularTools := make([]string, 0, 3)
			for i := 0; i < toolCount && i < 3; i++ {
				popularTools = append(popularTools, fmt.Sprintf("`%s`", swt.tools[i].Name))
			}
			sb.WriteString(fmt.Sprintf("  - Popular tools: %s\n", strings.Join(popularTools, ", ")))
		}
		sb.WriteString("\n")
	}

	// How to Use section
	sb.WriteString("## How to Use\n\n")
	sb.WriteString("When you need to use a tool:\n\n")
	sb.WriteString("1. Check the service list above to identify which MCP provides the tool\n")
	sb.WriteString("2. Read the detailed tool documentation from `tools/{mcp-name}.md`\n")
	sb.WriteString("3. Execute the tool using the syntax below\n\n")

	// Execution Syntax
	sb.WriteString("## Execution Syntax\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("python executor.py {mcp-name} {tool-name} '{json-params}'\n")
	sb.WriteString("```\n\n")

	// Examples
	sb.WriteString("Examples:\n")
	sb.WriteString("```bash\n")
	for _, swt := range services {
		if len(swt.tools) > 0 {
			tool := swt.tools[0]
			sb.WriteString(fmt.Sprintf("# Use %s\n", tool.Name))
			sb.WriteString(fmt.Sprintf("python executor.py %s %s '{...}'\n\n", swt.service.Name, tool.Name))
			break
		}
	}
	sb.WriteString("```\n\n")

	// Refresh Tool Docs
	sb.WriteString("## Refresh Tool Docs\n\n")
	sb.WriteString("If the MCP tools change, refresh the docs:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("python refresh_tool_docs.py\n")
	sb.WriteString("```\n\n")

	// Alternative: Use in Cursor
	sb.WriteString("## Alternative: Use in Cursor\n\n")
	sb.WriteString("Copy `mcp-config.json` to your Cursor MCP settings to use these services directly.\n")

	return sb.String()
}

func generateToolsMD(service *model.MCPService, tools []mcp.Tool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s Tools\n\n", service.DisplayName))
	sb.WriteString("## Available Tools\n\n")

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("### %s\n\n", tool.Name))
		if tool.Description != "" {
			sb.WriteString(tool.Description + "\n\n")
		}

		sb.WriteString("**Parameters:**\n")
		sb.WriteString("```json\n")
		schemaJSON, _ := json.MarshalIndent(tool.InputSchema, "", "  ")
		sb.WriteString(string(schemaJSON) + "\n")
		sb.WriteString("```\n\n")

		sb.WriteString("**Example:**\n")
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("python executor.py %s %s '{...}'\n", service.Name, tool.Name))
		sb.WriteString("```\n\n")
		sb.WriteString("---\n\n")
	}

	return sb.String()
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

const executorPy = `#!/usr/bin/env python3
"""
MCP Tool Executor - Zero-token execution script

Usage:
    python executor.py <mcp_name> <tool_name> <json_params>

Example:
    python executor.py github-mcp create_issue '{"repo": "owner/repo", "title": "Test"}'
"""

import sys
import json
import requests


def load_config():
    with open('mcp-config.json') as f:
        return json.load(f)


def call_tool(mcp_url, tool_name, params):
    """Call MCP tool via streamableHttp protocol"""
    response = requests.post(mcp_url, json={
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": tool_name,
            "arguments": params
        }
    }, headers={
        "Content-Type": "application/json"
    })
    return response.json()


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(__doc__)
        sys.exit(1)

    mcp_name = sys.argv[1]
    tool_name = sys.argv[2]
    params = json.loads(sys.argv[3])

    config = load_config()
    mcp_url = config["mcpServers"][mcp_name]["url"]

    result = call_tool(mcp_url, tool_name, params)
    print(json.dumps(result, indent=2))
`

const refreshToolDocsPy = `#!/usr/bin/env python3
"""
Refresh MCP tool documentation for all configured servers.

Usage:
    python refresh_tool_docs.py
"""

import json
import os
from pathlib import Path
import requests


def load_config():
    with open('mcp-config.json') as f:
        return json.load(f)


def fetch_tools(mcp_url):
    response = requests.post(mcp_url, json={
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/list",
        "params": {}
    }, headers={
        "Content-Type": "application/json"
    })
    return response.json().get("result", {}).get("tools", [])


def write_tools_md(mcp_name, tools, output_dir):
    lines = [f"# {mcp_name} Tools", "", "## Available Tools", ""]
    for tool in tools:
        lines.append(f"### {tool['name']}")
        lines.append("")
        lines.append(tool.get("description", ""))
        lines.append("")
        lines.append("**Parameters:**")
        lines.append("` + "```" + `json")
        lines.append(json.dumps(tool.get("inputSchema", {}), indent=2))
        lines.append("` + "```" + `")
        lines.append("")
        lines.append("---")
        lines.append("")
    output_path = output_dir / f"{mcp_name}.md"
    output_path.write_text("\n".join(lines))
    print(f"Updated: {output_path}")


if __name__ == "__main__":
    config = load_config()
    tools_dir = Path("tools")
    tools_dir.mkdir(exist_ok=True)

    for mcp_name, server in config["mcpServers"].items():
        try:
            tools = fetch_tools(server["url"])
            write_tools_md(mcp_name, tools, tools_dir)
        except Exception as e:
            print(f"Error fetching tools for {mcp_name}: {e}")

    print("Tool docs refreshed")
`
