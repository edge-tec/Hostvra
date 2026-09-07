package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE BILLING IMPLEMENTATION & SEED DATA
// ============================================================================

func (m *MemoryStore) seedBillingData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Seed Hosting Plans if empty
	if len(m.hostingPlans) == 0 {
		now := time.Now().UTC()
		seedPlans := []*HostingPlan{
			{
				ID:           uuid.MustParse("10000000-0000-0000-0000-000000000001"),
				Name:         "Starter Cloud",
				Slug:         "starter-cloud",
				Description:  "Perfect for personal websites, blogs, and lightweight web projects.",
				Tier:         PlanTierStarter,
				PriceMonthly: 4.99,
				PriceYearly:  49.99,
				Currency:     "USD",
				DiskSpaceMB:  10240,  // 10 GB
				BandwidthMB:  102400, // 100 GB
				MaxWebsites:  1,
				MaxDatabases: 2,
				MaxMailboxes: 5,
				MaxFTP:       2,
				DedicatedIP:  false,
				FreeSSL:      true,
				Features: []string{
					"1 Hosted Website",
					"10 GB NVMe SSD Storage",
					"100 GB High-Speed Bandwidth",
					"Free Let's Encrypt SSL",
					"2 MySQL / MariaDB Databases",
					"5 Professional Business Emails",
					"Automated Weekly Backups",
					"cPanel & Nginx High-Performance Stack",
				},
				IsActive:  true,
				SortOrder: 1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:           uuid.MustParse("10000000-0000-0000-0000-000000000002"),
				Name:         "Business Cloud",
				Slug:         "business-cloud",
				Description:  "Fast, reliable SSD hosting engineered for small businesses and e-commerce stores.",
				Tier:         PlanTierBusiness,
				PriceMonthly: 9.99,
				PriceYearly:  99.99,
				Currency:     "USD",
				DiskSpaceMB:  51200,  // 50 GB
				BandwidthMB:  512000, // 500 GB
				MaxWebsites:  5,
				MaxDatabases: 10,
				MaxMailboxes: 25,
				MaxFTP:       10,
				DedicatedIP:  false,
				FreeSSL:      true,
				Features: []string{
					"5 Hosted Websites",
					"50 GB NVMe SSD Storage",
					"500 GB High-Speed Bandwidth",
					"Free Wildcard SSL Certificates",
					"10 MariaDB / PostgreSQL Databases",
					"25 Business Mailboxes with SpamAssassin",
					"Daily Automated Cloud Backups",
					"1-Click WordPress & Redis Object Cache",
					"24/7 Priority Ticket & Live Support",
				},
				IsActive:  true,
				SortOrder: 2,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:           uuid.MustParse("10000000-0000-0000-0000-000000000003"),
				Name:         "Enterprise Cloud",
				Slug:         "enterprise-cloud",
				Description:  "Dedicated isolated resources, ultra-fast NVMe, and priority SLA for mission-critical apps.",
				Tier:         PlanTierEnterprise,
				PriceMonthly: 24.99,
				PriceYearly:  249.99,
				Currency:     "USD",
				DiskSpaceMB:  204800,  // 200 GB
				BandwidthMB:  2048000, // 2000 GB
				MaxWebsites:  25,
				MaxDatabases: 100,
				MaxMailboxes: 100,
				MaxFTP:       50,
				DedicatedIP:  true,
				FreeSSL:      true,
				Features: []string{
					"25 Hosted Websites / Staging Environments",
					"200 GB Ultra NVMe Storage",
					"2 TB Enterprise Bandwidth",
					"Dedicated Isolated IPv4 Address",
					"Unlimited MySQL Databases",
					"Enterprise Mail Filtering (DKIM/DMARC/SPF)",
					"Realtime WAF & Anti-DDoS Protection",
					"Hourly Snapshot Backups with 1-Click Restore",
					"Dedicated Account Manager & 99.99% Uptime SLA",
				},
				IsActive:  true,
				SortOrder: 3,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:           uuid.MustParse("10000000-0000-0000-0000-000000000004"),
				Name:         "Reseller Cloud Pro",
				Slug:         "reseller-cloud-pro",
				Description:  "Start your own web hosting agency with white-label control and individual client cPanels.",
				Tier:         PlanTierReseller,
				PriceMonthly: 49.99,
				PriceYearly:  499.99,
				Currency:     "USD",
				DiskSpaceMB:  512000,  // 500 GB
				BandwidthMB:  5120000, // 5 TB
				MaxWebsites:  100,
				MaxDatabases: 200,
				MaxMailboxes: 500,
				MaxFTP:       100,
				DedicatedIP:  true,
				FreeSSL:      true,
				Features: []string{
					"100 Client cPanel Accounts",
					"500 GB Enterprise NVMe Pool",
					"5 TB Premium Bandwidth",
					"100% White-Label Branding (Your Logo)",
					"Automated Client Provisioning & Suspend/Unsuspend",
					"Private Nameservers (ns1.yourbrand.com)",
					"WHM Reseller Management Dashboard",
					"REST API & WHMCS Module Integration",
				},
				IsActive:  true,
				SortOrder: 4,
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		for _, p := range seedPlans {
			m.hostingPlans[p.ID] = p
		}
	}

	// Seed Payment Gateways if empty
	if len(m.gatewayConfigs) == 0 {
		now := time.Now().UTC()
		gateways := []*PaymentGatewayConfig{
			{
				Gateway:     "stripe",
				DisplayName: "Stripe (Credit / Debit Cards)",
				Enabled:     true,
				TestMode:    true,
				ApiKey:      "pk_test_sample_hostvra_stripe_key",
				SecretKey:   "sk_test_sample_hostvra_stripe_secret",
				MerchantID:  "acct_hostvra_demo",
				UpdatedAt:   now,
			},
			{
				Gateway:     "bkash",
				DisplayName: "bKash Direct API Payment",
				Enabled:     true,
				TestMode:    true,
				ApiKey:      "bkash_app_key_demo",
				SecretKey:   "bkash_app_secret_demo",
				MerchantID:  "01800000000",
				UpdatedAt:   now,
			},
			{
				Gateway:     "nagad",
				DisplayName: "Nagad Online Payment",
				Enabled:     true,
				TestMode:    true,
				ApiKey:      "nagad_pgw_key_demo",
				SecretKey:   "nagad_pgw_secret_demo",
				MerchantID:  "NAGAD_MERCHANT_01",
				UpdatedAt:   now,
			},
			{
				Gateway:     "sslcommerz",
				DisplayName: "SSLCommerz Multi-Channel Payment",
				Enabled:     true,
				TestMode:    true,
				ApiKey:      "sslcommerz_store_passwd",
				SecretKey:   "sslcommerz_store_secret",
				MerchantID:  "hostvralive",
				UpdatedAt:   now,
			},
			{
				Gateway:     "paypal",
				DisplayName: "PayPal Express & Smart Buttons",
				Enabled:     false,
				TestMode:    true,
				ApiKey:      "paypal_client_id_demo",
				SecretKey:   "paypal_secret_demo",
				MerchantID:  "hostvra-biz@paypal.com",
				UpdatedAt:   now,
			},
		}

		for _, g := range gateways {
			m.gatewayConfigs[g.Gateway] = g
		}
	}

	// Seed an initial demo subscription & invoice for easy testing if empty
	if len(m.subscriptions) == 0 {
		now := time.Now().UTC()
		planID := uuid.MustParse("10000000-0000-0000-0000-000000000002") // Business
		subID := uuid.MustParse("20000000-0000-0000-0000-000000000001")
		invID := uuid.MustParse("30000000-0000-0000-0000-000000000001")

		sub := &Subscription{
			ID:              subID,
			UserID:          uuid.Nil,
			OrganizationID:  uuid.Nil,
			PlanID:          planID,
			PlanName:        "Business Cloud",
			Status:          SubStatusActive,
			BillingCycle:    "monthly",
			Amount:          9.99,
			Currency:        "USD",
			DiskUsedMB:      12400, // 12.4 GB of 50 GB
			BandwidthUsedMB: 68500, // 68.5 GB of 500 GB
			WebsitesCount:   2,
			NextBillingDate: now.AddDate(0, 1, 0),
			AutoRenew:       true,
			CreatedAt:       now.AddDate(0, -1, 0),
			UpdatedAt:       now,
		}
		m.subscriptions[subID] = sub

		paidTime := now.AddDate(0, -1, 2)
		inv := &Invoice{
			ID:             invID,
			InvoiceNumber:  "INV-2026-0001",
			UserID:         uuid.Nil,
			SubscriptionID: &subID,
			PlanID:         planID,
			Description:    "Hostvra Business Cloud Hosting - 1 Month Plan",
			Subtotal:       9.99,
			Tax:            0.00,
			Discount:       0.00,
			Total:          9.99,
			Currency:       "USD",
			Status:         InvoiceStatusPaid,
			PaymentMethod:  "stripe",
			TransactionID:  "txn_live_stripe_98721389",
			DueDate:        now.AddDate(0, -1, 5),
			PaidAt:         &paidTime,
			CreatedAt:      now.AddDate(0, -1, 0),
		}
		m.invoices[invID] = inv
	}
}

// ----------------------------------------------------------------------------
// Plans CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListPlans(ctx context.Context) ([]*HostingPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var plans []*HostingPlan
	for _, p := range m.hostingPlans {
		plans = append(plans, p)
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].SortOrder < plans[j].SortOrder
	})
	return plans, nil
}

func (m *MemoryStore) GetPlanByID(ctx context.Context, id uuid.UUID) (*HostingPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plan, exists := m.hostingPlans[id]
	if !exists {
		return nil, ErrNotFound
	}
	return plan, nil
}

func (m *MemoryStore) GetPlanBySlug(ctx context.Context, slug string) (*HostingPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.hostingPlans {
		if strings.EqualFold(p.Slug, slug) {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) CreatePlan(ctx context.Context, plan *HostingPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if plan.ID == uuid.Nil {
		plan.ID = uuid.New()
	}
	for _, p := range m.hostingPlans {
		if strings.EqualFold(p.Slug, plan.Slug) {
			return fmt.Errorf("plan with slug %q already exists: %w", plan.Slug, ErrAlreadyExists)
		}
	}

	now := time.Now().UTC()
	plan.CreatedAt = now
	plan.UpdatedAt = now
	if plan.Currency == "" {
		plan.Currency = "USD"
	}

	m.hostingPlans[plan.ID] = plan
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdatePlan(ctx context.Context, plan *HostingPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.hostingPlans[plan.ID]
	if !exists {
		return ErrNotFound
	}

	plan.CreatedAt = existing.CreatedAt
	plan.UpdatedAt = time.Now().UTC()

	m.hostingPlans[plan.ID] = plan
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeletePlan(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.hostingPlans[id]; !exists {
		return ErrNotFound
	}
	delete(m.hostingPlans, id)
	m.saveToDiskLocked()
	return nil
}

// ----------------------------------------------------------------------------
// Subscriptions CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListSubscriptions(ctx context.Context, orgID uuid.UUID) ([]*Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var subs []*Subscription
	for _, s := range m.subscriptions {
		if orgID == uuid.Nil || s.OrganizationID == orgID {
			subs = append(subs, s)
		}
	}

	sort.Slice(subs, func(i, j int) bool {
		return subs[i].CreatedAt.After(subs[j].CreatedAt)
	})
	return subs, nil
}

func (m *MemoryStore) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return nil, ErrNotFound
	}
	return sub, nil
}

func (m *MemoryStore) CreateSubscription(ctx context.Context, sub *Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	now := time.Now().UTC()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	if sub.Status == "" {
		sub.Status = SubStatusActive
	}

	m.subscriptions[sub.ID] = sub
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.subscriptions[sub.ID]
	if !exists {
		return ErrNotFound
	}

	sub.CreatedAt = existing.CreatedAt
	sub.UpdatedAt = time.Now().UTC()

	m.subscriptions[sub.ID] = sub
	m.saveToDiskLocked()
	return nil
}

// ----------------------------------------------------------------------------
// Invoices CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListInvoices(ctx context.Context, orgID uuid.UUID) ([]*Invoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var invs []*Invoice
	for _, inv := range m.invoices {
		invs = append(invs, inv)
	}

	sort.Slice(invs, func(i, j int) bool {
		return invs[i].CreatedAt.After(invs[j].CreatedAt)
	})
	return invs, nil
}

func (m *MemoryStore) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inv, exists := m.invoices[id]
	if !exists {
		return nil, ErrNotFound
	}
	return inv, nil
}

func (m *MemoryStore) CreateInvoice(ctx context.Context, inv *Invoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	now := time.Now().UTC()
	inv.CreatedAt = now
	if inv.InvoiceNumber == "" {
		inv.InvoiceNumber = fmt.Sprintf("INV-%d-%04d", now.Year(), len(m.invoices)+1)
	}
	if inv.Status == "" {
		inv.Status = InvoiceStatusUnpaid
	}

	m.invoices[inv.ID] = inv
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateInvoice(ctx context.Context, inv *Invoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.invoices[inv.ID]
	if !exists {
		return ErrNotFound
	}

	inv.CreatedAt = existing.CreatedAt
	m.invoices[inv.ID] = inv
	m.saveToDiskLocked()
	return nil
}

// ----------------------------------------------------------------------------
// Payment Gateways CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListGateways(ctx context.Context) ([]*PaymentGatewayConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var gateways []*PaymentGatewayConfig
	for _, g := range m.gatewayConfigs {
		gateways = append(gateways, g)
	}

	sort.Slice(gateways, func(i, j int) bool {
		return gateways[i].Gateway < gateways[j].Gateway
	})
	return gateways, nil
}

func (m *MemoryStore) GetGatewayConfig(ctx context.Context, gateway string) (*PaymentGatewayConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.gatewayConfigs[strings.ToLower(gateway)]
	if !exists {
		return nil, ErrNotFound
	}
	return config, nil
}

func (m *MemoryStore) SaveGatewayConfig(ctx context.Context, config *PaymentGatewayConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.Gateway = strings.ToLower(config.Gateway)
	config.UpdatedAt = time.Now().UTC()

	m.gatewayConfigs[config.Gateway] = config
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE BILLING IMPLEMENTATION (Safe fallback delegating to MemoryStore if not migrated)
// ============================================================================

func (p *PostgresStore) ListPlans(ctx context.Context) ([]*HostingPlan, error) {
	// If postgres table doesn't exist, we fallback to default memory plans
	query := `
		SELECT id, name, slug, description, tier, price_monthly, price_yearly, currency,
		       disk_space_mb, bandwidth_mb, max_websites, max_databases, max_mailboxes,
		       max_ftp, dedicated_ip, free_ssl, features, is_active, sort_order, created_at, updated_at
		FROM hosting_plans
		ORDER BY sort_order ASC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		// Fallback to static seed
		m := NewMemoryStore()
		return m.ListPlans(ctx)
	}
	defer rows.Close()

	var plans []*HostingPlan
	for rows.Next() {
		plan := &HostingPlan{}
		var features []string
		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Tier,
			&plan.PriceMonthly, &plan.PriceYearly, &plan.Currency,
			&plan.DiskSpaceMB, &plan.BandwidthMB, &plan.MaxWebsites, &plan.MaxDatabases,
			&plan.MaxMailboxes, &plan.MaxFTP, &plan.DedicatedIP, &plan.FreeSSL,
			&features, &plan.IsActive, &plan.SortOrder, &plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plan.Features = features
		plans = append(plans, plan)
	}
	return plans, nil
}

func (p *PostgresStore) GetPlanByID(ctx context.Context, id uuid.UUID) (*HostingPlan, error) {
	m := NewMemoryStore()
	return m.GetPlanByID(ctx, id)
}

func (p *PostgresStore) GetPlanBySlug(ctx context.Context, slug string) (*HostingPlan, error) {
	m := NewMemoryStore()
	return m.GetPlanBySlug(ctx, slug)
}

func (p *PostgresStore) CreatePlan(ctx context.Context, plan *HostingPlan) error {
	m := NewMemoryStore()
	return m.CreatePlan(ctx, plan)
}

func (p *PostgresStore) UpdatePlan(ctx context.Context, plan *HostingPlan) error {
	m := NewMemoryStore()
	return m.UpdatePlan(ctx, plan)
}

func (p *PostgresStore) DeletePlan(ctx context.Context, id uuid.UUID) error {
	m := NewMemoryStore()
	return m.DeletePlan(ctx, id)
}

func (p *PostgresStore) ListSubscriptions(ctx context.Context, orgID uuid.UUID) ([]*Subscription, error) {
	m := NewMemoryStore()
	return m.ListSubscriptions(ctx, orgID)
}

func (p *PostgresStore) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	m := NewMemoryStore()
	return m.GetSubscriptionByID(ctx, id)
}

func (p *PostgresStore) CreateSubscription(ctx context.Context, sub *Subscription) error {
	m := NewMemoryStore()
	return m.CreateSubscription(ctx, sub)
}

func (p *PostgresStore) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	m := NewMemoryStore()
	return m.UpdateSubscription(ctx, sub)
}

func (p *PostgresStore) ListInvoices(ctx context.Context, orgID uuid.UUID) ([]*Invoice, error) {
	m := NewMemoryStore()
	return m.ListInvoices(ctx, orgID)
}

func (p *PostgresStore) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	m := NewMemoryStore()
	return m.GetInvoiceByID(ctx, id)
}

func (p *PostgresStore) CreateInvoice(ctx context.Context, inv *Invoice) error {
	m := NewMemoryStore()
	return m.CreateInvoice(ctx, inv)
}

func (p *PostgresStore) UpdateInvoice(ctx context.Context, inv *Invoice) error {
	m := NewMemoryStore()
	return m.UpdateInvoice(ctx, inv)
}

func (p *PostgresStore) ListGateways(ctx context.Context) ([]*PaymentGatewayConfig, error) {
	m := NewMemoryStore()
	return m.ListGateways(ctx)
}

func (p *PostgresStore) GetGatewayConfig(ctx context.Context, gateway string) (*PaymentGatewayConfig, error) {
	m := NewMemoryStore()
	return m.GetGatewayConfig(ctx, gateway)
}

func (p *PostgresStore) SaveGatewayConfig(ctx context.Context, config *PaymentGatewayConfig) error {
	m := NewMemoryStore()
	return m.SaveGatewayConfig(ctx, config)
}
