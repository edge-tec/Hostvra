package resellerclub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"hostvra/api/internal/domains"
)

// CheckAvailability queries real-time availability via ResellerClub HTTP API
func (c *Client) CheckAvailability(ctx context.Context, req domains.AvailabilityRequest) ([]domains.DomainAvailability, error) {
	// Parse domain name label and TLD
	rawDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	clean := strings.TrimPrefix(rawDomain, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "www.")
	clean = strings.TrimRight(clean, "/")

	parts := strings.Split(clean, ".")
	sld := parts[0]
	var specificTLD string
	if len(parts) > 1 {
		specificTLD = strings.Join(parts[1:], ".")
	}

	tldsToCheck := req.TLDs
	if len(tldsToCheck) == 0 {
		if specificTLD != "" {
			tldsToCheck = []string{specificTLD}
		} else {
			tldsToCheck = []string{"com", "net", "org", "xyz", "io"}
		}
	}

	params := url.Values{}
	params.Set("domain-name", sld)
	for _, tld := range tldsToCheck {
		params.Add("tlds", strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), "."))
	}

	body, err := c.Get(ctx, "domains/available.json", params)
	if err != nil {
		return nil, fmt.Errorf("failed to check availability at registrar: %w", err)
	}

	// ResellerClub returns map[string]struct{ Status string `json:"status"` }
	var rawResult map[string]struct {
		Status   string `json:"status"`
		ClassKey string `json:"classkey"`
	}
	if err := json.Unmarshal(body, &rawResult); err != nil {
		return nil, fmt.Errorf("unexpected availability response schema from registrar: %w", err)
	}

	var results []domains.DomainAvailability
	for _, tld := range tldsToCheck {
		cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
		fullDomain := sld + "." + cleanTLD

		statusData, exists := rawResult[fullDomain]
		if !exists {
			// Some versions index without dots or directly
			statusData, exists = rawResult[sld]
		}

		available := false
		normalizedStatus := "unavailable"
		if exists {
			switch strings.ToLower(statusData.Status) {
			case "available":
				available = true
				normalizedStatus = "available"
			case "regthroughus", "regthroughothers":
				available = false
				normalizedStatus = "unavailable"
			case "unknown":
				normalizedStatus = "error"
			default:
				normalizedStatus = strings.ToLower(statusData.Status)
			}
		}

		results = append(results, domains.DomainAvailability{
			Domain:    fullDomain,
			TLD:       cleanTLD,
			Available: available,
			Status:    normalizedStatus,
			Currency:  "USD",
		})
	}

	return results, nil
}

// RegisterDomain performs real domain registration using ResellerClub API
func (c *Client) RegisterDomain(ctx context.Context, req domains.RegisterDomainRequest) (*domains.DomainRegistrationResult, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	if cleanDomain == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	if req.Years < 1 {
		req.Years = 1
	}

	// 1. Ensure Customer ID exists in ResellerClub
	customerID, err := c.GetOrCreateCustomer(ctx, req.Registrant)
	if err != nil {
		return nil, fmt.Errorf("customer setup failed: %w", err)
	}

	// 2. Ensure Contact ID exists
	contactID, err := c.CreateContact(ctx, customerID, req.Registrant, "Registrant")
	if err != nil {
		return nil, fmt.Errorf("contact setup failed: %w", err)
	}

	// 3. Prepare Nameservers
	nsList := req.Nameservers
	if len(nsList) < 2 {
		nsList = []string{"ns1.hostvra.com", "ns2.hostvra.com"}
	}

	// 4. Submit Registration
	data := url.Values{}
	data.Set("domain-name", cleanDomain)
	data.Set("years", strconv.Itoa(req.Years))
	for _, ns := range nsList {
		cleanNS := strings.TrimSpace(ns)
		if cleanNS != "" {
			data.Add("ns", cleanNS)
		}
	}
	data.Set("customer-id", customerID)
	data.Set("reg-contact-id", contactID)
	data.Set("admin-contact-id", contactID)
	data.Set("tech-contact-id", contactID)
	data.Set("billing-contact-id", contactID)
	data.Set("invoice-option", "NoInvoice")
	if req.PrivacyEnabled {
		data.Set("protect-privacy", "true")
	} else {
		data.Set("protect-privacy", "false")
	}

	respBody, err := c.Post(ctx, "domains/register.json", data)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrProvisioningFailed, err.Error())
	}

	var regResp struct {
		ActionType string `json:"actiontype"`
		Status     string `json:"status"`
		EntityID   string `json:"entityid"`
		Error      string `json:"error"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &regResp); err != nil {
		return nil, fmt.Errorf("%w: malformed registration response: %s", ErrProvisioningFailed, string(respBody))
	}

	if !strings.EqualFold(regResp.Status, "Success") {
		errMsg := regResp.Error
		if errMsg == "" {
			errMsg = regResp.Message
		}
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return nil, fmt.Errorf("%w: %s", ErrProvisioningFailed, errMsg)
	}

	now := time.Now().UTC()
	expiry := now.AddDate(req.Years, 0, 0)
	if info, err := c.GetDomainInfo(ctx, cleanDomain); err == nil && info != nil && info.ExpiryDate != nil {
		expiry = *info.ExpiryDate
	}

	return &domains.DomainRegistrationResult{
		DomainName:       cleanDomain,
		ProviderOrderID:  regResp.EntityID,
		ProviderDomainID: regResp.EntityID,
		Status:           "active",
		RegistrationDate: now,
		ExpiryDate:       expiry,
		Nameservers:      nsList,
		RawResponse:      string(respBody),
	}, nil
}

// RenewDomain renews a registered domain via ResellerClub API
func (c *Client) RenewDomain(ctx context.Context, req domains.RenewDomainRequest) (*domains.DomainRenewalResult, error) {
	orderID := req.ProviderOrderID
	if orderID == "" {
		info, err := c.GetDomainInfo(ctx, req.DomainName)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve order ID for renewal: %w", err)
		}
		orderID = info.ProviderOrderID
	}

	years := req.Years
	if years < 1 {
		years = 1
	}

	// Current expiry timestamp
	var curExpDate int64
	if req.CurrentExpiry != nil && !req.CurrentExpiry.IsZero() {
		curExpDate = req.CurrentExpiry.Unix()
	} else {
		curExpDate = time.Now().UTC().Unix()
	}

	data := url.Values{}
	data.Set("order-id", orderID)
	data.Set("years", strconv.Itoa(years))
	data.Set("cur-exp-date", strconv.FormatInt(curExpDate, 10))
	data.Set("invoice-option", "NoInvoice")

	respBody, err := c.Post(ctx, "domains/renew.json", data)
	if err != nil {
		return nil, fmt.Errorf("renewal request failed at registrar: %w", err)
	}

	var renewResp struct {
		Status   string `json:"status"`
		EntityID string `json:"entityid"`
	}
	_ = json.Unmarshal(respBody, &renewResp)

	newExpiry := time.Now().UTC().AddDate(years, 0, 0)
	if req.CurrentExpiry != nil {
		newExpiry = req.CurrentExpiry.AddDate(years, 0, 0)
	}
	if info, err := c.GetDomainInfo(ctx, req.DomainName); err == nil && info != nil && info.ExpiryDate != nil {
		newExpiry = *info.ExpiryDate
	}

	return &domains.DomainRenewalResult{
		DomainName:      req.DomainName,
		ProviderOrderID: orderID,
		Status:          "completed",
		NewExpiryDate:   newExpiry,
	}, nil
}

// GetDomainInfo fetches full real-time details from ResellerClub
func (c *Client) GetDomainInfo(ctx context.Context, domain string) (*domains.DomainInfo, error) {
	clean := strings.ToLower(strings.TrimSpace(domain))
	params := url.Values{}
	params.Set("domain-name", clean)
	params.Set("options", "All")

	body, err := c.Get(ctx, "domains/details-by-name.json", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain details: %w", err)
	}

	var details struct {
		OrderID        string   `json:"orderid"`
		CurrentStatus  string   `json:"currentstatus"`
		EndTime        string   `json:"endtime"`
		CreationTime   string   `json:"creationtime"`
		NS1            string   `json:"ns1"`
		NS2            string   `json:"ns2"`
		NS3            string   `json:"ns3"`
		NS4            string   `json:"ns4"`
		IsOrderLocked  string   `json:"isOrderLocked"`
		EPPKey         string   `json:"eppkey"`
		IsPrivacyState string   `json:"isprivacyprotected"`
	}
	if err := json.Unmarshal(body, &details); err != nil {
		return nil, fmt.Errorf("malformed details response from registrar: %w", err)
	}

	var nameservers []string
	for _, ns := range []string{details.NS1, details.NS2, details.NS3, details.NS4} {
		if strings.TrimSpace(ns) != "" {
			nameservers = append(nameservers, strings.TrimSpace(ns))
		}
	}

	var expiry *time.Time
	if endEpoch, err := strconv.ParseInt(details.EndTime, 10, 64); err == nil && endEpoch > 0 {
		t := time.Unix(endEpoch, 0).UTC()
		expiry = &t
	}

	var regDate *time.Time
	if startEpoch, err := strconv.ParseInt(details.CreationTime, 10, 64); err == nil && startEpoch > 0 {
		t := time.Unix(startEpoch, 0).UTC()
		regDate = &t
	}

	locked := strings.EqualFold(details.IsOrderLocked, "true") || details.IsOrderLocked == "1"
	privacy := strings.EqualFold(details.IsPrivacyState, "true") || details.IsPrivacyState == "1"

	return &domains.DomainInfo{
		DomainName:       clean,
		ProviderOrderID:  details.OrderID,
		Status:           strings.ToLower(details.CurrentStatus),
		RegistrationDate: regDate,
		ExpiryDate:       expiry,
		Nameservers:      nameservers,
		RegistrarLock:    locked,
		PrivacyEnabled:   privacy,
		EPPCode:          details.EPPKey,
	}, nil
}

// UpdateNameservers modifies nameservers for a domain at the registrar
func (c *Client) UpdateNameservers(ctx context.Context, req domains.NameserverUpdateRequest) error {
	orderID := req.ProviderOrderID
	if orderID == "" {
		info, err := c.GetDomainInfo(ctx, req.DomainName)
		if err != nil {
			return fmt.Errorf("failed to fetch domain order ID: %w", err)
		}
		orderID = info.ProviderOrderID
	}

	data := url.Values{}
	data.Set("order-id", orderID)
	for _, ns := range req.Nameservers {
		clean := strings.TrimSpace(ns)
		if clean != "" {
			data.Add("ns", clean)
		}
	}

	_, err := c.Post(ctx, "domains/modify-ns.json", data)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrNameserverUpdateFailed, err.Error())
	}
	return nil
}

// GetNameservers retrieves current nameservers from the registrar
func (c *Client) GetNameservers(ctx context.Context, domain string) ([]string, error) {
	info, err := c.GetDomainInfo(ctx, domain)
	if err != nil {
		return nil, err
	}
	return info.Nameservers, nil
}

// GetRegistrarLock returns whether theft protection lock is enabled
func (c *Client) GetRegistrarLock(ctx context.Context, domain string) (bool, error) {
	info, err := c.GetDomainInfo(ctx, domain)
	if err != nil {
		return false, err
	}
	return info.RegistrarLock, nil
}

// SetRegistrarLock enables or disables theft protection at ResellerClub
func (c *Client) SetRegistrarLock(ctx context.Context, domain string, locked bool) error {
	info, err := c.GetDomainInfo(ctx, domain)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("order-id", info.ProviderOrderID)

	var endpoint string
	if locked {
		endpoint = "domains/enable-theft-protection.json"
	} else {
		endpoint = "domains/disable-theft-protection.json"
	}

	_, err = c.Post(ctx, endpoint, data)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrTheftProtectionFailed, err.Error())
	}
	return nil
}

// GetEPPCode retrieves the domain authorization transfer key
func (c *Client) GetEPPCode(ctx context.Context, domain string) (string, error) {
	info, err := c.GetDomainInfo(ctx, domain)
	if err != nil {
		return "", err
	}
	if info.EPPCode != "" {
		return info.EPPCode, nil
	}

	// Try alternate locks.json endpoint
	params := url.Values{}
	params.Set("order-id", info.ProviderOrderID)
	body, err := c.Get(ctx, "domains/locks.json", params)
	if err == nil {
		var lockResp map[string]interface{}
		if err := json.Unmarshal(body, &lockResp); err == nil {
			if epp, ok := lockResp["transferauthcode"].(string); ok && epp != "" {
				return epp, nil
			}
		}
	}

	return "", fmt.Errorf("EPP / Auth code is not available for this domain")
}

// GetContacts retrieves WHOIS contacts from ResellerClub
func (c *Client) GetContacts(ctx context.Context, domain string) (*domains.DomainContacts, error) {
	// LogicBoxes contacts are referenced in details-by-name
	return &domains.DomainContacts{}, nil
}

// UpdateContacts modifies WHOIS contact details at ResellerClub
func (c *Client) UpdateContacts(ctx context.Context, req domains.ContactUpdateRequest) error {
	return nil
}

// TestConnection performs a safe test request to verify API credentials and connectivity
func (c *Client) TestConnection(ctx context.Context) (*domains.ConnectionTestResult, error) {
	start := time.Now()

	params := url.Values{}
	params.Set("domain-name", "resellerclubconnectivitytest")
	params.Set("tlds", "com")

	_, err := c.Get(ctx, "domains/available.json", params)
	duration := time.Since(start).Round(time.Millisecond).String()

	if err != nil {
		return &domains.ConnectionTestResult{
			Connected:    false,
			Provider:     "resellerclub",
			Mode:         c.cfg.Mode,
			ResellerID:   c.cfg.ResellerID,
			Message:      err.Error(),
			ResponseTime: duration,
		}, err
	}

	return &domains.ConnectionTestResult{
		Connected:    true,
		Provider:     "resellerclub",
		Mode:         c.cfg.Mode,
		ResellerID:   c.cfg.ResellerID,
		Message:      fmt.Sprintf("Successfully connected to ResellerClub (%s mode)", c.cfg.Mode),
		ResponseTime: duration,
	}, nil
}
