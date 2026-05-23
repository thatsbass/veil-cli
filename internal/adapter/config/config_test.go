package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONRepository_LoadDefaults(t *testing.T) {
	// Point configPath to a non-existent file so Load returns defaults.
	t.Setenv("HOME", t.TempDir())

	repo := NewJSONRepository()
	cfg, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIURL != defaultAPIURL {
		t.Errorf("default APIURL: got %q, want %q", cfg.APIURL, defaultAPIURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("default APIKey should be empty, got %q", cfg.APIKey)
	}
}

func TestJSONRepository_SaveAndLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	repo := NewJSONRepository()
	original := &Config{APIKey: "vl_live_test", APIURL: "https://test.veil.dev"}

	if err := repo.Save(original); err != nil {
		t.Fatal("Save:", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatal("Load:", err)
	}
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey: got %q, want %q", loaded.APIKey, original.APIKey)
	}
	if loaded.APIURL != original.APIURL {
		t.Errorf("APIURL: got %q, want %q", loaded.APIURL, original.APIURL)
	}
}

func TestJSONRepository_FilePermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	repo := NewJSONRepository()
	if err := repo.Save(&Config{APIKey: "key", APIURL: "url"}); err != nil {
		t.Fatal(err)
	}

	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".veil", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions: got %o, want 0600", perm)
	}
}

func TestJSONRepository_Delete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	repo := NewJSONRepository()
	if err := repo.Save(&Config{APIKey: "key", APIURL: "url"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(); err != nil {
		t.Fatal("Delete:", err)
	}

	// Load after delete should return defaults, not an error.
	cfg, err := repo.Load()
	if err != nil {
		t.Fatal("Load after delete:", err)
	}
	if cfg.APIKey != "" {
		t.Error("expected empty APIKey after delete")
	}
}

func TestJSONRepository_DeleteNonExistent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Delete on a non-existent file should not error.
	repo := NewJSONRepository()
	if err := repo.Delete(); err != nil {
		t.Errorf("Delete on non-existent file: %v", err)
	}
}
