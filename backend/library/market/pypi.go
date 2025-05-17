package market

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	// Assuming MCPServerInfo is in the same package, or import if it's moved to a common place.
	// For now, let's assume it's accessible as it's in the same package 'market'
	// If not, we'd need to define/import it.
	// MCPServerInfo is defined in npm.go
)

const (
	pythonVenvsBaseDir = "data/python_venvs" // Base directory for Python virtual environments
)

// CheckUVXAvailable checks if the 'uv' command is available.
func CheckUVXAvailable() bool {
	cmd := exec.Command("uv", "--version")
	if err := cmd.Run(); err != nil {
		// Consider logging the error for debugging, e.g., log.Printf("uv command not found: %v", err)
		return false
	}
	return true
}

// InstallPyPIPackage installs a Python package using uv, creates a virtual environment,
// and then attempts to initialize it as an MCP server.
// workDir is currently unused, venvsBaseDir is used instead.
func InstallPyPIPackage(ctx context.Context, packageName string, version string, workDir string, envVars map[string]string) (*MCPServerInfo, error) {
	if !CheckUVXAvailable() {
		return nil, fmt.Errorf("uv command is not available")
	}

	// Ensure the base directory for virtual environments exists
	if err := os.MkdirAll(pythonVenvsBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create python venvs base directory %s: %w", pythonVenvsBaseDir, err)
	}

	pkgVenvDir := filepath.Join(pythonVenvsBaseDir, packageName, "venv")

	// Create virtual environment using uv
	venvCmd := exec.CommandContext(ctx, "uv", "venv", pkgVenvDir)
	var stderrVenv bytes.Buffer
	venvCmd.Stderr = &stderrVenv
	if err := venvCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create virtual environment for %s at %s: %w, stderr: %s", packageName, pkgVenvDir, err, stderrVenv.String())
	}

	// Install package into the virtual environment
	// Construct package string (e.g., packageName or packageName==version)
	packageToInstall := packageName
	if version != "" && version != "latest" {
		packageToInstall = fmt.Sprintf("%s==%s", packageName, version)
	}

	pythonExecutable := filepath.Join(pkgVenvDir, "bin", "python")
	pipInstallCmd := exec.CommandContext(ctx, "uv", "pip", "install", packageToInstall, "--python", pythonExecutable)
	var stdoutPip, stderrPip bytes.Buffer
	pipInstallCmd.Stdout = &stdoutPip
	pipInstallCmd.Stderr = &stderrPip

	if err := pipInstallCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to install package %s: %w, stdout: %s, stderr: %s",
			packageToInstall, err, stdoutPip.String(), stderrPip.String())
	}

	// Assuming the executable name is the same as the package name
	// This might need to be more sophisticated later (e.g., from package metadata)
	executableName := packageName
	mcpCommandPath := filepath.Join(pkgVenvDir, "bin", executableName)

	// Prepare environment variables for the MCP client
	effectiveEnv := os.Environ() // Get current environment
	for key, value := range envVars {
		effectiveEnv = append(effectiveEnv, fmt.Sprintf("%s=%s", key, value))
	}

	// Use mark3labs/mcp-go to create stdio client
	// Note: NewStdioMCPClientFunc is used in npm.go for testability, consider similar here.
	mcpClient, err := client.NewStdioMCPClient(mcpCommandPath, effectiveEnv) // No separate args, command is the full path
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client for %s: %w", packageName, err)
	}
	defer mcpClient.Close()

	// Set context and timeout for MCP initialization
	// Using a shorter timeout for initialization as in npm.go
	initCtx, cancel := context.WithTimeout(ctx, 3*time.Minute) // Original was 3*time.Minute in npm.go
	defer cancel()

	// Start client (mcp-go Start might be called internally by Initialize or might need explicit call)
	// The npm.go version calls mcpClient.Start(runCtx)
	// Let's check mcp-go client.NewStdioMCPClient and Initialize docs.
	// Based on npm.go, Start() is called before Initialize()
	if err := mcpClient.Start(initCtx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client for %s: %w", packageName, err)
	}

	// Initialize client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "one-mcp", // Should this be configurable or a constant?
		Version: "1.0.0",   // App version
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{} // Define as needed

	initResult, err := mcpClient.Initialize(initCtx, initRequest)
	if err != nil {
		// This error might mean the installed package is not an MCP server,
		// or there was an issue communicating.
		// For now, we treat it as a failure to get MCPServerInfo.
		// If the goal is just to install, and MCP init is optional, this logic would change.
		return nil, fmt.Errorf("failed to initialize MCP client for %s (is it an MCP server?): %w. Install stdout: %s, Install stderr: %s",
			packageName, err, stdoutPip.String(), stderrPip.String())
	}

	// From initialization result, collect server info
	serverInfo := &MCPServerInfo{ // This type is from market package (defined in npm.go)
		Name:            initResult.ServerInfo.Name,
		Version:         initResult.ServerInfo.Version,
		ProtocolVersion: initResult.ProtocolVersion,
		Capabilities:    initResult.Capabilities,
	}

	// Optionally, could add to a client manager here if needed, similar to npm.go
	// manager := GetMCPClientManager()
	// if err := manager.InitializeClient(packageName, 0 /* serviceID, needs to be passed or handled */);

	return serverInfo, nil
}

// UninstallPyPIPackage (Placeholder for future implementation)
// This would involve removing the virtual environment.
func UninstallPyPIPackage(ctx context.Context, packageName string) error {
	pkgVenvDir := filepath.Join(pythonVenvsBaseDir, packageName, "venv")
	if _, err := os.Stat(pkgVenvDir); os.IsNotExist(err) {
		return fmt.Errorf("virtual environment for package %s does not exist at %s", packageName, pkgVenvDir)
	}

	// Remove the entire package-specific directory
	pkgBaseDir := filepath.Join(pythonVenvsBaseDir, packageName)
	if err := os.RemoveAll(pkgBaseDir); err != nil {
		return fmt.Errorf("failed to remove virtual environment for %s at %s: %w", packageName, pkgBaseDir, err)
	}
	return nil
}
