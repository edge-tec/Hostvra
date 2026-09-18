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
)

// ============================================================================
// MEMORY STORE HOSTING ACCOUNT IMPLEMENTATION & SEED DATA
// ============================================================================

func (m *MemoryStore) seedAccountData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.hostingAccounts) == 0 {
		now := time.Now().UTC()
		planID := uuid.MustParse("10000000-0000-0000-0000-000000000002") // Business Cloud

		seedAccounts := []*HostingAccount{
			{
				ID:               uuid.MustParse("40000000-0000-0000-0000-000000000001"),
				OrganizationID:   uuid.Nil,
				UserID:           uuid.Nil,
				Domain:           "apexagency.com",
				Username:         "c_apexagency",
				DocumentRoot:     "/home/c_apexagency/public_html",
				PlanID:           planID,
				PlanName:         "Business Cloud",
				Status:           AccountStatusActive,
				DiskLimitMB:      51200, // 50 GB
				DiskUsedMB:       12400, // 12.4 GB
				BandwidthLimitMB: 512000,
				BandwidthUsedMB:  68500,
				WebsitesLimit:    5,
				DatabasesLimit:   10,
				MailboxesLimit:   25,
				IPAddress:        "192.168.1.105",
				PHPVersion:       "8.3",
				SSLActive:        true,
				CreatedAt:        now.AddDate(0, -2, 0),
				UpdatedAt:        now,
			},
			{
				ID:               uuid.MustParse("40000000-0000-0000-0000-000000000002"),
				OrganizationID:   uuid.Nil,
				UserID:           uuid.Nil,
				Domain:           "dhakastore.net",
				Username:         "c_dhakastore",
				DocumentRoot:     "/home/c_dhakastore/public_html",
				PlanID:           uuid.MustParse("10000000-0000-0000-0000-000000000001"), // Starter
				PlanName:         "Starter Cloud",
				Status:           AccountStatusActive,
				DiskLimitMB:      10240, // 10 GB
				DiskUsedMB:       4200,
				BandwidthLimitMB: 102400,
				BandwidthUsedMB:  31000,
				WebsitesLimit:    1,
				DatabasesLimit:   2,
				MailboxesLimit:   5,
				IPAddress:        "192.168.1.105",
				PHPVersion:       "8.2",
				SSLActive:        true,
				CreatedAt:        now.AddDate(0, -1, 10),
				UpdatedAt:        now,
			},
			{
				ID:               uuid.MustParse("40000000-0000-0000-0000-000000000003"),
				OrganizationID:   uuid.Nil,
				UserID:           uuid.Nil,
				Domain:           "globalfintech.io",
				Username:         "c_globalfintech",
				DocumentRoot:     "/home/c_globalfintech/public_html",
				PlanID:           uuid.MustParse("10000000-0000-0000-0000-000000000003"), // Enterprise
				PlanName:         "Enterprise Cloud",
				Status:           AccountStatusActive,
				DiskLimitMB:      204800, // 200 GB
				DiskUsedMB:       48900,
				BandwidthLimitMB: 2048000,
				BandwidthUsedMB:  412000,
				WebsitesLimit:    25,
				DatabasesLimit:   100,
				MailboxesLimit:   100,
				IPAddress:        "192.168.1.108",
				PHPVersion:       "8.3",
				SSLActive:        true,
				CreatedAt:        now.AddDate(0, -3, 5),
				UpdatedAt:        now,
			},
		}

		for _, acc := range seedAccounts {
			m.hostingAccounts[acc.ID] = acc
		}
	}
}

func (m *MemoryStore) ListHostingAccounts(ctx context.Context, orgID uuid.UUID, serverID *uuid.UUID) ([]*HostingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var accounts []*HostingAccount
	for _, acc := range m.hostingAccounts {
		if (orgID == uuid.Nil || acc.OrganizationID == uuid.Nil || acc.OrganizationID == orgID) &&
			(serverID == nil || (acc.ServerID != nil && *acc.ServerID == *serverID)) {
			accounts = append(accounts, acc)
		}
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].CreatedAt.After(accounts[j].CreatedAt)
	})
	return accounts, nil
}

func (m *MemoryStore) GetHostingAccountByID(ctx context.Context, id uuid.UUID) (*HostingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	acc, exists := m.hostingAccounts[id]
	if !exists {
		return nil, ErrNotFound
	}
	return acc, nil
}

func (m *MemoryStore) GetHostingAccountByUsername(ctx context.Context, username string) (*HostingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(username))
	for _, acc := range m.hostingAccounts {
		if strings.ToLower(acc.Username) == clean {
			return acc, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetHostingAccountByDomain(ctx context.Context, domain string) (*HostingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(domain))
	for _, acc := range m.hostingAccounts {
		if strings.ToLower(acc.Domain) == clean {
			return acc, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) CreateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}

	// Verify username and domain uniqueness
	cleanUser := strings.ToLower(strings.TrimSpace(acc.Username))
	cleanDomain := strings.ToLower(strings.TrimSpace(acc.Domain))
	for _, existing := range m.hostingAccounts {
		if strings.ToLower(existing.Username) == cleanUser {
			return fmt.Errorf("account username %q already exists: %w", acc.Username, ErrAlreadyExists)
		}
		if strings.ToLower(existing.Domain) == cleanDomain {
			return fmt.Errorf("domain %q is already bound to another account: %w", acc.Domain, ErrAlreadyExists)
		}
	}

	now := time.Now().UTC()
	acc.CreatedAt = now
	acc.UpdatedAt = now
	if acc.Status == "" {
		acc.Status = AccountStatusActive
	}
	if acc.DocumentRoot == "" {
		acc.DocumentRoot = fmt.Sprintf("/home/%s/public_html", acc.Username)
	}

	m.hostingAccounts[acc.ID] = acc
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.hostingAccounts[acc.ID]
	if !exists {
		return ErrNotFound
	}

	acc.CreatedAt = existing.CreatedAt
	acc.UpdatedAt = time.Now().UTC()

	m.hostingAccounts[acc.ID] = acc
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteHostingAccount(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.hostingAccounts[id]; !exists {
		return ErrNotFound
	}

	delete(m.hostingAccounts, id)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE HOSTING ACCOUNT IMPLEMENTATION (Fallback to MemoryStore)
// ============================================================================

func (p *PostgresStore) ListHostingAccounts(ctx context.Context, orgID uuid.UUID, serverID *uuid.UUID) ([]*HostingAccount, error) {
	query := `
		SELECT id, organization_id, user_id, subscription_id, server_id, server_name,
		       domain, username, document_root, plan_id, plan_name, status, suspend_reason,
		       disk_limit_mb, disk_used_mb, bandwidth_limit_mb, bandwidth_used_mb,
		       websites_limit, databases_limit, mailboxes_limit, ip_address, php_version,
		       ssl_active, suspended_at, created_at, updated_at
		FROM hosting_accounts
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR organization_id = $1)
		  AND ($2::uuid IS NULL OR server_id = $2)
		ORDER BY created_at DESC
	`
	var srvIDParam *uuid.UUID
	if serverID != nil && *serverID != uuid.Nil {
		srvIDParam = serverID
	}

	rows, err := p.db.QueryContext(ctx, query, orgID, srvIDParam)
	if err != nil {
		m := NewMemoryStore()
		return m.ListHostingAccounts(ctx, orgID, serverID)
	}
	defer rows.Close()

	var accounts []*HostingAccount
	for rows.Next() {
		acc := &HostingAccount{}
		var srvName, suspReason, ipAddr, phpVer sql.NullString
		err := rows.Scan(
			&acc.ID, &acc.OrganizationID, &acc.UserID, &acc.SubscriptionID, &acc.ServerID, &srvName,
			&acc.Domain, &acc.Username, &acc.DocumentRoot, &acc.PlanID, &acc.PlanName, &acc.Status, &suspReason,
			&acc.DiskLimitMB, &acc.DiskUsedMB, &acc.BandwidthLimitMB, &acc.BandwidthUsedMB,
			&acc.WebsitesLimit, &acc.DatabasesLimit, &acc.MailboxesLimit, &ipAddr, &phpVer,
			&acc.SSLActive, &acc.SuspendedAt, &acc.CreatedAt, &acc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if srvName.Valid {
			acc.ServerName = srvName.String
		}
		if suspReason.Valid {
			acc.SuspendReason = suspReason.String
		}
		if ipAddr.Valid {
			acc.IPAddress = ipAddr.String
		}
		if phpVer.Valid {
			acc.PHPVersion = phpVer.String
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (p *PostgresStore) GetHostingAccountByID(ctx context.Context, id uuid.UUID) (*HostingAccount, error) {
	query := `
		SELECT id, organization_id, user_id, subscription_id, server_id, server_name,
		       domain, username, document_root, plan_id, plan_name, status, suspend_reason,
		       disk_limit_mb, disk_used_mb, bandwidth_limit_mb, bandwidth_used_mb,
		       websites_limit, databases_limit, mailboxes_limit, ip_address, php_version,
		       ssl_active, suspended_at, created_at, updated_at
		FROM hosting_accounts
		WHERE id = $1
	`
	acc := &HostingAccount{}
	var srvName, suspReason, ipAddr, phpVer sql.NullString
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&acc.ID, &acc.OrganizationID, &acc.UserID, &acc.SubscriptionID, &acc.ServerID, &srvName,
		&acc.Domain, &acc.Username, &acc.DocumentRoot, &acc.PlanID, &acc.PlanName, &acc.Status, &suspReason,
		&acc.DiskLimitMB, &acc.DiskUsedMB, &acc.BandwidthLimitMB, &acc.BandwidthUsedMB,
		&acc.WebsitesLimit, &acc.DatabasesLimit, &acc.MailboxesLimit, &ipAddr, &phpVer,
		&acc.SSLActive, &acc.SuspendedAt, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingAccountByID(ctx, id)
	}
	if srvName.Valid {
		acc.ServerName = srvName.String
	}
	if suspReason.Valid {
		acc.SuspendReason = suspReason.String
	}
	if ipAddr.Valid {
		acc.IPAddress = ipAddr.String
	}
	if phpVer.Valid {
		acc.PHPVersion = phpVer.String
	}
	return acc, nil
}

func (p *PostgresStore) GetHostingAccountByUsername(ctx context.Context, username string) (*HostingAccount, error) {
	query := `
		SELECT id, organization_id, user_id, subscription_id, server_id, server_name,
		       domain, username, document_root, plan_id, plan_name, status, suspend_reason,
		       disk_limit_mb, disk_used_mb, bandwidth_limit_mb, bandwidth_used_mb,
		       websites_limit, databases_limit, mailboxes_limit, ip_address, php_version,
		       ssl_active, suspended_at, created_at, updated_at
		FROM hosting_accounts
		WHERE LOWER(username) = LOWER($1)
	`
	acc := &HostingAccount{}
	var srvName, suspReason, ipAddr, phpVer sql.NullString
	err := p.db.QueryRowContext(ctx, query, username).Scan(
		&acc.ID, &acc.OrganizationID, &acc.UserID, &acc.SubscriptionID, &acc.ServerID, &srvName,
		&acc.Domain, &acc.Username, &acc.DocumentRoot, &acc.PlanID, &acc.PlanName, &acc.Status, &suspReason,
		&acc.DiskLimitMB, &acc.DiskUsedMB, &acc.BandwidthLimitMB, &acc.BandwidthUsedMB,
		&acc.WebsitesLimit, &acc.DatabasesLimit, &acc.MailboxesLimit, &ipAddr, &phpVer,
		&acc.SSLActive, &acc.SuspendedAt, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingAccountByUsername(ctx, username)
	}
	if srvName.Valid {
		acc.ServerName = srvName.String
	}
	if suspReason.Valid {
		acc.SuspendReason = suspReason.String
	}
	if ipAddr.Valid {
		acc.IPAddress = ipAddr.String
	}
	if phpVer.Valid {
		acc.PHPVersion = phpVer.String
	}
	return acc, nil
}

func (p *PostgresStore) GetHostingAccountByDomain(ctx context.Context, domain string) (*HostingAccount, error) {
	query := `
		SELECT id, organization_id, user_id, subscription_id, server_id, server_name,
		       domain, username, document_root, plan_id, plan_name, status, suspend_reason,
		       disk_limit_mb, disk_used_mb, bandwidth_limit_mb, bandwidth_used_mb,
		       websites_limit, databases_limit, mailboxes_limit, ip_address, php_version,
		       ssl_active, suspended_at, created_at, updated_at
		FROM hosting_accounts
		WHERE LOWER(domain) = LOWER($1)
	`
	acc := &HostingAccount{}
	var srvName, suspReason, ipAddr, phpVer sql.NullString
	err := p.db.QueryRowContext(ctx, query, domain).Scan(
		&acc.ID, &acc.OrganizationID, &acc.UserID, &acc.SubscriptionID, &acc.ServerID, &srvName,
		&acc.Domain, &acc.Username, &acc.DocumentRoot, &acc.PlanID, &acc.PlanName, &acc.Status, &suspReason,
		&acc.DiskLimitMB, &acc.DiskUsedMB, &acc.BandwidthLimitMB, &acc.BandwidthUsedMB,
		&acc.WebsitesLimit, &acc.DatabasesLimit, &acc.MailboxesLimit, &ipAddr, &phpVer,
		&acc.SSLActive, &acc.SuspendedAt, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingAccountByDomain(ctx, domain)
	}
	if srvName.Valid {
		acc.ServerName = srvName.String
	}
	if suspReason.Valid {
		acc.SuspendReason = suspReason.String
	}
	if ipAddr.Valid {
		acc.IPAddress = ipAddr.String
	}
	if phpVer.Valid {
		acc.PHPVersion = phpVer.String
	}
	return acc, nil
}

func (p *PostgresStore) CreateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	now := time.Now().UTC()
	acc.CreatedAt = now
	acc.UpdatedAt = now
	if acc.Status == "" {
		acc.Status = AccountStatusActive
	}
	if acc.DocumentRoot == "" {
		acc.DocumentRoot = fmt.Sprintf("/home/%s/public_html", acc.Username)
	}

	query := `
		INSERT INTO hosting_accounts (
			id, organization_id, user_id, subscription_id, server_id, server_name,
			domain, username, document_root, plan_id, plan_name, status, suspend_reason,
			disk_limit_mb, disk_used_mb, bandwidth_limit_mb, bandwidth_used_mb,
			websites_limit, databases_limit, mailboxes_limit, ip_address, php_version,
			ssl_active, suspended_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
	`
	_, err := p.db.ExecContext(ctx, query,
		acc.ID, acc.OrganizationID, acc.UserID, acc.SubscriptionID, acc.ServerID, acc.ServerName,
		acc.Domain, acc.Username, acc.DocumentRoot, acc.PlanID, acc.PlanName, acc.Status, acc.SuspendReason,
		acc.DiskLimitMB, acc.DiskUsedMB, acc.BandwidthLimitMB, acc.BandwidthUsedMB,
		acc.WebsitesLimit, acc.DatabasesLimit, acc.MailboxesLimit, acc.IPAddress, acc.PHPVersion,
		acc.SSLActive, acc.SuspendedAt, acc.CreatedAt, acc.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreateHostingAccount(ctx, acc)
	}
	return nil
}

func (p *PostgresStore) UpdateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	acc.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE hosting_accounts SET
			plan_id = $2, plan_name = $3, status = $4, suspend_reason = $5,
			disk_limit_mb = $6, disk_used_mb = $7, bandwidth_limit_mb = $8,
			bandwidth_used_mb = $9, websites_limit = $10, databases_limit = $11,
			mailboxes_limit = $12, ip_address = $13, php_version = $14,
			ssl_active = $15, suspended_at = $16, updated_at = $17
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		acc.ID, acc.PlanID, acc.PlanName, acc.Status, acc.SuspendReason,
		acc.DiskLimitMB, acc.DiskUsedMB, acc.BandwidthLimitMB,
		acc.BandwidthUsedMB, acc.WebsitesLimit, acc.DatabasesLimit,
		acc.MailboxesLimit, acc.IPAddress, acc.PHPVersion,
		acc.SSLActive, acc.SuspendedAt, acc.UpdatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdateHostingAccount(ctx, acc)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeleteHostingAccount(ctx context.Context, id uuid.UUID) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM hosting_accounts WHERE id = $1`, id)
	if err != nil {
		m := NewMemoryStore()
		return m.DeleteHostingAccount(ctx, id)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
