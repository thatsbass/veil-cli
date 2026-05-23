package config

import "github.com/thatsbass/veil-cli/internal/domain"

const defaultAPIURL = "https://api.veil.dev"

// Config is a type alias for the canonical domain.Config.
type Config = domain.Config

func defaults() *Config {
	return &Config{APIURL: defaultAPIURL}
}
