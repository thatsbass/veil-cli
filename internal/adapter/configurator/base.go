package configurator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// baseConfigurator provides the shared behaviour for all ToolConfigurator
// implementations. Concrete configurators supply only the fields that differ:
// binary name, config path, env variable, and read/write strategies.
type baseConfigurator struct {
	name    string
	slug    string
	binary  string
	pathFn  func() (string, error)
	writeFn func(path, apiURL, apiKey string) error
	readKey func(path string) string
	envKey  string
	testCmd string
}

func (b *baseConfigurator) Name() string    { return b.name }
func (b *baseConfigurator) Slug() string    { return b.slug }
func (b *baseConfigurator) TestCmd() string { return b.testCmd }

func (b *baseConfigurator) ConfigPath() (string, error) {
	return b.pathFn()
}

func (b *baseConfigurator) Detect() bool {
	if _, err := exec.LookPath(b.binary); err == nil {
		return true
	}
	path, err := b.pathFn()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func (b *baseConfigurator) ExistingKey() string {
	if k := os.Getenv(b.envKey); k != "" {
		return k
	}
	path, err := b.pathFn()
	if err != nil {
		return ""
	}
	return b.readKey(path)
}

func (b *baseConfigurator) Configure(apiURL, apiKey string) error {
	path, err := b.pathFn()
	if err != nil {
		return fmt.Errorf("%s.Configure: %w", b.name, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("%s.Configure: %w", b.name, err)
	}
	if err := backupFile(path); err != nil {
		return err
	}
	return b.writeFn(path, apiURL, apiKey)
}

func (b *baseConfigurator) Restore() error {
	path, err := b.pathFn()
	if err != nil {
		return fmt.Errorf("%s.Restore: %w", b.name, err)
	}
	return restoreFile(path)
}

// homePath resolves a path relative to the current user's home directory.
func homePath(rel string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("homePath: %w", err)
	}
	return filepath.Join(home, rel), nil
}
