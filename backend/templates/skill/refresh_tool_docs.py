#!/usr/bin/env python3
"""
Refresh MCP tool documentation for all configured servers.

Usage:
    python refresh_tool_docs.py

Note: Uses Python standard library only (no external dependencies).
      For YAML output, install pyyaml: pip install pyyaml
"""

import json
import os
import urllib.request
import urllib.error
from pathlib import Path

try:
    import yaml
    HAS_YAML = True
except ImportError:
    HAS_YAML = False

# Get the directory where this script is located
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))


def load_config():
    config_path = os.path.join(SCRIPT_DIR, 'mcp-config.json')
    with open(config_path) as f:
        return json.load(f)


class MCPClient:
    """Simple MCP client that handles session management using standard library."""
    
    def __init__(self, base_url):
        self.base_url = base_url
        self.session_id = None
    
    def _post(self, data, headers=None):
        """Make a POST request using urllib."""
        req_headers = {"Content-Type": "application/json"}
        if headers:
            req_headers.update(headers)
        
        request = urllib.request.Request(
            self.base_url,
            data=json.dumps(data).encode('utf-8'),
            headers=req_headers,
            method='POST'
        )
        
        with urllib.request.urlopen(request) as response:
            response_headers = dict(response.headers)
            body = json.loads(response.read().decode('utf-8'))
            return body, response_headers
    
    def initialize(self):
        """Initialize MCP session and get session ID."""
        body, headers = self._post({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {
                    "name": "refresh-tool-docs",
                    "version": "1.0.0"
                }
            }
        })
        
        self.session_id = headers.get("Mcp-Session-Id")
        if not self.session_id:
            raise Exception("No session ID in response")
        
        return body
    
    def list_tools(self):
        """List all available tools."""
        if not self.session_id:
            self.initialize()
        
        body, _ = self._post({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list",
            "params": {}
        }, headers={"Mcp-Session-Id": self.session_id})
        
        if "error" in body:
            raise Exception(f"MCP error: {body['error']}")
        
        return body.get("result", {}).get("tools", [])


def fetch_tools(mcp_url):
    """Fetch tools from MCP server using proper session management."""
    client = MCPClient(mcp_url)
    return client.list_tools()


def convert_schema_to_yaml_params(input_schema):
    """Convert inputSchema to compact YAML format."""
    props = input_schema.get("properties", {})
    required = set(input_schema.get("required", []))
    
    params = {}
    for name, prop in props.items():
        param = {"type": prop.get("type", "string")}
        if "description" in prop:
            param["desc"] = prop["description"]
        if "enum" in prop:
            param["enum"] = prop["enum"]
        if "default" in prop:
            param["default"] = prop["default"]
        if name in required:
            param["required"] = True
        params[name] = param
    
    return params


def write_tools_md(mcp_name, tools, output_dir):
    lines = [f"# {mcp_name} Tools", ""]
    
    for tool in tools:
        lines.append(f"## {tool['name']}")
        lines.append("")
        if tool.get("description"):
            lines.append(tool["description"])
            lines.append("")
        
        input_schema = tool.get("inputSchema", {})
        if input_schema.get("properties"):
            lines.append("**Params:**")
            if HAS_YAML:
                lines.append("```yaml")
                params = convert_schema_to_yaml_params(input_schema)
                lines.append(yaml.dump(params, allow_unicode=True, default_flow_style=False).rstrip())
            else:
                lines.append("```json")
                lines.append(json.dumps(input_schema, indent=2))
            lines.append("```")
            lines.append("")
    
    output_path = output_dir / f"{mcp_name}.md"
    output_path.write_text("\n".join(lines), encoding="utf-8")
    print(f"Updated: {output_path}")


if __name__ == "__main__":
    config = load_config()
    tools_dir = Path(SCRIPT_DIR) / "tools"
    tools_dir.mkdir(exist_ok=True)

    for mcp_name, server in config["mcpServers"].items():
        try:
            tools = fetch_tools(server["url"])
            write_tools_md(mcp_name, tools, tools_dir)
        except Exception as e:
            print(f"Error fetching tools for {mcp_name}: {e}")

    print("Tool docs refreshed")
