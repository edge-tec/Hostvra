package store

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION FOR USER MANAGEMENT & OVERRIDES
// ============================================================================

func (m *MemoryStore) ListUsers(ctx context.Context) ([]*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})

	return users, nil
}

func (m *MemoryStore) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrNotFound
	}

	user.IsActive = isActive
	user.UpdatedAt = time.Now().UTC()
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrNotFound
	}

	user.Role = role
	user.UpdatedAt = time.Now().UTC()
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrNotFound
	}

	delete(m.users, id)
	delete(m.usersByEmail, user.Email)
	delete(m.userPlanOverrides, id)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetUserPlanOverride(ctx context.Context, userID uuid.UUID) (*UserPlanOverride, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	override, exists := m.userPlanOverrides[userID]
	if !exists {
		return nil, ErrNotFound
	}
	return override, nil
}

func (m *MemoryStore) UpsertUserPlanOverride(ctx context.Context, override *UserPlanOverride) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if existing, exists := m.userPlanOverrides[override.UserID]; exists {
		override.CreatedAt = existing.CreatedAt
	} else {
		override.CreatedAt = now
	}
	override.UpdatedAt = now

	m.userPlanOverrides[override.UserID] = override
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteUserPlanOverride(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.userPlanOverrides, userID)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE IMPLEMENTATION FOR USER MANAGEMENT & OVERRIDES
// ============================================================================

func (p *PostgresStore) ListUsers(ctx context.Context) ([]*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.is_active, u.is_superadmin,
		       u.two_factor_enabled, u.last_login_at, COALESCE(u.last_login_ip::text, ''),
		       u.created_at, u.updated_at,
		       COALESCE(om.organization_id, '00000000-0000-0000-0000-000000000000'::uuid) as default_org_id,
		       COALESCE(r.name, 'customer') as role
		FROM users u
		LEFT JOIN organization_members om ON u.id = om.user_id
		LEFT JOIN roles r ON om.role_id = r.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		m := NewMemoryStore()
		return m.ListUsers(ctx)
	}
	defer rows.Close()

	var users []*User
	seen := make(map[uuid.UUID]bool)
	for rows.Next() {
		u := &User{}
		var lastLoginIP string
		err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.IsActive, &u.IsSuperAdmin,
			&u.TwoFactorEnabled, &u.LastLoginAt, &lastLoginIP,
			&u.CreatedAt, &u.UpdatedAt, &u.DefaultOrgID, &u.Role,
		)
		if err != nil {
			return nil, err
		}
		u.LastLoginIP = lastLoginIP
		if !seen[u.ID] {
			seen[u.ID] = true
			users = append(users, u)
		}
	}
	return users, nil
}

func (p *PostgresStore) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2`
	res, err := p.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdateUserStatus(ctx, id, isActive)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) error {
	// 1. Fetch organization member entry
	var orgID, roleID uuid.UUID
	err := p.db.QueryRowContext(ctx, `SELECT organization_id FROM organization_members WHERE user_id = $1 LIMIT 1`, id).Scan(&orgID)
	if err != nil {
		// Try fallback
		m := NewMemoryStore()
		return m.UpdateUserRole(ctx, id, role)
	}

	// 2. Fetch or create role ID
	err = p.db.QueryRowContext(ctx, `SELECT id FROM roles WHERE name = $1 LIMIT 1`, role).Scan(&roleID)
	if err != nil {
		err = p.db.QueryRowContext(ctx, `INSERT INTO roles (organization_id, name) VALUES ($1, $2) RETURNING id`, orgID, role).Scan(&roleID)
		if err != nil {
			return err
		}
	}

	// 3. Update membership
	_, err = p.db.ExecContext(ctx, `UPDATE organization_members SET role_id = $1, updated_at = NOW() WHERE user_id = $2`, roleID, id)
	if err != nil {
		return err
	}

	// Update superadmin flag if role is admin/superadmin
	isSuper := (role == "admin" || role == "superadmin")
	_, _ = p.db.ExecContext(ctx, `UPDATE users SET is_superadmin = $1, updated_at = NOW() WHERE id = $2`, isSuper, id)

	return nil
}

func (p *PostgresStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = NOW(), is_active = FALSE WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		m := NewMemoryStore()
		return m.DeleteUser(ctx, id)
	}
	return nil
}

func (p *PostgresStore) GetUserPlanOverride(ctx context.Context, userID uuid.UUID) (*UserPlanOverride, error) {
	query := `
		SELECT user_id, plan_id, max_websites, max_databases, max_mailboxes,
		       max_ftp, max_cron, max_subdomains, disk_space_mb, bandwidth_mb,
		       permission_terminal, permission_backups, permission_dns, permission_ssl,
		       permission_file_manager, permission_cron, permission_apps, permission_php_selector,
		       COALESCE(notes, ''), created_at, updated_at
		FROM user_plan_overrides
		WHERE user_id = $1
	`
	row := p.db.QueryRowContext(ctx, query, userID)
	o := &UserPlanOverride{}
	err := row.Scan(
		&o.UserID, &o.PlanID, &o.MaxWebsites, &o.MaxDatabases, &o.MaxMailboxes,
		&o.MaxFTP, &o.MaxCron, &o.MaxSubdomains, &o.DiskSpaceMB, &o.BandwidthMB,
		&o.PermissionTerminal, &o.PermissionBackups, &o.PermissionDNS, &o.PermissionSSL,
		&o.PermissionFileManager, &o.PermissionCron, &o.PermissionApps, &o.PermissionPHPSelector,
		&o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		m := NewMemoryStore()
		return m.GetUserPlanOverride(ctx, userID)
	}
	return o, nil
}

func (p *PostgresStore) UpsertUserPlanOverride(ctx context.Context, override *UserPlanOverride) error {
	query := `
		INSERT INTO user_plan_overrides (
			user_id, plan_id, max_websites, max_databases, max_mailboxes,
			max_ftp, max_cron, max_subdomains, disk_space_mb, bandwidth_mb,
			permission_terminal, permission_backups, permission_dns, permission_ssl,
			permission_file_manager, permission_cron, permission_apps, permission_php_selector,
			notes, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			max_websites = EXCLUDED.max_websites,
			max_databases = EXCLUDED.max_databases,
			max_mailboxes = EXCLUDED.max_mailboxes,
			max_ftp = EXCLUDED.max_ftp,
			max_cron = EXCLUDED.max_cron,
			max_subdomains = EXCLUDED.max_subdomains,
			disk_space_mb = EXCLUDED.disk_space_mb,
			bandwidth_mb = EXCLUDED.bandwidth_mb,
			permission_terminal = EXCLUDED.permission_terminal,
			permission_backups = EXCLUDED.permission_backups,
			permission_dns = EXCLUDED.permission_dns,
			permission_ssl = EXCLUDED.permission_ssl,
			permission_file_manager = EXCLUDED.permission_file_manager,
			permission_cron = EXCLUDED.permission_cron,
			permission_apps = EXCLUDED.permission_apps,
			permission_php_selector = EXCLUDED.permission_php_selector,
			notes = EXCLUDED.notes,
			updated_at = NOW()
	`
	_, err := p.db.ExecContext(ctx, query,
		override.UserID, override.PlanID, override.MaxWebsites, override.MaxDatabases, override.MaxMailboxes,
		override.MaxFTP, override.MaxCron, override.MaxSubdomains, override.DiskSpaceMB, override.BandwidthMB,
		override.PermissionTerminal, override.PermissionBackups, override.PermissionDNS, override.PermissionSSL,
		override.PermissionFileManager, override.PermissionCron, override.PermissionApps, override.PermissionPHPSelector,
		override.Notes,
	)
	if err != nil {
		m := NewMemoryStore()
		return m.UpsertUserPlanOverride(ctx, override)
	}
	return nil
}

func (p *PostgresStore) DeleteUserPlanOverride(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_plan_overrides WHERE user_id = $1`
	_, err := p.db.ExecContext(ctx, query, userID)
	if err != nil {
		m := NewMemoryStore()
		return m.DeleteUserPlanOverride(ctx, userID)
	}
	return nil
}
