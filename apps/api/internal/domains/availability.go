package domains

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AvailabilityService struct {
	registrar  DomainRegistrar
	pricing    *PricingEngine
	httpClient *http.Client
}

func NewAvailabilityService(r DomainRegistrar, pe *PricingEngine) *AvailabilityService {
	return &AvailabilityService{
		registrar: r,
		pricing:   pe,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// SearchDomain performs real-time availability check across multiple TLDs
// using the configured registrar with automated real-time DNS & RDAP fallback.
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

	popularTLDs := []string{"com", "net", "org", "xyz", "io", "co", "tech", "online"}

	tldsToCheck := requestedTLDs
	if len(tldsToCheck) == 0 {
		if specificTLD != "" {
			tldsToCheck = []string{specificTLD}
			for _, p := range popularTLDs {
				if p != specificTLD {
					tldsToCheck = append(tldsToCheck, p)
				}
			}
		} else {
			tldsToCheck = popularTLDs
		}
	}

	var results []DomainAvailability
	var registrarErr error

	// 1. Try configured registrar first if available
	if s.registrar != nil {
		regCtx, cancel := context.WithTimeout(ctx, 3500*time.Millisecond)
		results, registrarErr = s.registrar.CheckAvailability(regCtx, AvailabilityRequest{
			DomainName: sld,
			TLDs:       tldsToCheck,
		})
		cancel()
	}

	// 2. If registrar is missing, timed out, or returned all errors -> run Real-Time Live Resolver
	needsFallback := registrarErr != nil || len(results) == 0
	if !needsFallback {
		allFailed := true
		for _, r := range results {
			if r.Status != "error" {
				allFailed = false
				break
			}
		}
		needsFallback = allFailed
	}

	if needsFallback {
		results = s.resolveRealtimeAvailability(ctx, sld, tldsToCheck)
	}

	// 3. Attach Hostvra selling pricing from database/catalog & enrich metadata
	for i := range results {
		res := &results[i]
		cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(res.TLD)), ".")
		res.TLD = cleanTLD
		res.Currency = "USD"

		switch cleanTLD {
		case "com", "net", "org", "io", "xyz":
			res.IsPopular = true
		}

		if calcPrice, err := s.pricing.CalculateRegistrationPrice(ctx, cleanTLD, 1); err == nil {
			res.RegisterPrice = calcPrice.UnitPrice
			res.Currency = calcPrice.Currency
		}
		if renewPrice, errRenew := s.pricing.CalculateRenewalPrice(ctx, cleanTLD, 1); errRenew == nil {
			res.RenewPrice = renewPrice.UnitPrice
		}
		if transferPrice, errTrans := s.pricing.CalculateTransferPrice(ctx, cleanTLD); errTrans == nil {
			res.TransferPrice = transferPrice.UnitPrice
		}
	}

	return results, nil
}

// resolveRealtimeAvailability queries real-time DNS (NS, A, SOA) and RDAP in parallel
func (s *AvailabilityService) resolveRealtimeAvailability(ctx context.Context, sld string, tlds []string) []DomainAvailability {
	results := make([]DomainAvailability, len(tlds))
	var wg sync.WaitGroup

	for i, tld := range tlds {
		wg.Add(1)
		cleanTLD := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
		fullDomain := sld + "." + cleanTLD

		go func(idx int, targetDomain, targetTLD string) {
			defer wg.Done()
			available := s.checkDomainRealtimeDNSAndRDAP(ctx, targetDomain)
			status := "unavailable"
			if available {
				status = "available"
			}

			results[idx] = DomainAvailability{
				Domain:    targetDomain,
				TLD:       targetTLD,
				Available: available,
				Status:    status,
			}
		}(i, fullDomain, cleanTLD)
	}

	wg.Wait()
	return results
}

// checkDomainRealtimeDNSAndRDAP checks authoritative DNS and RDAP
func (s *AvailabilityService) checkDomainRealtimeDNSAndRDAP(ctx context.Context, domain string) bool {
	// 1. Check DNS Nameservers
	ns, err := net.DefaultResolver.LookupNS(ctx, domain)
	if err == nil && len(ns) > 0 {
		return false // Definitely taken
	}

	// 2. Check DNS Host IP
	ips, err := net.DefaultResolver.LookupHost(ctx, domain)
	if err == nil && len(ips) > 0 {
		return false // Definitely taken
	}

	// 3. Fast RDAP query to confirm availability
	rdapURL := "https://rdap.org/domain/" + domain
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rdapURL, nil)
	if err == nil {
		req.Header.Set("Accept", "application/rdap+json, application/json")
		resp, rdapErr := s.httpClient.Do(req)
		if rdapErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				// RDAP 404 Not Found explicitly means domain is available
				return true
			}
			if resp.StatusCode == http.StatusOK {
				// RDAP 200 OK means domain is registered
				return false
			}
		}
	}

	// If DNS has no NS/IP and RDAP did not find active registration, it is available
	return true
}
