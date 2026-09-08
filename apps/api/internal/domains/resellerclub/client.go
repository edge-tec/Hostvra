package resellerclub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hostvra/api/internal/domains"
)

const (
	DefaultSandboxBaseURL    = "https://test.httpapi.com/api/"
	DefaultProductionBaseURL = "https://httpapi.com/api/"
	DefaultTimeout           = 30 * time.Second
)

type Config struct {
	ResellerID string        `json:"reseller_id"`
	APIKey     string        `json:"api_key"`
	Mode       string        `json:"mode"` // "sandbox" or "production"
	BaseURL    string        `json:"base_url"`
	Timeout    time.Duration `json:"timeout"`
}

type Client struct {
	cfg        Config
	httpClient *http.Client
	baseURL    string
}

var _ domains.DomainRegistrar = (*Client)(nil)

func NewClient(cfg Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = "sandbox"
	}
	cfg.Mode = mode

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		if mode == "production" {
			baseURL = DefaultProductionBaseURL
		} else {
			baseURL = DefaultSandboxBaseURL
		}
	} else {
		// Strict Endpoint Isolation Validation (Section 21)
		if mode == "production" && strings.Contains(baseURL, "test.httpapi.com") {
			return nil, fmt.Errorf("endpoint conflict: production mode cannot target sandbox URL (%s)", baseURL)
		}
		if mode == "sandbox" && strings.Contains(baseURL, "httpapi.com") && !strings.Contains(baseURL, "test.httpapi.com") {
			return nil, fmt.Errorf("endpoint conflict: sandbox mode cannot target production URL (%s)", baseURL)
		}
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &Client{
		cfg:     cfg,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// Get executes a GET request to the ResellerClub API with credentials attached
func (c *Client) Get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	// Append authentication credentials
	params.Set("auth-userid", c.cfg.ResellerID)
	params.Set("api-key", c.cfg.APIKey)

	cleanEndpoint := strings.TrimPrefix(endpoint, "/")
	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, cleanEndpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("User-Agent", "Hostvra-DomainRegistrar/1.0 (LogicBoxes-API-Client; +https://hostvra.com)")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	slog.Debug("Executing ResellerClub GET request",
		"endpoint", cleanEndpoint,
		"mode", c.cfg.Mode,
		"reseller_id", c.cfg.ResellerID,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrRegistrarTimeout
		}
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseAPIError(resp.StatusCode, body)
	}

	// LogicBoxes API often returns HTTP 200 with {"status": "ERROR", "message": "..."}
	var lbErr LogicBoxesErrorResponse
	if err := json.Unmarshal(body, &lbErr); err == nil {
		if strings.EqualFold(lbErr.Status, "ERROR") || strings.EqualFold(lbErr.Status, "Failed") {
			return nil, ParseAPIError(resp.StatusCode, body)
		}
	}

	return body, nil
}

// Post executes a POST request with form-url-encoded body and credentials attached
func (c *Client) Post(ctx context.Context, endpoint string, data url.Values) ([]byte, error) {
	if data == nil {
		data = url.Values{}
	}
	// Append authentication credentials
	data.Set("auth-userid", c.cfg.ResellerID)
	data.Set("api-key", c.cfg.APIKey)

	cleanEndpoint := strings.TrimPrefix(endpoint, "/")
	reqURL := fmt.Sprintf("%s%s", c.baseURL, cleanEndpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Hostvra-DomainRegistrar/1.0 (LogicBoxes-API-Client; +https://hostvra.com)")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	slog.Debug("Executing ResellerClub POST request",
		"endpoint", cleanEndpoint,
		"mode", c.cfg.Mode,
		"reseller_id", c.cfg.ResellerID,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, ErrRegistrarTimeout
		}
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseAPIError(resp.StatusCode, body)
	}

	// LogicBoxes API often returns HTTP 200 with {"status": "ERROR", "message": "..."}
	var lbPostErr LogicBoxesErrorResponse
	if err := json.Unmarshal(body, &lbPostErr); err == nil {
		if strings.EqualFold(lbPostErr.Status, "ERROR") || strings.EqualFold(lbPostErr.Status, "Failed") {
			return nil, ParseAPIError(resp.StatusCode, body)
		}
	}

	return body, nil
}

func (c *Client) GetMode() string {
	return c.cfg.Mode
}

func (c *Client) GetResellerID() string {
	return c.cfg.ResellerID
}

func (c *Client) GetBaseURL() string {
	return c.baseURL
}
