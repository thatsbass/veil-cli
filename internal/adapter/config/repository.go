package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Repository is the storage contract for CLI configuration.
type Repository interface {
	Load() (*Config, error)
	Save(*Config) error
	Delete() error
}

type jsonRepository struct{}

// NewJSONRepository returns a Repository that persists to ~/.veil/config.json
// with 0600 permissions.
func NewJSONRepository() Repository {
	return &jsonRepository{}
}

func (r *jsonRepository) Load() (*Config, error) {
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

func (r *jsonRepository) Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	return nil
}

func (r *jsonRepository) Delete() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config.Delete: %w", err)
	}
	return nil
}

// Load reads config from the default JSON path. Prefer injecting a Repository
// over calling this convenience helper.
func Load() (*Config, error) { return NewJSONRepository().Load() }

// Delete removes the default config file. Prefer injecting a Repository.
func Delete() error { return NewJSONRepository().Delete() }

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config.configPath: %w", err)
	}
	return filepath.Join(home, ".veil", "config.json"), nil
}
