package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestDomainRegistrarHandler_Flow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewDomainRegistrarHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         userID,
				OrganizationID: orgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Get("/domains/search", h.SearchDomains)
	r.Get("/domains/whois", h.WhoisLookup)
	r.Get("/domains/tlds", h.ListTLDs)
	r.Put("/domains/tlds/{tld}", h.UpdateTLD)
	r.Get("/domains/registrars", h.ListRegistrars)
	r.Put("/domains/registrars/{registrar}", h.UpdateRegistrar)
	r.Post("/domains/order", h.OrderDomain)

	// 1. Search domains
	t.Run("SearchDomains", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/domains/search?query=hostvratestdomain2026", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool                     `json:"success"`
			Data    []DomainSearchResultItem `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data) == 0 {
			t.Fatalf("expected search results, got 0")
		}
	})

	// 2. Whois lookup
	t.Run("WhoisLookup", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/domains/whois?domain=example.com", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool              `json:"success"`
			Data    store.WhoisRecord `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Data.Domain != "example.com" {
			t.Fatalf("expected example.com, got %s", resp.Data.Domain)
		}
	})

	// 3. List & Update TLD pricing
	t.Run("TLDPricing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/domains/tlds", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var listResp struct {
			Success bool               `json:"success"`
			Data    []store.TLDPricing `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(listResp.Data) == 0 {
			t.Fatalf("expected seeded TLDs, got 0")
		}

		// Update .com price
		updateBody := map[string]interface{}{
			"register_price": 14.50,
			"renew_price":    15.00,
			"transfer_price": 13.50,
			"enabled":        true,
			"is_popular":     true,
		}
		buf, _ := json.Marshal(updateBody)
		reqUpdate := httptest.NewRequest("PUT", "/domains/tlds/.com", bytes.NewReader(buf))
		reqUpdate.Header.Set("Content-Type", "application/json")
		recUpdate := httptest.NewRecorder()
		r.ServeHTTP(recUpdate, reqUpdate)

		if recUpdate.Code != http.StatusOK {
			t.Fatalf("expected 200 on update, got %d: %s", recUpdate.Code, recUpdate.Body.String())
		}
	})

	// 4. List & Update Registrars
	t.Run("Registrars", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/domains/registrars", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var regResp struct {
			Success bool                          `json:"success"`
			Data    []store.DomainRegistrarConfig `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &regResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(regResp.Data) == 0 {
			t.Fatalf("expected seeded registrars, got 0")
		}

		// Update Namecheap registrar
		updateBody := map[string]interface{}{
			"api_user":   "hostvra_admin",
			"api_key":    "nc_sec_key_123456",
			"sandbox":    true,
			"enabled":    true,
			"is_default": true,
		}
		buf, _ := json.Marshal(updateBody)
		reqUpdate := httptest.NewRequest("PUT", "/domains/registrars/namecheap", bytes.NewReader(buf))
		reqUpdate.Header.Set("Content-Type", "application/json")
		recUpdate := httptest.NewRecorder()
		r.ServeHTTP(recUpdate, reqUpdate)

		if recUpdate.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recUpdate.Code, recUpdate.Body.String())
		}
	})

	// 5. Order domain
	t.Run("OrderDomain", func(t *testing.T) {
		orderBody := map[string]interface{}{
			"domain":         "hostvratest2026.com",
			"action":         "register",
			"years":          1,
			"whois_privacy":  true,
			"auto_renew":     true,
			"client_name":    "Mizanur Rahman",
			"client_email":   "mizan@example.com",
			"client_phone":   "+8801700000000",
			"client_address": "Dhaka, Bangladesh",
			"payment_method": "bkash",
		}
		buf, _ := json.Marshal(orderBody)
		reqOrder := httptest.NewRequest("POST", "/domains/order", bytes.NewReader(buf))
		reqOrder.Header.Set("Content-Type", "application/json")
		recOrder := httptest.NewRecorder()
		r.ServeHTTP(recOrder, reqOrder)

		if recOrder.Code != http.StatusCreated {
			t.Fatalf("expected 201 created, got %d: %s", recOrder.Code, recOrder.Body.String())
		}

		var orderResp struct {
			Success bool `json:"success"`
			Data    struct {
				Domain  string        `json:"domain"`
				Years   int           `json:"years"`
				Amount  float64       `json:"amount"`
				Invoice store.Invoice `json:"invoice"`
				Message string        `json:"message"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recOrder.Body.Bytes(), &orderResp); err != nil {
			t.Fatalf("failed to decode order response: %v", err)
		}

		if orderResp.Data.Domain != "hostvratest2026.com" {
			t.Fatalf("expected domain hostvratest2026.com, got %s", orderResp.Data.Domain)
		}
		if orderResp.Data.Amount <= 0 {
			t.Fatalf("expected order amount > 0, got %f", orderResp.Data.Amount)
		}
	})

	// 4. Tenant Isolation Security Test (Section 28)
	t.Run("TenantIsolationSecurity", func(t *testing.T) {
		ownerUserID := uuid.New()
		attackerUserID := uuid.New()

		domainID := uuid.New()
		domainRecord := &store.Domain{
			ID:               domainID,
			UserID:           ownerUserID,
			DomainName:       "victim-domain.com",
			TLD:              "com",
			Registrar:        "resellerclub",
			ProviderOrderID:  "prov-12345",
			Status:           "active",
			RegistrarLock:    true,
		}
		_ = s.CreateDomain(context.Background(), domainRecord)

		authRouter := chi.NewRouter()
		var currentCtxUser uuid.UUID
		var currentCtxRole string

		authRouter.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
					UserID:         currentCtxUser,
					OrganizationID: uuid.New(),
					Role:           currentCtxRole,
				})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})

		authRouter.Get("/domains/{id}", h.GetDomain)
		authRouter.Get("/domains/{id}/nameservers", h.GetNameservers)
		authRouter.Get("/domains/{id}/dns", h.ListDNSRecords)
		authRouter.Get("/domains/{id}/lock", h.GetLock)
		authRouter.Get("/domains/{id}/epp-code", h.GetEPPCode)
		authRouter.Get("/domains/{id}/contacts", h.GetContacts)

		// Attacker (User B) attempts to access Victim (User A) endpoints
		currentCtxUser = attackerUserID
		currentCtxRole = "member"

		endpoints := []string{
			"/domains/" + domainID.String(),
			"/domains/" + domainID.String() + "/nameservers",
			"/domains/" + domainID.String() + "/dns",
			"/domains/" + domainID.String() + "/lock",
			"/domains/" + domainID.String() + "/epp-code",
			"/domains/" + domainID.String() + "/contacts",
		}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest("GET", endpoint, nil)
			rec := httptest.NewRecorder()
			authRouter.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("Security Breach! Expected 403 Forbidden for endpoint %s, but got %d: %s", endpoint, rec.Code, rec.Body.String())
			}
		}

		// Legitimate Owner (User A) access
		currentCtxUser = ownerUserID
		currentCtxRole = "client"

		reqOwner := httptest.NewRequest("GET", "/domains/"+domainID.String(), nil)
		recOwner := httptest.NewRecorder()
		authRouter.ServeHTTP(recOwner, reqOwner)
		if recOwner.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for legitimate owner, got %d", recOwner.Code)
		}
	})
}
