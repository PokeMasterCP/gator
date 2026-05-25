package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	configFileName := ".gatorconfig.json"
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(homeDir, configFileName), nil
}

func ReadConfig() (Config, error) {
	var config Config
	configPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	confBytes, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(confBytes, &config); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return config, nil
}

func writeConfig(conf Config) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing to config file: %w", err)
	}

	return nil
}

func (c *Config) SetUser(user string) error {
	c.CurrentUserName = user
	err := writeConfig(*c)
	if err != nil {
		return err
	}

	return nil
}
