package proxy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/stretchr/testify/require"
)

func TestStderrLogAggregatorPreservesCompletePythonTraceback(t *testing.T) {
	aggregator := &stderrLogAggregator{}
	var messages []string
	for _, line := range []string{
		"Traceback (most recent call last):",
		`  File "/app/server.py", line 12, in <module>`,
		"    from mcp.shared.exceptions import McpError",
		"ImportError: cannot import name 'McpError' from 'mcp.shared.exceptions'",
		"server restarting",
	} {
		messages = append(messages, aggregator.add(line)...)
	}
	messages = append(messages, aggregator.flush()...)

	require.Equal(t, []string{
		"Traceback (most recent call last):\n" +
			`  File "/app/server.py", line 12, in <module>` + "\n" +
			"    from mcp.shared.exceptions import McpError\n" +
			"ImportError: cannot import name 'McpError' from 'mcp.shared.exceptions'",
		"server restarting",
	}, messages)
}

func TestStderrLogAggregatorEmitsOrdinaryLinesSeparately(t *testing.T) {
	aggregator := &stderrLogAggregator{}

	require.Equal(t, []string{"server starting"}, aggregator.add("server starting"))
	require.Equal(t, []string{"server ready"}, aggregator.add("server ready"))
	require.Empty(t, aggregator.flush())
}

func TestShouldRetryUVXWithMCPV1ForSDK2ImportError(t *testing.T) {
	stderr := `ImportError: cannot import name 'McpError' from 'mcp.shared.exceptions' (` +
		`/root/.cache/uv/archive/lib/python3.13/site-packages/mcp/shared/exceptions.py). Did you mean: 'MCPError'?`

	require.True(t, shouldRetryUVXWithMCPV1("uvx", []string{"arbitrary-mcp-server"}, stderr))
	require.True(t, shouldRetryUVXWithMCPV1("/usr/local/bin/uvx", []string{"another-server"}, stderr))
}

func TestShouldRetryUVXWithMCPV1ForRemovedServerAPI(t *testing.T) {
	stderr := `AttributeError: 'Server' object has no attribute 'list_tools'`

	require.True(t, shouldRetryUVXWithMCPV1("uvx", []string{"unknown-mcp-server"}, stderr))
}

func TestShouldRetryUVXWithMCPV1RejectsUnrelatedOrExplicitlyConstrainedCommands(t *testing.T) {
	compatibilityError := `ImportError: cannot import name 'McpError' from 'mcp.shared.exceptions'`

	require.False(t, shouldRetryUVXWithMCPV1("npx", []string{"some-server"}, compatibilityError))
	require.False(t, shouldRetryUVXWithMCPV1("uvx", []string{"some-server"}, "connection refused"))
	require.False(t, shouldRetryUVXWithMCPV1("uvx", []string{"--with", "mcp>=2", "some-server"}, compatibilityError))
	require.False(t, shouldRetryUVXWithMCPV1("uvx", []string{"--with=mcp==1.29.0", "some-server"}, compatibilityError))
	require.False(t, shouldRetryUVXWithMCPV1("uvx", []string{"--with", "mcp @ https://example.test/mcp.whl", "some-server"}, compatibilityError))
}

func TestWithMCPV1ConstraintPreservesUVXArguments(t *testing.T) {
	original := []string{"--from", "git+https://example.test/server.git", "custom-server", "--verbose"}

	constrained := withMCPV1Constraint(original)

	require.Equal(t, []string{
		"--with", "mcp<2",
		"--from", "git+https://example.test/server.git", "custom-server", "--verbose",
	}, constrained)
	require.Equal(t, []string{"--from", "git+https://example.test/server.git", "custom-server", "--verbose"}, original)
}

func TestUVXServiceRetriesWithMCPV1AfterSDK2CompatibilityFailure(t *testing.T) {
	uvxPath, err := exec.LookPath("uvx")
	if err != nil {
		t.Skip("uvx is not available")
	}

	packageDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "pyproject.toml"), []byte(`[build-system]
requires = ["setuptools>=68"]
build-backend = "setuptools.build_meta"

[project]
name = "one-mcp-uvx-compat-test"
version = "0.0.1"
dependencies = ["mcp>=1.23"]

[project.scripts]
one-mcp-uvx-compat-test = "compat_server:main"
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "compat_server.py"), []byte(`import asyncio
from mcp.shared.exceptions import McpError
from mcp.server import Server
from mcp.server.stdio import stdio_server

async def run():
    server = Server("compat-test")
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())

def main():
    asyncio.run(run())
`), 0o600))

	service := &model.MCPService{
		Name:            "generic-uvx-compat-test",
		Type:            model.ServiceTypeStdio,
		Command:         uvxPath,
		ArgsJSON:        fmt.Sprintf(`["--refresh","--from",%q,"one-mcp-uvx-compat-test"]`, packageDir),
		DefaultEnvsJSON: `{}`,
	}
	originalSQLitePath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "uvx-compat.db")
	require.NoError(t, model.InitDB())
	t.Cleanup(func() {
		common.SQLitePath = originalSQLitePath
	})
	require.NoError(t, model.CreateService(service))

	handshakeCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	server, client, _, _, serverInfo, err := createActualMcpGoServerAndClientUncached(
		handshakeCtx,
		handshakeCtx,
		"uvx-compat-test",
		service,
		"uvx-compat-test",
	)
	require.NoError(t, err)
	require.NotNil(t, server)
	require.NotNil(t, client)
	require.NotNil(t, serverInfo)
	require.Equal(t, "compat-test", serverInfo.Name)
	require.NoError(t, client.Close())
}
