package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	configDirName = ".config/hostbox"
	configFile    = "config.json"
	tokenFile     = "token"
)

// configRoot can be overridden for testing.
var configRoot = ""

func configDirPath() string {
	if configRoot != "" {
		return configRoot
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDirName)
}

func configFilePath() string {
	return filepath.Join(configDirPath(), configFile)
}

func tokenFilePath() string {
	return filepath.Join(configDirPath(), tokenFile)
}

type Config struct {
	ServerURL string `json:"server_url"`
	path      string
}

func Load() (*Config, error) {
	data, err := os.ReadFile(configFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{path: configFilePath()}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.path = configFilePath()
	return &cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(configDirPath(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFilePath(), data, 0600)
}

func SaveToken(token string) error {
	if err := os.MkdirAll(configDirPath(), 0700); err != nil {
		return err
	}
	return os.WriteFile(tokenFilePath(), []byte(token), 0600)
}

func LoadToken() (string, error) {
	data, err := os.ReadFile(tokenFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func ClearToken() error {
	err := os.Remove(tokenFilePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
