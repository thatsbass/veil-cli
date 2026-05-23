package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thatsbass/veil-cli/internal/domain"
)

// Client is an HTTP client for the Veil API server.
// It maintains two underlying HTTP clients: one with a 10 s timeout for
// regular requests, and one without a timeout for long-lived SSE streams.
type Client struct {
	baseURL string
	apiKey  string

	httpClient   *http.Client
	streamClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:      baseURL,
		apiKey:       apiKey,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		streamClient: &http.Client{},
	}
}

// All response DTOs are defined in the domain package; re-exported here for
// backward compatibility with code that imports this adapter directly.
type (
	Status             = domain.Status
	Stats              = domain.Stats
	BillingPlan        = domain.BillingPlan
	LogEvent           = domain.LogEvent
	DeviceAuthResponse = domain.DeviceAuthResponse
	TokenResponse      = domain.TokenResponse
)

func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	var s Status
	if err := c.get(ctx, "/health", &s); err != nil {
		return nil, fmt.Errorf("api.GetStatus: %w", err)
	}
	return &s, nil
}

func (c *Client) GetStats(ctx context.Context) (*Stats, error) {
	var s Stats
	if err := c.get(ctx, "/v1/usage", &s); err != nil {
		return nil, fmt.Errorf("api.GetStats: %w", err)
	}
	return &s, nil
}

func (c *Client) GetBillingPlan(ctx context.Context) (*BillingPlan, error) {
	var p BillingPlan
	if err := c.get(ctx, "/v1/billing/plan", &p); err != nil {
		return nil, fmt.Errorf("api.GetBillingPlan: %w", err)
	}
	return &p, nil
}

// GetLogs opens a long-lived SSE connection and streams raw event lines into
// the events channel. The stream runs until ctx is cancelled. Callers must
// provide a buffered channel to avoid blocking the underlying HTTP reader.
func (c *Client) GetLogs(ctx context.Context, events chan<- string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/logs", nil)
	if err != nil {
		return fmt.Errorf("api.GetLogs: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("api.GetLogs: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			events <- strings.TrimPrefix(line, "data: ")
		}
	}
	return scanner.Err()
}

// InitiateDeviceAuth requests a new device authorization session and returns
// the verification code the user must enter in their browser.
func (c *Client) InitiateDeviceAuth(ctx context.Context) (*DeviceAuthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/device", nil)
	if err != nil {
		return nil, fmt.Errorf("api.InitiateDeviceAuth: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api.InitiateDeviceAuth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api.InitiateDeviceAuth: server returned %d", resp.StatusCode)
	}
	var out DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("api.InitiateDeviceAuth: %w", err)
	}
	return &out, nil
}

// PollDeviceToken checks whether the user has completed the device
// authorization flow. It returns the API key once the user approves.
func (c *Client) PollDeviceToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	url := c.baseURL + "/auth/device/token?device_code=" + deviceCode
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("api.PollDeviceToken: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api.PollDeviceToken: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api.PollDeviceToken: server returned %d", resp.StatusCode)
	}
	var out TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("api.PollDeviceToken: %w", err)
	}
	return &out, nil
}

// FormatLogEvent parses a raw SSE data string and returns a human-readable line.
// Falls back to printing the raw string if JSON parsing fails.
func FormatLogEvent(raw string) string {
	var e LogEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return raw
	}
	ts := e.Timestamp
	if len(ts) > 19 {
		ts = ts[:19]
	}
	return fmt.Sprintf("%-19s  %-12s  %-10s  %4dms  %s",
		ts, e.Type, e.Provider, e.LatencyMS, e.Status)
}

// --- internal ---

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) setHeaders(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}
