// Package domain contains the core business types used across all layers.
// These types have no external dependencies and represent the ubiquitous
// language of the Veil domain.
package domain

// Config holds the Veil CLI settings.
type Config struct {
	APIKey string `json:"api_key"`
	APIURL string `json:"api_url"`
}
