package store

import (
	"context"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// MEMORY STORE TLD & REGISTRAR IMPLEMENTATION & SEED DATA
// ============================================================================

func (m *MemoryStore) seedTLDData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	// Seed TLD Pricings if empty
	if len(m.tldPricings) == 0 {
		seedTLDs := []*TLDPricing{
			{
				TLD:           ".com",
				RegisterPrice: 9.99,
				RenewPrice:    13.99,
				TransferPrice: 9.99,
				Currency:      "USD",
				IsPopular:     true,
				Category:      "popular",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".net",
				RegisterPrice: 11.99,
				RenewPrice:    15.99,
				TransferPrice: 11.99,
				Currency:      "USD",
				IsPopular:     true,
				Category:      "popular",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".org",
				RegisterPrice: 12.49,
				RenewPrice:    14.99,
				TransferPrice: 12.49,
				Currency:      "USD",
				IsPopular:     true,
				Category:      "popular",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".xyz",
				RegisterPrice: 1.99,
				RenewPrice:    11.99,
				TransferPrice: 10.99,
				Currency:      "USD",
				IsPopular:     true,
				Category:      "tech",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".io",
				RegisterPrice: 34.99,
				RenewPrice:    39.99,
				TransferPrice: 34.99,
				Currency:      "USD",
				IsPopular:     true,
				Category:      "tech",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".com.bd",
				RegisterPrice: 12.00,
				RenewPrice:    12.00,
				TransferPrice: 0.00,
				Currency:      "USD",
				IsPopular:     true,
				Category:      "country",
				MinYears:      2,
				UpdatedAt:     now,
			},
			{
				TLD:           ".co",
				RegisterPrice: 8.99,
				RenewPrice:    24.99,
				TransferPrice: 22.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "business",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".tech",
				RegisterPrice: 3.99,
				RenewPrice:    19.99,
				TransferPrice: 18.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "tech",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".store",
				RegisterPrice: 2.99,
				RenewPrice:    29.99,
				TransferPrice: 24.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "ecommerce",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".online",
				RegisterPrice: 1.99,
				RenewPrice:    29.99,
				TransferPrice: 24.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "business",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".info",
				RegisterPrice: 4.99,
				RenewPrice:    19.99,
				TransferPrice: 17.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "popular",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".app",
				RegisterPrice: 14.99,
				RenewPrice:    16.99,
				TransferPrice: 14.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "tech",
				MinYears:      1,
				UpdatedAt:     now,
			},
			{
				TLD:           ".dev",
				RegisterPrice: 14.99,
				RenewPrice:    16.99,
				TransferPrice: 14.99,
				Currency:      "USD",
				IsPopular:     false,
				Category:      "tech",
				MinYears:      1,
				UpdatedAt:     now,
			},
		}

		for _, t := range seedTLDs {
			m.tldPricings[strings.ToLower(t.TLD)] = t
		}
	}

	// Seed Registrars if empty
	if len(m.registrarConfigs) == 0 {
		registrars := []*DomainRegistrarConfig{
			{
				Registrar:   "namecheap",
				DisplayName: "Namecheap API Integration",
				Enabled:     true,
				TestMode:    true,
				APIUser:     "hostvra_demo_user",
				APIKey:      "nc_sandbox_key_9823472918",
				ClientIP:    "127.0.0.1",
				UpdatedAt:   now,
			},
			{
				Registrar:   "resellerclub",
				DisplayName: "ResellerClub / LogicBoxes HTTP API",
				Enabled:     true,
				TestMode:    true,
				APIUser:     "rc_reseller_54129",
				APIKey:      "rc_api_key_demo_secret",
				ClientIP:    "127.0.0.1",
				UpdatedAt:   now,
			},
			{
				Registrar:   "cloudflare",
				DisplayName: "Cloudflare Registrar API",
				Enabled:     false,
				TestMode:    true,
				APIUser:     "cf_account_hostvra",
				APIKey:      "cf_token_demo_9823487",
				ClientIP:    "127.0.0.1",
				UpdatedAt:   now,
			},
			{
				Registrar:   "enom",
				DisplayName: "Enom Domain Reseller",
				Enabled:     false,
				TestMode:    true,
				APIUser:     "enom_uid_demo",
				APIKey:      "enom_pw_demo",
				ClientIP:    "127.0.0.1",
				UpdatedAt:   now,
			},
		}

		for _, r := range registrars {
			m.registrarConfigs[strings.ToLower(r.Registrar)] = r
		}
	}
}

// ----------------------------------------------------------------------------
// TLD Pricing CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListTLDPricings(ctx context.Context) ([]*TLDPricing, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*TLDPricing
	for _, p := range m.tldPricings {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].IsPopular != list[j].IsPopular {
			return list[i].IsPopular
		}
		return list[i].RegisterPrice < list[j].RegisterPrice
	})
	return list, nil
}

func (m *MemoryStore) GetTLDPricing(ctx context.Context, tld string) (*TLDPricing, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(tld))
	if !strings.HasPrefix(clean, ".") {
		clean = "." + clean
	}

	p, exists := m.tldPricings[clean]
	if !exists {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) SaveTLDPricing(ctx context.Context, pricing *TLDPricing) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.ToLower(strings.TrimSpace(pricing.TLD))
	if !strings.HasPrefix(clean, ".") {
		clean = "." + clean
	}
	pricing.TLD = clean
	pricing.UpdatedAt = time.Now().UTC()

	m.tldPricings[clean] = pricing
	m.saveToDiskLocked()
	return nil
}

// ----------------------------------------------------------------------------
// Registrar Configs CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListRegistrarConfigs(ctx context.Context) ([]*DomainRegistrarConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainRegistrarConfig
	for _, r := range m.registrarConfigs {
		list = append(list, r)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Registrar < list[j].Registrar
	})
	return list, nil
}

func (m *MemoryStore) GetRegistrarConfig(ctx context.Context, registrar string) (*DomainRegistrarConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(registrar))
	r, exists := m.registrarConfigs[clean]
	if !exists {
		return nil, ErrNotFound
	}
	return r, nil
}

func (m *MemoryStore) SaveRegistrarConfig(ctx context.Context, config *DomainRegistrarConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.Registrar = strings.ToLower(strings.TrimSpace(config.Registrar))
	config.UpdatedAt = time.Now().UTC()

	m.registrarConfigs[config.Registrar] = config
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE IMPLEMENTATION (Delegates to MemoryStore)
// ============================================================================

func (p *PostgresStore) ListTLDPricings(ctx context.Context) ([]*TLDPricing, error) {
	m := NewMemoryStore()
	return m.ListTLDPricings(ctx)
}

func (p *PostgresStore) GetTLDPricing(ctx context.Context, tld string) (*TLDPricing, error) {
	m := NewMemoryStore()
	return m.GetTLDPricing(ctx, tld)
}

func (p *PostgresStore) SaveTLDPricing(ctx context.Context, pricing *TLDPricing) error {
	m := NewMemoryStore()
	return m.SaveTLDPricing(ctx, pricing)
}

func (p *PostgresStore) ListRegistrarConfigs(ctx context.Context) ([]*DomainRegistrarConfig, error) {
	m := NewMemoryStore()
	return m.ListRegistrarConfigs(ctx)
}

func (p *PostgresStore) GetRegistrarConfig(ctx context.Context, registrar string) (*DomainRegistrarConfig, error) {
	m := NewMemoryStore()
	return m.GetRegistrarConfig(ctx, registrar)
}

func (p *PostgresStore) SaveRegistrarConfig(ctx context.Context, config *DomainRegistrarConfig) error {
	m := NewMemoryStore()
	return m.SaveRegistrarConfig(ctx, config)
}
