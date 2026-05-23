package configurator

// NewCodex returns a configurator for Codex CLI. It writes api_base_url and
// api_key pairs into ~/.codex/config.toml.
func NewCodex() ToolConfigurator {
	return &baseConfigurator{
		name:    "Codex CLI",
		slug:    "codex",
		binary:  "codex",
		envKey:  "OPENAI_API_KEY",
		testCmd: "codex --version",
		pathFn:  func() (string, error) { return homePath(".codex/config.toml") },
		readKey: func(path string) string { return readFlatKey(path, "api_key", " = ") },
		writeFn: func(path, apiURL, apiKey string) error {
			return updateFlatFile(path, map[string]string{
				"api_base_url": apiURL,
				"api_key":      apiKey,
			}, " = ", true)
		},
	}
}
