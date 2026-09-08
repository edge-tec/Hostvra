package resellerclub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"hostvra/api/internal/domains"
)

// GetDNSRecords searches and retrieves DNS records from ResellerClub DNS service
func (c *Client) GetDNSRecords(ctx context.Context, domain string) ([]domains.DNSRecord, error) {
	clean := strings.ToLower(strings.TrimSpace(domain))
	params := url.Values{}
	params.Set("domain-name", clean)
	params.Set("no-of-records", "100")
	params.Set("page-no", "1")

	body, err := c.Get(ctx, "dns/manage/search-records.json", params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dns records: %w", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(body, &rawMap); err != nil {
		return nil, fmt.Errorf("malformed dns search response: %w", err)
	}

	var records []domains.DNSRecord
	// Parse records by type from map
	for recType, val := range rawMap {
		if recMap, ok := val.(map[string]interface{}); ok {
			for recID, recData := range recMap {
				if d, ok := recData.(map[string]interface{}); ok {
					host, _ := d["host"].(string)
					value, _ := d["value"].(string)
					ttl := 3600
					if ttlStr, ok := d["timetolive"].(string); ok {
						if t, err := strconv.Atoi(ttlStr); err == nil {
							ttl = t
						}
					}
					priority := 0
					if prioStr, ok := d["priority"].(string); ok {
						if p, err := strconv.Atoi(prioStr); err == nil {
							priority = p
						}
					}

					records = append(records, domains.DNSRecord{
						ID:       recID,
						Type:     strings.ToUpper(recType),
						Name:     host,
						Value:    value,
						TTL:      ttl,
						Priority: priority,
					})
				}
			}
		}
	}

	return records, nil
}

// CreateDNSRecord creates a DNS record in ResellerClub
func (c *Client) CreateDNSRecord(ctx context.Context, req domains.DNSRecordRequest) error {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	recType := strings.ToUpper(strings.TrimSpace(req.Type))

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	data := url.Values{}
	data.Set("domain-name", cleanDomain)
	data.Set("host", req.Name)
	data.Set("value", req.Value)
	data.Set("ttl", strconv.Itoa(ttl))

	var endpoint string
	switch recType {
	case "A":
		endpoint = "dns/manage/add-ipv4-record.json"
	case "AAAA":
		endpoint = "dns/manage/add-ipv6-record.json"
	case "CNAME":
		endpoint = "dns/manage/add-cname-record.json"
	case "MX":
		endpoint = "dns/manage/add-mx-record.json"
		priority := req.Priority
		if priority <= 0 {
			priority = 10
		}
		data.Set("priority", strconv.Itoa(priority))
	case "TXT":
		endpoint = "dns/manage/add-txt-record.json"
	case "NS":
		endpoint = "dns/manage/add-ns-record.json"
	case "SRV":
		endpoint = "dns/manage/add-srv-record.json"
		data.Set("priority", strconv.Itoa(req.Priority))
	default:
		return fmt.Errorf("unsupported DNS record type: %s", recType)
	}

	_, err := c.Post(ctx, endpoint, data)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDNSUpdateFailed, err.Error())
	}
	return nil
}

// UpdateDNSRecord modifies an existing DNS record in ResellerClub
func (c *Client) UpdateDNSRecord(ctx context.Context, req domains.DNSRecordUpdateRequest) error {
	cleanDomain := strings.ToLower(strings.TrimSpace(req.DomainName))
	recType := strings.ToUpper(strings.TrimSpace(req.Type))

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	data := url.Values{}
	data.Set("domain-name", cleanDomain)
	data.Set("host", req.Name)
	data.Set("new-value", req.Value)
	data.Set("current-value", req.Value)
	data.Set("ttl", strconv.Itoa(ttl))

	var endpoint string
	switch recType {
	case "A":
		endpoint = "dns/manage/modify-ipv4-record.json"
	case "AAAA":
		endpoint = "dns/manage/modify-ipv6-record.json"
	case "CNAME":
		endpoint = "dns/manage/modify-cname-record.json"
	case "MX":
		endpoint = "dns/manage/modify-mx-record.json"
		data.Set("priority", strconv.Itoa(req.Priority))
	case "TXT":
		endpoint = "dns/manage/modify-txt-record.json"
	case "NS":
		endpoint = "dns/manage/modify-ns-record.json"
	case "SRV":
		endpoint = "dns/manage/modify-srv-record.json"
		data.Set("priority", strconv.Itoa(req.Priority))
	default:
		return fmt.Errorf("unsupported DNS record type: %s", recType)
	}

	_, err := c.Post(ctx, endpoint, data)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDNSUpdateFailed, err.Error())
	}
	return nil
}

// DeleteDNSRecord removes a DNS record in ResellerClub by finding the record and calling the type-specific deletion endpoint
func (c *Client) DeleteDNSRecord(ctx context.Context, domain string, recordID string) error {
	cleanDomain := strings.ToLower(strings.TrimSpace(domain))

	records, err := c.GetDNSRecords(ctx, cleanDomain)
	if err != nil {
		return err
	}

	var target *domains.DNSRecord
	for i := range records {
		if records[i].ID == recordID {
			target = &records[i]
			break
		}
	}

	if target == nil {
		// Record already deleted or not present at registrar
		return nil
	}

	data := url.Values{}
	data.Set("domain-name", cleanDomain)
	data.Set("host", target.Name)
	data.Set("value", target.Value)

	var endpoint string
	switch strings.ToUpper(target.Type) {
	case "A":
		endpoint = "dns/manage/delete-ipv4-record.json"
	case "AAAA":
		endpoint = "dns/manage/delete-ipv6-record.json"
	case "CNAME":
		endpoint = "dns/manage/delete-cname-record.json"
	case "MX":
		endpoint = "dns/manage/delete-mx-record.json"
	case "TXT":
		endpoint = "dns/manage/delete-txt-record.json"
	case "NS":
		endpoint = "dns/manage/delete-ns-record.json"
	case "SRV":
		endpoint = "dns/manage/delete-srv-record.json"
	default:
		endpoint = "dns/manage/delete-ipv4-record.json"
	}

	_, err = c.Post(ctx, endpoint, data)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDNSUpdateFailed, err.Error())
	}
	return nil
}
