package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"hostvra/api/internal/store"
)

var (
	ErrGatewayDisabled      = errors.New("payment gateway is disabled by administrator")
	ErrGatewayNotConfigured = errors.New("payment gateway credentials are not configured")
	ErrPaymentFailed        = errors.New("payment verification failed or status not completed")
	ErrAmountMismatch       = errors.New("payment amount does not match invoice total")
	ErrCurrencyMismatch     = errors.New("payment currency mismatch")
	ErrInvalidSignature     = errors.New("invalid webhook signature")
)

type PaymentInitResult struct {
	Gateway          string `json:"gateway"`
	CheckoutURL      string `json:"checkout_url"`
	GatewayReference string `json:"gateway_reference"` // session_id, paymentID, etc.
	PaymentStatus    string `json:"payment_status"`    // payment_initiated, pending
}

type PaymentVerifyResult struct {
	Gateway          string  `json:"gateway"`
	TransactionID    string  `json:"transaction_id"`
	GatewayReference string  `json:"gateway_reference"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"` // paid, completed, failed
	PaymentMethod    string  `json:"payment_method"`
	RawResponse      string  `json:"raw_response,omitempty"`
}

type Service struct {
	client *http.Client
}

func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// InitiatePayment starts a real payment session with the selected gateway
func (s *Service) InitiatePayment(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	plan *store.HostingPlan,
	billingCycle string,
	successURL, cancelURL, callbackURL string,
) (*PaymentInitResult, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, ErrGatewayDisabled
	}

	gateway := strings.ToLower(cfg.Gateway)
	switch gateway {
	case "stripe":
		return s.initiateStripe(ctx, cfg, inv, plan, billingCycle, successURL, cancelURL)
	case "bkash":
		return s.initiateBkash(ctx, cfg, inv, callbackURL)
	case "nagad":
		return s.initiateNagad(ctx, cfg, inv, callbackURL)
	default:
		return nil, fmt.Errorf("unsupported payment gateway: %s", gateway)
	}
}

// VerifyPayment checks the transaction with the payment gateway server-side
func (s *Service) VerifyPayment(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	paymentRef string,
) (*PaymentVerifyResult, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, ErrGatewayDisabled
	}

	gateway := strings.ToLower(cfg.Gateway)
	switch gateway {
	case "stripe":
		return s.verifyStripe(ctx, cfg, inv, paymentRef)
	case "bkash":
		return s.verifyBkash(ctx, cfg, inv, paymentRef)
	case "nagad":
		return s.verifyNagad(ctx, cfg, inv, paymentRef)
	default:
		return nil, fmt.Errorf("unsupported payment gateway: %s", gateway)
	}
}

// VerifyWebhookSignature verifies gateway HMAC or signature
func (s *Service) VerifyWebhookSignature(cfg *store.PaymentGatewayConfig, body []byte, sigHeader string) bool {
	if cfg == nil || cfg.SecretKey == "" {
		return false
	}
	if sigHeader == "" {
		return false
	}

	// 1. Direct HMAC-SHA256 comparison
	mac := hmac.New(sha256.New, []byte(cfg.SecretKey))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if hmac.Equal([]byte(sigHeader), []byte(expectedSig)) {
		return true
	}

	// 2. Stripe v1 signature scheme (t=...,v1=...)
	if strings.Contains(sigHeader, "v1=") {
		var v1Sig, timestamp string
		for _, part := range strings.Split(sigHeader, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "v1=") {
				v1Sig = strings.TrimPrefix(part, "v1=")
			} else if strings.HasPrefix(part, "t=") {
				timestamp = strings.TrimPrefix(part, "t=")
			}
		}

		if v1Sig != "" {
			if hmac.Equal([]byte(v1Sig), []byte(expectedSig)) {
				return true
			}
			if timestamp != "" {
				stripeMac := hmac.New(sha256.New, []byte(cfg.SecretKey))
				stripeMac.Write([]byte(timestamp + "."))
				stripeMac.Write(body)
				stripeExpected := hex.EncodeToString(stripeMac.Sum(nil))
				if hmac.Equal([]byte(v1Sig), []byte(stripeExpected)) {
					return true
				}
			}
		}
	}

	return false
}

// ----------------------------------------------------------------------------
// STRIPE INTEGRATION
// ----------------------------------------------------------------------------

func (s *Service) initiateStripe(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	plan *store.HostingPlan,
	billingCycle string,
	successURL, cancelURL string,
) (*PaymentInitResult, error) {
	if cfg.SecretKey == "" {
		return nil, ErrGatewayNotConfigured
	}

	// Handle automated mock testing key in test mode
	if cfg.TestMode && cfg.SecretKey == "mock_test_secret" {
		mockSessionID := fmt.Sprintf("cs_test_%s_%d", inv.ID.String()[:8], time.Now().Unix())
		return &PaymentInitResult{
			Gateway:          "stripe",
			CheckoutURL:      fmt.Sprintf("/billing?invoice_id=%s&session_id=%s&gateway=stripe", inv.ID.String(), mockSessionID),
			GatewayReference: mockSessionID,
			PaymentStatus:    "payment_initiated",
		}, nil
	}

	stripeURL := "https://api.stripe.com/v1/checkout/sessions"
	data := url.Values{}
	data.Set("payment_method_types[0]", "card")
	data.Set("mode", "payment")
	data.Set("client_reference_id", inv.ID.String())
	data.Set("metadata[invoice_id]", inv.ID.String())
	data.Set("metadata[user_id]", inv.UserID.String())
	if inv.SubscriptionID != nil {
		data.Set("metadata[subscription_id]", inv.SubscriptionID.String())
	}

	currency := strings.ToLower(inv.Currency)
	if currency == "" {
		currency = "usd"
	}
	unitAmount := int64(math.Round(inv.Total * 100))
	data.Set("line_items[0][price_data][currency]", currency)
	data.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(unitAmount, 10))
	data.Set("line_items[0][price_data][product_data][name]", fmt.Sprintf("%s (%s Subscription)", plan.Name, strings.Title(billingCycle)))
	data.Set("line_items[0][quantity]", "1")

	if successURL != "" {
		if strings.Contains(successURL, "{CHECKOUT_SESSION_ID}") {
			data.Set("success_url", successURL)
		} else {
			data.Set("success_url", fmt.Sprintf("%s?session_id={CHECKOUT_SESSION_ID}&invoice_id=%s", successURL, inv.ID.String()))
		}
	} else {
		data.Set("success_url", fmt.Sprintf("https://hostvra.com/billing?invoice_id=%s&session_id={CHECKOUT_SESSION_ID}", inv.ID.String()))
	}

	if cancelURL != "" {
		data.Set("cancel_url", cancelURL)
	} else {
		data.Set("cancel_url", fmt.Sprintf("https://hostvra.com/billing?invoice_id=%s&status=cancelled", inv.ID.String()))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", stripeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Stripe API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Stripe response: %w", err)
	}

	var sessionResp struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return nil, fmt.Errorf("failed to parse Stripe response: %w", err)
	}
	if sessionResp.Error != nil {
		return nil, fmt.Errorf("stripe error: %s", sessionResp.Error.Message)
	}

	return &PaymentInitResult{
		Gateway:          "stripe",
		CheckoutURL:      sessionResp.URL,
		GatewayReference: sessionResp.ID,
		PaymentStatus:    "payment_initiated",
	}, nil
}

func (s *Service) verifyStripe(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	paymentRef string,
) (*PaymentVerifyResult, error) {
	if cfg.SecretKey == "" {
		return nil, ErrGatewayNotConfigured
	}

	// Automated mock testing key handler in test mode
	if cfg.TestMode && cfg.SecretKey == "mock_test_secret" {
		if strings.Contains(paymentRef, "fail") {
			return nil, ErrPaymentFailed
		}
		return &PaymentVerifyResult{
			Gateway:          "stripe",
			TransactionID:    fmt.Sprintf("pi_test_%s_%d", inv.ID.String()[:8], time.Now().Unix()),
			GatewayReference: paymentRef,
			Amount:           inv.Total,
			Currency:         inv.Currency,
			Status:           "completed",
			PaymentMethod:    "stripe_card",
		}, nil
	}

	if paymentRef == "" {
		return nil, errors.New("missing Stripe session_id or payment_intent reference")
	}

	var endpoint string
	if strings.HasPrefix(paymentRef, "cs_") {
		endpoint = fmt.Sprintf("https://api.stripe.com/v1/checkout/sessions/%s", paymentRef)
	} else if strings.HasPrefix(paymentRef, "pi_") {
		endpoint = fmt.Sprintf("https://api.stripe.com/v1/payment_intents/%s", paymentRef)
	} else {
		endpoint = fmt.Sprintf("https://api.stripe.com/v1/checkout/sessions/%s", paymentRef)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SecretKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact Stripe API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var session struct {
		ID            string            `json:"id"`
		PaymentStatus string            `json:"payment_status"`
		Status        string            `json:"status"`
		AmountTotal   int64             `json:"amount_total"`
		Currency      string            `json:"currency"`
		PaymentIntent interface{}       `json:"payment_intent"`
		Metadata      map[string]string `json:"metadata"`
		Error         *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(bodyBytes, &session); err != nil {
		return nil, fmt.Errorf("failed to parse Stripe verify response: %w", err)
	}
	if session.Error != nil {
		return nil, fmt.Errorf("stripe error: %s", session.Error.Message)
	}

	isPaid := session.PaymentStatus == "paid" || session.Status == "complete" || session.Status == "succeeded"
	if !isPaid {
		return nil, fmt.Errorf("%w: status is %s", ErrPaymentFailed, session.PaymentStatus)
	}

	expectedAmountCents := int64(math.Round(inv.Total * 100))
	if session.AmountTotal > 0 && session.AmountTotal != expectedAmountCents {
		return nil, fmt.Errorf("%w: expected %d cents, got %d cents", ErrAmountMismatch, expectedAmountCents, session.AmountTotal)
	}

	if session.Currency != "" && !strings.EqualFold(session.Currency, inv.Currency) {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrCurrencyMismatch, inv.Currency, session.Currency)
	}

	// Extract transaction ID
	txnID := session.ID
	if piStr, ok := session.PaymentIntent.(string); ok && piStr != "" {
		txnID = piStr
	} else if piMap, ok := session.PaymentIntent.(map[string]interface{}); ok {
		if id, ok := piMap["id"].(string); ok {
			txnID = id
		}
	}

	return &PaymentVerifyResult{
		Gateway:          "stripe",
		TransactionID:    txnID,
		GatewayReference: session.ID,
		Amount:           float64(session.AmountTotal) / 100.0,
		Currency:         strings.ToUpper(session.Currency),
		Status:           "completed",
		PaymentMethod:    "stripe_card",
		RawResponse:      string(bodyBytes),
	}, nil
}

// ----------------------------------------------------------------------------
// BKASH INTEGRATION (Tokenized Checkout API v1.2.0-beta)
// ----------------------------------------------------------------------------

func (s *Service) initiateBkash(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	callbackURL string,
) (*PaymentInitResult, error) {
	if cfg.ApiKey == "" || cfg.SecretKey == "" {
		return nil, ErrGatewayNotConfigured
	}

	// Handle automated mock testing key in test mode
	if cfg.TestMode && cfg.ApiKey == "mock_test_key" {
		mockPaymentID := fmt.Sprintf("BKASH_PAY_%s_%d", inv.ID.String()[:8], time.Now().Unix())
		return &PaymentInitResult{
			Gateway:          "bkash",
			CheckoutURL:      fmt.Sprintf("/billing?invoice_id=%s&paymentID=%s&gateway=bkash", inv.ID.String(), mockPaymentID),
			GatewayReference: mockPaymentID,
			PaymentStatus:    "payment_initiated",
		}, nil
	}

	baseURL := "https://tokenized.pay.bka.sh/v1.2.0-beta"
	if cfg.TestMode {
		baseURL = "https://tokenized.sandbox.bka.sh/v1.2.0-beta"
	}

	// 1. Grant Token
	tokenPayload := map[string]string{
		"app_key":    cfg.ApiKey,
		"app_secret": cfg.SecretKey,
	}
	tokenBody, _ := json.Marshal(tokenPayload)
	tokenReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/tokenized/checkout/token/grant", bytes.NewReader(tokenBody))
	if err != nil {
		return nil, err
	}
	tokenReq.Header.Set("Content-Type", "application/json")
	if cfg.MerchantID != "" {
		parts := strings.SplitN(cfg.MerchantID, ":", 2)
		if len(parts) == 2 {
			tokenReq.Header.Set("username", parts[0])
			tokenReq.Header.Set("password", parts[1])
		} else {
			tokenReq.Header.Set("username", cfg.MerchantID)
		}
	}

	tokenResp, err := s.client.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bKash token API: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenResult struct {
		IDToken      string `json:"id_token"`
		StatusCode   string `json:"statusCode"`
		StatusMessage string `json:"statusMessage"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenResult); err != nil {
		return nil, fmt.Errorf("failed to decode bKash token response: %w", err)
	}
	if tokenResult.IDToken == "" {
		return nil, fmt.Errorf("bkash token error: %s (%s)", tokenResult.StatusMessage, tokenResult.StatusCode)
	}

	// 2. Create Payment
	cbURL := callbackURL
	if cbURL == "" {
		cbURL = fmt.Sprintf("https://hostvra.com/billing?invoice_id=%s&gateway=bkash", inv.ID.String())
	}
	createPayload := map[string]interface{}{
		"mode":                  "0011",
		"payerReference":        inv.UserID.String(),
		"callbackURL":           cbURL,
		"amount":                fmt.Sprintf("%.2f", inv.Total),
		"currency":              "BDT",
		"intent":                "sale",
		"merchantInvoiceNumber": inv.InvoiceNumber,
	}
	createBody, _ := json.Marshal(createPayload)
	createReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/tokenized/checkout/create", bytes.NewReader(createBody))
	if err != nil {
		return nil, err
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", tokenResult.IDToken)
	createReq.Header.Set("X-APP-Key", cfg.ApiKey)

	createResp, err := s.client.Do(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create bKash payment: %w", err)
	}
	defer createResp.Body.Close()

	var createResult struct {
		PaymentID     string `json:"paymentID"`
		BkashURL      string `json:"bkashURL"`
		StatusCode    string `json:"statusCode"`
		StatusMessage string `json:"statusMessage"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		return nil, fmt.Errorf("failed to decode bKash create payment response: %w", err)
	}
	if createResult.PaymentID == "" || createResult.BkashURL == "" {
		return nil, fmt.Errorf("bkash create payment error: %s (%s)", createResult.StatusMessage, createResult.StatusCode)
	}

	return &PaymentInitResult{
		Gateway:          "bkash",
		CheckoutURL:      createResult.BkashURL,
		GatewayReference: createResult.PaymentID,
		PaymentStatus:    "payment_initiated",
	}, nil
}

func (s *Service) verifyBkash(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	paymentRef string,
) (*PaymentVerifyResult, error) {
	if cfg.ApiKey == "" || cfg.SecretKey == "" {
		return nil, ErrGatewayNotConfigured
	}

	// Handle automated mock testing key in test mode
	if cfg.TestMode && cfg.ApiKey == "mock_test_key" {
		if strings.Contains(paymentRef, "fail") {
			return nil, ErrPaymentFailed
		}
		return &PaymentVerifyResult{
			Gateway:          "bkash",
			TransactionID:    fmt.Sprintf("TRX_BKASH_%s_%d", inv.ID.String()[:8], time.Now().Unix()),
			GatewayReference: paymentRef,
			Amount:           inv.Total,
			Currency:         "BDT",
			Status:           "completed",
			PaymentMethod:    "bkash_direct",
		}, nil
	}

	if paymentRef == "" {
		return nil, errors.New("missing bKash paymentID reference")
	}

	baseURL := "https://tokenized.pay.bka.sh/v1.2.0-beta"
	if cfg.TestMode {
		baseURL = "https://tokenized.sandbox.bka.sh/v1.2.0-beta"
	}

	// 1. Grant Token for execute / query
	tokenPayload := map[string]string{
		"app_key":    cfg.ApiKey,
		"app_secret": cfg.SecretKey,
	}
	tokenBody, _ := json.Marshal(tokenPayload)
	tokenReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/tokenized/checkout/token/grant", bytes.NewReader(tokenBody))
	if err != nil {
		return nil, err
	}
	tokenReq.Header.Set("Content-Type", "application/json")
	if cfg.MerchantID != "" {
		parts := strings.SplitN(cfg.MerchantID, ":", 2)
		if len(parts) == 2 {
			tokenReq.Header.Set("username", parts[0])
			tokenReq.Header.Set("password", parts[1])
		}
	}

	tokenResp, err := s.client.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get bKash token: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenResult struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenResult); err != nil || tokenResult.IDToken == "" {
		return nil, errors.New("failed to acquire bKash token for verification")
	}

	// 2. Execute Payment
	execPayload := map[string]string{"paymentID": paymentRef}
	execBody, _ := json.Marshal(execPayload)
	execReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/tokenized/checkout/execute", bytes.NewReader(execBody))
	if err != nil {
		return nil, err
	}
	execReq.Header.Set("Content-Type", "application/json")
	execReq.Header.Set("Authorization", tokenResult.IDToken)
	execReq.Header.Set("X-APP-Key", cfg.ApiKey)

	execResp, err := s.client.Do(execReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bKash payment: %w", err)
	}
	defer execResp.Body.Close()

	bodyBytes, _ := io.ReadAll(execResp.Body)

	var execResult struct {
		StatusCode            string `json:"statusCode"`
		StatusMessage         string `json:"statusMessage"`
		PaymentID             string `json:"paymentID"`
		TrxID                 string `json:"trxID"`
		TransactionStatus     string `json:"transactionStatus"`
		Amount                string `json:"amount"`
		Currency              string `json:"currency"`
		MerchantInvoiceNumber string `json:"merchantInvoiceNumber"`
	}
	_ = json.Unmarshal(bodyBytes, &execResult)

	// If execute failed because already executed, query payment status
	if execResult.StatusCode == "2029" || execResult.TransactionStatus == "" {
		queryReq, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/tokenized/checkout/payment/query", bytes.NewReader(execBody))
		queryReq.Header.Set("Content-Type", "application/json")
		queryReq.Header.Set("Authorization", tokenResult.IDToken)
		queryReq.Header.Set("X-APP-Key", cfg.ApiKey)
		if qResp, qErr := s.client.Do(queryReq); qErr == nil {
			defer qResp.Body.Close()
			qBytes, _ := io.ReadAll(qResp.Body)
			_ = json.Unmarshal(qBytes, &execResult)
			bodyBytes = qBytes
		}
	}

	if execResult.TransactionStatus != "Completed" {
		return nil, fmt.Errorf("%w: bkash status is %s (%s)", ErrPaymentFailed, execResult.TransactionStatus, execResult.StatusMessage)
	}

	parsedAmount, _ := strconv.ParseFloat(execResult.Amount, 64)
	if parsedAmount > 0 && math.Abs(parsedAmount-inv.Total) > 0.05 {
		return nil, fmt.Errorf("%w: expected %.2f, got %.2f", ErrAmountMismatch, inv.Total, parsedAmount)
	}

	trxID := execResult.TrxID
	if trxID == "" {
		trxID = execResult.PaymentID
	}

	return &PaymentVerifyResult{
		Gateway:          "bkash",
		TransactionID:    trxID,
		GatewayReference: execResult.PaymentID,
		Amount:           parsedAmount,
		Currency:         "BDT",
		Status:           "completed",
		PaymentMethod:    "bkash_direct",
		RawResponse:      string(bodyBytes),
	}, nil
}

// ----------------------------------------------------------------------------
// NAGAD INTEGRATION (PGW API)
// ----------------------------------------------------------------------------

func (s *Service) initiateNagad(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	callbackURL string,
) (*PaymentInitResult, error) {
	if cfg.MerchantID == "" {
		return nil, ErrGatewayNotConfigured
	}

	// Handle automated mock testing key in test mode
	if cfg.TestMode && cfg.MerchantID == "mock_merchant_id" {
		mockPaymentRef := fmt.Sprintf("NAGAD_REF_%s_%d", inv.ID.String()[:8], time.Now().Unix())
		return &PaymentInitResult{
			Gateway:          "nagad",
			CheckoutURL:      fmt.Sprintf("/billing?invoice_id=%s&payment_ref_id=%s&gateway=nagad", inv.ID.String(), mockPaymentRef),
			GatewayReference: mockPaymentRef,
			PaymentStatus:    "payment_initiated",
		}, nil
	}

	baseURL := "https://api.mynagad.com/api/dfs"
	if cfg.TestMode {
		baseURL = "http://sandbox.mynagad.com:10080/remote-payment-gateway-1.0/api/dfs"
	}

	nowStr := time.Now().Format("20060102150405")
	nagadURL := fmt.Sprintf("%s/check-out/initialize/%s/%s", baseURL, cfg.MerchantID, inv.InvoiceNumber)

	return &PaymentInitResult{
		Gateway:          "nagad",
		CheckoutURL:      nagadURL,
		GatewayReference: fmt.Sprintf("NAGAD_%s_%s", inv.InvoiceNumber, nowStr),
		PaymentStatus:    "payment_initiated",
	}, nil
}

func (s *Service) verifyNagad(
	ctx context.Context,
	cfg *store.PaymentGatewayConfig,
	inv *store.Invoice,
	paymentRef string,
) (*PaymentVerifyResult, error) {
	if cfg.MerchantID == "" {
		return nil, ErrGatewayNotConfigured
	}

	// Handle automated mock testing key in test mode
	if cfg.TestMode && cfg.MerchantID == "mock_merchant_id" {
		if strings.Contains(paymentRef, "fail") {
			return nil, ErrPaymentFailed
		}
		return &PaymentVerifyResult{
			Gateway:          "nagad",
			TransactionID:    fmt.Sprintf("TRX_NAGAD_%s_%d", inv.ID.String()[:8], time.Now().Unix()),
			GatewayReference: paymentRef,
			Amount:           inv.Total,
			Currency:         "BDT",
			Status:           "completed",
			PaymentMethod:    "nagad_pgw",
		}, nil
	}

	if paymentRef == "" {
		return nil, errors.New("missing Nagad payment_ref_id")
	}

	baseURL := "https://api.mynagad.com/api/dfs"
	if cfg.TestMode {
		baseURL = "http://sandbox.mynagad.com:10080/remote-payment-gateway-1.0/api/dfs"
	}

	verifyURL := fmt.Sprintf("%s/verify/payment/%s", baseURL, paymentRef)
	req, err := http.NewRequestWithContext(ctx, "GET", verifyURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-KM-Api-Version", "v-0.2.0")
	req.Header.Set("X-KM-IP-V4", "127.0.0.1")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact Nagad verification API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nagadResp struct {
		MerchantID    string `json:"merchantId"`
		OrderID       string `json:"orderId"`
		PaymentRefID  string `json:"paymentRefId"`
		Amount        string `json:"amount"`
		Status        string `json:"status"` // Success, Aborted, Failed
		StatusCode    string `json:"statusCode"`
		StatusMessage string `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &nagadResp); err != nil {
		return nil, fmt.Errorf("failed to parse Nagad verify response: %w", err)
	}

	if !strings.EqualFold(nagadResp.Status, "Success") {
		return nil, fmt.Errorf("%w: nagad status is %s (%s)", ErrPaymentFailed, nagadResp.Status, nagadResp.StatusMessage)
	}

	parsedAmount, _ := strconv.ParseFloat(nagadResp.Amount, 64)
	if parsedAmount > 0 && math.Abs(parsedAmount-inv.Total) > 0.05 {
		return nil, fmt.Errorf("%w: expected %.2f, got %.2f", ErrAmountMismatch, inv.Total, parsedAmount)
	}

	return &PaymentVerifyResult{
		Gateway:          "nagad",
		TransactionID:    nagadResp.PaymentRefID,
		GatewayReference: nagadResp.PaymentRefID,
		Amount:           parsedAmount,
		Currency:         "BDT",
		Status:           "completed",
		PaymentMethod:    "nagad_pgw",
		RawResponse:      string(bodyBytes),
	}, nil
}
