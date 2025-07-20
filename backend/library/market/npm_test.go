package market

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFindMCPConfigInReadme(t *testing.T) {
	// 测试情况1：常规MCP配置
	readme1 := `
# My Package

This is an example package.

## Configuration

Add the following to your configuration:

` + "```json" + `
{
  "mcpServers": {
    "gitlab": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-gitlab"
      ],
      "env": {
        "GITLAB_PERSONAL_ACCESS_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_API_URL": "https://gitlab.com/api/v4"
      }
    }
  }
}
` + "```" + `

More text here.
`

	config1 := findMCPConfigInReadme(readme1)
	if config1 == nil {
		t.Error("Expected to find MCP config in readme1, but got nil")
	} else {
		// 验证提取的配置
		if len(config1.MCPServers) != 1 {
			t.Errorf("Expected 1 server in config, got %d", len(config1.MCPServers))
		}

		server, exists := config1.MCPServers["gitlab"]
		if !exists {
			t.Error("Expected gitlab server in config, but not found")
		} else {
			if server.Command != "npx" {
				t.Errorf("Expected command to be 'npx', got '%s'", server.Command)
			}

			if len(server.Args) != 2 {
				t.Errorf("Expected 2 args, got %d", len(server.Args))
			}

			if len(server.Env) != 2 {
				t.Errorf("Expected 2 env vars, got %d", len(server.Env))
			}

			if _, exists := server.Env["GITLAB_PERSONAL_ACCESS_TOKEN"]; !exists {
				t.Error("Expected GITLAB_PERSONAL_ACCESS_TOKEN in env vars, but not found")
			}

			if _, exists := server.Env["GITLAB_API_URL"]; !exists {
				t.Error("Expected GITLAB_API_URL in env vars, but not found")
			}
		}
	}

	// 测试情况2：没有MCP配置的README
	readme2 := `
# My Package

This is an example package with no MCP config.

## Usage

Some usage instructions.
`

	config2 := findMCPConfigInReadme(readme2)
	if config2 != nil {
		t.Error("Expected not to find MCP config in readme2, but got a config")
	}

	// 测试情况3：不完整的JSON
	readme3 := `
# My Package

This has an incomplete mcpServers section:

` + "```json" + `
{
  "mcpServers": {
    "gitlab": {
      "command": "npx"
` + "```" + `

More text here.
`

	config3 := findMCPConfigInReadme(readme3)
	if config3 != nil {
		t.Error("Expected not to find valid MCP config in readme3 due to incomplete JSON, but got a config")
	}
}

func TestGetEnvVarsFromMCPConfig(t *testing.T) {
	// 创建测试MCP配置
	config := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"service1": {
				Command: "npx",
				Args:    []string{"-y", "package1"},
				Env: map[string]string{
					"API_KEY": "value1",
					"API_URL": "https://example.com",
					"DEBUG":   "true",
				},
			},
			"service2": {
				Command: "npm",
				Args:    []string{"run", "start"},
				Env: map[string]string{
					"PORT":     "3000",
					"NODE_ENV": "production",
					"API_KEY":  "value1", // 重复的环境变量
				},
			},
		},
	}

	// 获取环境变量
	envVars := GetEnvVarsFromMCPConfig(config)

	// 预期有5个唯一的环境变量
	expectedVarCount := 5
	if len(envVars) != expectedVarCount {
		t.Errorf("Expected %d unique env vars, got %d", expectedVarCount, len(envVars))
	}

	// 验证环境变量名称
	expectedVars := map[string]bool{
		"API_KEY":  true,
		"API_URL":  true,
		"DEBUG":    true,
		"PORT":     true,
		"NODE_ENV": true,
	}

	for _, envVar := range envVars {
		if !expectedVars[envVar] {
			t.Errorf("Unexpected env var: %s", envVar)
		}
	}
}

func TestFindJSONBlocksInText(t *testing.T) {
	// 测试文本
	text := `
Some text before.

{
  "key1": "value1",
  "mcpServers": {
    "service1": {
      "command": "npx"
    }
  }
}

Some text in between.

{
  "anotherKey": "value2"
}

Another mcpServers mention outside a valid JSON block.
`

	// 查找包含mcpServers的JSON块
	blocks := findJSONBlocksInText(text, "mcpServers")

	// 应该找到一个块
	if len(blocks) != 1 {
		t.Errorf("Expected 1 JSON block with mcpServers, got %d", len(blocks))
	}

	// 验证块内容
	if len(blocks) > 0 && !strings.Contains(blocks[0], "service1") {
		t.Error("Expected JSON block to contain 'service1', but it doesn't")
	}
}

func TestInstallNPMPackage(t *testing.T) {
	// 跳过实际执行，仅在明确指定TEST_NPM_INSTALL环境变量时运行
	if os.Getenv("TEST_NPM_INSTALL") != "true" {
		t.Skip("Skipping npm package installation test. Set TEST_NPM_INSTALL=true to run.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试安装一个简单的npm包
	packageName := "chalk"
	version := "4.1.2"     // 指定一个特定版本以确保测试一致性
	command := "npx"       // 默认命令
	args := []string{"-y"} // 默认参数
	workDir := ""          // 使用默认工作目录
	envVars := map[string]string{
		"TEST_ENV_VAR": "test_value",
	}

	serverInfo, err := InstallNPMPackage(ctx, packageName, version, command, args, workDir, envVars)
	if err != nil {
		t.Fatalf("Failed to install npm package: %v", err)
	}

	// 验证返回的服务器信息
	if serverInfo == nil {
		t.Fatal("Expected server info, got nil")
	}

	// 输出服务器信息
	t.Logf("Server Name: %s", serverInfo.Name)
	t.Logf("Server Version: %s", serverInfo.Version)
	t.Logf("Protocol Version: %s", serverInfo.ProtocolVersion)
}

func TestCheckNPXAvailable(t *testing.T) {
	// 跳过实际执行，仅在明确指定TEST_NPX_CHECK环境变量时运行
	if os.Getenv("TEST_NPX_CHECK") != "true" {
		t.Skip("Skipping npx availability check test. Set TEST_NPX_CHECK=true to run.")
	}

	isAvailable := CheckNPXAvailable()

	// 这个测试的目的是验证函数是否成功运行，而不是验证npx是否确实可用
	// 因为在不同环境中npx可能存在也可能不存在
	t.Logf("NPX availability: %v", isAvailable)
}

func TestGetInstalledMCPServersFromDB(t *testing.T) {
	// 跳过实际执行，仅在明确指定TEST_MCP_SERVERS_DB环境变量时运行
	if os.Getenv("TEST_MCP_SERVERS_DB") != "true" {
		t.Skip("Skipping MCP servers from DB test. Set TEST_MCP_SERVERS_DB=true to run.")
	}

	servers, err := GetInstalledMCPServersFromDB()
	if err != nil {
		t.Fatalf("Failed to get installed MCP servers from DB: %v", err)
	}

	t.Logf("Found %d MCP servers in database:", len(servers))
	for packageName, serviceID := range servers {
		t.Logf("- %s (Service ID: %d)", packageName, serviceID)
	}
}
