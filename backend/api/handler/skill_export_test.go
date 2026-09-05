package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestSkillExportOmitsGloballyDisabledTools(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "skill-tool-policy.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, model.InitDB())

	svc := &model.MCPService{Name: "skill-policy-svc", DisplayName: "Skill Policy Svc", Type: model.ServiceTypeStdio, Command: "echo", ArgsJSON: "[]", Enabled: true}
	require.NoError(t, model.CreateService(svc))
	group := &model.MCPServiceGroup{UserID: 1, Name: "skill-policy-group", DisplayName: "Skill Policy Group", Enabled: true}
	group.SetServiceIDs([]int64{svc.ID})
	require.NoError(t, group.Insert())
	proxy.GetToolsCacheManager().SetServiceTools(svc.ID, &proxy.ToolsCacheEntry{Tools: []mcp.Tool{
		{Name: "allowed-tool", Description: "allowed description"},
		{Name: "skill-blocked-tool", Description: "blocked description"},
	}})
	t.Cleanup(func() { proxy.GetToolsCacheManager().DeleteServiceTools(svc.ID) })
	_, err := model.SetMCPToolPolicy(svc.ID, "skill-blocked-tool", false, 1)
	require.NoError(t, err)

	archive, err := buildSkillZip(context.Background(), group, &model.User{}, "http://localhost:3000")
	require.NoError(t, err)
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)

	var toolsDoc string
	for _, file := range reader.File {
		if file.Name != "tools/skill-policy-svc.md" {
			continue
		}
		opened, openErr := file.Open()
		require.NoError(t, openErr)
		contents, readErr := io.ReadAll(opened)
		require.NoError(t, readErr)
		require.NoError(t, opened.Close())
		toolsDoc = string(contents)
	}
	require.Contains(t, toolsDoc, "allowed-tool")
	require.NotContains(t, toolsDoc, "skill-blocked-tool")
}
