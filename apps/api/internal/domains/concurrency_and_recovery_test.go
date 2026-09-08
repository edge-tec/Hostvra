package domains_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/domains"
	"hostvra/api/internal/store"
)

// controlledMockRegistrar provides deterministic state for concurrency & failure injection testing
type controlledMockRegistrar struct {
	mu                  sync.Mutex
	registrationCalls   int32
	activeRegistrations map[string]*domains.DomainInfo
	failRegisterWith    error
}

func newControlledMockRegistrar() *controlledMockRegistrar {
	return &controlledMockRegistrar{
		activeRegistrations: make(map[string]*domains.DomainInfo),
	}
}

func (m *controlledMockRegistrar) CheckAvailability(ctx context.Context, req domains.AvailabilityRequest) ([]domains.DomainAvailability, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []domains.DomainAvailability
	for _, tld := range req.TLDs {
		dom := req.DomainName + "." + tld
		_, exists := m.activeRegistrations[dom]
		results = append(results, domains.DomainAvailability{
			Domain:        dom,
			TLD:           tld,
			Available:     !exists,
			Status:        "available",
			RegisterPrice: 12.00,
			Currency:      "USD",
		})
	}
	return results, nil
}

func (m *controlledMockRegistrar) RegisterDomain(ctx context.Context, req domains.RegisterDomainRequest) (*domains.DomainRegistrationResult, error) {
	atomic.AddInt32(&m.registrationCalls, 1)

	m.mu.Lock()
	if m.failRegisterWith != nil {
		err := m.failRegisterWith
		m.mu.Unlock()
		return nil, err
	}

	now := time.Now().UTC()
	exp := now.AddDate(req.Years, 0, 0)
	m.activeRegistrations[req.DomainName] = &domains.DomainInfo{
		DomainName:       req.DomainName,
		ProviderOrderID:  "rc-order-" + uuid.New().String()[:8],
		Status:           "Active",
		RegistrationDate: &now,
		ExpiryDate:       &exp,
		Nameservers:      req.Nameservers,
		RegistrarLock:    true,
	}
	info := m.activeRegistrations[req.DomainName]
	m.mu.Unlock()

	return &domains.DomainRegistrationResult{
		DomainName:       req.DomainName,
		ProviderOrderID:  info.ProviderOrderID,
		ProviderDomainID: info.ProviderOrderID,
		Status:           "active",
		RegistrationDate: *info.RegistrationDate,
		ExpiryDate:       *info.ExpiryDate,
		Nameservers:      req.Nameservers,
	}, nil
}

func (m *controlledMockRegistrar) RenewDomain(ctx context.Context, req domains.RenewDomainRequest) (*domains.DomainRenewalResult, error) {
	exp := time.Now().UTC().AddDate(req.Years, 0, 0)
	return &domains.DomainRenewalResult{
		DomainName:      req.DomainName,
		ProviderOrderID: req.ProviderOrderID,
		Status:          "completed",
		NewExpiryDate:   exp,
	}, nil
}

func (m *controlledMockRegistrar) TransferDomain(ctx context.Context, req domains.TransferDomainRequest) (*domains.DomainTransferResult, error) {
	return &domains.DomainTransferResult{
		DomainName:      req.DomainName,
		ProviderOrderID: "rc-transfer-" + uuid.New().String()[:8],
		Status:          "submitted",
	}, nil
}

func (m *controlledMockRegistrar) GetDomainInfo(ctx context.Context, domain string) (*domains.DomainInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if info, ok := m.activeRegistrations[domain]; ok {
		return info, nil
	}
	return nil, errors.New("domain not found at registrar")
}

func (m *controlledMockRegistrar) UpdateNameservers(ctx context.Context, req domains.NameserverUpdateRequest) error {
	return nil
}

func (m *controlledMockRegistrar) GetNameservers(ctx context.Context, domain string) ([]string, error) {
	return []string{"ns1.hostvra.com", "ns2.hostvra.com"}, nil
}

func (m *controlledMockRegistrar) GetRegistrarLock(ctx context.Context, domain string) (bool, error) {
	return true, nil
}

func (m *controlledMockRegistrar) SetRegistrarLock(ctx context.Context, domain string, locked bool) error {
	return nil
}

func (m *controlledMockRegistrar) GetEPPCode(ctx context.Context, domain string) (string, error) {
	return "TEST-AUTH-KEY-XYZ", nil
}

func (m *controlledMockRegistrar) GetContacts(ctx context.Context, domain string) (*domains.DomainContacts, error) {
	return &domains.DomainContacts{}, nil
}

func (m *controlledMockRegistrar) UpdateContacts(ctx context.Context, req domains.ContactUpdateRequest) error {
	return nil
}

func (m *controlledMockRegistrar) GetDNSRecords(ctx context.Context, domain string) ([]domains.DNSRecord, error) {
	return []domains.DNSRecord{}, nil
}

func (m *controlledMockRegistrar) CreateDNSRecord(ctx context.Context, req domains.DNSRecordRequest) error {
	return nil
}

func (m *controlledMockRegistrar) UpdateDNSRecord(ctx context.Context, req domains.DNSRecordUpdateRequest) error {
	return nil
}

func (m *controlledMockRegistrar) DeleteDNSRecord(ctx context.Context, domain string, recordID string) error {
	return nil
}

func (m *controlledMockRegistrar) TestConnection(ctx context.Context) (*domains.ConnectionTestResult, error) {
	return &domains.ConnectionTestResult{Connected: true, Mode: "test", ResellerID: "12345"}, nil
}

// ----------------------------------------------------------------------------
// 1. Concurrency Test: 10 Simultaneous Domain Registrations
// ----------------------------------------------------------------------------
func TestConcurrentDomainRegistrations(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	mockReg := newControlledMockRegistrar()

	svc := domains.NewService(st, mockReg, "test-secret-key-32-chars-length!!")

	// Set TLD Pricing
	_ = st.SaveTLDPricing(ctx, &store.TLDPricing{
		TLD:           "com",
		RegisterPrice: 15.00,
		RenewPrice:    15.00,
		TransferPrice: 15.00,
		Currency:      "USD",
	})

	userID := uuid.New()
	targetDomain := "concurrent-test.com"

	validContact := &domains.ContactInfo{
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john.doe@example.com",
		Phone:      "15551234567",
		Address1:   "100 Main St",
		City:       "New York",
		State:      "NY",
		PostalCode: "10001",
		Country:    "US",
	}

	// Create a single paid order
	orderRes, err := svc.Orders.CreateRegistrationOrder(ctx, domains.CreateRegistrationOrderRequest{
		UserID:        userID,
		DomainName:    targetDomain,
		Years:         1,
		Registrant:    validContact,
		PaymentMethod: "stripe",
	})
	if err != nil {
		t.Fatalf("Failed to create domain order: %v", err)
	}

	order := orderRes.Order
	order.PaymentStatus = "paid"
	_ = st.UpdateDomainOrder(ctx, order)

	// Simulate 10 concurrent provisioning workers racing to process the exact same order
	numWorkers := 10
	var wg sync.WaitGroup
	var successfulRegistrations int32
	var rejectedErrors int32

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := svc.Provisioning.ProcessPaidOrder(ctx, order.ID, nil, nil)
			if err == nil {
				atomic.AddInt32(&successfulRegistrations, 1)
			} else {
				atomic.AddInt32(&rejectedErrors, 1)
			}
		}()
	}

	wg.Wait()

	// Exactly 1 call to registrar RegisterDomain must occur across all concurrent attempts
	if mockReg.registrationCalls != 1 {
		t.Errorf("Expected registrar to be invoked exactly 1 time, got %d", mockReg.registrationCalls)
	}

	// Verify database state: exactly 1 active domain record created
	dom, err := st.GetDomainByName(ctx, targetDomain)
	if err != nil || dom == nil {
		t.Fatalf("Expected registered domain record in DB, got error: %v", err)
	}
	if dom.Status != "active" {
		t.Errorf("Expected domain status 'active', got '%s'", dom.Status)
	}
}

// ----------------------------------------------------------------------------
// 2. Failure Injection: Uncertain Registration Network Drop Recovery
// ----------------------------------------------------------------------------
func TestUncertainRegistrationRecovery(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	mockReg := newControlledMockRegistrar()

	svc := domains.NewService(st, mockReg, "test-secret-key-32-chars-length!!")

	_ = st.SaveTLDPricing(ctx, &store.TLDPricing{
		TLD:           "net",
		RegisterPrice: 18.00,
		RenewPrice:    18.00,
		TransferPrice: 18.00,
		Currency:      "USD",
	})

	userID := uuid.New()
	targetDomain := "uncertain-recovery.net"

	validContact := &domains.ContactInfo{
		FirstName:  "Jane",
		LastName:   "Smith",
		Email:      "jane.smith@example.com",
		Phone:      "15559876543",
		Address1:   "200 Market St",
		City:       "San Francisco",
		State:      "CA",
		PostalCode: "94105",
		Country:    "US",
	}

	orderRes, err := svc.Orders.CreateRegistrationOrder(ctx, domains.CreateRegistrationOrderRequest{
		UserID:        userID,
		DomainName:    targetDomain,
		Years:         1,
		Registrant:    validContact,
		PaymentMethod: "stripe",
	})
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}
	order := orderRes.Order
	order.PaymentStatus = "paid"
	_ = st.UpdateDomainOrder(ctx, order)

	// Simulate Scenario: ResellerClub actually registered the domain, but network connection dropped
	now := time.Now().UTC()
	exp := now.AddDate(1, 0, 0)
	mockReg.activeRegistrations[targetDomain] = &domains.DomainInfo{
		DomainName:       targetDomain,
		ProviderOrderID:  "rc-recovered-9999",
		Status:           "Active",
		RegistrationDate: &now,
		ExpiryDate:       &exp,
		Nameservers:      []string{"ns1.hostvra.com", "ns2.hostvra.com"},
		RegistrarLock:    true,
	}

	// Make RegisterDomain fail with a network timeout error
	mockReg.failRegisterWith = errors.New("network timeout communicating with registrar")

	// ProcessPaidOrder should check GetDomainInfo, detect it is actually Active at the provider, and recover!
	err = svc.Provisioning.ProcessPaidOrder(ctx, order.ID, nil, nil)
	if err != nil {
		t.Fatalf("ProcessPaidOrder failed to recover uncertain registration: %v", err)
	}

	// Verify order was marked completed
	savedOrder, err := st.GetDomainOrderByID(ctx, order.ID)
	if err != nil || savedOrder.ProvisioningStatus != "completed" {
		t.Errorf("Expected order completed, got status %s", savedOrder.ProvisioningStatus)
	}

	// Verify domain was persisted as active with recovered provider order ID
	d, err := st.GetDomainByName(ctx, targetDomain)
	if err != nil || d == nil {
		t.Fatalf("Domain record missing after recovery: %v", err)
	}
	if d.ProviderOrderID != "rc-recovered-9999" {
		t.Errorf("Expected provider order ID 'rc-recovered-9999', got '%s'", d.ProviderOrderID)
	}
}

// ----------------------------------------------------------------------------
// 3. Duplicate Order Prevention Test
// ----------------------------------------------------------------------------
func TestDuplicateActiveDomainOrderPrevention(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	mockReg := newControlledMockRegistrar()

	svc := domains.NewService(st, mockReg, "test-secret-key-32-chars-length!!")

	_ = st.SaveTLDPricing(ctx, &store.TLDPricing{
		TLD:           "org",
		RegisterPrice: 20.00,
		RenewPrice:    20.00,
		TransferPrice: 20.00,
		Currency:      "USD",
	})

	userID := uuid.New()
	domainName := "active-check.org"

	// Create and register the domain
	domainRecord := &store.Domain{
		ID:         uuid.New(),
		UserID:     userID,
		DomainName: domainName,
		TLD:        "org",
		Registrar:  "resellerclub",
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	_ = st.CreateDomain(ctx, domainRecord)

	validContact := &domains.ContactInfo{
		FirstName:  "Alice",
		LastName:   "Wonder",
		Email:      "alice@example.org",
		Phone:      "15553334444",
		Address1:   "300 Elm St",
		City:       "Austin",
		State:      "TX",
		PostalCode: "78701",
		Country:    "US",
	}

	// Attempt to create a new registration order for the same active domain
	_, err := svc.Orders.CreateRegistrationOrder(ctx, domains.CreateRegistrationOrderRequest{
		UserID:        uuid.New(), // User B
		DomainName:    domainName,
		Years:         1,
		Registrant:    validContact,
		PaymentMethod: "paypal",
	})

	if err == nil {
		t.Fatalf("Expected error when ordering an already active domain in Hostvra, but got nil")
	}

	expectedSubstr := fmt.Sprintf("domain %s is already registered and active in Hostvra", domainName)
	if err.Error() != expectedSubstr {
		t.Errorf("Expected error '%s', got '%s'", expectedSubstr, err.Error())
	}
}
