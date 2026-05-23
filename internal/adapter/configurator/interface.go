package configurator

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ToolConfigurator adapts a local AI tool (Claude Code, Cursor, etc.) to
// route API calls through the Veil gateway.
type ToolConfigurator interface {
	Name() string
	Slug() string
	Detect() bool
	ExistingKey() string
	Configure(apiURL, apiKey string) error
	Restore() error
	ConfigPath() (string, error)
	TestCmd() string
}

// All returns every supported tool configurator in display order.
func All() []ToolConfigurator {
	return []ToolConfigurator{NewClaude(), NewCodex(), NewCursor(), NewAider()}
}

// ── File manipulation helpers ──

func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("backup %s: %w", path, err)
	}
	return os.WriteFile(path+".veil.bak", data, 0600)
}

func restoreFile(path string) error {
	data, err := os.ReadFile(path + ".veil.bak")
	if err != nil {
		return fmt.Errorf("restore %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0600)
}

// mergeJSON reads a JSON object from path, applies the given updates, and writes
// the result back. Dotted keys are expanded into nested objects:
// "openai.apiBase" becomes {"openai": {"apiBase": "..."}}.
func mergeJSON(path string, updates map[string]any) error {
	m := map[string]any{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("mergeJSON read %s: %w", path, err)
	}
	if err == nil {
		_ = json.Unmarshal(data, &m) // best-effort; corrupt file → start fresh
	}
	for k, v := range updates {
		setNestedValue(m, k, v)
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("mergeJSON marshal: %w", err)
	}
	return os.WriteFile(path, out, 0600)
}

// setNestedValue writes value into m at the given dotted key path.
func setNestedValue(m map[string]any, key string, value any) {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		m[key] = value
		return
	}
	parent, child := key[:idx], key[idx+1:]
	sub, _ := m[parent].(map[string]any)
	if sub == nil {
		sub = map[string]any{}
	}
	setNestedValue(sub, child, value)
	m[parent] = sub
}

// readJSONKey reads a single string value at a dotted key path from a JSON file.
// It returns "" if the file is missing, malformed, or the key does not exist.
func readJSONKey(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	return getNestedValue(m, key)
}

// getNestedValue reads a string value at a dotted key path from m.
func getNestedValue(m map[string]any, key string) string {
	idx := strings.IndexByte(key, '.')
	if idx < 0 {
		v, _ := m[key].(string)
		return v
	}
	parent, child := key[:idx], key[idx+1:]
	sub, ok := m[parent].(map[string]any)
	if !ok {
		return ""
	}
	return getNestedValue(sub, child)
}

// updateFlatFile updates key=value pairs in a flat config file (TOML or YAML).
// Existing lines are updated in place; missing keys are appended. The sep
// parameter controls the format: " = " for TOML (quoted values), ": " for YAML.
func updateFlatFile(path string, updates map[string]string, sep string, quoted bool) error {
	content := ""
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("updateFlatFile read %s: %w", path, err)
	}
	if err == nil {
		content = string(data)
	}

	// Pre-compile one regex per key to avoid recompilation in the inner loop.
	type keyPattern struct {
		key     string
		pattern *regexp.Regexp
	}
	patterns := make([]keyPattern, 0, len(updates))
	for k := range updates {
		re, err := regexp.Compile(`(?i)^` + regexp.QuoteMeta(k) + `\s*` + regexp.QuoteMeta(strings.TrimSpace(sep)))
		if err != nil {
			return fmt.Errorf("updateFlatFile compile pattern for %q: %w", k, err)
		}
		patterns = append(patterns, keyPattern{key: k, pattern: re})
	}

	remaining := make(map[string]string, len(updates))
	for k, v := range updates {
		remaining[k] = v
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		for _, kp := range patterns {
			if _, done := remaining[kp.key]; !done {
				continue
			}
			if kp.pattern.MatchString(line) {
				lines[i] = formatFlatLine(kp.key, remaining[kp.key], sep, quoted)
				delete(remaining, kp.key)
				break
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(lines, "\n"))
	if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
		sb.WriteString("\n")
	}
	for k, v := range remaining {
		sb.WriteString(formatFlatLine(k, v, sep, quoted))
		sb.WriteString("\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

// readFlatKey reads a single value from a flat config file.
// (?m) makes ^ match the start of each line, not just the start of the string.
func readFlatKey(path, key, sep string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	pattern := regexp.MustCompile(`(?im)^` + regexp.QuoteMeta(key) + `\s*` +
		regexp.QuoteMeta(strings.TrimSpace(sep)) + `\s*["']?([^"'\r\n]+)["']?`)
	m := pattern.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func formatFlatLine(key, value, sep string, quoted bool) string {
	if quoted {
		return key + sep + `"` + value + `"`
	}
	return key + sep + value
}
