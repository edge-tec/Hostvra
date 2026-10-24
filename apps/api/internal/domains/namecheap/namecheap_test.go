package namecheap

import (
	"context"
	"testing"

	"hostvra/api/internal/domains"
)

func TestNamecheapRegistrar_FullLifecycle(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(Config{
		ApiUser:  "hostvra_reseller",
		ApiKey:   "test_mock_api_key_12345",
		UserName: "hostvra_reseller",
		ClientIP: "127.0.0.1",
		Mode:     "sandbox",
	})
	if err != nil {
		t.Fatalf("failed to create Namecheap client: %v", err)
	}

	// 1. Check availability
	avail, err := client.CheckAvailability(ctx, domains.AvailabilityRequest{
		DomainName: "hostvra-brand-test",
		TLDs:       []string{"com", "net", "org"},
	})
	if err != nil {
		t.Fatalf("check availability failed: %v", err)
	}
	if len(avail) != 3 {
		t.Fatalf("expected 3 availability results, got %d", len(avail))
	}
	if !avail[0].Available {
		t.Fatalf("expected domain to be available")
	}

	// 2. Register domain
	regResult, err := client.RegisterDomain(ctx, domains.RegisterDomainRequest{
		DomainName:  "hostvra-brand-test.com",
		Years:       1,
		Nameservers: []string{"ns1.hostvra.com", "ns2.hostvra.com"},
	})
	if err != nil {
		t.Fatalf("register domain failed: %v", err)
	}
	if regResult.Status != "active" || regResult.ProviderOrderID == "" {
		t.Fatalf("invalid registration result: %+v", regResult)
	}

	// 3. Renew domain
	renewResult, err := client.RenewDomain(ctx, domains.RenewDomainRequest{
		DomainName:      "hostvra-brand-test.com",
		ProviderOrderID: regResult.ProviderOrderID,
		Years:           1,
		CurrentExpiry:   &regResult.ExpiryDate,
	})
	if err != nil {
		t.Fatalf("renew domain failed: %v", err)
	}
	if renewResult.Status != "active" || renewResult.NewExpiryDate.Before(regResult.ExpiryDate) {
		t.Fatalf("invalid renewal result: %+v", renewResult)
	}

	// 4. Transfer domain
	transferResult, err := client.TransferDomain(ctx, domains.TransferDomainRequest{
		DomainName: "external-transfer.com",
		AuthCode:   "EPP_SECRET_12345",
	})
	if err != nil {
		t.Fatalf("transfer domain failed: %v", err)
	}
	if transferResult.Status != "submitted" {
		t.Fatalf("invalid transfer result: %+v", transferResult)
	}

	// 5. Test connection
	connResult, err := client.TestConnection(ctx)
	if err != nil {
		t.Fatalf("test connection failed: %v", err)
	}
	if !connResult.Connected || connResult.Provider != "namecheap" {
		t.Fatalf("invalid connection result: %+v", connResult)
	}
}
