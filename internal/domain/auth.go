package domain

// DeviceAuthResponse is the response body of POST /auth/device.
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse is the response body of GET /auth/device/token.
type TokenResponse struct {
	Status string `json:"status"`
	APIKey string `json:"api_key,omitempty"`
}
