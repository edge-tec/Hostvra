package ssl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DNSProvider abstracts DNS record creation and deletion for DNS-01 challenges
type DNSProvider interface {
	CreateTXTRecord(ctx context.Context, domain, fqdn, value string) error
	DeleteTXTRecord(ctx context.Context, domain, fqdn, value string) error
}

// ============================================================================
// CLOUDFLARE DNS PROVIDER
// ============================================================================

type CloudflareProvider struct {
	APIToken   string
	ZoneID     string
	BaseURL    string
	HTTPClient *http.Client
	mu         sync.Mutex
	recordIDs  map[string]string // maps fqdn+value -> record_id
}

func NewCloudflareProvider(apiToken, zoneID string) *CloudflareProvider {
	return &CloudflareProvider{
		APIToken:   apiToken,
		ZoneID:     zoneID,
		BaseURL:    "https://api.cloudflare.com/client/v4",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		recordIDs:  make(map[string]string),
	}
}

type cfZoneResponse struct {
	Success bool `json:"success"`
	Result  []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type cfRecordResponse struct {
	Success bool `json:"success"`
	Result  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (cf *CloudflareProvider) resolveZoneID(ctx context.Context, domain string) (string, error) {
	if cf.ZoneID != "" {
		return cf.ZoneID, nil
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid domain name: %s", domain)
	}
	rootDomain := strings.Join(parts[len(parts)-2:], ".")

	url := fmt.Sprintf("%s/zones?name=%s", cf.BaseURL, rootDomain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cf.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cf.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloudflare api error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var zResp cfZoneResponse
	if err := json.Unmarshal(body, &zResp); err != nil {
		return "", fmt.Errorf("failed to parse cloudflare response: %w", err)
	}

	if !zResp.Success || len(zResp.Result) == 0 {
		return "", fmt.Errorf("cloudflare zone not found for %s", rootDomain)
	}

	cf.ZoneID = zResp.Result[0].ID
	return cf.ZoneID, nil
}

func (cf *CloudflareProvider) CreateTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	zoneID, err := cf.resolveZoneID(ctx, domain)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"type":    "TXT",
		"name":    fqdn,
		"content": value,
		"ttl":     120,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", cf.BaseURL, zoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cf.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cf.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create txt record on cloudflare: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var recResp cfRecordResponse
	if err := json.Unmarshal(body, &recResp); err != nil {
		return fmt.Errorf("invalid json response from cloudflare: %w", err)
	}

	if !recResp.Success {
		var errMsgs []string
		for _, e := range recResp.Errors {
			errMsgs = append(errMsgs, e.Message)
		}
		return fmt.Errorf("cloudflare error creating txt record: %s", strings.Join(errMsgs, ", "))
	}

	cf.mu.Lock()
	cf.recordIDs[fqdn+":"+value] = recResp.Result.ID
	cf.mu.Unlock()

	return nil
}

func (cf *CloudflareProvider) DeleteTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	cf.mu.Lock()
	recordID, ok := cf.recordIDs[fqdn+":"+value]
	cf.mu.Unlock()

	if !ok || recordID == "" {
		return nil // nothing to delete or already cleaned up
	}

	zoneID, err := cf.resolveZoneID(ctx, domain)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cf.BaseURL, zoneID, recordID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cf.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cf.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete txt record from cloudflare: %w", err)
	}
	defer resp.Body.Close()

	cf.mu.Lock()
	delete(cf.recordIDs, fqdn+":"+value)
	cf.mu.Unlock()

	return nil
}

// ============================================================================
// DIGITALOCEAN DNS PROVIDER
// ============================================================================

type DigitalOceanProvider struct {
	APIToken   string
	BaseURL    string
	HTTPClient *http.Client
	mu         sync.Mutex
	recordIDs  map[string]int
}

func NewDigitalOceanProvider(apiToken string) *DigitalOceanProvider {
	return &DigitalOceanProvider{
		APIToken:   apiToken,
		BaseURL:    "https://api.digitalocean.com/v2",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		recordIDs:  make(map[string]int),
	}
}

type doRecordResponse struct {
	DomainRecord struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
		Name string `json:"name"`
		Data string `json:"data"`
	} `json:"domain_record"`
	Message string `json:"message"`
}

func (do *DigitalOceanProvider) CreateTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid domain: %s", domain)
	}
	rootDomain := strings.Join(parts[len(parts)-2:], ".")

	subdomain := strings.TrimSuffix(fqdn, "."+rootDomain)
	if subdomain == fqdn {
		subdomain = "@"
	}

	payload := map[string]interface{}{
		"type": "TXT",
		"name": subdomain,
		"data": value,
		"ttl":  120,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/domains/%s/records", do.BaseURL, rootDomain)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+do.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := do.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("digitalocean api error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("digitalocean error (%d): %s", resp.StatusCode, string(body))
	}

	var recResp doRecordResponse
	if err := json.Unmarshal(body, &recResp); err != nil {
		return fmt.Errorf("failed to parse digitalocean response: %w", err)
	}

	do.mu.Lock()
	do.recordIDs[fqdn+":"+value] = recResp.DomainRecord.ID
	do.mu.Unlock()

	return nil
}

func (do *DigitalOceanProvider) DeleteTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	do.mu.Lock()
	recordID, ok := do.recordIDs[fqdn+":"+value]
	do.mu.Unlock()

	if !ok || recordID == 0 {
		return nil
	}

	parts := strings.Split(domain, ".")
	rootDomain := strings.Join(parts[len(parts)-2:], ".")

	url := fmt.Sprintf("%s/domains/%s/records/%d", do.BaseURL, rootDomain, recordID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+do.APIToken)

	resp, err := do.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete record from digitalocean: %w", err)
	}
	defer resp.Body.Close()

	do.mu.Lock()
	delete(do.recordIDs, fqdn+":"+value)
	do.mu.Unlock()

	return nil
}

// ============================================================================
// LOCAL BIND9 / DNS FILE PROVIDER
// ============================================================================

type LocalDNSProvider struct {
	ZoneDir string
}

func NewLocalDNSProvider(zoneDir string) *LocalDNSProvider {
	if zoneDir == "" {
		zoneDir = "/etc/bind/zones"
	}
	return &LocalDNSProvider{ZoneDir: zoneDir}
}

func (l *LocalDNSProvider) CreateTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	_ = os.MkdirAll(l.ZoneDir, 0755)
	zoneFile := filepath.Join(l.ZoneDir, domain+".db")

	entry := fmt.Sprintf("\n; ACME DNS-01 Challenge\n%s. 120 IN TXT \"%s\"\n", fqdn, value)
	f, err := os.OpenFile(zoneFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open local zone file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to append txt record to zone: %w", err)
	}
	return nil
}

func (l *LocalDNSProvider) DeleteTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	zoneFile := filepath.Join(l.ZoneDir, domain+".db")
	data, err := os.ReadFile(zoneFile)
	if err != nil {
		return nil // nothing to delete if file doesn't exist
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	needle := fmt.Sprintf("%s. 120 IN TXT \"%s\"", fqdn, value)

	for _, line := range lines {
		if !strings.Contains(line, needle) && !strings.Contains(line, "; ACME DNS-01 Challenge") {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(zoneFile, []byte(strings.Join(newLines, "\n")), 0644)
}

// ============================================================================
// MANUAL CHALLENGE STORE
// ============================================================================

type ManualProvider struct {
	challenges map[string]*ChallengeInfo
	mu         sync.RWMutex
}

func NewManualProvider() *ManualProvider {
	return &ManualProvider{
		challenges: make(map[string]*ChallengeInfo),
	}
}

func (m *ManualProvider) StoreChallenge(c *ChallengeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.challenges[c.Domain] = c
}

func (m *ManualProvider) GetChallenge(domain string) (*ChallengeInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.challenges[domain]
	return c, ok
}

func (m *ManualProvider) DeleteChallenge(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.challenges, domain)
}

func (m *ManualProvider) CreateTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	// For manual provider, record is kept in memory waiting for user to publish it on their DNS registrar
	return nil
}

func (m *ManualProvider) DeleteTXTRecord(ctx context.Context, domain, fqdn, value string) error {
	m.DeleteChallenge(domain)
	return nil
}
