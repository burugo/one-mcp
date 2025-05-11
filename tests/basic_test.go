package tests

import (
	"context"
	"one-mcp/backend/common"
	"one-mcp/backend/model"
	"os"
	"testing"

	"github.com/burugo/thing"
	thingRedis "github.com/burugo/thing/drivers/cache/redis"
	"github.com/burugo/thing/drivers/db/sqlite"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	common.SQLitePath = ":memory:"
	if os.Getenv("REDIS_CONN_STRING") == "" {
		common.RedisEnabled = false
		common.RDB = nil
	}
	dbAdapter, err := sqlite.NewSQLiteAdapter(":memory:")
	if err != nil {
		panic(err)
	}
	var cacheClient thing.CacheClient = nil
	if common.RedisEnabled && common.RDB != nil {
		cacheClient, _ = thingRedis.NewClient(common.RDB, nil)
	}
	thing.Configure(dbAdapter, cacheClient)

	// 初始化 model 层 ORM
	if err := model.InitDB(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestRedisConnection(t *testing.T) {
	os.Setenv("REDIS_CONN_STRING", "redis://localhost:6379/0")
	err := common.InitRedisClient()
	if !common.RedisEnabled {
		t.Skip("Redis not enabled, skipping test")
	}
	assert.NoError(t, err)
	err = common.RDB.Set(context.Background(), "test-key", "test-value", 0).Err()
	assert.NoError(t, err)
	val, err := common.RDB.Get(context.Background(), "test-key").Result()
	assert.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

func TestPasswordHash(t *testing.T) {
	hash, err := common.Password2Hash("testpass")
	assert.NoError(t, err)
	assert.True(t, common.ValidatePasswordAndHash("testpass", hash))
	assert.False(t, common.ValidatePasswordAndHash("wrongpass", hash))
}

func TestUserInsertAndQuery(t *testing.T) {
	// 需要 SQLite 环境支持
	os.Setenv("SQLITE_PATH", "test.db")
	user := &model.User{
		Username: "testuser",
		Password: "testpass",
		Email:    "test@example.com",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	err := user.Insert()
	assert.NoError(t, err)
	queryUser := &model.User{Username: "testuser"}
	err = queryUser.FillUserByUsername()
	assert.NoError(t, err)
	assert.Equal(t, user.Email, queryUser.Email)
	// 清理
	// TODO: 删除测试用户
}

// TODO: 增加 HTTP 路由和中间件测试

func TestGetUserByIdAndDeleteUserById(t *testing.T) {
	// 需要 SQLite 环境支持
	os.Setenv("SQLITE_PATH", "test.db")
	user := &model.User{
		Username: "testuser",
		Password: "testpass",
		Email:    "test@example.com",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	err := user.Insert()
	assert.NoError(t, err)

	gotUser, err := model.GetUserById(int64(user.ID), false)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, gotUser.ID)

	err = model.DeleteUserById(int64(user.ID))
	assert.NoError(t, err)

	// 清理
	// TODO: 删除测试用户
}
