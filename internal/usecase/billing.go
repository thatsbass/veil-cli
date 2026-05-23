package usecase

import (
	"context"

	"github.com/thatsbass/veil-cli/internal/domain"
	"github.com/thatsbass/veil-cli/internal/ports"
)

// BillingResult packages the active plan with current usage for the
// `/billing` REPL card.
type BillingResult struct {
	PlanID     string
	PriceUSD   float64
	TokenQuota int64
	Stats      *domain.Stats
}

// GetBillingOverview fetches the billing plan and current usage from the
// Veil API. Both calls must succeed for a result to be returned.
func GetBillingOverview(ctx context.Context, client ports.GatewayClient) (*BillingResult, error) {
	plan, err := client.GetBillingPlan(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := client.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return &BillingResult{
		PlanID:     plan.ID,
		PriceUSD:   plan.PriceUSD,
		TokenQuota: plan.TokenQuota,
		Stats:      stats,
	}, nil
}
