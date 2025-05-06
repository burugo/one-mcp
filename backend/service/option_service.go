package service

import (
	"one-mcp/backend/model"
)

// AllOption returns all options
func AllOption() ([]*model.Option, error) {
	return model.OptionDB.All()
}

// UpdateOption updates an option in the database and in memory
func UpdateOption(key string, value string) error {
	options, err := model.OptionDB.Where("key = ?", key).Fetch(0, 1)
	var option *model.Option
	if err != nil {
		return err
	}
	if len(options) == 0 {
		option = &model.Option{Key: key, Value: value}
	} else {
		option = options[0]
		option.Value = value
	}
	err = model.OptionDB.Save(option)
	if err != nil {
		return err
	}
	model.UpdateOptionMap(key, value)
	return nil
}
