package model

import (
	"encoding/gob"
	"one-mcp/backend/common"

	"github.com/burugo/thing"
	redisCache "github.com/burugo/thing/drivers/cache/redis"
	"github.com/burugo/thing/drivers/db/sqlite"
)

// 全局变量用于兼容旧代码，后续可逐步移除
// var DB *gorm.DB

func init() {
	gob.Register(ServiceCategory(""))
	gob.Register(ServiceType(""))
	gob.Register(EnvVarDefinition{})
	gob.Register(ClientTemplateDetail{})
}
func createRootAccountIfNeed() error {
	// 检查是否有用户，无则创建 root 用户
	userThing, err := thing.Use[*User]()
	if err != nil {
		return err
	}
	users, err := userThing.Query(thing.QueryParams{}).Fetch(0, 1)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		common.SysLog("no user exists, create a root user for you: username is root, password is 123456")
		hashedPassword, err := common.Password2Hash("123456")
		if err != nil {
			return err
		}
		rootUser := &User{
			Username:    "root",
			Password:    hashedPassword,
			Role:        common.RoleRootUser,
			Status:      common.UserStatusEnabled,
			DisplayName: "Root User",
			Email:       "root@localhost",
			GitHubId:    "",
			WeChatId:    "",
			Token:       "",
		}
		err = userThing.Save(rootUser)
		if err != nil {
			return err
		}
	}
	return nil
}

func InitDB() (err error) {
	dbAdapter, err := sqlite.NewSQLiteAdapter(common.SQLitePath)
	if err != nil {
		common.FatalLog(err)
		return err
	}
	var cacheClient thing.CacheClient = nil
	if common.RedisEnabled && common.RDB != nil {
		cacheClient, err = redisCache.NewClient(common.RDB, nil)
		if err != nil {
			return err
		}
	}
	thing.Configure(dbAdapter, cacheClient)

	// 1. AutoMigrate all models first
	err = thing.AutoMigrate(&User{}, &Option{}, &MCPService{}, &UserConfig{}, &ConfigService{})
	if err != nil {
		return err
	}

	// 2. Initialize all ORM instances
	if err := UserInit(); err != nil {
		return err
	}
	if err := OptionInit(); err != nil {
		return err
	}
	// InitOptionMapFromDB should be called after OptionInit and AutoMigrate
	if err := InitOptionMapFromDB(); err != nil {
		return err
	}
	if err := MCPServiceInit(); err != nil {
		return err
	}
	if err := ConfigServiceInit(); err != nil {
		return err
	}
	if err := UserConfigInit(); err != nil {
		return err
	}

	// 3. Perform data-dependent operations like creating a root account
	return createRootAccountIfNeed()
}

func CloseDB() error {
	// Thing ORM 不需要显式关闭 DB，若后续有需要可补充
	return nil
}
