package domains

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

var (
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	labelRegex  = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
)

func ValidateDomainName(domain string) (clean string, tld string, err error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	domain = strings.TrimRight(domain, "/")

	if len(domain) < 3 || len(domain) > 253 {
		return "", "", errors.New("domain name length must be between 3 and 253 characters")
	}

	if !domainRegex.MatchString(domain) {
		return "", "", errors.New("invalid domain name syntax")
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", "", errors.New("domain must include a top-level domain extension (e.g. .com)")
	}

	for _, part := range parts {
		if !labelRegex.MatchString(part) {
			return "", "", fmt.Errorf("invalid domain label: %s", part)
		}
	}

	tld = strings.Join(parts[1:], ".")
	return domain, tld, nil
}

func ValidateContact(contact *ContactInfo, contactType string) error {
	if contact == nil {
		return fmt.Errorf("%s contact is required", contactType)
	}
	if strings.TrimSpace(contact.FirstName) == "" {
		return fmt.Errorf("%s contact first name is required", contactType)
	}
	if strings.TrimSpace(contact.LastName) == "" {
		return fmt.Errorf("%s contact last name is required", contactType)
	}
	if _, err := mail.ParseAddress(contact.Email); err != nil {
		return fmt.Errorf("%s contact email is invalid: %w", contactType, err)
	}
	if strings.TrimSpace(contact.Phone) == "" {
		return fmt.Errorf("%s contact phone number is required", contactType)
	}
	if strings.TrimSpace(contact.Address1) == "" {
		return fmt.Errorf("%s contact address is required", contactType)
	}
	if strings.TrimSpace(contact.City) == "" {
		return fmt.Errorf("%s contact city is required", contactType)
	}
	if strings.TrimSpace(contact.Country) == "" {
		return fmt.Errorf("%s contact country is required", contactType)
	}
	if len(strings.TrimSpace(contact.Country)) != 2 {
		return fmt.Errorf("%s contact country must be a 2-letter ISO country code", contactType)
	}
	return nil
}
