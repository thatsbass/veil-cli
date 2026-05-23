package domain

// Status is the response body of GET /health.
type Status struct {
	Status string `json:"status"`
}

// Stats is the response body of GET /v1/usage.
type Stats struct {
	UsedTokens  int64  `json:"used_tokens"`
	QuotaTokens int64  `json:"quota_tokens"`
	Percent     int    `json:"percent"`
	ResetsAt    string `json:"resets_at"`
}

// BillingPlan is the response body of GET /v1/billing/plan.
type BillingPlan struct {
	ID         string  `json:"id"`
	PriceUSD   float64 `json:"price_usd"`
	TokenQuota int64   `json:"token_quota"`
	MaxAPIKeys int     `json:"max_api_keys"`
}

// LogEvent is a single SSE event received from GET /v1/logs.
type LogEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
	LatencyMS int    `json:"latency_ms"`
	Status    string `json:"status"`
}
