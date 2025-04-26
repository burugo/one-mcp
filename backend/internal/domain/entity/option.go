package model

import (
	"strconv"
	"strings"

	"one-cmp/backend/internal/common"
	"one-cmp/backend/internal/library/db"
)

// OptionMap stores system options, accessible via common.OptionMapRWMutex
var OptionMap map[string]string

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	var err error
	err = db.DB.Find(&options).Error
	return options, err
}

func UpdateOption(key string, value string) error {
	option := Option{
		Key: key,
	}
	db.DB.FirstOrCreate(&option, Option{Key: key})
	option.Value = value
	db.DB.Save(&option)
	// Update the central OptionMap (accessed via mutex)
	updateOptionMapValue(key, value)
	return nil
}

// updateOptionMapValue updates ONLY the in-memory OptionMap.
// The logic to update global variables in common package is REMOVED
// to prevent import cycles and tight coupling.
// Common package or other services should read from model.OptionMap directly.
func updateOptionMapValue(key string, value string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if OptionMap == nil {
		OptionMap = make(map[string]string)
	}
	OptionMap[key] = value
	// common.OptionMap[key] = value // REMOVED - Avoid direct update across packages here.
	// REMOVED logic that updated common.* variables
}
