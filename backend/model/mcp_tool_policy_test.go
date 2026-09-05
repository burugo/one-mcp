package model

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"one-mcp/backend/common"

	"github.com/stretchr/testify/require"
)

func TestMCPToolPolicyQueriesReturnMoreThanOneDatabasePage(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "tool-policy-pagination.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, InitDB())

	for index := 0; index < 305; index++ {
		_, err := SetMCPToolPolicy(77, fmt.Sprintf("tool-%03d", index), false, 1)
		require.NoError(t, err)
	}

	servicePolicies, err := GetMCPToolPoliciesForService(77)
	require.NoError(t, err)
	require.Len(t, servicePolicies, 305)
	allPolicies, err := GetAllMCPToolPolicies()
	require.NoError(t, err)
	require.Len(t, allPolicies, 305)
}

func TestSetMCPToolPolicyConcurrentUpsertKeepsOneRow(t *testing.T) {
	originalPath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "tool-policy-concurrent.db")
	t.Cleanup(func() { common.SQLitePath = originalPath })
	require.NoError(t, InitDB())

	var waitGroup sync.WaitGroup
	errors := make(chan error, 20)
	for index := 0; index < 20; index++ {
		waitGroup.Add(1)
		go func(enabled bool) {
			defer waitGroup.Done()
			_, err := SetMCPToolPolicy(88, "shared-tool", enabled, 1)
			errors <- err
		}(index%2 == 0)
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	policies, err := GetMCPToolPoliciesForService(88)
	require.NoError(t, err)
	require.Len(t, policies, 1)
}
