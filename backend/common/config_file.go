package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/ini.v1"
)

const defaultConfigTemplate = "PORT=3000\nSQLITE_PATH=data/one-mcp.db\nENABLE_GZIP=true\nJWT_SECRET=%s\n"

func loadConfigFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "one-mcp", "config.ini")
	if err := ensureConfigFile(configPath); err != nil {
		return err
	}

	configMap, err := parseIniConfig(configPath)
	if err != nil {
		return err
	}

	if err := applyConfigMap(configMap); err != nil {
		return fmt.Errorf("apply config file %s: %w", configPath, err)
	}

	return nil
}

func ensureConfigFile(configPath string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config directory %s: %w", configDir, err)
	}

	configFile, err := os.OpenFile(configPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("create config file %s: %w", configPath, err)
	}
	defer configFile.Close()

	if _, err := configFile.WriteString(fmt.Sprintf(defaultConfigTemplate, uuid.New().String())); err != nil {
		return fmt.Errorf("write default config file %s: %w", configPath, err)
	}

	return nil
}

func parseIniConfig(path string) (map[string]string, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("parse ini config %s: %w", path, err)
	}

	configMap := make(map[string]string)
	for _, section := range cfg.Sections() {
		for _, key := range section.Keys() {
			configKey := strings.ToUpper(strings.TrimSpace(key.Name()))
			if configKey == "" {
				continue
			}
			configMap[configKey] = strings.TrimSpace(key.Value())
		}
	}

	return configMap, nil
}

func applyConfigMap(configMap map[string]string) error {
	if configValue, ok := configMap["SESSION_SECRET"]; ok && configValue != "" {
		SessionSecret = configValue
	}

	if configValue, ok := configMap["SQLITE_PATH"]; ok && configValue != "" {
		SQLitePath = configValue
	}

	if configValue, ok := configMap["JWT_SECRET"]; ok && configValue != "" {
		JWTSecret = configValue
	}

	if configValue, ok := configMap["JWT_REFRESH_SECRET"]; ok && configValue != "" {
		JWTRefreshSecret = configValue
	} else if configValue, ok := configMap["JWT_SECRET"]; ok && configValue != "" {
		JWTRefreshSecret = configValue
	}

	if configValue, ok := configMap["PORT"]; ok && configValue != "" {
		portInt, err := strconv.Atoi(configValue)
		if err != nil {
			return fmt.Errorf("invalid value for PORT: %w", err)
		}
		*Port = portInt
	}

	if configValue, ok := configMap["ENABLE_GZIP"]; ok && configValue != "" {
		enableGzipBool, err := strconv.ParseBool(configValue)
		if err != nil {
			return fmt.Errorf("invalid value for ENABLE_GZIP: %w", err)
		}
		*EnableGzip = enableGzipBool
	}

	return nil
}
