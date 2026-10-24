package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

func TestPaymentGateways_AllFiveInitiateAndVerify(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	invoice := &store.Invoice{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		Total:    25.00,
		Currency: "USD",
	}

	plan := &store.HostingPlan{
		ID:   uuid.New(),
		Name: "Pro Cloud Plan",
	}

	tests := []struct {
		name        string
		cfg         *store.PaymentGatewayConfig
		invCurrency string
		expectedGw  string
	}{
		{
			name: "Stripe Gateway",
			cfg: &store.PaymentGatewayConfig{
				Gateway:   "stripe",
				Enabled:   true,
				TestMode:  true,
				SecretKey: "mock_test_secret",
			},
			invCurrency: "USD",
			expectedGw:  "stripe",
		},
		{
			name: "bKash Direct API",
			cfg: &store.PaymentGatewayConfig{
				Gateway:   "bkash",
				Enabled:   true,
				TestMode:  true,
				ApiKey:    "mock_test_key",
				SecretKey: "mock_test_secret",
			},
			invCurrency: "BDT",
			expectedGw:  "bkash",
		},
		{
			name: "Nagad PGW",
			cfg: &store.PaymentGatewayConfig{
				Gateway:    "nagad",
				Enabled:    true,
				TestMode:   true,
				MerchantID: "mock_merchant_id",
			},
			invCurrency: "BDT",
			expectedGw:  "nagad",
		},
		{
			name: "SSLCommerz Multi-Channel",
			cfg: &store.PaymentGatewayConfig{
				Gateway:    "sslcommerz",
				Enabled:    true,
				TestMode:   true,
				MerchantID: "mock_merchant_id",
				SecretKey:  "mock_test_secret",
			},
			invCurrency: "BDT",
			expectedGw:  "sslcommerz",
		},
		{
			name: "PayPal Express",
			cfg: &store.PaymentGatewayConfig{
				Gateway:   "paypal",
				Enabled:   true,
				TestMode:  true,
				ApiKey:    "mock_test_key",
				SecretKey: "mock_test_secret",
			},
			invCurrency: "USD",
			expectedGw:  "paypal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			invoice.Currency = tc.invCurrency
			initRes, err := service.InitiatePayment(ctx, tc.cfg, invoice, plan, "monthly", "http://localhost:3000/success", "http://localhost:3000/cancel", "http://localhost:8080/api/v1/billing/webhook")
			if err != nil {
				t.Fatalf("InitiatePayment failed for %s: %v", tc.expectedGw, err)
			}
			if initRes.Gateway != tc.expectedGw {
				t.Errorf("Expected gateway %s, got %s", tc.expectedGw, initRes.Gateway)
			}
			if initRes.CheckoutURL == "" {
				t.Errorf("Expected non-empty CheckoutURL for %s", tc.expectedGw)
			}
			if initRes.GatewayReference == "" {
				t.Errorf("Expected non-empty GatewayReference for %s", tc.expectedGw)
			}

			// Verify Payment
			verifyRes, err := service.VerifyPayment(ctx, tc.cfg, invoice, initRes.GatewayReference)
			if err != nil {
				t.Fatalf("VerifyPayment failed for %s: %v", tc.expectedGw, err)
			}
			if verifyRes.Gateway != tc.expectedGw {
				t.Errorf("Expected verify gateway %s, got %s", tc.expectedGw, verifyRes.Gateway)
			}
			if verifyRes.Status != "completed" && verifyRes.Status != "paid" {
				t.Errorf("Expected status completed/paid for %s, got %s", tc.expectedGw, verifyRes.Status)
			}
		})
	}
}
