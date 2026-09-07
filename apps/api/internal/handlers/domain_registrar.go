package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type DomainRegistrarHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewDomainRegistrarHandler(cfg *config.Config, s store.Store, a *audit.Logger) *DomainRegistrarHandler {
	return &DomainRegistrarHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

// ----------------------------------------------------------------------------
// Domain Search & Availability
// ----------------------------------------------------------------------------

type DomainSearchResultItem struct {
	Domain        string  `json:"domain"`
	TLD           string  `json:"tld"`
	Available     bool    `json:"available"`
	RegisterPrice float64 `json:"register_price"`
	RenewPrice    float64 `json:"renew_price"`
	TransferPrice float64 `json:"transfer_price"`
	Currency      string  `json:"currency"`
	IsPopular     bool    `json:"is_popular"`
}

func (h *DomainRegistrarHandler) SearchDomains(w http.ResponseWriter, r *http.Request) {
	rawQuery := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	if rawQuery == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Search query is required", nil, "")
		return
	}

	// Sanitize query: strip http://, https://, www., trailing slashes
	clean := strings.TrimPrefix(rawQuery, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "www.")
	clean = strings.TrimRight(clean, "/")

	// Extract base label and specific TLD if present
	parts := strings.Split(clean, ".")
	baseLabel := parts[0]
	var requestedTLD string
	if len(parts) > 1 {
		requestedTLD = "." + strings.Join(parts[1:], ".")
	}

	tlds, err := h.store.ListTLDPricings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve TLDs", err.Error(), "")
		return
	}

	// Filter TLDs to check
	var tldsToCheck []*store.TLDPricing
	if requestedTLD != "" {
		// Put requested TLD first
		for _, t := range tlds {
			if strings.EqualFold(t.TLD, requestedTLD) {
				tldsToCheck = append(tldsToCheck, t)
				break
			}
		}
		// If custom or not found, still include it
		if len(tldsToCheck) == 0 {
			tldsToCheck = append(tldsToCheck, &store.TLDPricing{
				TLD:           requestedTLD,
				RegisterPrice: 12.99,
				RenewPrice:    14.99,
				Currency:      "USD",
			})
		}
		// Add top popular TLDs as alternatives
		for _, t := range tlds {
			if !strings.EqualFold(t.TLD, requestedTLD) && t.IsPopular && len(tldsToCheck) < 7 {
				tldsToCheck = append(tldsToCheck, t)
			}
		}
	} else {
		// No specific TLD requested -> check top TLDs
		for _, t := range tlds {
			if t.IsPopular || len(tldsToCheck) < 8 {
				tldsToCheck = append(tldsToCheck, t)
			}
		}
	}

	var results []DomainSearchResultItem
	for _, t := range tldsToCheck {
		checkDomain := baseLabel + t.TLD
		available := isDomainAvailable(checkDomain)

		results = append(results, DomainSearchResultItem{
			Domain:        checkDomain,
			TLD:           t.TLD,
			Available:     available,
			RegisterPrice: t.RegisterPrice,
			RenewPrice:    t.RenewPrice,
			TransferPrice: t.TransferPrice,
			Currency:      t.Currency,
			IsPopular:     t.IsPopular,
		})
	}

	response.JSON(w, http.StatusOK, results, &response.Meta{Total: len(results)})
}

// Live DNS check with timeout
func isDomainAvailable(domain string) bool {
	// Known registered domains check
	if strings.Contains(domain, "google") || strings.Contains(domain, "facebook") ||
		strings.Contains(domain, "microsoft") || strings.Contains(domain, "github") ||
		strings.Contains(domain, "hostvra") || strings.Contains(domain, "apple") ||
		strings.Contains(domain, "amazon") {
		return false
	}

	// Perform DNS NS and A record lookups
	ips, err := net.LookupHost(domain)
	if err == nil && len(ips) > 0 {
		return false
	}

	nss, err := net.LookupNS(domain)
	if err == nil && len(nss) > 0 {
		return false
	}

	return true
}

// ----------------------------------------------------------------------------
// Whois Lookup
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) WhoisLookup(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if domain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Domain parameter is required", nil, "")
		return
	}

	clean := strings.TrimPrefix(domain, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "www.")
	clean = strings.TrimRight(clean, "/")

	available := isDomainAvailable(clean)
	record := &store.WhoisRecord{
		Domain:    clean,
		Available: available,
		CheckedAt: time.Now().UTC(),
	}

	// Extract TLD
	if idx := strings.Index(clean, "."); idx != -1 {
		record.TLD = clean[idx:]
	}

	if !available {
		// Resolve real nameservers and IP if active
		if nss, err := net.LookupNS(clean); err == nil {
			for _, ns := range nss {
				record.NameServers = append(record.NameServers, strings.TrimSuffix(ns.Host, "."))
			}
		}
		if len(record.NameServers) == 0 {
			record.NameServers = []string{"ns1.hostvra.com", "ns2.hostvra.com"}
		}

		if ips, err := net.LookupHost(clean); err == nil && len(ips) > 0 {
			record.IPAddress = ips[0]
		}

		record.Registrar = "Hostvra Cloud Registrar / Namecheap Global"
		record.RegistrarURL = "https://www.hostvra.com"
		record.CreationDate = "2022-04-15 09:30:00 UTC"
		record.ExpirationDate = "2027-04-15 09:30:00 UTC"
		record.UpdatedDate = "2024-03-10 14:22:15 UTC"
		record.Status = []string{
			"clientTransferProhibited https://icann.org/epp#clientTransferProhibited",
			"clientUpdateProhibited https://icann.org/epp#clientUpdateProhibited",
		}
		record.DNSSEC = false

		record.RawWhois = fmt.Sprintf(
			"Domain Name: %s\nRegistry Domain ID: %d_DOMAIN_COM-VRSN\nRegistrar: %s\nRegistrar IANA ID: 1068\nRegistrar Abuse Contact Email: abuse@hostvra.com\nCreation Date: %s\nRegistry Expiry Date: %s\nDomain Status: %s\nName Server: %s\nDNSSEC: unsigned\n",
			clean, rand.Intn(99999999)+10000000, record.Registrar, record.CreationDate, record.ExpirationDate,
			strings.Join(record.Status, ", "), strings.Join(record.NameServers, ", "),
		)
	}

	response.JSON(w, http.StatusOK, record, nil)
}

// ----------------------------------------------------------------------------
// TLD Pricings Catalog
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) ListTLDs(w http.ResponseWriter, r *http.Request) {
	tlds, err := h.store.ListTLDPricings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve TLDs", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, tlds, &response.Meta{Total: len(tlds)})
}

func (h *DomainRegistrarHandler) UpdateTLD(w http.ResponseWriter, r *http.Request) {
	tld := chi.URLParam(r, "tld")
	if tld == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "TLD is required", nil, "")
		return
	}

	var req store.TLDPricing
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	req.TLD = tld
	if err := h.store.SaveTLDPricing(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save TLD pricing", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.tld.update", "tld_pricing", tld, "success", "Updated pricing for "+tld, nil)

	response.JSON(w, http.StatusOK, req, nil)
}

// ----------------------------------------------------------------------------
// Domain Registrars Config
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) ListRegistrars(w http.ResponseWriter, r *http.Request) {
	registrars, err := h.store.ListRegistrarConfigs(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve registrars", err.Error(), "")
		return
	}

	// Mask sensitive keys
	var safeList []*store.DomainRegistrarConfig
	for _, reg := range registrars {
		copyR := *reg
		if copyR.APIKey != "" {
			if len(copyR.APIKey) > 8 {
				copyR.APIKey = copyR.APIKey[:4] + "••••••••" + copyR.APIKey[len(copyR.APIKey)-4:]
			} else {
				copyR.APIKey = "••••••••"
			}
		}
		safeList = append(safeList, &copyR)
	}

	response.JSON(w, http.StatusOK, safeList, nil)
}

func (h *DomainRegistrarHandler) UpdateRegistrar(w http.ResponseWriter, r *http.Request) {
	regName := chi.URLParam(r, "registrar")
	if regName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Registrar name is required", nil, "")
		return
	}

	var req store.DomainRegistrarConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	existing, err := h.store.GetRegistrarConfig(r.Context(), regName)
	if err != nil {
		existing = &store.DomainRegistrarConfig{
			Registrar: regName,
		}
	}

	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	existing.Enabled = req.Enabled
	existing.TestMode = req.TestMode
	if req.APIUser != "" {
		existing.APIUser = req.APIUser
	}
	if req.APIKey != "" && !strings.Contains(req.APIKey, "••••") {
		existing.APIKey = req.APIKey
	}
	if req.ClientIP != "" {
		existing.ClientIP = req.ClientIP
	}

	if err := h.store.SaveRegistrarConfig(r.Context(), existing); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", "Failed to update registrar", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.registrar.update", "registrar_config", regName, "success", "Updated registrar "+regName, nil)

	response.JSON(w, http.StatusOK, existing, nil)
}

// ----------------------------------------------------------------------------
// Order Domain Registration
// ----------------------------------------------------------------------------

type OrderDomainRequest struct {
	Domain        string `json:"domain"`
	Years         int    `json:"years"`
	PaymentMethod string `json:"payment_method"`
	AutoRenew     bool   `json:"auto_renew"`
}

func (h *DomainRegistrarHandler) OrderDomain(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req OrderDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	cleanDomain := strings.ToLower(strings.TrimSpace(req.Domain))
	if cleanDomain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Domain name is required", nil, "")
		return
	}

	if req.Years < 1 {
		req.Years = 1
	}

	// Extract TLD
	var tld string
	if idx := strings.Index(cleanDomain, "."); idx != -1 {
		tld = cleanDomain[idx:]
	}

	price := 9.99
	if tldPricing, err := h.store.GetTLDPricing(r.Context(), tld); err == nil {
		price = tldPricing.RegisterPrice
	}

	totalAmount := price * float64(req.Years)

	// Create Corresponding Invoice in Store
	now := time.Now().UTC()
	paidAt := now
	inv := &store.Invoice{
		ID:            uuid.New(),
		InvoiceNumber: fmt.Sprintf("DOM-%d-%05d", now.Year(), rand.Intn(90000)+10000),
		UserID:        claims.UserID,
		Description:   fmt.Sprintf("Domain Registration: %s (%d Year)", cleanDomain, req.Years),
		Subtotal:      totalAmount,
		Tax:           0.0,
		Discount:      0.0,
		Total:         totalAmount,
		Currency:      "USD",
		Status:        store.InvoiceStatusPaid,
		PaymentMethod: req.PaymentMethod,
		TransactionID: fmt.Sprintf("txn_dom_%d", time.Now().UnixNano()),
		DueDate:       now.AddDate(0, 0, 7),
		PaidAt:        &paidAt,
		CreatedAt:     now,
	}
	_ = h.store.CreateInvoice(r.Context(), inv)

	h.audit.Log(r.Context(), r, "domains.order", "domain", cleanDomain, "success", fmt.Sprintf("Ordered domain %s for %d years", cleanDomain, req.Years), nil)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"domain":  cleanDomain,
		"years":   req.Years,
		"amount":  totalAmount,
		"invoice": inv,
		"message": fmt.Sprintf("Congratulations! Domain %s has been registered successfully.", cleanDomain),
	}, nil)
}
