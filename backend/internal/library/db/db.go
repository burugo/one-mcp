package db

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"

	"one-cmp/backend/internal/common"
	"one-cmp/backend/internal/domain/model"
)

var DB *gorm.DB

func createRootAccountIfNeed() error {
	var userCount int64
	DB.Model(&model.User{}).Count(&userCount)
	if userCount == 0 {
		common.SysLog("no user exists, create a root/admin user for you: username is root, password is 123456")
		hashedPassword, err := common.Password2Hash("123456")
		if err != nil {
			return err
		}
		adminUser := model.User{
			Username:    "root",
			Password:    hashedPassword,
			Role:        model.RoleAdminUser,
			Status:      model.UserStatusEnabled,
			DisplayName: "Root User",
			Email:       "admin@localhost",
		}
		if err := DB.Create(&adminUser).Error; err != nil {
			return err
		}
		common.SysLog("Root/admin user created successfully.")
	}
	return nil
}

func InitOptionMap() error {
	common.OptionMapRWMutex.Lock()
	model.OptionMap = make(map[string]string)
	var options []model.Option
	err := DB.Find(&options).Error
	if err != nil {
		common.OptionMapRWMutex.Unlock()
		return fmt.Errorf("failed to query options: %w", err)
	}
	for _, option := range options {
		model.OptionMap[option.Key] = option.Value
	}
	common.OptionMapRWMutex.Unlock()
	common.SysLog("options loaded from database into model.OptionMap.")
	return nil
}

func CountTable(tableName string) (num int64) {
	DB.Table(tableName).Count(&num)
	return
}

func InitDB() (err error) {
	var dbInstance *gorm.DB
	dsn := os.Getenv("SQL_DSN")

	if dsn != "" {
		common.SysLog("Using MySQL database")
		dbInstance, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			PrepareStmt: true,
		})
	} else {
		common.SysLog("SQL_DSN not set, using SQLite as database: " + common.SQLitePath)
		dbInstance, err = gorm.Open(sqlite.Open(common.SQLitePath), &gorm.Config{
			PrepareStmt: true,
		})
	}

	if err != nil {
		common.FatalLog("failed to connect database: " + err.Error())
		return err
	}

	DB = dbInstance

	err = DB.AutoMigrate(
		&model.User{},
		&model.MCPService{},
		&model.UserConfig{},
		&model.ConfigService{},
		&model.Option{},
		&model.File{},
	)
	if err != nil {
		common.FatalLog("failed to auto migrate database schema: " + err.Error())
		return err
	}

	if err = createRootAccountIfNeed(); err != nil {
		common.FatalLog("failed to create root account: " + err.Error())
		return err
	}

	if err = InitOptionMap(); err != nil {
		common.FatalLog("failed to initialize options: " + err.Error())
		return err
	}

	common.SysLog("Database initialized successfully.")
	return nil
}

func CloseDB() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	common.SysLog("Closing database connection.")
	err = sqlDB.Close()
	return err
}
