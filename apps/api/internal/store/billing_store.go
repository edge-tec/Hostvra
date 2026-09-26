package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
				ID:            uuid.MustParse("10000000-0000-0000-0000-000000000001"),
				Name:          "Starter Cloud",
				Slug:          "starter-cloud",
				Description:   "Perfect for personal websites, blogs, and lightweight web projects.",
				Tier:          PlanTierStarter,
				PriceMonthly:  4.99,
				PriceYearly:   49.99,
				Currency:      "USD",
				SetupFee:      0.00,
				TrialAllowed:  true,
				TrialDays:     14,
				IsFeatured:    false,
				CPULimit:      1.0,
				RAMLimitMB:    1024,
				DiskSpaceMB:   10240,  // 10 GB
				BandwidthMB:   102400, // 100 GB
				MaxWebsites:   1,
				MaxDatabases:  2,
				MaxMailboxes:  5,
				MaxFTP:        2,
				MaxCron:       5,
				MaxSubdomains: 10,
				DedicatedIP:   false,
				FreeSSL:       true,
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
				ID:            uuid.MustParse("10000000-0000-0000-0000-000000000002"),
				Name:          "Business Cloud",
				Slug:          "business-cloud",
				Description:   "Fast, reliable SSD hosting engineered for small businesses and e-commerce stores.",
				Tier:          PlanTierBusiness,
				PriceMonthly:  9.99,
				PriceYearly:   99.99,
				Currency:      "USD",
				SetupFee:      0.00,
				TrialAllowed:  true,
				TrialDays:     14,
				IsFeatured:    true,
				CPULimit:      2.0,
				RAMLimitMB:    2048,
				DiskSpaceMB:   51200,  // 50 GB
				BandwidthMB:   512000, // 500 GB
				MaxWebsites:   5,
				MaxDatabases:  10,
				MaxMailboxes:  25,
				MaxFTP:        10,
				MaxCron:       15,
				MaxSubdomains: 25,
				DedicatedIP:   false,
				FreeSSL:       true,
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
				ID:            uuid.MustParse("10000000-0000-0000-0000-000000000003"),
				Name:          "Enterprise Cloud",
				Slug:          "enterprise-cloud",
				Description:   "Dedicated isolated resources, ultra-fast NVMe, and priority SLA for mission-critical apps.",
				Tier:          PlanTierEnterprise,
				PriceMonthly:  24.99,
				PriceYearly:   249.99,
				Currency:      "USD",
				SetupFee:      0.00,
				TrialAllowed:  false,
				TrialDays:     0,
				IsFeatured:    false,
				CPULimit:      4.0,
				RAMLimitMB:    4096,
				DiskSpaceMB:   204800,  // 200 GB
				BandwidthMB:   2048000, // 2000 GB
				MaxWebsites:   25,
				MaxDatabases:  100,
				MaxMailboxes:  100,
				MaxFTP:        50,
				MaxCron:       50,
				MaxSubdomains: 100,
				DedicatedIP:   true,
				FreeSSL:       true,
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
				ID:            uuid.MustParse("10000000-0000-0000-0000-000000000004"),
				Name:          "Reseller Cloud Pro",
				Slug:          "reseller-cloud-pro",
				Description:   "Start your own web hosting agency with white-label control and individual client cPanels.",
				Tier:          PlanTierReseller,
				PriceMonthly:  49.99,
				PriceYearly:   499.99,
				Currency:      "USD",
				SetupFee:      0.00,
				TrialAllowed:  false,
				TrialDays:     0,
				IsFeatured:    false,
				CPULimit:      8.0,
				RAMLimitMB:    16384,
				DiskSpaceMB:   512000,  // 500 GB
				BandwidthMB:   5120000, // 5 TB
				MaxWebsites:   100,
				MaxDatabases:  200,
				MaxMailboxes:  500,
				MaxFTP:        100,
				MaxCron:       100,
				MaxSubdomains: 500,
				DedicatedIP:   true,
				FreeSSL:       true,
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
				DisplayName: "Stripe Payment Gateway",
				Enabled:     false,
				TestMode:    true,
				ApiKey:      "",
				SecretKey:   "",
				MerchantID:  "",
				UpdatedAt:   now,
			},
			{
				Gateway:     "bkash",
				DisplayName: "bKash Direct API Payment",
				Enabled:     false,
				TestMode:    true,
				ApiKey:      "",
				SecretKey:   "",
				MerchantID:  "",
				UpdatedAt:   now,
			},
			{
				Gateway:     "nagad",
				DisplayName: "Nagad Online Payment",
				Enabled:     false,
				TestMode:    true,
				ApiKey:      "",
				SecretKey:   "",
				MerchantID:  "",
				UpdatedAt:   now,
			},
			{
				Gateway:     "sslcommerz",
				DisplayName: "SSLCommerz Multi-Channel Payment",
				Enabled:     false,
				TestMode:    true,
				ApiKey:      "",
				SecretKey:   "",
				MerchantID:  "",
				UpdatedAt:   now,
			},
			{
				Gateway:     "paypal",
				DisplayName: "PayPal Express & Smart Buttons",
				Enabled:     false,
				TestMode:    true,
				ApiKey:      "",
				SecretKey:   "",
				MerchantID:  "",
				UpdatedAt:   now,
			},
		}

		for _, g := range gateways {
			m.gatewayConfigs[g.Gateway] = g
		}
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

func (m *MemoryStore) ListAllSubscriptions(ctx context.Context) ([]*Subscription, error) {
	return m.ListSubscriptions(ctx, uuid.Nil)
}

func (m *MemoryStore) ListTrials(ctx context.Context) ([]*Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var trials []*Subscription
	for _, s := range m.subscriptions {
		if s.Status == SubStatusTrial || s.TrialEndsAt != nil {
			trials = append(trials, s)
		}
	}
	sort.Slice(trials, func(i, j int) bool {
		return trials[i].CreatedAt.After(trials[j].CreatedAt)
	})
	return trials, nil
}

func (m *MemoryStore) GetTrialSettings(ctx context.Context) (*TrialSettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.trialSettings == nil {
		return &TrialSettings{
			Enabled:              true,
			DefaultDays:          14,
			RequirePaymentMethod: false,
			OneTrialPerCustomer:  true,
			AutoSuspendOnExpiry:  true,
			UpdatedAt:            time.Now().UTC(),
		}, nil
	}
	copy := *m.trialSettings
	return &copy, nil
}

func (m *MemoryStore) SaveTrialSettings(ctx context.Context, settings *TrialSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	settings.UpdatedAt = time.Now().UTC()
	copy := *settings
	m.trialSettings = &copy
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
	query := `
		SELECT id, name, slug, description, tier, price_monthly, price_yearly, currency,
		       disk_space_mb, bandwidth_mb, max_websites, max_databases, max_mailboxes,
		       max_ftp, dedicated_ip, free_ssl, features, is_active, sort_order, created_at, updated_at
		FROM hosting_plans
		WHERE id = $1
	`
	plan := &HostingPlan{}
	var features []string
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Tier,
		&plan.PriceMonthly, &plan.PriceYearly, &plan.Currency,
		&plan.DiskSpaceMB, &plan.BandwidthMB, &plan.MaxWebsites, &plan.MaxDatabases,
		&plan.MaxMailboxes, &plan.MaxFTP, &plan.DedicatedIP, &plan.FreeSSL,
		pq.Array(&features), &plan.IsActive, &plan.SortOrder, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetPlanByID(ctx, id)
	}
	plan.Features = features
	return plan, nil
}

func (p *PostgresStore) GetPlanBySlug(ctx context.Context, slug string) (*HostingPlan, error) {
	query := `
		SELECT id, name, slug, description, tier, price_monthly, price_yearly, currency,
		       disk_space_mb, bandwidth_mb, max_websites, max_databases, max_mailboxes,
		       max_ftp, dedicated_ip, free_ssl, features, is_active, sort_order, created_at, updated_at
		FROM hosting_plans
		WHERE slug = $1
	`
	plan := &HostingPlan{}
	var features []string
	err := p.db.QueryRowContext(ctx, query, slug).Scan(
		&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Tier,
		&plan.PriceMonthly, &plan.PriceYearly, &plan.Currency,
		&plan.DiskSpaceMB, &plan.BandwidthMB, &plan.MaxWebsites, &plan.MaxDatabases,
		&plan.MaxMailboxes, &plan.MaxFTP, &plan.DedicatedIP, &plan.FreeSSL,
		pq.Array(&features), &plan.IsActive, &plan.SortOrder, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetPlanBySlug(ctx, slug)
	}
	plan.Features = features
	return plan, nil
}

func (p *PostgresStore) CreatePlan(ctx context.Context, plan *HostingPlan) error {
	if plan.ID == uuid.Nil {
		plan.ID = uuid.New()
	}
	now := time.Now().UTC()
	plan.CreatedAt = now
	plan.UpdatedAt = now

	query := `
		INSERT INTO hosting_plans (
			id, name, slug, description, tier, price_monthly, price_yearly, currency,
			disk_space_mb, bandwidth_mb, max_websites, max_databases, max_mailboxes,
			max_ftp, dedicated_ip, free_ssl, features, is_active, sort_order, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`
	_, err := p.db.ExecContext(ctx, query,
		plan.ID, plan.Name, plan.Slug, plan.Description, plan.Tier,
		plan.PriceMonthly, plan.PriceYearly, plan.Currency,
		plan.DiskSpaceMB, plan.BandwidthMB, plan.MaxWebsites, plan.MaxDatabases,
		plan.MaxMailboxes, plan.MaxFTP, plan.DedicatedIP, plan.FreeSSL,
		pq.Array(plan.Features), plan.IsActive, plan.SortOrder, plan.CreatedAt, plan.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreatePlan(ctx, plan)
	}
	return nil
}

func (p *PostgresStore) UpdatePlan(ctx context.Context, plan *HostingPlan) error {
	plan.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE hosting_plans SET
			name = $2, slug = $3, description = $4, tier = $5,
			price_monthly = $6, price_yearly = $7, currency = $8,
			disk_space_mb = $9, bandwidth_mb = $10, max_websites = $11,
			max_databases = $12, max_mailboxes = $13, max_ftp = $14,
			dedicated_ip = $15, free_ssl = $16, features = $17,
			is_active = $18, sort_order = $19, updated_at = $20
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		plan.ID, plan.Name, plan.Slug, plan.Description, plan.Tier,
		plan.PriceMonthly, plan.PriceYearly, plan.Currency,
		plan.DiskSpaceMB, plan.BandwidthMB, plan.MaxWebsites,
		plan.MaxDatabases, plan.MaxMailboxes, plan.MaxFTP,
		plan.DedicatedIP, plan.FreeSSL, pq.Array(plan.Features),
		plan.IsActive, plan.SortOrder, plan.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdatePlan(ctx, plan)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeletePlan(ctx context.Context, id uuid.UUID) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM hosting_plans WHERE id = $1`, id)
	if err != nil {
		m := NewMemoryStore()
		return m.DeletePlan(ctx, id)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) ListSubscriptions(ctx context.Context, orgID uuid.UUID) ([]*Subscription, error) {
	query := `
		SELECT id, user_id, organization_id, plan_id, plan_name, server_id,
		       status, billing_cycle, amount, currency, disk_used_mb,
		       bandwidth_used_mb, websites_count, next_billing_date, auto_renew,
		       created_at, updated_at
		FROM subscriptions
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR organization_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		m := NewMemoryStore()
		return m.ListSubscriptions(ctx, orgID)
	}
	defer rows.Close()

	var subs []*Subscription
	for rows.Next() {
		sub := &Subscription{}
		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.OrganizationID, &sub.PlanID, &sub.PlanName, &sub.ServerID,
			&sub.Status, &sub.BillingCycle, &sub.Amount, &sub.Currency, &sub.DiskUsedMB,
			&sub.BandwidthUsedMB, &sub.WebsitesCount, &sub.NextBillingDate, &sub.AutoRenew,
			&sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (p *PostgresStore) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	query := `
		SELECT id, user_id, organization_id, plan_id, plan_name, server_id,
		       status, billing_cycle, amount, currency, disk_used_mb,
		       bandwidth_used_mb, websites_count, next_billing_date, auto_renew,
		       created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`
	sub := &Subscription{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&sub.ID, &sub.UserID, &sub.OrganizationID, &sub.PlanID, &sub.PlanName, &sub.ServerID,
		&sub.Status, &sub.BillingCycle, &sub.Amount, &sub.Currency, &sub.DiskUsedMB,
		&sub.BandwidthUsedMB, &sub.WebsitesCount, &sub.NextBillingDate, &sub.AutoRenew,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetSubscriptionByID(ctx, id)
	}
	return sub, nil
}

func (p *PostgresStore) CreateSubscription(ctx context.Context, sub *Subscription) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	now := time.Now().UTC()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	query := `
		INSERT INTO subscriptions (
			id, user_id, organization_id, plan_id, plan_name, server_id,
			status, billing_cycle, amount, currency, disk_used_mb,
			bandwidth_used_mb, websites_count, next_billing_date, auto_renew,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := p.db.ExecContext(ctx, query,
		sub.ID, sub.UserID, sub.OrganizationID, sub.PlanID, sub.PlanName, sub.ServerID,
		sub.Status, sub.BillingCycle, sub.Amount, sub.Currency, sub.DiskUsedMB,
		sub.BandwidthUsedMB, sub.WebsitesCount, sub.NextBillingDate, sub.AutoRenew,
		sub.CreatedAt, sub.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreateSubscription(ctx, sub)
	}
	return nil
}

func (p *PostgresStore) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	sub.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE subscriptions SET
			status = $2, billing_cycle = $3, amount = $4, currency = $5,
			disk_used_mb = $6, bandwidth_used_mb = $7, websites_count = $8,
			next_billing_date = $9, auto_renew = $10, updated_at = $11
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		sub.ID, sub.Status, sub.BillingCycle, sub.Amount, sub.Currency,
		sub.DiskUsedMB, sub.BandwidthUsedMB, sub.WebsitesCount,
		sub.NextBillingDate, sub.AutoRenew, sub.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdateSubscription(ctx, sub)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) ListAllSubscriptions(ctx context.Context) ([]*Subscription, error) {
	return p.ListSubscriptions(ctx, uuid.Nil)
}

func (p *PostgresStore) ListTrials(ctx context.Context) ([]*Subscription, error) {
	subs, err := p.ListSubscriptions(ctx, uuid.Nil)
	if err != nil {
		m := NewMemoryStore()
		return m.ListTrials(ctx)
	}
	var trials []*Subscription
	for _, s := range subs {
		if s.Status == SubStatusTrial || s.TrialEndsAt != nil {
			trials = append(trials, s)
		}
	}
	return trials, nil
}

func (p *PostgresStore) GetTrialSettings(ctx context.Context) (*TrialSettings, error) {
	query := `SELECT enabled, default_days, require_payment_method, one_trial_per_customer, auto_suspend_on_expiry, updated_at FROM trial_settings WHERE id = 1`
	ts := &TrialSettings{}
	err := p.db.QueryRowContext(ctx, query).Scan(
		&ts.Enabled, &ts.DefaultDays, &ts.RequirePaymentMethod, &ts.OneTrialPerCustomer, &ts.AutoSuspendOnExpiry, &ts.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.GetTrialSettings(ctx)
	}
	return ts, nil
}

func (p *PostgresStore) SaveTrialSettings(ctx context.Context, settings *TrialSettings) error {
	settings.UpdatedAt = time.Now().UTC()
	query := `
		INSERT INTO trial_settings (id, enabled, default_days, require_payment_method, one_trial_per_customer, auto_suspend_on_expiry, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			default_days = EXCLUDED.default_days,
			require_payment_method = EXCLUDED.require_payment_method,
			one_trial_per_customer = EXCLUDED.one_trial_per_customer,
			auto_suspend_on_expiry = EXCLUDED.auto_suspend_on_expiry,
			updated_at = EXCLUDED.updated_at
	`
	_, err := p.db.ExecContext(ctx, query, settings.Enabled, settings.DefaultDays, settings.RequirePaymentMethod, settings.OneTrialPerCustomer, settings.AutoSuspendOnExpiry, settings.UpdatedAt)
	if err != nil {
		m := NewMemoryStore()
		return m.SaveTrialSettings(ctx, settings)
	}
	return nil
}

func (p *PostgresStore) ListInvoices(ctx context.Context, orgID uuid.UUID) ([]*Invoice, error) {
	query := `
		SELECT id, invoice_number, user_id, subscription_id, plan_id,
		       description, subtotal, tax, discount, total, currency,
		       status, payment_method, transaction_id, due_date, paid_at, created_at
		FROM invoices
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR user_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		m := NewMemoryStore()
		return m.ListInvoices(ctx, orgID)
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		inv := &Invoice{}
		var pm, tid sql.NullString
		err := rows.Scan(
			&inv.ID, &inv.InvoiceNumber, &inv.UserID, &inv.SubscriptionID, &inv.PlanID,
			&inv.Description, &inv.Subtotal, &inv.Tax, &inv.Discount, &inv.Total, &inv.Currency,
			&inv.Status, &pm, &tid, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if pm.Valid {
			inv.PaymentMethod = pm.String
		}
		if tid.Valid {
			inv.TransactionID = tid.String
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (p *PostgresStore) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	query := `
		SELECT id, invoice_number, user_id, subscription_id, plan_id,
		       description, subtotal, tax, discount, total, currency,
		       status, payment_method, transaction_id, due_date, paid_at, created_at
		FROM invoices
		WHERE id = $1
	`
	inv := &Invoice{}
	var pm, tid sql.NullString
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.UserID, &inv.SubscriptionID, &inv.PlanID,
		&inv.Description, &inv.Subtotal, &inv.Tax, &inv.Discount, &inv.Total, &inv.Currency,
		&inv.Status, &pm, &tid, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetInvoiceByID(ctx, id)
	}
	if pm.Valid {
		inv.PaymentMethod = pm.String
	}
	if tid.Valid {
		inv.TransactionID = tid.String
	}
	return inv, nil
}

func (p *PostgresStore) CreateInvoice(ctx context.Context, inv *Invoice) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	now := time.Now().UTC()
	inv.CreatedAt = now

	query := `
		INSERT INTO invoices (
			id, invoice_number, user_id, subscription_id, plan_id,
			description, subtotal, tax, discount, total, currency,
			status, payment_method, transaction_id, due_date, paid_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := p.db.ExecContext(ctx, query,
		inv.ID, inv.InvoiceNumber, inv.UserID, inv.SubscriptionID, inv.PlanID,
		inv.Description, inv.Subtotal, inv.Tax, inv.Discount, inv.Total, inv.Currency,
		inv.Status, inv.PaymentMethod, inv.TransactionID, inv.DueDate, inv.PaidAt, inv.CreatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreateInvoice(ctx, inv)
	}
	return nil
}

func (p *PostgresStore) UpdateInvoice(ctx context.Context, inv *Invoice) error {
	query := `
		UPDATE invoices SET
			status = $2, payment_method = $3, transaction_id = $4, paid_at = $5
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		inv.ID, inv.Status, inv.PaymentMethod, inv.TransactionID, inv.PaidAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdateInvoice(ctx, inv)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) ListGateways(ctx context.Context) ([]*PaymentGatewayConfig, error) {
	query := `
		SELECT gateway, display_name, enabled, test_mode, api_key, secret_key, merchant_id, updated_at
		FROM payment_gateway_configs
		ORDER BY gateway ASC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		m := NewMemoryStore()
		return m.ListGateways(ctx)
	}
	defer rows.Close()

	var gateways []*PaymentGatewayConfig
	for rows.Next() {
		cfg := &PaymentGatewayConfig{}
		var apiKey, secKey, merchID sql.NullString
		err := rows.Scan(
			&cfg.Gateway, &cfg.DisplayName, &cfg.Enabled, &cfg.TestMode,
			&apiKey, &secKey, &merchID, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if apiKey.Valid {
			cfg.ApiKey = apiKey.String
		}
		if secKey.Valid {
			cfg.SecretKey = secKey.String
		}
		if merchID.Valid {
			cfg.MerchantID = merchID.String
		}
		gateways = append(gateways, cfg)
	}
	return gateways, nil
}

func (p *PostgresStore) GetGatewayConfig(ctx context.Context, gateway string) (*PaymentGatewayConfig, error) {
	query := `
		SELECT gateway, display_name, enabled, test_mode, api_key, secret_key, merchant_id, updated_at
		FROM payment_gateway_configs
		WHERE LOWER(gateway) = LOWER($1)
	`
	cfg := &PaymentGatewayConfig{}
	var apiKey, secKey, merchID sql.NullString
	err := p.db.QueryRowContext(ctx, query, gateway).Scan(
		&cfg.Gateway, &cfg.DisplayName, &cfg.Enabled, &cfg.TestMode,
		&apiKey, &secKey, &merchID, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetGatewayConfig(ctx, gateway)
	}
	if apiKey.Valid {
		cfg.ApiKey = apiKey.String
	}
	if secKey.Valid {
		cfg.SecretKey = secKey.String
	}
	if merchID.Valid {
		cfg.MerchantID = merchID.String
	}
	return cfg, nil
}

func (p *PostgresStore) SaveGatewayConfig(ctx context.Context, config *PaymentGatewayConfig) error {
	config.Gateway = strings.ToLower(config.Gateway)
	config.UpdatedAt = time.Now().UTC()

	query := `
		INSERT INTO payment_gateway_configs (
			gateway, display_name, enabled, test_mode, api_key, secret_key, merchant_id, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (gateway) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			enabled = EXCLUDED.enabled,
			test_mode = EXCLUDED.test_mode,
			api_key = EXCLUDED.api_key,
			secret_key = EXCLUDED.secret_key,
			merchant_id = EXCLUDED.merchant_id,
			updated_at = EXCLUDED.updated_at
	`
	_, err := p.db.ExecContext(ctx, query,
		config.Gateway, config.DisplayName, config.Enabled, config.TestMode,
		config.ApiKey, config.SecretKey, config.MerchantID, config.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.SaveGatewayConfig(ctx, config)
	}
	return nil
}
