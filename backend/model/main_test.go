package model

import (
	"database/sql"
	"path/filepath"
	"testing"

	"one-mcp/backend/common"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestInitDBBackfillsOAuthColumnsForExistingServices(t *testing.T) {
	originalSQLitePath := common.SQLitePath
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	common.SQLitePath = dbPath
	t.Cleanup(func() {
		common.SQLitePath = originalSQLitePath
	})

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE mcp_services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted BOOLEAN NOT NULL,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT NOT NULL,
		category TEXT NOT NULL,
		icon TEXT NOT NULL,
		default_on BOOLEAN NOT NULL,
		admin_only BOOLEAN NOT NULL,
		order_num INTEGER NOT NULL,
		enabled BOOLEAN NOT NULL,
		type TEXT NOT NULL,
		command TEXT NOT NULL,
		args_json TEXT NOT NULL DEFAULT '{}',
		allow_user_override BOOLEAN NOT NULL,
		client_config_templates TEXT NOT NULL,
		required_env_vars_json TEXT NOT NULL,
		package_manager TEXT NOT NULL,
		source_package_name TEXT NOT NULL,
		installed_version TEXT NOT NULL,
		installer_user_id INTEGER NOT NULL,
		default_envs_json TEXT NOT NULL DEFAULT '{}',
		headers_json TEXT NOT NULL DEFAULT '{}',
		rpd_limit INTEGER NOT NULL DEFAULT 0
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO mcp_services (
		created_at, updated_at, deleted, name, display_name, description, category, icon, default_on,
		admin_only, order_num, enabled, type, command, args_json,
		allow_user_override, client_config_templates, required_env_vars_json,
		package_manager, source_package_name, installed_version, installer_user_id,
		default_envs_json, headers_json, rpd_limit
	) VALUES (CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0, 'legacy-service', 'Legacy Service', '', '', '', 0, 0, 0, 1,
		'sse', 'https://example.test/mcp', '{}', 0, '', '', '', '', '1.0.0', 1,
		'{}', '{}', 0)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	require.NoError(t, InitDB())

	service, err := GetServiceByName("legacy-service")
	require.NoError(t, err)
	require.False(t, service.OAuthEnabled)
	require.Empty(t, service.OAuthScopes)
	require.Equal(t, "not_configured", service.OAuthAuthStatus)
}
