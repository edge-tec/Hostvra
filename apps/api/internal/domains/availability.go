package domains

import (
	"context"
	"strings"
)

type AvailabilityService struct {
	registrar DomainRegistrar
	pricing   *PricingEngine
}

func NewAvailabilityService(r DomainRegistrar, pe *PricingEngine) *AvailabilityService {
	return &AvailabilityService{
		registrar: r,
		pricing:   pe,
	}
}

// SearchDomain performs real-time availability check and enriches with Hostvra pricing
func (s *AvailabilityService) SearchDomain(ctx context.Context, query string, requestedTLDs []string) ([]DomainAvailability, error) {
	cleanDomain := strings.ToLower(strings.TrimSpace(query))
	cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
	cleanDomain = strings.TrimPrefix(cleanDomain, "http://")
	cleanDomain = strings.TrimPrefix(cleanDomain, "www.")
	cleanDomain = strings.TrimRight(cleanDomain, "/")

	parts := strings.Split(cleanDomain, ".")
	sld := parts[0]
	var specificTLD string
	if len(parts) > 1 {
		specificTLD = strings.Join(parts[1:], ".")
	}

	tldsToCheck := requestedTLDs
	if len(tldsToCheck) == 0 {
		if specificTLD != "" {
			tldsToCheck = []string{specificTLD}
		} else {
			tldsToCheck = []string{"com", "net", "org", "xyz", "io", "co", "tech", "online"}
		}
	}

	// Query real registrar API
	results, err := s.registrar.CheckAvailability(ctx, AvailabilityRequest{
		DomainName: sld,
		TLDs:       tldsToCheck,
	})
	if err != nil {
		// Return safe error entries, NEVER false availability
		var errorResults []DomainAvailability
		for _, tld := range tldsToCheck {
			cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
			errorResults = append(errorResults, DomainAvailability{
				Domain:    sld + "." + cleanTLD,
				TLD:       cleanTLD,
				Available: false,
				Status:    "error",
				Message:   "Unable to check domain availability at the moment",
				Currency:  "USD",
			})
		}
		return errorResults, nil
	}

	// Attach Hostvra selling pricing from database
	for i := range results {
		res := &results[i]
		calcPrice, err := s.pricing.CalculateRegistrationPrice(ctx, res.TLD, 1)
		if err == nil {
			res.RegisterPrice = calcPrice.UnitPrice
			renewPrice, errRenew := s.pricing.CalculateRenewalPrice(ctx, res.TLD, 1)
			if errRenew == nil {
				res.RenewPrice = renewPrice.UnitPrice
			}
			transferPrice, errTrans := s.pricing.CalculateTransferPrice(ctx, res.TLD)
			if errTrans == nil {
				res.TransferPrice = transferPrice.UnitPrice
			}
			res.Currency = calcPrice.Currency
		}
	}

	return results, nil
}
