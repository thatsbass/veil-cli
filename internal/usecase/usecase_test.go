package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/thatsbass/veil-cli/internal/adapter/api"
	"github.com/thatsbass/veil-cli/internal/domain"
	"github.com/thatsbass/veil-cli/internal/ports"
)

// Compile-time check: *api.Client satisfies ports.GatewayClient.
var _ ports.GatewayClient = (*api.Client)(nil)

// mockClient implements ports.GatewayClient for tests.
type mockClient struct {
	statusFn      func(context.Context) (*domain.Status, error)
	statsFn       func(context.Context) (*domain.Stats, error)
	billingPlanFn func(context.Context) (*domain.BillingPlan, error)
}

func (m *mockClient) GetStatus(ctx context.Context) (*domain.Status, error) {
	return m.statusFn(ctx)
}
func (m *mockClient) GetStats(ctx context.Context) (*domain.Stats, error) {
	return m.statsFn(ctx)
}
func (m *mockClient) GetBillingPlan(ctx context.Context) (*domain.BillingPlan, error) {
	return m.billingPlanFn(ctx)
}
func (m *mockClient) GetLogs(_ context.Context, _ chan<- string) error { return nil }
func (m *mockClient) InitiateDeviceAuth(_ context.Context) (*api.DeviceAuthResponse, error) {
	return nil, nil
}
func (m *mockClient) PollDeviceToken(_ context.Context, _ string) (*api.TokenResponse, error) {
	return nil, nil
}

func TestCheckHealth_Success(t *testing.T) {
	client := &mockClient{
		statusFn: func(_ context.Context) (*api.Status, error) {
			return &api.Status{Status: "ok"}, nil
		},
	}
	result, err := CheckHealth(context.Background(), client, "https://api.veil.dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status: got %q, want %q", result.Status, "ok")
	}
	if result.APIURL != "https://api.veil.dev" {
		t.Errorf("apiURL: got %q, want %q", result.APIURL, "https://api.veil.dev")
	}
}

func TestCheckHealth_Unreachable(t *testing.T) {
	client := &mockClient{
		statusFn: func(_ context.Context) (*api.Status, error) {
			return nil, errors.New("connection refused")
		},
	}
	_, err := CheckHealth(context.Background(), client, "https://bad.url")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDashboard_Success(t *testing.T) {
	client := &mockClient{
		statusFn: func(_ context.Context) (*api.Status, error) {
			return &api.Status{Status: "ok"}, nil
		},
		statsFn: func(_ context.Context) (*api.Stats, error) {
			return &api.Stats{
				UsedTokens:  500_000,
				QuotaTokens: 1_000_000,
				Percent:     50,
				ResetsAt:    "2026-06-01",
			}, nil
		},
	}
	result, err := GetDashboard(context.Background(), client, "https://api.veil.dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status: got %q, want %q", result.Status, "ok")
	}
	if result.UsedTokens != 500_000 {
		t.Errorf("usedTokens: got %d, want 500000", result.UsedTokens)
	}
	if result.Percent != 50 {
		t.Errorf("percent: got %d, want 50", result.Percent)
	}
}

func TestGetDashboard_StatsUnavailable(t *testing.T) {
	client := &mockClient{
		statusFn: func(_ context.Context) (*api.Status, error) {
			return &api.Status{Status: "ok"}, nil
		},
		statsFn: func(_ context.Context) (*api.Stats, error) {
			return nil, errors.New("unavailable")
		},
	}
	result, err := GetDashboard(context.Background(), client, "https://api.veil.dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UsedTokens != 0 {
		t.Error("expected zero used tokens when stats unavailable")
	}
}

func TestGetBillingOverview_Success(t *testing.T) {
	client := &mockClient{
		billingPlanFn: func(_ context.Context) (*api.BillingPlan, error) {
			return &api.BillingPlan{
				ID:         "pro",
				PriceUSD:   29.0,
				TokenQuota: 10_000_000,
			}, nil
		},
		statsFn: func(_ context.Context) (*api.Stats, error) {
			return &api.Stats{
				UsedTokens: 3_000_000,
				Percent:    30,
				ResetsAt:   "2026-06-01",
			}, nil
		},
	}
	result, err := GetBillingOverview(context.Background(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PlanID != "pro" {
		t.Errorf("planID: got %q, want %q", result.PlanID, "pro")
	}
	if result.PriceUSD != 29.0 {
		t.Errorf("priceUSD: got %f, want 29.0", result.PriceUSD)
	}
	if result.Stats.Percent != 30 {
		t.Errorf("stats.percent: got %d, want 30", result.Stats.Percent)
	}
}

func TestGetBillingOverview_PlanError(t *testing.T) {
	client := &mockClient{
		billingPlanFn: func(_ context.Context) (*api.BillingPlan, error) {
			return nil, errors.New("not found")
		},
	}
	_, err := GetBillingOverview(context.Background(), client)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
