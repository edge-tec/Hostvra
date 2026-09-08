package domains

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type mockRegistrar struct {
	availableResult []DomainAvailability
	registeredOrder string
}

func (m *mockRegistrar) CheckAvailability(ctx context.Context, req AvailabilityRequest) ([]DomainAvailability, error) {
	return m.availableResult, nil
}

func (m *mockRegistrar) RegisterDomain(ctx context.Context, req RegisterDomainRequest) (*DomainRegistrationResult, error) {
	return &DomainRegistrationResult{
		DomainName:       req.DomainName,
		ProviderOrderID:  "provider-order-12345",
		ProviderDomainID: "provider-order-12345",
		Status:           "active",
		Nameservers:      req.Nameservers,
	}, nil
}

func (m *mockRegistrar) RenewDomain(ctx context.Context, req RenewDomainRequest) (*DomainRenewalResult, error) {
	return &DomainRenewalResult{
		DomainName:      req.DomainName,
		ProviderOrderID: req.ProviderOrderID,
		Status:          "completed",
	}, nil
}

func (m *mockRegistrar) TransferDomain(ctx context.Context, req TransferDomainRequest) (*DomainTransferResult, error) {
	return &DomainTransferResult{
		DomainName:      req.DomainName,
		ProviderOrderID: "transfer-12345",
		Status:          "submitted",
	}, nil
}

func (m *mockRegistrar) GetDomainInfo(ctx context.Context, domain string) (*DomainInfo, error) {
	return &DomainInfo{
		DomainName:      domain,
		ProviderOrderID: "provider-order-12345",
		Status:          "active",
		Nameservers:     []string{"ns1.hostvra.com", "ns2.hostvra.com"},
		RegistrarLock:   true,
	}, nil
}

func (m *mockRegistrar) UpdateNameservers(ctx context.Context, req NameserverUpdateRequest) error {
	return nil
}

func (m *mockRegistrar) GetNameservers(ctx context.Context, domain string) ([]string, error) {
	return []string{"ns1.hostvra.com", "ns2.hostvra.com"}, nil
}

func (m *mockRegistrar) GetRegistrarLock(ctx context.Context, domain string) (bool, error) {
	return true, nil
}

func (m *mockRegistrar) SetRegistrarLock(ctx context.Context, domain string, locked bool) error {
	return nil
}

func (m *mockRegistrar) GetEPPCode(ctx context.Context, domain string) (string, error) {
	return "SECURE-AUTH-KEY-123", nil
}

func (m *mockRegistrar) GetContacts(ctx context.Context, domain string) (*DomainContacts, error) {
	return &DomainContacts{}, nil
}

func (m *mockRegistrar) UpdateContacts(ctx context.Context, req ContactUpdateRequest) error {
	return nil
}

func (m *mockRegistrar) GetDNSRecords(ctx context.Context, domain string) ([]DNSRecord, error) {
	return []DNSRecord{}, nil
}

func (m *mockRegistrar) CreateDNSRecord(ctx context.Context, req DNSRecordRequest) error {
	return nil
}

func (m *mockRegistrar) UpdateDNSRecord(ctx context.Context, req DNSRecordUpdateRequest) error {
	return nil
}

func (m *mockRegistrar) DeleteDNSRecord(ctx context.Context, domain string, recordID string) error {
	return nil
}

func (m *mockRegistrar) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	return &ConnectionTestResult{Connected: true, Provider: "test", Message: "Connected"}, nil
}

func TestDomainValidation(t *testing.T) {
	clean, tld, err := ValidateDomainName("   https://MyTestDomain.COM/  ")
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if clean != "mytestdomain.com" {
		t.Errorf("expected clean domain mytestdomain.com, got %s", clean)
	}
	if tld != "com" {
		t.Errorf("expected tld com, got %s", tld)
	}

	// Invalid domains
	if _, _, err := ValidateDomainName("invalid_domain!.com"); err == nil {
		t.Errorf("expected error for invalid characters")
	}
	if _, _, err := ValidateDomainName("nodot"); err == nil {
		t.Errorf("expected error for missing TLD")
	}
}

func TestPricingEngine(t *testing.T) {
	s := store.NewMemoryStore()
	pe := NewPricingEngine(s)

	price, err := pe.CalculateRegistrationPrice(context.Background(), "com", 2)
	if err != nil {
		t.Fatalf("calculate price failed: %v", err)
	}

	if price.UnitPrice <= 0 {
		t.Errorf("expected positive unit price, got %f", price.UnitPrice)
	}
	if price.TotalPrice != roundMoney(price.UnitPrice*2) {
		t.Errorf("total price mismatch: %f != %f", price.TotalPrice, price.UnitPrice*2)
	}
}

func TestOrderAndProvisioningFlow(t *testing.T) {
	s := store.NewMemoryStore()
	mockReg := &mockRegistrar{}
	svc := NewService(s, mockReg, "test-secret-encryption-key-min32bytes")

	userID := uuid.New()
	contact := &ContactInfo{
		FirstName:  "Jane",
		LastName:   "Doe",
		Email:      "jane@hostvra.local",
		Phone:      "15559876543",
		Address1:   "100 Cloud Street",
		City:       "Austin",
		State:      "TX",
		PostalCode: "78701",
		Country:    "US",
	}

	// 1. Create registration order
	res, err := svc.Orders.CreateRegistrationOrder(context.Background(), CreateRegistrationOrderRequest{
		UserID:        userID,
		DomainName:    "myhostvratest.com",
		Years:         1,
		Registrant:    contact,
		PaymentMethod: "stripe",
	})
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	if res.Order.PaymentStatus != "pending" {
		t.Errorf("expected pending payment status, got %s", res.Order.PaymentStatus)
	}

	// 2. Try provisioning unpaid order -> must fail fast!
	err = svc.Provisioning.ProcessPaidOrder(context.Background(), res.Order.ID, contact, nil)
	if err == nil {
		t.Fatalf("expected error provisioning unpaid order, got nil")
	}

	// 3. Simulate payment webhook marked order as paid
	res.Order.PaymentStatus = "paid"
	_ = s.UpdateDomainOrder(context.Background(), res.Order)

	// 4. Now provisioning must succeed!
	err = svc.Provisioning.ProcessPaidOrder(context.Background(), res.Order.ID, contact, nil)
	if err != nil {
		t.Fatalf("expected successful provisioning, got: %v", err)
	}

	// 5. Verify domain was created in store
	dom, err := s.GetDomainByName(context.Background(), "myhostvratest.com")
	if err != nil {
		t.Fatalf("registered domain not found in store: %v", err)
	}
	if dom.Status != "active" {
		t.Errorf("expected active domain status, got %s", dom.Status)
	}

	// 6. Test idempotent retry: calling again on completed order must not fail or duplicate
	err = svc.Provisioning.ProcessPaidOrder(context.Background(), res.Order.ID, contact, nil)
	if err != nil {
		t.Fatalf("expected idempotent success on completed order, got: %v", err)
	}
}
