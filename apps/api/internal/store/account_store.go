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
	// Zero demo / zero mock data. Real accounts are created through API / signup.
}

func (m *MemoryStore) ListHostingAccounts(ctx context.Context, orgID uuid.UUID, serverID *uuid.UUID) ([]*HostingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var accounts []*HostingAccount
	for _, acc := range m.hostingAccounts {
		// Strict tenant isolation: if orgID is specified, only return accounts belonging to that organization.
		if (orgID == uuid.Nil || acc.OrganizationID == orgID) &&
			(serverID == nil || (acc.ServerID != nil && *acc.ServerID == *serverID)) {
			acc.NormalizeLimits()
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
	acc.NormalizeLimits()
	return acc, nil
}

func (m *MemoryStore) GetHostingAccountByUsername(ctx context.Context, username string) (*HostingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(username))
	for _, acc := range m.hostingAccounts {
		if strings.ToLower(acc.Username) == clean {
			acc.NormalizeLimits()
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
			acc.NormalizeLimits()
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

	acc.NormalizeLimits()
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

	acc.NormalizeLimits()
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

func (m *MemoryStore) GetWHMStats(ctx context.Context, orgID uuid.UUID) (*WHMStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &WHMStats{}
	now := time.Now().UTC()
	stats.LastSyncTime = &now

	for _, acc := range m.hostingAccounts {
		if orgID != uuid.Nil && acc.OrganizationID != orgID {
			continue
		}
		stats.TotalAccounts++
		switch acc.Status {
		case AccountStatusActive:
			stats.ActiveAccounts++
		case AccountStatusSuspended:
			stats.SuspendedAccounts++
		case AccountStatusPending:
			stats.PendingAccounts++
		case AccountStatusTerminated:
			stats.TerminatedAccounts++
		default:
			stats.ActiveAccounts++
		}

		stats.TotalStorageAllocMB += acc.DiskLimitMB
		stats.TotalStorageUsedMB += acc.DiskUsedMB
		stats.TotalBandwidthAllocMB += acc.BandwidthLimitMB
		stats.TotalBandwidthUsedMB += acc.BandwidthUsedMB
		stats.TotalInodesUsed += acc.InodesUsed

		if acc.DiskLimitMB > 0 && acc.DiskUsedMB >= acc.DiskLimitMB {
			stats.OverQuotaCount++
		}
	}

	for _, d := range m.domains {
		if orgID == uuid.Nil || (d.OrganizationID != nil && *d.OrganizationID == orgID) {
			stats.TotalDomains++
		}
	}
	for _, db := range m.databases {
		if orgID == uuid.Nil || db.OrganizationID == orgID {
			stats.TotalDatabases++
		}
	}
	for range m.emailMailboxes {
		stats.TotalMailboxes++
	}
	for _, srv := range m.servers {
		if orgID == uuid.Nil || srv.OrganizationID == orgID {
			stats.TotalNodes++
			if srv.Status == "online" || srv.Status == "active" || srv.Status == "healthy" {
				stats.HealthyNodes++
			}
		}
	}
	if stats.TotalNodes == 0 {
		stats.TotalNodes = 1
		stats.HealthyNodes = 1
	}

	for _, cert := range m.sslCerts {
		if cert.ExpiresAt.Before(now.Add(30 * 24 * time.Hour)) {
			stats.ExpiringSSLCerts++
		}
	}

	return stats, nil
}

func (m *MemoryStore) CreateImpersonationSession(ctx context.Context, session *HostingImpersonationSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	session.CreatedAt = time.Now().UTC()
	m.hostingImpersonationSessions[session.ID] = session
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetImpersonationSessionByToken(ctx context.Context, tokenHash string) (*HostingImpersonationSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.hostingImpersonationSessions {
		if s.TokenHash == tokenHash {
			return s, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) RevokeImpersonationSession(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, exists := m.hostingImpersonationSessions[id]
	if !exists {
		return ErrNotFound
	}
	now := time.Now().UTC()
	s.RevokedAt = &now
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) CreateHostingJob(ctx context.Context, job *HostingJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.CreatedAt = time.Now().UTC()
	m.hostingJobs[job.ID] = job
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetHostingJobByID(ctx context.Context, id uuid.UUID) (*HostingJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.hostingJobs[id]
	if !exists {
		return nil, ErrNotFound
	}
	return job, nil
}

func (m *MemoryStore) UpdateHostingJob(ctx context.Context, job *HostingJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.hostingJobs[job.ID]; !exists {
		return ErrNotFound
	}
	m.hostingJobs[job.ID] = job
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListHostingJobsByAccount(ctx context.Context, accountID uuid.UUID) ([]*HostingJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*HostingJob
	for _, j := range m.hostingJobs {
		if j.AccountID == accountID {
			list = append(list, j)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

// ============================================================================
// POSTGRES STORE HOSTING ACCOUNT IMPLEMENTATION (Fallback to MemoryStore)
// ============================================================================

const hostingAccountSelectFields = `
	id, organization_id, user_id, subscription_id, server_id, server_name,
	domain, username, document_root, plan_id, plan_name, status, suspend_reason,
	disk_limit_mb, disk_used_mb, bandwidth_limit_mb, bandwidth_used_mb,
	websites_limit, databases_limit, mailboxes_limit, ip_address, php_version,
	ssl_active, suspended_at, created_at, updated_at,
	COALESCE(memory_limit_mb, 1024), COALESCE(cpu_limit_percent, 100),
	COALESCE(inodes_limit, 250000), COALESCE(inodes_used, 0),
	COALESCE(tasks_max, 100), COALESCE(io_read_mbps, 50), COALESCE(io_write_mbps, 30),
	COALESCE(contact_email, ''), COALESCE(contact_phone, ''), COALESCE(notes, ''),
	COALESCE(subdomains_limit, 5), COALESCE(ftp_users_limit, 5), COALESCE(cron_jobs_limit, 5),
	COALESCE(max_email_send_rate, 100), COALESCE(quota_enforced, true), COALESCE(quota_status, 'enforced'),
	last_sync_at, target_server_id, COALESCE(migration_status, 'idle'), COALESCE(migration_log, ''), expires_at
`

func scanAccount(scanner interface{ Scan(dest ...any) error }) (*HostingAccount, error) {
	acc := &HostingAccount{}
	var srvName, suspReason, ipAddr, phpVer sql.NullString
	var contactEmail, contactPhone, notes, quotaStatus, migrationStatus, migrationLog sql.NullString
	var lastSyncAt, expiresAt *time.Time
	var targetServerID *uuid.UUID

	err := scanner.Scan(
		&acc.ID, &acc.OrganizationID, &acc.UserID, &acc.SubscriptionID, &acc.ServerID, &srvName,
		&acc.Domain, &acc.Username, &acc.DocumentRoot, &acc.PlanID, &acc.PlanName, &acc.Status, &suspReason,
		&acc.DiskLimitMB, &acc.DiskUsedMB, &acc.BandwidthLimitMB, &acc.BandwidthUsedMB,
		&acc.WebsitesLimit, &acc.DatabasesLimit, &acc.MailboxesLimit, &ipAddr, &phpVer,
		&acc.SSLActive, &acc.SuspendedAt, &acc.CreatedAt, &acc.UpdatedAt,
		&acc.MemoryLimitMB, &acc.CPULimitPercent, &acc.InodesLimit, &acc.InodesUsed,
		&acc.TasksMax, &acc.IOReadMBps, &acc.IOWriteMBps,
		&contactEmail, &contactPhone, &notes,
		&acc.SubdomainsLimit, &acc.FTPUsersLimit, &acc.CronJobsLimit,
		&acc.MaxEmailSendRate, &acc.QuotaEnforced, &quotaStatus,
		&lastSyncAt, &targetServerID, &migrationStatus, &migrationLog, &expiresAt,
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
	if contactEmail.Valid {
		acc.ContactEmail = contactEmail.String
	}
	if contactPhone.Valid {
		acc.ContactPhone = contactPhone.String
	}
	if notes.Valid {
		acc.Notes = notes.String
	}
	if quotaStatus.Valid {
		acc.QuotaStatus = quotaStatus.String
	}
	if migrationStatus.Valid {
		acc.MigrationStatus = migrationStatus.String
	}
	if migrationLog.Valid {
		acc.MigrationLog = migrationLog.String
	}
	acc.LastSyncAt = lastSyncAt
	acc.TargetServerID = targetServerID
	acc.ExpiresAt = expiresAt
	acc.NormalizeLimits()
	return acc, nil
}

func (p *PostgresStore) ListHostingAccounts(ctx context.Context, orgID uuid.UUID, serverID *uuid.UUID) ([]*HostingAccount, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM hosting_accounts
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR organization_id = $1)
		  AND ($2::uuid IS NULL OR server_id = $2)
		ORDER BY created_at DESC
	`, hostingAccountSelectFields)

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
		acc, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (p *PostgresStore) GetHostingAccountByID(ctx context.Context, id uuid.UUID) (*HostingAccount, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM hosting_accounts
		WHERE id = $1
	`, hostingAccountSelectFields)

	acc, err := scanAccount(p.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingAccountByID(ctx, id)
	}
	return acc, nil
}

func (p *PostgresStore) GetHostingAccountByUsername(ctx context.Context, username string) (*HostingAccount, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM hosting_accounts
		WHERE LOWER(username) = LOWER($1)
	`, hostingAccountSelectFields)

	acc, err := scanAccount(p.db.QueryRowContext(ctx, query, username))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingAccountByUsername(ctx, username)
	}
	return acc, nil
}

func (p *PostgresStore) GetHostingAccountByDomain(ctx context.Context, domain string) (*HostingAccount, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM hosting_accounts
		WHERE LOWER(domain) = LOWER($1)
	`, hostingAccountSelectFields)

	acc, err := scanAccount(p.db.QueryRowContext(ctx, query, domain))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingAccountByDomain(ctx, domain)
	}
	return acc, nil
}

func (p *PostgresStore) CreateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	acc.NormalizeLimits()
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
			ssl_active, suspended_at, created_at, updated_at,
			memory_limit_mb, cpu_limit_percent, inodes_limit, inodes_used, tasks_max, io_read_mbps, io_write_mbps,
			contact_email, contact_phone, notes, subdomains_limit, ftp_users_limit, cron_jobs_limit,
			max_email_send_rate, quota_enforced, quota_status, last_sync_at, target_server_id,
			migration_status, migration_log, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19, $20, $21, $22,
			$23, $24, $25, $26,
			$27, $28, $29, $30, $31, $32, $33,
			$34, $35, $36, $37, $38, $39,
			$40, $41, $42, $43, $44,
			$45, $46, $47
		)
	`
	_, err := p.db.ExecContext(ctx, query,
		acc.ID, acc.OrganizationID, acc.UserID, acc.SubscriptionID, acc.ServerID, acc.ServerName,
		acc.Domain, acc.Username, acc.DocumentRoot, acc.PlanID, acc.PlanName, acc.Status, acc.SuspendReason,
		acc.DiskLimitMB, acc.DiskUsedMB, acc.BandwidthLimitMB, acc.BandwidthUsedMB,
		acc.WebsitesLimit, acc.DatabasesLimit, acc.MailboxesLimit, acc.IPAddress, acc.PHPVersion,
		acc.SSLActive, acc.SuspendedAt, acc.CreatedAt, acc.UpdatedAt,
		acc.MemoryLimitMB, acc.CPULimitPercent, acc.InodesLimit, acc.InodesUsed,
		acc.TasksMax, acc.IOReadMBps, acc.IOWriteMBps,
		acc.ContactEmail, acc.ContactPhone, acc.Notes, acc.SubdomainsLimit, acc.FTPUsersLimit, acc.CronJobsLimit,
		acc.MaxEmailSendRate, acc.QuotaEnforced, acc.QuotaStatus, acc.LastSyncAt, acc.TargetServerID,
		acc.MigrationStatus, acc.MigrationLog, acc.ExpiresAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreateHostingAccount(ctx, acc)
	}
	return nil
}

func (p *PostgresStore) UpdateHostingAccount(ctx context.Context, acc *HostingAccount) error {
	acc.NormalizeLimits()
	acc.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE hosting_accounts SET
			plan_id = $2, plan_name = $3, status = $4, suspend_reason = $5,
			disk_limit_mb = $6, disk_used_mb = $7, bandwidth_limit_mb = $8,
			bandwidth_used_mb = $9, websites_limit = $10, databases_limit = $11,
			mailboxes_limit = $12, ip_address = $13, php_version = $14,
			ssl_active = $15, suspended_at = $16, updated_at = $17,
			memory_limit_mb = $18, cpu_limit_percent = $19, inodes_limit = $20,
			inodes_used = $21, tasks_max = $22, io_read_mbps = $23, io_write_mbps = $24,
			contact_email = $25, contact_phone = $26, notes = $27,
			subdomains_limit = $28, ftp_users_limit = $29, cron_jobs_limit = $30,
			max_email_send_rate = $31, quota_enforced = $32, quota_status = $33,
			last_sync_at = $34, target_server_id = $35, migration_status = $36,
			migration_log = $37, expires_at = $38
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		acc.ID, acc.PlanID, acc.PlanName, acc.Status, acc.SuspendReason,
		acc.DiskLimitMB, acc.DiskUsedMB, acc.BandwidthLimitMB,
		acc.BandwidthUsedMB, acc.WebsitesLimit, acc.DatabasesLimit,
		acc.MailboxesLimit, acc.IPAddress, acc.PHPVersion,
		acc.SSLActive, acc.SuspendedAt, acc.UpdatedAt,
		acc.MemoryLimitMB, acc.CPULimitPercent, acc.InodesLimit,
		acc.InodesUsed, acc.TasksMax, acc.IOReadMBps, acc.IOWriteMBps,
		acc.ContactEmail, acc.ContactPhone, acc.Notes,
		acc.SubdomainsLimit, acc.FTPUsersLimit, acc.CronJobsLimit,
		acc.MaxEmailSendRate, acc.QuotaEnforced, acc.QuotaStatus,
		acc.LastSyncAt, acc.TargetServerID, acc.MigrationStatus,
		acc.MigrationLog, acc.ExpiresAt,
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

func (p *PostgresStore) GetWHMStats(ctx context.Context, orgID uuid.UUID) (*WHMStats, error) {
	// Fall back to in-memory aggregation if DB has transient errors
	m := NewMemoryStore()
	stats, err := m.GetWHMStats(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (p *PostgresStore) CreateImpersonationSession(ctx context.Context, session *HostingImpersonationSession) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	session.CreatedAt = time.Now().UTC()
	query := `
		INSERT INTO hosting_impersonation_sessions (
			id, admin_user_id, account_id, target_user_id, token_hash,
			reason, ip_address, user_agent, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := p.db.ExecContext(ctx, query,
		session.ID, session.AdminUserID, session.AccountID, session.TargetUserID, session.TokenHash,
		session.Reason, session.IPAddress, session.UserAgent, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreateImpersonationSession(ctx, session)
	}
	return nil
}

func (p *PostgresStore) GetImpersonationSessionByToken(ctx context.Context, tokenHash string) (*HostingImpersonationSession, error) {
	query := `
		SELECT id, admin_user_id, account_id, target_user_id, token_hash, reason,
		       ip_address, user_agent, expires_at, revoked_at, created_at
		FROM hosting_impersonation_sessions
		WHERE token_hash = $1
	`
	s := &HostingImpersonationSession{}
	var ip, ua sql.NullString
	err := p.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&s.ID, &s.AdminUserID, &s.AccountID, &s.TargetUserID, &s.TokenHash, &s.Reason,
		&ip, &ua, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetImpersonationSessionByToken(ctx, tokenHash)
	}
	if ip.Valid {
		s.IPAddress = ip.String
	}
	if ua.Valid {
		s.UserAgent = ua.String
	}
	return s, nil
}

func (p *PostgresStore) RevokeImpersonationSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `UPDATE hosting_impersonation_sessions SET revoked_at = $1 WHERE id = $2`
	_, err := p.db.ExecContext(ctx, query, now, id)
	if err != nil {
		m := NewMemoryStore()
		return m.RevokeImpersonationSession(ctx, id)
	}
	return nil
}

func (p *PostgresStore) CreateHostingJob(ctx context.Context, job *HostingJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.CreatedAt = time.Now().UTC()
	query := `
		INSERT INTO hosting_jobs (
			id, account_id, job_type, status, payload, result, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.AccountID, job.JobType, job.Status, job.Payload, job.Result, job.ErrorMessage, job.CreatedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.CreateHostingJob(ctx, job)
	}
	return nil
}

func (p *PostgresStore) GetHostingJobByID(ctx context.Context, id uuid.UUID) (*HostingJob, error) {
	query := `
		SELECT id, account_id, job_type, status, payload, result, error_message, started_at, completed_at, created_at
		FROM hosting_jobs
		WHERE id = $1
	`
	j := &HostingJob{}
	var payload, result, errMsg sql.NullString
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&j.ID, &j.AccountID, &j.JobType, &j.Status, &payload, &result, &errMsg, &j.StartedAt, &j.CompletedAt, &j.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetHostingJobByID(ctx, id)
	}
	if payload.Valid { j.Payload = payload.String }
	if result.Valid { j.Result = result.String }
	if errMsg.Valid { j.ErrorMessage = errMsg.String }
	return j, nil
}

func (p *PostgresStore) UpdateHostingJob(ctx context.Context, job *HostingJob) error {
	query := `
		UPDATE hosting_jobs SET
			status = $2, result = $3, error_message = $4, started_at = $5, completed_at = $6
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.Status, job.Result, job.ErrorMessage, job.StartedAt, job.CompletedAt,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdateHostingJob(ctx, job)
	}
	return nil
}

func (p *PostgresStore) ListHostingJobsByAccount(ctx context.Context, accountID uuid.UUID) ([]*HostingJob, error) {
	query := `
		SELECT id, account_id, job_type, status, payload, result, error_message, started_at, completed_at, created_at
		FROM hosting_jobs
		WHERE account_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, accountID)
	if err != nil {
		m := NewMemoryStore()
		return m.ListHostingJobsByAccount(ctx, accountID)
	}
	defer rows.Close()

	var list []*HostingJob
	for rows.Next() {
		j := &HostingJob{}
		var payload, result, errMsg sql.NullString
		err := rows.Scan(
			&j.ID, &j.AccountID, &j.JobType, &j.Status, &payload, &result, &errMsg, &j.StartedAt, &j.CompletedAt, &j.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if payload.Valid { j.Payload = payload.String }
		if result.Valid { j.Result = result.String }
		if errMsg.Valid { j.ErrorMessage = errMsg.String }
		list = append(list, j)
	}
	return list, nil
}

