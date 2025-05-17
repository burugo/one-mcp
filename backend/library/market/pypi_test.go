package market

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	// "one-mcp/backend/model" // Will be needed for MCPServerInfo if testing client init
)

// Mocking exec.Command for testing
var (
	mockExecCommand func(ctx context.Context, command string, args ...string) *exec.Cmd
	stdLookPath     = exec.LookPath // Keep original for restoring
)

func mockLookPath(file string) (string, error) {
	if file == "uv" {
		if mockUVPathError != nil {
			return "", mockUVPathError
		}
		return "/fake/path/to/uv", nil
	}
	return stdLookPath(file)
}

var mockUVPathError error

func TestMain(m *testing.M) {
	// Setup: Replace exec.LookPath with our mock
	execLookPath = mockLookPath
	
	// Run tests
	code := m.Run()
	
	// Teardown: Restore original exec.LookPath
	execLookPath = stdLookPath
	mockUVPathError = nil
	os.Exit(code)
}


func TestCheckUVXAvailable(t *testing.T) {
	tests := []struct {
		name          string
		mockError     error
		expectedAvail bool
	}{
		{"uv found", nil, true},
		{"uv not found", errors.New("uv not found in PATH"), false},
	}

	originalLookPath := execLookPath
	defer func() { execLookPath = originalLookPath }() // Restore

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUVPathError = tt.mockError
			execLookPath = mockLookPath // Ensure our mock is used for this subtest's context

			available := CheckUVXAvailable()
			if available != tt.expectedAvail {
				t.Errorf("CheckUVXAvailable() got = %v, want %v", available, tt.expectedAvail)
			}
		})
	}
}

// Helper for InstallPyPIPackage tests
type cmdMocker struct {
	ctx     context.Context
	command string
	args    []string
	stdOut  string
	stdErr  string
	exitErr error
}

func (cm *cmdMocker) CombinedOutput() ([]byte, error) {
	var output []byte
	output = append(output, []byte(cm.stdOut)...)
	if cm.stdErr != "" {
		if len(output) > 0 && !strings.HasSuffix(string(output), "\n") {
			output = append(output, '\n')
		}
		output = append(output, []byte(cm.stdErr)...)
	}
	return output, cm.exitErr
}

func (cm *cmdMocker) Start() error { return nil }
func (cm *cmdMocker) Wait() error  { return cm.exitErr }
func (cm *cmdMocker) Run() error   { 
	_, err := cm.CombinedOutput()
	return err
}
func (cm *cmdMocker) Output() ([]byte, error) {
	return cm.CombinedOutput()
}


func TestInstallPyPIPackage_Success(t *testing.T) {
	originalExecCommand := execCommand
	mockCmdOutput := "Successfully installed package"
	mockCmdExitError := error(nil)

	execCommand = func(ctx context.Context, command string, args ...string) cmdRunner {
		// Basic validation of command and args if needed
		if command != "uv" || args[0] != "pip" || args[1] != "install" {
			t.Fatalf("execCommand called with unexpected command/args: %s %v", command, args)
		}
		return &cmdMocker{ctx: ctx, command: command, args: args, stdOut: mockCmdOutput, exitErr: mockCmdExitError}
	}
	defer func() { execCommand = originalExecCommand }()

	// Mock mcp-go client initialization if InstallPyPIPackage attempts it
	// For now, assume it doesn't or we are not testing that part deeply yet.
	// originalNewClientFn := newClientFn
	// newClientFn = func(serverPath string, serverInfo model.MCPServerInfo, timeout time.Duration) (MCPClient, error) {
	// 	 return &mockMCPClient{}, nil
	// }
	// defer func() { newClientFn = originalNewClientFn }()


	ctx := context.Background()
	packageName := "test-package"
	version := "1.0.0"
	envVars := map[string]string{"API_KEY": "testkey"}
	venvName := packageName + "-" + strings.ReplaceAll(version, ".", "_")
	expectedVenvPath := filepath.Join("data", "python_venvs", venvName)


	// Ensure the venv directory does not exist to test its creation path
	os.RemoveAll(expectedVenvPath)

	serverInfo, logs, err := InstallPyPIPackage(ctx, packageName, version, envVars)

	if err != nil {
		t.Fatalf("InstallPyPIPackage() unexpected error: %v, logs: %s", err, strings.Join(logs, "\n"))
	}

	if serverInfo == nil {
		t.Errorf("InstallPyPIPackage() serverInfo is nil, want non-nil (even if empty if not an MCP server)")
	}
	// Further assertions on serverInfo if mcp-go client part is active
	// e.g., if serverInfo.Name == "" for non-mcp package but command success


	foundLog := false
	for _, log := range logs {
		if strings.Contains(log, mockCmdOutput) {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Errorf("Expected log output not found. Logs: %s", strings.Join(logs, "\n"))
	}

	// Check if venv path was created (basic check)
	// if _, errStat := os.Stat(expectedVenvPath); os.IsNotExist(errStat) {
	// 	t.Errorf("Expected virtual environment directory to be created at %s, but it wasn't", expectedVenvPath)
	// }
	// Note: Actual venv creation involves more than just the dir. This test mocks out the `uv venv` and `uv pip install` calls.
	// The test above focuses on the `uv pip install` part. A separate test for venv creation path might be needed if that logic is complex.
}


func TestInstallPyPIPackage_InstallFails(t *testing.T) {
	originalExecCommand := execCommand
	mockCmdErrorOutput := "Error: Failed to install package"
	mockCmdExitError := errors.New("exit status 1")

	execCommand = func(ctx context.Context, command string, args ...string) cmdRunner {
		return &cmdMocker{ctx: ctx, command: command, args: args, stdErr: mockCmdErrorOutput, exitErr: mockCmdExitError}
	}
	defer func() { execCommand = originalExecCommand }()

	ctx := context.Background()
	packageName := "failing-package"
	version := "1.0.0"
	
	_, logs, err := InstallPyPIPackage(ctx, packageName, version, nil)

	if err == nil {
		t.Fatalf("InstallPyPIPackage() expected an error, but got nil")
	}

	if !strings.Contains(err.Error(), "failed to install package") {
		t.Errorf("Error message mismatch. Got: %v, Expected to contain: 'failed to install package'", err)
	}
	
	foundLog := false
	for _, log := range logs {
		if strings.Contains(log, mockCmdErrorOutput) {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Errorf("Expected error log output not found. Logs: %s", strings.Join(logs, "\n"))
	}
}


// TODO: Add tests for UninstallPyPIPackage (placeholder function for now)
// TODO: Add tests for specific mcp-go client interaction if/when that part is more fleshed out in InstallPyPIPackage
// TODO: Test cases where venv creation part of InstallPyPIPackage fails (if uv venv is called explicitly and can be mocked)



</rewritten_file> 