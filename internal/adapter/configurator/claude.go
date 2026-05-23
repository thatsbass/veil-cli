package configurator

// NewClaude returns a configurator for Claude Code. It writes apiBaseUrl and
// apiKey into ~/.claude/settings.json.
func NewClaude() ToolConfigurator {
	return &baseConfigurator{
		name:    "Claude Code",
		slug:    "claude",
		binary:  "claude",
		envKey:  "ANTHROPIC_API_KEY",
		testCmd: `claude "hello"`,
		pathFn:  func() (string, error) { return homePath(".claude/settings.json") },
		readKey: func(path string) string { return readJSONKey(path, "apiKey") },
		writeFn: func(path, apiURL, apiKey string) error {
			return mergeJSON(path, map[string]any{
				"apiBaseUrl": apiURL,
				"apiKey":     apiKey,
			})
		},
	}
}
