package resellerclub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hostvra/api/internal/domains"
)

func TestResellerClubClient_CheckAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth credentials passed
		if r.URL.Query().Get("auth-userid") != "test-reseller-123" {
			t.Errorf("expected auth-userid test-reseller-123, got %s", r.URL.Query().Get("auth-userid"))
		}
		if r.URL.Query().Get("api-key") != "secret-api-key" {
			t.Errorf("expected api-key secret-api-key, got %s", r.URL.Query().Get("api-key"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"testexample.com": {"status": "available", "classkey": "domcno"},
			"testexample.net": {"status": "regthroughothers", "classkey": "domcno"}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		ResellerID: "test-reseller-123",
		APIKey:     "secret-api-key",
		Mode:       "sandbox",
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	results, err := client.CheckAvailability(context.Background(), domains.AvailabilityRequest{
		DomainName: "testexample",
		TLDs:       []string{"com", "net"},
	})
	if err != nil {
		t.Fatalf("check availability failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Domain != "testexample.com" || !results[0].Available {
		t.Errorf("expected testexample.com available, got %+v", results[0])
	}
	if results[1].Domain != "testexample.net" || results[1].Available {
		t.Errorf("expected testexample.net unavailable, got %+v", results[1])
	}
}

func TestResellerClubClient_ErrorParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": "ERROR", "message": "Authentication Failed. Invalid auth-userid or api-key."}`))
	}))
	defer server.Close()

	client, _ := NewClient(Config{
		ResellerID: "invalid-id",
		APIKey:     "wrong-key",
		Mode:       "sandbox",
		BaseURL:    server.URL,
	})

	_, err := client.CheckAvailability(context.Background(), domains.AvailabilityRequest{
		DomainName: "anydomain",
		TLDs:       []string{"com"},
	})
	if err == nil {
		t.Fatalf("expected authentication error, got nil")
	}

	if !strings.Contains(err.Error(), "registrar authentication failed") {
		t.Errorf("expected normalized auth error, got: %v", err)
	}
}

func TestResellerClubClient_RegisterDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "customers/details.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"customerid": "99991", "useremail": "client@example.com"}`))
			return
		}
		if strings.Contains(r.URL.Path, "contacts/add.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`"88881"`))
			return
		}
		if strings.Contains(r.URL.Path, "domains/register.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"actiontype": "AddNewDomain",
				"status": "Success",
				"entityid": "55551",
				"description": "myawesomecloud.com"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, _ := NewClient(Config{
		ResellerID: "test-reseller",
		APIKey:     "test-key",
		BaseURL:    server.URL,
	})

	res, err := client.RegisterDomain(context.Background(), domains.RegisterDomainRequest{
		DomainName:  "myawesomecloud.com",
		Years:       1,
		Nameservers: []string{"ns1.hostvra.com", "ns2.hostvra.com"},
		Registrant: &domains.ContactInfo{
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "client@example.com",
			Phone:      "15551234567",
			Address1:   "123 Street",
			City:       "Dallas",
			State:      "TX",
			PostalCode: "75001",
			Country:    "US",
		},
	})
	if err != nil {
		t.Fatalf("expected successful registration, got: %v", err)
	}

	if res.ProviderOrderID != "55551" {
		t.Errorf("expected entity ID 55551, got %s", res.ProviderOrderID)
	}
	if res.Status != "active" {
		t.Errorf("expected active status, got %s", res.Status)
	}
}

// TestResellerClubClient_EndpointProtection verifies Sandbox/Production endpoint isolation (Section 21)
func TestResellerClubClient_EndpointProtection(t *testing.T) {
	// 1. Sandbox mode defaults to test.httpapi.com
	sbClient, err := NewClient(Config{
		ResellerID: "12345",
		APIKey:     "testkey",
		Mode:       "sandbox",
	})
	if err != nil {
		t.Fatalf("unexpected error creating sandbox client: %v", err)
	}
	if sbClient.GetBaseURL() != DefaultSandboxBaseURL {
		t.Errorf("expected sandbox base URL %s, got %s", DefaultSandboxBaseURL, sbClient.GetBaseURL())
	}

	// 2. Production mode defaults to httpapi.com
	prodClient, err := NewClient(Config{
		ResellerID: "12345",
		APIKey:     "testkey",
		Mode:       "production",
	})
	if err != nil {
		t.Fatalf("unexpected error creating production client: %v", err)
	}
	if prodClient.GetBaseURL() != DefaultProductionBaseURL {
		t.Errorf("expected production base URL %s, got %s", DefaultProductionBaseURL, prodClient.GetBaseURL())
	}

	// 3. Sandbox mode cannot accidentally target production URL
	_, err = NewClient(Config{
		ResellerID: "12345",
		APIKey:     "testkey",
		Mode:       "sandbox",
		BaseURL:    "https://httpapi.com/api/",
	})
	if err == nil {
		t.Fatalf("expected error when configuring sandbox mode with production URL, got nil")
	}

	// 4. Production mode cannot accidentally target sandbox URL
	_, err = NewClient(Config{
		ResellerID: "12345",
		APIKey:     "testkey",
		Mode:       "production",
		BaseURL:    "https://test.httpapi.com/api/",
	})
	if err == nil {
		t.Fatalf("expected error when configuring production mode with sandbox URL, got nil")
	}
}

func TestResellerClubClient_CloudflareBlockParsing(t *testing.T) {
	cfHTML := `<!DOCTYPE html><html><head><title>Attention Required! | Cloudflare</title></head><body>
		<h1>Sorry, you have been blocked</h1>
		<div id="cf-footer-item-ip">Your IP: <span id="cf-footer-ip">13.140.157.238</span></div>
		<div>Cloudflare Ray ID: <strong class="font-semibold">a37db599ac4a5010</strong></div>
	</body></html>`

	err := ParseAPIError(403, []byte(cfHTML))
	if err == nil {
		t.Fatalf("expected error from ParseAPIError, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Cloudflare blocked API request (HTTP 403)") {
		t.Errorf("expected clean Cloudflare message, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "13.140.157.238") {
		t.Errorf("expected server IP to be extracted, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "a37db599ac4a5010") {
		t.Errorf("expected Ray ID to be extracted, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Authorized IP Addresses") {
		t.Errorf("expected instructions to whitelist IP, got: %s", errMsg)
	}
}

func TestResellerClubClient_TestConnectionCredentials(t *testing.T) {
	client, err := NewClient(Config{
		ResellerID: "",
		APIKey:     "",
		Mode:       "sandbox",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	res, err := client.TestConnection(context.Background())
	if err == nil {
		t.Fatalf("expected error when credentials missing, got nil")
	}
	if res == nil {
		t.Fatalf("expected structured ConnectionTestResult, got nil")
	}
	if res.Connected {
		t.Errorf("expected Connected=false")
	}
	if !strings.Contains(res.Message, "RESELLERCLUB_RESELLER_ID") {
		t.Errorf("expected helpful message regarding missing env vars, got: %s", res.Message)
	}
}

