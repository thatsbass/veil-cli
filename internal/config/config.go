package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultAPIURL = "https://api.veil.dev"

// Config holds the local Veil CLI configuration stored at ~/.veil/config.json.
type Config struct {
	APIKey string `json:"api_key"`
	APIURL string `json:"api_url"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config.configPath: %w", err)
	}
	return filepath.Join(home, ".veil", "config.json"), nil
}

// Exists reports whether a config file is present.
func Exists() bool {
	path, err := configPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Load reads the config file. Returns defaults if the file does not exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}
	return cfg, nil
}

// Save writes the config to ~/.veil/config.json.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	return nil
}

// Delete removes ~/.veil/config.json.
func Delete() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config.Delete: %w", err)
	}
	return nil
}

func defaults() *Config {
	return &Config{APIURL: defaultAPIURL}
}
