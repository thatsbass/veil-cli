// Package ports defines the interfaces ("ports") the usecase layer depends on.
// Every adapter under internal/adapter implements one of these contracts.
//
// This package must never import any adapter or delivery package.
package ports

import (
	"context"

	"github.com/thatsbass/veil-cli/internal/adapter/config"
	"github.com/thatsbass/veil-cli/internal/domain"
)

// GatewayClient is the contract for communicating with the Veil API.
type GatewayClient interface {
	GetStatus(ctx context.Context) (*domain.Status, error)
	GetStats(ctx context.Context) (*domain.Stats, error)
	GetBillingPlan(ctx context.Context) (*domain.BillingPlan, error)
	GetLogs(ctx context.Context, events chan<- string) error
	InitiateDeviceAuth(ctx context.Context) (*domain.DeviceAuthResponse, error)
	PollDeviceToken(ctx context.Context, deviceCode string) (*domain.TokenResponse, error)
}

// ConfigRepository is the contract for loading and saving CLI configuration.
type ConfigRepository = config.Repository
