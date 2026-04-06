package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const configFileName string = ".gatorconfig.json"

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("reading home dir: %w", err)
	}
	return filepath.Join(homeDir, configFileName), nil
}

func ReadConfig() (Config, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	return Config{}, nil
}
