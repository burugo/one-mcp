package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/library/market"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type npmRegistryProbe struct{ urls []string }

func (p *npmRegistryProbe) RoundTrip(r *http.Request) (*http.Response, error) {
	p.urls = append(p.urls, r.URL.String())
	return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"Not found"}`)), Request: r}, nil
}

func TestInstallCustomGitHubSourcePreservesNPXArguments(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "github-install.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())
	// Only the executable boundary is replaced. No package is fetched or run.
	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "npx"), []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo fixture; exit 0; fi\nwhile [ ! -f \"$ONE_MCP_INSTALL_RELEASE\" ]; do /bin/sleep 0.01; done\nexit 1\n"), 0755))
	t.Setenv("PATH", binDir)
	probe := &npmRegistryProbe{}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = probe
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	for _, tc := range []struct {
		name, source, packageName string
		args                      []string
		accepted                  bool
	}{
		{"github", "custom", "github:owner/repo", []string{"-y", "github:owner/repo"}, true},
		{"github-ref", "custom", "github:owner/another#v1.2.3", []string{"--yes", "github:owner/another#v1.2.3", "--port", "4321"}, true},
		{"registry", "custom", "missing-package@1.2.3", nil, false},
		{"scoped-registry", "custom", "@scope/missing@1.2.3", nil, false},
		{"marketplace", "marketplace", "github:owner/repo", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			releaseFile := filepath.Join(t.TempDir(), "release")
			t.Setenv("ONE_MCP_INSTALL_RELEASE", releaseFile)
			probe.urls = nil
			request := newJSONRequest(t, http.MethodPost, "/api/mcp_market/install_or_add_service", map[string]any{
				"source_type": tc.source, "package_manager": "npm", "package_name": tc.packageName, "custom_args": tc.args,
			})
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Set("user_id", int64(10001))
			ctx.Set("lang", "en")
			InstallOrAddService(ctx)
			if !tc.accepted {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Len(t, probe.urls, 1, "registry packages must still be validated")
				require.Contains(t, recorder.Body.String(), "404")
				return
			}
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.Empty(t, probe.urls, "GitHub sources must not be looked up in npm registry")
			var response struct {
				Success bool `json:"success"`
				Data    struct {
					ServiceID int64 `json:"mcp_service_id"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.True(t, response.Success)
			manager := market.GetInstallationManager()
			task, ok := manager.GetTaskStatus(response.Data.ServiceID)
			require.True(t, ok)
			finished := false
			t.Cleanup(func() {
				require.NoError(t, os.WriteFile(releaseFile, nil, 0600))
				if !finished {
					select {
					case <-task.CompletionNotify:
					case <-time.After(5 * time.Second):
						t.Error("fixture did not stop during cleanup")
					}
				}
				manager.CleanupTask(response.Data.ServiceID)
			})
			stored, err := model.GetServiceByID(response.Data.ServiceID)
			require.NoError(t, err)
			endpoint, err := url.Parse("http://localhost/proxy/" + stored.Name + "/mcp")
			require.NoError(t, err)
			require.Empty(t, endpoint.Fragment, "source refs must not turn the proxy path into a URL fragment")
			require.NoError(t, os.WriteFile(releaseFile, nil, 0600))
			// Wait for the fixture process failure and all task DB writes before cleanup.
			select {
			case completed := <-task.CompletionNotify:
				finished = true
				require.Equal(t, market.StatusFailed, completed.Status)
				require.Contains(t, completed.Error, "failed to initialize MCP client")
				require.Contains(t, completed.Output, completed.Error)
				require.Equal(t, "npx", completed.Command)
				require.Equal(t, tc.packageName, completed.PackageName)
				require.Equal(t, tc.args, completed.Args)
			case <-time.After(5 * time.Second):
				t.Fatal("fixture installation did not finish")
			}
			statusRecorder := httptest.NewRecorder()
			statusCtx, _ := gin.CreateTestContext(statusRecorder)
			statusCtx.Request = httptest.NewRequest(http.MethodGet, "/api/mcp_market/installation_status?service_id="+strconv.FormatInt(response.Data.ServiceID, 10), nil)
			GetInstallationStatus(statusCtx)
			require.Equal(t, http.StatusOK, statusRecorder.Code)
			var status struct {
				Data struct {
					Status string `json:"status"`
					Error  string `json:"error"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(statusRecorder.Body.Bytes(), &status))
			require.Equal(t, string(market.StatusFailed), status.Data.Status)
			require.Contains(t, status.Data.Error, "failed to initialize MCP client")
		})
	}
}
