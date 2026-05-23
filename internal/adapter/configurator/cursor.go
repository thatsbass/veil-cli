package configurator

// NewCursor returns a configurator for the Cursor editor. It writes nested
// JSON keys ("openai.apiBase", "openai.apiKey") into ~/.cursor/settings.json.
func NewCursor() ToolConfigurator {
	return &baseConfigurator{
		name:    "Cursor",
		slug:    "cursor",
		binary:  "cursor",
		envKey:  "OPENAI_API_KEY",
		testCmd: "cursor .",
		pathFn:  func() (string, error) { return homePath(".cursor/settings.json") },
		readKey: func(path string) string { return readJSONKey(path, "openai.apiKey") },
		writeFn: func(path, apiURL, apiKey string) error {
			return mergeJSON(path, map[string]any{
				"openai.apiBase": apiURL,
				"openai.apiKey":  apiKey,
			})
		},
	}
}
