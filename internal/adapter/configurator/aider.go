package configurator

// NewAider returns a configurator for Aider. It writes openai-api-base and
// openai-api-key pairs (bare YAML values) into ~/.aider.conf.yml.
func NewAider() ToolConfigurator {
	return &baseConfigurator{
		name:    "Aider",
		slug:    "aider",
		binary:  "aider",
		envKey:  "OPENAI_API_KEY",
		testCmd: "aider --version",
		pathFn:  func() (string, error) { return homePath(".aider.conf.yml") },
		readKey: func(path string) string { return readFlatKey(path, "openai-api-key", ": ") },
		writeFn: func(path, apiURL, apiKey string) error {
			return updateFlatFile(path, map[string]string{
				"openai-api-base": apiURL,
				"openai-api-key":  apiKey,
			}, ": ", false)
		},
	}
}
