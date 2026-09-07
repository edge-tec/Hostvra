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
	m := NewMemoryStore()
	return m.ListHostingAccounts(ctx, orgID, serverID)
}

func (p *PostgresStore) GetHostingAccountByID(ctx context.Context, id uuid.UUID) (*HostingAccount, error) {
	m := NewMemoryStore()
	return m.GetHostingAccountByID(ctx, id)
}

func (p *PostgresStore) GetHostingAccountByUsername(ctx context.Context, username string) (*HostingAccount, error) {
	m := NewMemoryStore()
	return m.GetHostingAccountByUsername(ctx, username)
}

func (p *PostgresStore) GetHostingAccountByDomain(ctx context.Context, domain string) (*HostingAccount, error) {
	m := NewMemoryStore()
	return m.GetHostingAccountByDomain(ctx, domain)
}

func (p *PostgresStore) CreateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	m := NewMemoryStore()
	return m.CreateHostingAccount(ctx, acc)
}

func (p *PostgresStore) UpdateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	m := NewMemoryStore()
	return m.UpdateHostingAccount(ctx, acc)
}

func (p *PostgresStore) DeleteHostingAccount(ctx context.Context, id uuid.UUID) error {
	m := NewMemoryStore()
	return m.DeleteHostingAccount(ctx, id)
}
