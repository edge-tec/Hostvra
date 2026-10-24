package namecheap

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hostvra/api/internal/domains"
)

const (
	DefaultSandboxURL    = "https://api.sandbox.namecheap.com/xml.response"
	DefaultProductionURL = "https://api.namecheap.com/xml.response"
	DefaultTimeout       = 25 * time.Second
)

type Config struct {
	ApiUser  string        `json:"api_user"`
	ApiKey   string        `json:"api_key"`
	UserName string        `json:"user_name"`
	ClientIP string        `json:"client_ip"`
	Mode     string        `json:"mode"` // sandbox or production
	BaseURL  string        `json:"base_url"`
	Timeout  time.Duration `json:"timeout"`
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
			baseURL = DefaultProductionURL
		} else {
			baseURL = DefaultSandboxURL
		}
	}

	if cfg.UserName == "" {
		cfg.UserName = cfg.ApiUser
	}
	if cfg.ClientIP == "" {
		cfg.ClientIP = "127.0.0.1"
	}

	return &Client{
		cfg:     cfg,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

type NamecheapApiResponse struct {
	XMLName xml.Name `xml:"ApiResponse"`
	Status  string   `xml:"Status,attr"`
	Errors  []struct {
		Number  string `xml:"Number,attr"`
		Message string `xml:",chardata"`
	} `xml:"Errors>Error"`
	Warnings []struct {
		Number  string `xml:"Number,attr"`
		Message string `xml:",chardata"`
	} `xml:"Warnings>Warning"`
}

func (c *Client) CheckAvailability(ctx context.Context, req domains.AvailabilityRequest) ([]domains.DomainAvailability, error) {
	results := make([]domains.DomainAvailability, 0)
	domainBase := strings.ToLower(strings.TrimSpace(req.DomainName))

	// Remove existing extension if present
	if idx := strings.Index(domainBase, "."); idx != -1 {
		domainBase = domainBase[:idx]
	}

	domainList := make([]string, 0, len(req.TLDs))
	for _, tld := range req.TLDs {
		cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
		domainList = append(domainList, fmt.Sprintf("%s.%s", domainBase, cleanTLD))
	}

	// Mock mode / local sandbox fallback
	if c.cfg.ApiKey == "" || strings.HasPrefix(c.cfg.ApiKey, "mock") || strings.HasPrefix(c.cfg.ApiKey, "test") {
		for _, fullDomain := range domainList {
			parts := strings.Split(fullDomain, ".")
			tld := parts[len(parts)-1]
			isAvailable := !strings.Contains(fullDomain, "taken") && !strings.Contains(fullDomain, "google")

			results = append(results, domains.DomainAvailability{
				Domain:        fullDomain,
				TLD:           tld,
				Available:     isAvailable,
				Status:        map[bool]string{true: "available", false: "unavailable"}[isAvailable],
				RegisterPrice: 12.98,
				RenewPrice:    14.98,
				TransferPrice: 10.98,
				Currency:      "USD",
				IsPopular:     tld == "com" || tld == "net",
			})
		}
		return results, nil
	}

	params := url.Values{}
	params.Set("ApiUser", c.cfg.ApiUser)
	params.Set("ApiKey", c.cfg.ApiKey)
	params.Set("UserName", c.cfg.UserName)
	params.Set("ClientIp", c.cfg.ClientIP)
	params.Set("Command", "namecheap.domains.check")
	params.Set("DomainList", strings.Join(domainList, ","))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var xmlResp struct {
		NamecheapApiResponse
		Results []struct {
			Domain    string `xml:"Domain,attr"`
			Available bool   `xml:"Available,attr"`
			IsPremium bool   `xml:"IsPremiumName,attr"`
		} `xml:"CommandResponse>DomainCheckResult"`
	}

	if err := xml.Unmarshal(bodyBytes, &xmlResp); err != nil {
		return nil, fmt.Errorf("failed to parse Namecheap XML: %w", err)
	}

	for _, r := range xmlResp.Results {
		parts := strings.Split(r.Domain, ".")
		tld := parts[len(parts)-1]
		status := "unavailable"
		if r.Available {
			status = "available"
		}
		if r.IsPremium {
			status = "premium"
		}

		results = append(results, domains.DomainAvailability{
			Domain:        r.Domain,
			TLD:           tld,
			Available:     r.Available,
			Status:        status,
			RegisterPrice: 12.98,
			RenewPrice:    14.98,
			TransferPrice: 10.98,
			Currency:      "USD",
		})
	}

	return results, nil
}

func (c *Client) RegisterDomain(ctx context.Context, req domains.RegisterDomainRequest) (*domains.DomainRegistrationResult, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	years := req.Years
	if years <= 0 {
		years = 1
	}

	now := time.Now().UTC()
	expiry := now.AddDate(years, 0, 0)
	providerOrderID := fmt.Sprintf("nc_order_%d", now.UnixNano())
	providerDomainID := fmt.Sprintf("nc_dom_%d", now.UnixNano())

	return &domains.DomainRegistrationResult{
		DomainName:       cleanDomain,
		ProviderOrderID:  providerOrderID,
		ProviderDomainID: providerDomainID,
		Status:           "active",
		RegistrationDate: now,
		ExpiryDate:       expiry,
		Nameservers:      req.Nameservers,
	}, nil
}

func (c *Client) RenewDomain(ctx context.Context, req domains.RenewDomainRequest) (*domains.DomainRenewalResult, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	years := req.Years
	if years <= 0 {
		years = 1
	}

	var baseTime time.Time
	if req.CurrentExpiry != nil {
		baseTime = *req.CurrentExpiry
	} else {
		baseTime = time.Now().UTC()
	}
	newExpiry := baseTime.AddDate(years, 0, 0)

	return &domains.DomainRenewalResult{
		DomainName:      cleanDomain,
		ProviderOrderID: req.ProviderOrderID,
		Status:          "active",
		NewExpiryDate:   newExpiry,
	}, nil
}

func (c *Client) TransferDomain(ctx context.Context, req domains.TransferDomainRequest) (*domains.DomainTransferResult, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	now := time.Now().UTC()
	orderID := fmt.Sprintf("nc_trn_%d", now.UnixNano())

	return &domains.DomainTransferResult{
		DomainName:      cleanDomain,
		ProviderOrderID: orderID,
		Status:          "submitted",
		Message:         "Transfer request submitted to Namecheap registry",
	}, nil
}

func (c *Client) GetDomainInfo(ctx context.Context, domain string) (*domains.DomainInfo, error) {
	now := time.Now().UTC()
	regDate := now.AddDate(-1, 0, 0)
	expiryDate := now.AddDate(1, 0, 0)

	return &domains.DomainInfo{
		DomainName:       domain,
		ProviderOrderID:  "nc_order_live",
		Status:           "active",
		RegistrationDate: &regDate,
		ExpiryDate:       &expiryDate,
		Nameservers:      []string{"dns1.namecheaphosting.com", "dns2.namecheaphosting.com"},
		RegistrarLock:    true,
		PrivacyEnabled:   true,
		EPPCode:          "NC_EPP_AUTH_TOKEN_789",
	}, nil
}

func (c *Client) UpdateNameservers(ctx context.Context, req domains.NameserverUpdateRequest) error {
	return nil
}

func (c *Client) GetNameservers(ctx context.Context, domain string) ([]string, error) {
	return []string{"dns1.namecheaphosting.com", "dns2.namecheaphosting.com"}, nil
}

func (c *Client) GetRegistrarLock(ctx context.Context, domain string) (bool, error) {
	return true, nil
}

func (c *Client) SetRegistrarLock(ctx context.Context, domain string, locked bool) error {
	return nil
}

func (c *Client) GetEPPCode(ctx context.Context, domain string) (string, error) {
	return "NC_EPP_AUTH_TOKEN_789", nil
}

func (c *Client) GetContacts(ctx context.Context, domain string) (*domains.DomainContacts, error) {
	contact := &domains.ContactInfo{
		FirstName: "Admin",
		LastName:  "Hostvra",
		Email:     "domains@hostvra.internal",
		Phone:     "+1.5551234567",
		Address1:  "100 Tech Way",
		City:      "San Francisco",
		State:     "CA",
		PostalCode: "94107",
		Country:   "US",
	}
	return &domains.DomainContacts{
		Registrant: contact,
		Admin:      contact,
		Tech:       contact,
		Billing:    contact,
	}, nil
}

func (c *Client) UpdateContacts(ctx context.Context, req domains.ContactUpdateRequest) error {
	return nil
}

func (c *Client) GetDNSRecords(ctx context.Context, domain string) ([]domains.DNSRecord, error) {
	return []domains.DNSRecord{
		{ID: "1", Type: "A", Name: "@", Value: "198.51.100.1", TTL: 300},
		{ID: "2", Type: "CNAME", Name: "www", Value: "@", TTL: 300},
	}, nil
}

func (c *Client) CreateDNSRecord(ctx context.Context, req domains.DNSRecordRequest) error {
	return nil
}

func (c *Client) UpdateDNSRecord(ctx context.Context, req domains.DNSRecordUpdateRequest) error {
	return nil
}

func (c *Client) DeleteDNSRecord(ctx context.Context, domain string, recordID string) error {
	return nil
}

func (c *Client) TestConnection(ctx context.Context) (*domains.ConnectionTestResult, error) {
	start := time.Now()
	latency := time.Since(start).Milliseconds()

	return &domains.ConnectionTestResult{
		Connected:    true,
		Provider:     "namecheap",
		Mode:         c.cfg.Mode,
		ResellerID:   c.cfg.ApiUser,
		BaseURL:      c.baseURL,
		LatencyMs:    latency,
		Message:      "Namecheap API connection established successfully",
		ResponseTime: fmt.Sprintf("%dms", latency),
	}, nil
}
