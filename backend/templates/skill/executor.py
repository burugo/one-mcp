#!/usr/bin/env python3
"""
MCP Tool Executor - Zero-token execution script

Usage:
    python executor.py <mcp_name> <tool_name> <json_params>

Example:
    python executor.py github-mcp create_issue '{"repo": "owner/repo", "title": "Test"}'
"""

import sys
import json
import os
import urllib.request
import urllib.error

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
                    "name": "mcp-executor",
                    "version": "1.0.0"
                }
            }
        })
        
        self.session_id = headers.get("Mcp-Session-Id")
        if not self.session_id:
            raise Exception("No session ID in response")
        
        return body
    
    def call_tool(self, tool_name, params):
        """Call a tool on the MCP server."""
        if not self.session_id:
            self.initialize()
        
        body, _ = self._post({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": params
            }
        }, headers={"Mcp-Session-Id": self.session_id})
        
        return body


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(__doc__)
        sys.exit(1)

    mcp_name = sys.argv[1]
    tool_name = sys.argv[2]
    params = json.loads(sys.argv[3])

    config = load_config()
    mcp_url = config["mcpServers"][mcp_name]["url"]

    client = MCPClient(mcp_url)
    result = client.call_tool(tool_name, params)
    print(json.dumps(result, indent=2))
