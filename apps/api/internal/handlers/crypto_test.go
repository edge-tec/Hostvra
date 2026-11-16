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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/crypto"
	"hostvra/api/internal/store"
)

func setupTestCryptoHandler(t *testing.T) (*CryptoHandler, store.Store) {
	st := store.NewMemoryStoreWithPath(t.TempDir() + "/store.json")
	cfg := &config.Config{
		JWTSecret: "test-secret-32-chars-long-12345678",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(st, logger)

	cryptoSvc := crypto.NewCryptoService(st, func(ctx context.Context, invoice *store.CryptoPaymentInvoice) error {
		return nil
	})

	// Pre-seed an available deposit address
	_ = st.CreateDepositAddress(context.Background(), &store.CryptoDepositAddress{
		ID:         uuid.New(),
		NetworkID:  "tron",
		CurrencyID: "USDT_TRC20",
		Address:    "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn",
		Status:     "available",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})

	handler := NewCryptoHandler(cfg, st, auditLogger, cryptoSvc)
	return handler, st
}

func TestCryptoHandler_ListCurrencies(t *testing.T) {
	h, _ := setupTestCryptoHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/crypto/currencies", nil)
	w := httptest.NewRecorder()

	h.ListCurrencies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.Data) == 0 {
		t.Fatalf("expected currencies list, got 0")
	}
}

func TestCryptoHandler_CreateAndGetInvoice(t *testing.T) {
	h, _ := setupTestCryptoHandler(t)

	// 1. Create Invoice
	body := map[string]interface{}{
		"payable_type":    "invoice",
		"payable_id":      "INV-9999",
		"fiat_amount_usd": 100.0,
		"network_id":      "tron",
		"currency_id":     "USDT_TRC20",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/crypto/invoices", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.CreateInvoice(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createRes struct {
		Data struct {
			InvoiceCode    string  `json:"invoice_code"`
			DepositAddress string  `json:"deposit_address"`
			CryptoAmount   float64 `json:"crypto_amount"`
			Status         string  `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createRes); err != nil {
		t.Fatalf("failed to parse invoice creation response: %v", err)
	}

	code := createRes.Data.InvoiceCode
	if code == "" || createRes.Data.DepositAddress == "" {
		t.Fatalf("expected non-empty invoice code & address, got: %+v", createRes.Data)
	}

	// 2. Get Invoice by Code
	r := chi.NewRouter()
	r.Get("/api/v1/crypto/invoices/{code}", h.GetInvoice)

	getReq := httptest.NewRequest("GET", "/api/v1/crypto/invoices/"+code, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on get invoice, got %d: %s", getW.Code, getW.Body.String())
	}

	// 3. Verify Payment with Mock TxHash
	r.Post("/api/v1/crypto/invoices/{code}/verify", h.VerifyInvoice)
	verifyBody := map[string]interface{}{
		"tx_hash": "0xabc123",
	}
	vBytes, _ := json.Marshal(verifyBody)
	vReq := httptest.NewRequest("POST", "/api/v1/crypto/invoices/"+code+"/verify", bytes.NewReader(vBytes))
	vW := httptest.NewRecorder()
	r.ServeHTTP(vW, vReq)

	if vW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on verify, got %d: %s", vW.Code, vW.Body.String())
	}
}

func TestCryptoHandler_AdminEndpoints(t *testing.T) {
	h, _ := setupTestCryptoHandler(t)

	// List Invoices
	req := httptest.NewRequest("GET", "/api/v1/admin/crypto/invoices", nil)
	w := httptest.NewRecorder()
	h.AdminListInvoices(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on AdminListInvoices, got %d", w.Code)
	}

	// Save Treasury Wallet
	treasuryBody := map[string]interface{}{
		"network_id": "tron",
		"name":       "Trust Wallet Primary",
		"address":    "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn",
		"is_active":  true,
	}
	tBytes, _ := json.Marshal(treasuryBody)
	tReq := httptest.NewRequest("POST", "/api/v1/admin/crypto/treasuries", bytes.NewReader(tBytes))
	tW := httptest.NewRecorder()
	h.AdminSaveTreasury(tW, tReq)
	if tW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on AdminSaveTreasury, got %d: %s", tW.Code, tW.Body.String())
	}

	// Settings Get & Update
	sReq := httptest.NewRequest("GET", "/api/v1/admin/crypto/settings", nil)
	sW := httptest.NewRecorder()
	h.AdminGetSettings(sW, sReq)
	if sW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on AdminGetSettings, got %d", sW.Code)
	}

	uBody := map[string]string{
		"invoice_expiry_minutes": "45",
	}
	uBytes, _ := json.Marshal(uBody)
	uReq := httptest.NewRequest("PUT", "/api/v1/admin/crypto/settings", bytes.NewReader(uBytes))
	uW := httptest.NewRecorder()
	h.AdminUpdateSettings(uW, uReq)
	if uW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on AdminUpdateSettings, got %d", uW.Code)
	}

	// 1-Click Quick Setup
	quickBody := map[string]string{
		"tron_address": "TYDzsYUE4jEJTXsyNDoja2d1M7T4ZkYjHn",
		"evm_address":  "0x55d398326f99059fF775485246999027B3197955",
		"btc_address":  "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
	}
	qBytes, _ := json.Marshal(quickBody)
	qReq := httptest.NewRequest("POST", "/api/v1/admin/crypto/treasuries/quick-setup", bytes.NewReader(qBytes))
	qW := httptest.NewRecorder()
	h.AdminQuickSetupTreasuries(qW, qReq)
	if qW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on AdminQuickSetupTreasuries, got %d: %s", qW.Code, qW.Body.String())
	}

	// Verify all networks are active now
	listReq := httptest.NewRequest("GET", "/api/v1/crypto/currencies", nil)
	listW := httptest.NewRecorder()
	h.ListCurrencies(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on ListCurrencies, got %d", listW.Code)
	}
	var currenciesRes []map[string]interface{}
	if err := json.Unmarshal(listW.Body.Bytes(), &currenciesRes); err != nil {
		t.Fatalf("failed to decode currencies response: %v", err)
	}
	hasActive := false
	for _, c := range currenciesRes {
		if hasT, ok := c["has_treasury"].(bool); ok && hasT {
			hasActive = true
			break
		}
	}
	if !hasActive {
		t.Fatalf("expected at least one currency to have has_treasury = true after quick setup")
	}
}
