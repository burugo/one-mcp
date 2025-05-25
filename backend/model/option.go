package model

import (
	"fmt"
	"one-mcp/backend/common"
	"strconv"

	"github.com/burugo/thing"
)

// OptionMap stores system options, accessible via common.OptionMapRWMutex
var OptionMap = common.OptionMap

type Option struct {
	thing.BaseModel
	Key   string `json:"key" db:"key,unique"`
	Value string `json:"value"`
}

// DB interface functions to avoid circular imports
// These will be called by code in library/db

// UpdateOptionMap updates the central OptionMap with a new option value
func UpdateOptionMap(key string, value string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	OptionMap[key] = value
}

// The functions below will be implemented in service/option_service.go
// to avoid circular dependencies

// AllOption - moved to service
// UpdateOption - moved to service

var OptionDB *thing.Thing[*Option]

// OptionInit 用于在 InitDB 时初始化 OptionDB
func OptionInit() error {
	var err error
	OptionDB, err = thing.Use[*Option]()
	if err != nil {
		return err
	}
	return nil
}

// InitOptionMap initializes the option map
func InitOptionMap() error {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	common.OptionMap["ServerAddress"] = common.ServerAddress
	common.OptionMap["Port"] = strconv.Itoa(*common.Port)
	if err := InitOptionMapFromDB(); err != nil {
		common.SysError(fmt.Sprintf("Failed to initialize option map from database: %v", err))
		return err
	}
	return nil
}

// InitOptionMapFromDB loads all options from the database into OptionMap
func InitOptionMapFromDB() error {
	if OptionDB == nil {
		return nil // OptionDB not initialized
	}
	options, err := OptionDB.All()
	if err != nil {
		return err
	}
	for _, opt := range options {
		OptionMap[opt.Key] = opt.Value
	}
	return nil
}
