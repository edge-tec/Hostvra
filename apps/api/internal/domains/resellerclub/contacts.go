package resellerclub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"hostvra/api/internal/domains"
)

type CustomerDetailsResponse struct {
	CustomerID string `json:"customerid"`
	UserEmail  string `json:"useremail"`
	Name       string `json:"name"`
	Status     string `json:"status"`
}

// GetOrCreateCustomer ensures a customer account exists in ResellerClub for the given registrant
func (c *Client) GetOrCreateCustomer(ctx context.Context, contact *domains.ContactInfo) (string, error) {
	if contact == nil || contact.Email == "" {
		return "", fmt.Errorf("registrant contact email is required")
	}

	// 1. Try to find existing customer by email
	params := url.Values{}
	params.Set("username", contact.Email)

	body, err := c.Get(ctx, "customers/details.json", params)
	if err == nil {
		var details CustomerDetailsResponse
		if err := json.Unmarshal(body, &details); err == nil && details.CustomerID != "" {
			return details.CustomerID, nil
		}
	}

	// 2. If not found, create new customer via customers/v2/signup.json
	fullName := strings.TrimSpace(contact.FirstName + " " + contact.LastName)
	if fullName == "" {
		fullName = "Domain Registrant"
	}
	company := contact.Organization
	if company == "" {
		company = fullName
	}
	phoneCC, phoneNum := splitPhone(contact.Phone, contact.Country)

	// Generate secure random password for customer account
	randomBytes := make([]byte, 12)
	_, _ = rand.Read(randomBytes)
	passwd := "Hv!" + hex.EncodeToString(randomBytes)

	data := url.Values{}
	data.Set("username", contact.Email)
	data.Set("passwd", passwd)
	data.Set("name", fullName)
	data.Set("company", company)
	data.Set("address-line-1", defaultIfEmpty(contact.Address1, "100 Main St"))
	if contact.Address2 != "" {
		data.Set("address-line-2", contact.Address2)
	}
	data.Set("city", defaultIfEmpty(contact.City, "City"))
	data.Set("state", defaultIfEmpty(contact.State, "State"))
	data.Set("country", defaultIfEmpty(contact.Country, "US"))
	data.Set("zipcode", defaultIfEmpty(contact.PostalCode, "10001"))
	data.Set("phone-cc", phoneCC)
	data.Set("phone", phoneNum)
	data.Set("lang-pref", "en")

	signupResp, err := c.Post(ctx, "customers/v2/signup.json", data)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrCustomerCreationFailed, err.Error())
	}

	// LogicBoxes returns either the raw integer customer ID or JSON integer
	customerIDStr := strings.Trim(strings.TrimSpace(string(signupResp)), `"`)
	if _, err := strconv.Atoi(customerIDStr); err != nil {
		var intID int
		if err := json.Unmarshal(signupResp, &intID); err == nil && intID > 0 {
			customerIDStr = strconv.Itoa(intID)
		} else {
			return "", fmt.Errorf("%w: unexpected customer signup response: %s", ErrCustomerCreationFailed, string(signupResp))
		}
	}

	return customerIDStr, nil
}

// CreateContact creates a WHOIS contact object for a customer
func (c *Client) CreateContact(ctx context.Context, customerID string, contact *domains.ContactInfo, contactType string) (string, error) {
	if contact == nil {
		return "", fmt.Errorf("contact details are required")
	}

	fullName := strings.TrimSpace(contact.FirstName + " " + contact.LastName)
	if fullName == "" {
		fullName = "Domain " + contactType
	}
	company := contact.Organization
	if company == "" {
		company = fullName
	}
	phoneCC, phoneNum := splitPhone(contact.Phone, contact.Country)

	data := url.Values{}
	data.Set("name", fullName)
	data.Set("company", company)
	data.Set("email", contact.Email)
	data.Set("address-line-1", defaultIfEmpty(contact.Address1, "100 Main St"))
	if contact.Address2 != "" {
		data.Set("address-line-2", contact.Address2)
	}
	data.Set("city", defaultIfEmpty(contact.City, "City"))
	data.Set("state", defaultIfEmpty(contact.State, "State"))
	data.Set("country", defaultIfEmpty(contact.Country, "US"))
	data.Set("zipcode", defaultIfEmpty(contact.PostalCode, "10001"))
	data.Set("phone-cc", phoneCC)
	data.Set("phone", phoneNum)
	data.Set("customer-id", customerID)
	data.Set("type", "Contact")

	resp, err := c.Post(ctx, "contacts/add.json", data)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrContactCreationFailed, err.Error())
	}

	contactIDStr := strings.Trim(strings.TrimSpace(string(resp)), `"`)
	if _, err := strconv.Atoi(contactIDStr); err != nil {
		var intID int
		if err := json.Unmarshal(resp, &intID); err == nil && intID > 0 {
			contactIDStr = strconv.Itoa(intID)
		} else {
			return "", fmt.Errorf("%w: unexpected contact create response: %s", ErrContactCreationFailed, string(resp))
		}
	}

	return contactIDStr, nil
}

func defaultIfEmpty(val, def string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return def
	}
	return val
}

func splitPhone(phone, country string) (cc, num string) {
	clean := strings.TrimPrefix(strings.TrimSpace(phone), "+")
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")

	if strings.HasPrefix(clean, "880") {
		return "880", strings.TrimPrefix(clean, "880")
	}
	if strings.HasPrefix(clean, "1") && len(clean) == 11 {
		return "1", clean[1:]
	}
	if country == "BD" {
		if strings.HasPrefix(clean, "0") {
			return "880", clean[1:]
		}
		return "880", clean
	}
	if len(clean) >= 10 {
		return "1", clean[len(clean)-10:]
	}
	return "1", "5551234567"
}
