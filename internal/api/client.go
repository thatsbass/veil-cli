package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client talks to the Veil API server.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Status is the response from GET /health.
type Status struct {
	Status string `json:"status"`
}

// Stats is the response from GET /api/usage.
type Stats struct {
	UsedTokens  int64  `json:"used_tokens"`
	QuotaTokens int64  `json:"quota_tokens"`
	Percent     int    `json:"percent"`
	ResetsAt    string `json:"resets_at"`
}

func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	var s Status
	if err := c.get(ctx, "/health", &s); err != nil {
		return nil, fmt.Errorf("api.GetStatus: %w", err)
	}
	return &s, nil
}

func (c *Client) GetStats(ctx context.Context) (*Stats, error) {
	var s Stats
	if err := c.get(ctx, "/api/usage", &s); err != nil {
		return nil, fmt.Errorf("api.GetStats: %w", err)
	}
	return &s, nil
}

// GetLogs streams server-sent events into the events channel until ctx is cancelled.
func (c *Client) GetLogs(ctx context.Context, events chan<- string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/logs", nil)
	if err != nil {
		return fmt.Errorf("api.GetLogs: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
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

// DeviceAuthResponse is returned by POST /auth/device.
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse is returned by GET /auth/device/token.
type TokenResponse struct {
	Status string `json:"status"`
	APIKey string `json:"api_key,omitempty"`
}

// InitiateDeviceAuth starts a new device authorization session.
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

// PollDeviceToken polls for the API key after the user approves the CLI session.
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
