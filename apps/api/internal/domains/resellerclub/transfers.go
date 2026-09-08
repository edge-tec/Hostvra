package resellerclub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"hostvra/api/internal/domains"
)

// TransferDomain initiates an inbound transfer at ResellerClub
func (c *Client) TransferDomain(ctx context.Context, req domains.TransferDomainRequest) (*domains.DomainTransferResult, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	authCode := strings.TrimSpace(req.AuthCode)
	if cleanDomain == "" {
		return nil, fmt.Errorf("domain name is required")
	}
	if authCode == "" {
		return nil, fmt.Errorf("%w: auth code cannot be empty", ErrInvalidAuthCode)
	}

	// 1. Ensure Customer exists
	customerID, err := c.GetOrCreateCustomer(ctx, req.Registrant)
	if err != nil {
		return nil, fmt.Errorf("customer setup failed: %w", err)
	}

	// 2. Ensure Contact exists
	contactID, err := c.CreateContact(ctx, customerID, req.Registrant, "TransferRegistrant")
	if err != nil {
		return nil, fmt.Errorf("contact setup failed: %w", err)
	}

	nsList := req.Nameservers
	if len(nsList) < 2 {
		nsList = []string{"ns1.hostvra.com", "ns2.hostvra.com"}
	}

	data := url.Values{}
	data.Set("domain-name", cleanDomain)
	data.Set("auth-code", authCode)
	data.Set("customer-id", customerID)
	data.Set("reg-contact-id", contactID)
	data.Set("admin-contact-id", contactID)
	data.Set("tech-contact-id", contactID)
	data.Set("billing-contact-id", contactID)
	for _, ns := range nsList {
		cleanNS := strings.TrimSpace(ns)
		if cleanNS != "" {
			data.Add("ns", cleanNS)
		}
	}
	data.Set("invoice-option", "NoInvoice")
	if req.PrivacyEnabled {
		data.Set("protect-privacy", "true")
	} else {
		data.Set("protect-privacy", "false")
	}

	respBody, err := c.Post(ctx, "domains/transfer.json", data)
	if err != nil {
		return nil, fmt.Errorf("transfer submission failed at registrar: %w", err)
	}

	var transResp struct {
		ActionType string `json:"actiontype"`
		Status     string `json:"status"`
		EntityID   string `json:"entityid"`
		Error      string `json:"error"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &transResp); err != nil {
		return nil, fmt.Errorf("malformed transfer response: %s", string(respBody))
	}

	if !strings.EqualFold(transResp.Status, "Success") {
		errMsg := transResp.Error
		if errMsg == "" {
			errMsg = transResp.Message
		}
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return nil, fmt.Errorf("transfer failed: %s", errMsg)
	}

	return &domains.DomainTransferResult{
		DomainName:      cleanDomain,
		ProviderOrderID: transResp.EntityID,
		Status:          "submitted",
		Message:         "Domain transfer has been successfully submitted to the registrar",
	}, nil
}
