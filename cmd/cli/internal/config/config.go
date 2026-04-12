package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	configDir  = ".config/hostbox"
	configFile = "config.json"
	tokenFile  = "token"
)

type Config struct {
	ServerURL string `json:"server_url"`
	path      string
}

func configDirPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configDir)
}

func configFilePath() string {
	return filepath.Join(configDirPath(), configFile)
}

func tokenFilePath() string {
	return filepath.Join(configDirPath(), tokenFile)
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

// Token management — stored as a plain file with 0600 perms.
// Future: integrate OS keyring for better security.

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
