package configurator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- mergeJSON tests ---

func TestMergeJSON_FlatKeys(t *testing.T) {
	path := tmpFile(t, "")
	if err := mergeJSON(path, map[string]any{"apiKey": "key1", "apiBaseUrl": "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	m := readJSONMap(t, path)
	assertEqual(t, "apiKey", m["apiKey"], "key1")
	assertEqual(t, "apiBaseUrl", m["apiBaseUrl"], "https://example.com")
}

func TestMergeJSON_NestedKeys(t *testing.T) {
	path := tmpFile(t, "")
	if err := mergeJSON(path, map[string]any{
		"openai.apiBase": "https://api.veil.dev",
		"openai.apiKey":  "vl_live_test",
	}); err != nil {
		t.Fatal(err)
	}

	m := readJSONMap(t, path)
	openai, ok := m["openai"].(map[string]any)
	if !ok {
		t.Fatal("expected nested openai object")
	}
	assertEqual(t, "openai.apiBase", openai["apiBase"], "https://api.veil.dev")
	assertEqual(t, "openai.apiKey", openai["apiKey"], "vl_live_test")
}

func TestMergeJSON_PreservesExistingKeys(t *testing.T) {
	initial := `{"theme": "dark", "fontSize": 14}`
	path := tmpFile(t, initial)
	if err := mergeJSON(path, map[string]any{"apiKey": "newkey"}); err != nil {
		t.Fatal(err)
	}
	m := readJSONMap(t, path)
	assertEqual(t, "theme", m["theme"], "dark")
	assertEqual(t, "apiKey", m["apiKey"], "newkey")
}

func TestMergeJSON_OverwritesNestedKey(t *testing.T) {
	initial := `{"openai": {"apiBase": "old", "apiKey": "oldkey"}}`
	path := tmpFile(t, initial)
	if err := mergeJSON(path, map[string]any{"openai.apiKey": "newkey"}); err != nil {
		t.Fatal(err)
	}
	m := readJSONMap(t, path)
	openai := m["openai"].(map[string]any)
	assertEqual(t, "openai.apiBase preserved", openai["apiBase"], "old")
	assertEqual(t, "openai.apiKey updated", openai["apiKey"], "newkey")
}

func TestReadJSONKey_Nested(t *testing.T) {
	content := `{"openai": {"apiKey": "vl_live_abc"}}`
	path := tmpFile(t, content)
	got := readJSONKey(path, "openai.apiKey")
	assertEqual(t, "readJSONKey nested", got, "vl_live_abc")
}

func TestReadJSONKey_Missing(t *testing.T) {
	path := tmpFile(t, `{"other": "value"}`)
	got := readJSONKey(path, "openai.apiKey")
	assertEqual(t, "readJSONKey missing", got, "")
}

// --- updateFlatFile tests (TOML) ---

func TestUpdateFlatFile_TOML_New(t *testing.T) {
	path := tmpFile(t, "")
	if err := updateFlatFile(path, map[string]string{
		"api_key":      "vl_live_test",
		"api_base_url": "https://api.veil.dev",
	}, " = ", true); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "api_key", readFlatKey(path, "api_key", " = "), "vl_live_test")
	assertEqual(t, "api_base_url", readFlatKey(path, "api_base_url", " = "), "https://api.veil.dev")
}

func TestUpdateFlatFile_TOML_UpdateExisting(t *testing.T) {
	initial := "api_key = \"oldkey\"\nother = \"preserved\"\n"
	path := tmpFile(t, initial)
	if err := updateFlatFile(path, map[string]string{"api_key": "newkey"}, " = ", true); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "api_key updated", readFlatKey(path, "api_key", " = "), "newkey")
	assertEqual(t, "other preserved", readFlatKey(path, "other", " = "), "preserved")
}

// --- updateFlatFile tests (YAML) ---

func TestUpdateFlatFile_YAML_New(t *testing.T) {
	path := tmpFile(t, "")
	if err := updateFlatFile(path, map[string]string{
		"openai-api-key":  "vl_live_test",
		"openai-api-base": "https://api.veil.dev",
	}, ": ", false); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "openai-api-key", readFlatKey(path, "openai-api-key", ": "), "vl_live_test")
	assertEqual(t, "openai-api-base", readFlatKey(path, "openai-api-base", ": "), "https://api.veil.dev")
}

func TestUpdateFlatFile_YAML_UpdateExisting(t *testing.T) {
	initial := "openai-api-key: oldkey\nsome-other: preserved\n"
	path := tmpFile(t, initial)
	if err := updateFlatFile(path, map[string]string{"openai-api-key": "newkey"}, ": ", false); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "openai-api-key updated", readFlatKey(path, "openai-api-key", ": "), "newkey")
	assertEqual(t, "some-other preserved", readFlatKey(path, "some-other", ": "), "preserved")
}

// --- configurator integration tests (filesystem) ---

func TestClaudeConfigurator_Configure(t *testing.T) {
	dir := t.TempDir()
	c := &baseConfigurator{
		name:    "Claude Code",
		slug:    "claude",
		binary:  "claude",
		envKey:  "ANTHROPIC_API_KEY",
		testCmd: `claude "hello"`,
		pathFn:  func() (string, error) { return filepath.Join(dir, "settings.json"), nil },
		readKey: func(path string) string { return readJSONKey(path, "apiKey") },
		writeFn: func(path, apiURL, apiKey string) error {
			return mergeJSON(path, map[string]any{"apiBaseUrl": apiURL, "apiKey": apiKey})
		},
	}
	if err := c.Configure("https://api.veil.dev", "vl_live_test"); err != nil {
		t.Fatal(err)
	}
	got := readJSONKey(filepath.Join(dir, "settings.json"), "apiKey")
	assertEqual(t, "apiKey", got, "vl_live_test")
}

func TestCursorConfigurator_Configure_Nested(t *testing.T) {
	dir := t.TempDir()
	c := &baseConfigurator{
		name:   "Cursor",
		slug:   "cursor",
		binary: "cursor",
		envKey: "OPENAI_API_KEY",
		pathFn: func() (string, error) { return filepath.Join(dir, "settings.json"), nil },
		readKey: func(path string) string { return readJSONKey(path, "openai.apiKey") },
		writeFn: func(path, apiURL, apiKey string) error {
			return mergeJSON(path, map[string]any{"openai.apiBase": apiURL, "openai.apiKey": apiKey})
		},
	}
	if err := c.Configure("https://api.veil.dev", "vl_live_cursor"); err != nil {
		t.Fatal(err)
	}
	got := readJSONKey(filepath.Join(dir, "settings.json"), "openai.apiKey")
	assertEqual(t, "nested apiKey", got, "vl_live_cursor")
	got = readJSONKey(filepath.Join(dir, "settings.json"), "openai.apiBase")
	assertEqual(t, "nested apiBase", got, "https://api.veil.dev")
}

func TestConfigurator_Backup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"existing": true}`), 0600); err != nil {
		t.Fatal(err)
	}
	c := &baseConfigurator{
		name:    "Claude Code",
		pathFn:  func() (string, error) { return path, nil },
		writeFn: func(path, apiURL, apiKey string) error { return mergeJSON(path, map[string]any{"apiKey": apiKey}) },
	}
	if err := c.Configure("url", "newkey"); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(path + ".veil.bak")
	if err != nil {
		t.Fatal("backup not created:", err)
	}
	if string(bak) != `{"existing": true}` {
		t.Errorf("unexpected backup content: %s", bak)
	}
}

// --- helpers ---

func tmpFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "veil-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	return f.Name()
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON in %s: %v", path, err)
	}
	return m
}

func assertEqual(t *testing.T, label string, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", label, got, want)
	}
}
