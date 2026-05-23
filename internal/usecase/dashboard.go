package usecase

import (
	"context"

	"github.com/thatsbass/veil-cli/internal/ports"
)

// DashboardResult bundles server status with monthly token usage for the
// `veil up` and `/status` commands.
type DashboardResult struct {
	APIURL      string
	Status      string
	UsedTokens  int64
	QuotaTokens int64
	Percent     int
	ResetsAt    string
}

// GetDashboard fetches server status and monthly usage in a single call.
// A failure to fetch stats is non-fatal: the returned result contains zero
// values for all usage fields.
func GetDashboard(ctx context.Context, client ports.GatewayClient, apiURL string) (*DashboardResult, error) {
	status, err := client.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	result := &DashboardResult{APIURL: apiURL, Status: status.Status}

	if stats, err := client.GetStats(ctx); err == nil {
		result.UsedTokens = stats.UsedTokens
		result.QuotaTokens = stats.QuotaTokens
		result.Percent = stats.Percent
		result.ResetsAt = stats.ResetsAt
	}
	return result, nil
}
