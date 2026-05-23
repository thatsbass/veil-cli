package usecase

import (
	"context"

	"github.com/thatsbass/veil-cli/internal/ports"
)

// HealthResult contains the outcome of a server health check.
type HealthResult struct {
	APIURL string
	Status string
}

// CheckHealth calls GET /health on the gateway and returns the server status.
// It returns an error if the server is unreachable.
func CheckHealth(ctx context.Context, client ports.GatewayClient, apiURL string) (*HealthResult, error) {
	status, err := client.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &HealthResult{APIURL: apiURL, Status: status.Status}, nil
}
