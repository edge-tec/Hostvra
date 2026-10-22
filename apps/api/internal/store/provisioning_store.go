package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION
// ============================================================================

func (m *MemoryStore) CreateResellerProfile(ctx context.Context, profile *ResellerProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	if profile.CustomNameservers == nil {
		profile.CustomNameservers = []string{"ns1.hostvra.com", "ns2.hostvra.com"}
	}
	if profile.BrandName == "" {
		profile.BrandName = "Hostvra"
	}
	if profile.CustomEmailSender == "" {
		profile.CustomEmailSender = "noreply@hostvra.com"
	}

	m.resellerProfiles[profile.ID] = profile
	m.resellerProfilesByUser[profile.UserID] = profile.ID
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetResellerProfileByUserID(ctx context.Context, userID uuid.UUID) (*ResellerProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	profID, ok := m.resellerProfilesByUser[userID]
	if !ok {
		return nil, ErrNotFound
	}
	prof, ok := m.resellerProfiles[profID]
	if !ok {
		return nil, ErrNotFound
	}
	return prof, nil
}

func (m *MemoryStore) GetResellerProfileByID(ctx context.Context, id uuid.UUID) (*ResellerProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prof, ok := m.resellerProfiles[id]
	if !ok {
		return nil, ErrNotFound
	}
	return prof, nil
}

func (m *MemoryStore) ListResellerProfiles(ctx context.Context, orgID uuid.UUID) ([]*ResellerProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*ResellerProfile
	for _, prof := range m.resellerProfiles {
		if prof.OrganizationID == orgID {
			list = append(list, prof)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateResellerProfile(ctx context.Context, profile *ResellerProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.resellerProfiles[profile.ID]
	if !ok {
		return ErrNotFound
	}
	profile.UpdatedAt = time.Now().UTC()
	profile.CreatedAt = existing.CreatedAt
	m.resellerProfiles[profile.ID] = profile
	m.resellerProfilesByUser[profile.UserID] = profile.ID
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetResellerUsage(ctx context.Context, resellerUserID uuid.UUID) (*ResellerUsageSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	profID, ok := m.resellerProfilesByUser[resellerUserID]
	if !ok {
		return nil, ErrNotFound
	}
	prof := m.resellerProfiles[profID]

	// Count client accounts and resources owned by this reseller
	usedAccounts := 0
	var usedDiskMB int64 = 0
	var usedBandwidthMB int64 = 0

	// Find all sub-users belonging to this reseller
	subUserIDs := make(map[uuid.UUID]bool)
	for _, u := range m.users {
		if u.ResellerID != nil && *u.ResellerID == resellerUserID {
			subUserIDs[u.ID] = true
		}
	}

	// Calculate disk and bandwidth from hosting accounts
	for _, acc := range m.hostingAccounts {
		if acc.Status != "terminated" && (subUserIDs[acc.UserID] || acc.UserID == resellerUserID) {
			usedAccounts++
			usedDiskMB += acc.DiskUsedMB
			usedBandwidthMB += acc.BandwidthUsedMB
		}
	}

	return &ResellerUsageSummary{
		MaxAccounts:          prof.MaxAccounts,
		UsedAccounts:         usedAccounts,
		AllocatedDiskMB:      prof.AllocatedDiskMB,
		UsedDiskMB:           usedDiskMB,
		AllocatedBandwidthMB: prof.AllocatedBandwidthMB,
		UsedBandwidthMB:      usedBandwidthMB,
		AllowOverselling:     prof.AllowOverselling,
	}, nil
}

func (m *MemoryStore) CreateProvisioningIntegration(ctx context.Context, integration *ProvisioningIntegration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if integration.ID == uuid.Nil {
		integration.ID = uuid.New()
	}
	now := time.Now().UTC()
	integration.CreatedAt = now
	integration.UpdatedAt = now
	m.provisioningIntegrations[integration.ID] = integration
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetProvisioningIntegrationByID(ctx context.Context, id uuid.UUID) (*ProvisioningIntegration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.provisioningIntegrations[id]
	if !ok {
		return nil, ErrNotFound
	}
	return item, nil
}

func (m *MemoryStore) ListProvisioningIntegrations(ctx context.Context, orgID uuid.UUID) ([]*ProvisioningIntegration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*ProvisioningIntegration
	for _, item := range m.provisioningIntegrations {
		if orgID == uuid.Nil || item.OrganizationID == orgID {
			list = append(list, item)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateProvisioningIntegration(ctx context.Context, integration *ProvisioningIntegration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.provisioningIntegrations[integration.ID]
	if !ok {
		return ErrNotFound
	}
	integration.UpdatedAt = time.Now().UTC()
	integration.CreatedAt = existing.CreatedAt
	m.provisioningIntegrations[integration.ID] = integration
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteProvisioningIntegration(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.provisioningIntegrations[id]; !ok {
		return ErrNotFound
	}
	delete(m.provisioningIntegrations, id)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) LogProvisioningWebhook(ctx context.Context, log *ProvisioningWebhookLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	m.provisioningWebhookLogs = append([]*ProvisioningWebhookLog{log}, m.provisioningWebhookLogs...)
	if len(m.provisioningWebhookLogs) > 1000 {
		m.provisioningWebhookLogs = m.provisioningWebhookLogs[:1000]
	}
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListProvisioningWebhookLogs(ctx context.Context, orgID uuid.UUID, limit int) ([]*ProvisioningWebhookLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	var list []*ProvisioningWebhookLog
	for _, item := range m.provisioningWebhookLogs {
		if orgID == uuid.Nil || item.OrganizationID == orgID {
			list = append(list, item)
			if len(list) >= limit {
				break
			}
		}
	}
	return list, nil
}

func (m *MemoryStore) GetProvisioningWebhookLogByID(ctx context.Context, id uuid.UUID) (*ProvisioningWebhookLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, item := range m.provisioningWebhookLogs {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, ErrNotFound
}

// ============================================================================
// POSTGRES STORE IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) CreateResellerProfile(ctx context.Context, profile *ResellerProfile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	nsJSON, err := json.Marshal(profile.CustomNameservers)
	if err != nil {
		nsJSON = []byte(`["ns1.hostvra.com", "ns2.hostvra.com"]`)
	}

	query := `
		INSERT INTO reseller_profiles (
			id, organization_id, user_id, max_accounts, allocated_disk_mb,
			allocated_bandwidth_mb, allow_overselling, brand_name, brand_logo_url,
			custom_nameservers, custom_email_sender, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err = p.db.ExecContext(ctx, query,
		profile.ID, profile.OrganizationID, profile.UserID, profile.MaxAccounts,
		profile.AllocatedDiskMB, profile.AllocatedBandwidthMB, profile.AllowOverselling,
		profile.BrandName, profile.BrandLogoURL, nsJSON, profile.CustomEmailSender,
		profile.IsActive, profile.CreatedAt, profile.UpdatedAt,
	)
	return err
}

func (p *PostgresStore) GetResellerProfileByUserID(ctx context.Context, userID uuid.UUID) (*ResellerProfile, error) {
	query := `
		SELECT id, organization_id, user_id, max_accounts, allocated_disk_mb,
		       allocated_bandwidth_mb, allow_overselling, brand_name, brand_logo_url,
		       custom_nameservers, custom_email_sender, is_active, created_at, updated_at
		FROM reseller_profiles
		WHERE user_id = $1
	`
	var prof ResellerProfile
	var nsJSON []byte
	var brandLogo sql.NullString

	err := p.db.QueryRowContext(ctx, query, userID).Scan(
		&prof.ID, &prof.OrganizationID, &prof.UserID, &prof.MaxAccounts,
		&prof.AllocatedDiskMB, &prof.AllocatedBandwidthMB, &prof.AllowOverselling,
		&prof.BrandName, &brandLogo, &nsJSON, &prof.CustomEmailSender,
		&prof.IsActive, &prof.CreatedAt, &prof.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if brandLogo.Valid {
		prof.BrandLogoURL = brandLogo.String
	}
	_ = json.Unmarshal(nsJSON, &prof.CustomNameservers)
	return &prof, nil
}

func (p *PostgresStore) GetResellerProfileByID(ctx context.Context, id uuid.UUID) (*ResellerProfile, error) {
	query := `
		SELECT id, organization_id, user_id, max_accounts, allocated_disk_mb,
		       allocated_bandwidth_mb, allow_overselling, brand_name, brand_logo_url,
		       custom_nameservers, custom_email_sender, is_active, created_at, updated_at
		FROM reseller_profiles
		WHERE id = $1
	`
	var prof ResellerProfile
	var nsJSON []byte
	var brandLogo sql.NullString

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&prof.ID, &prof.OrganizationID, &prof.UserID, &prof.MaxAccounts,
		&prof.AllocatedDiskMB, &prof.AllocatedBandwidthMB, &prof.AllowOverselling,
		&prof.BrandName, &brandLogo, &nsJSON, &prof.CustomEmailSender,
		&prof.IsActive, &prof.CreatedAt, &prof.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if brandLogo.Valid {
		prof.BrandLogoURL = brandLogo.String
	}
	_ = json.Unmarshal(nsJSON, &prof.CustomNameservers)
	return &prof, nil
}

func (p *PostgresStore) ListResellerProfiles(ctx context.Context, orgID uuid.UUID) ([]*ResellerProfile, error) {
	query := `
		SELECT id, organization_id, user_id, max_accounts, allocated_disk_mb,
		       allocated_bandwidth_mb, allow_overselling, brand_name, brand_logo_url,
		       custom_nameservers, custom_email_sender, is_active, created_at, updated_at
		FROM reseller_profiles
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ResellerProfile
	for rows.Next() {
		var prof ResellerProfile
		var nsJSON []byte
		var brandLogo sql.NullString

		if err := rows.Scan(
			&prof.ID, &prof.OrganizationID, &prof.UserID, &prof.MaxAccounts,
			&prof.AllocatedDiskMB, &prof.AllocatedBandwidthMB, &prof.AllowOverselling,
			&prof.BrandName, &brandLogo, &nsJSON, &prof.CustomEmailSender,
			&prof.IsActive, &prof.CreatedAt, &prof.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if brandLogo.Valid {
			prof.BrandLogoURL = brandLogo.String
		}
		_ = json.Unmarshal(nsJSON, &prof.CustomNameservers)
		list = append(list, &prof)
	}
	return list, nil
}

func (p *PostgresStore) UpdateResellerProfile(ctx context.Context, profile *ResellerProfile) error {
	profile.UpdatedAt = time.Now().UTC()
	nsJSON, err := json.Marshal(profile.CustomNameservers)
	if err != nil {
		nsJSON = []byte(`["ns1.hostvra.com", "ns2.hostvra.com"]`)
	}

	query := `
		UPDATE reseller_profiles
		SET max_accounts = $2, allocated_disk_mb = $3, allocated_bandwidth_mb = $4,
		    allow_overselling = $5, brand_name = $6, brand_logo_url = $7,
		    custom_nameservers = $8, custom_email_sender = $9, is_active = $10,
		    updated_at = $11
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		profile.ID, profile.MaxAccounts, profile.AllocatedDiskMB, profile.AllocatedBandwidthMB,
		profile.AllowOverselling, profile.BrandName, profile.BrandLogoURL, nsJSON,
		profile.CustomEmailSender, profile.IsActive, profile.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) GetResellerUsage(ctx context.Context, resellerUserID uuid.UUID) (*ResellerUsageSummary, error) {
	prof, err := p.GetResellerProfileByUserID(ctx, resellerUserID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT COUNT(DISTINCT ha.id) as used_accounts,
		       COALESCE(SUM(ha.disk_used_mb), 0) as used_disk,
		       COALESCE(SUM(ha.bandwidth_used_mb), 0) as used_bw
		FROM hosting_accounts ha
		LEFT JOIN users u ON u.id = ha.user_id
		WHERE (u.reseller_id = $1 OR ha.user_id = $1) AND ha.status != 'terminated'
	`
	var usedAccounts int
	var usedDiskMB, usedBwMB int64
	err = p.db.QueryRowContext(ctx, query, resellerUserID).Scan(&usedAccounts, &usedDiskMB, &usedBwMB)
	if err != nil {
		return nil, err
	}

	return &ResellerUsageSummary{
		MaxAccounts:          prof.MaxAccounts,
		UsedAccounts:         usedAccounts,
		AllocatedDiskMB:      prof.AllocatedDiskMB,
		UsedDiskMB:           usedDiskMB,
		AllocatedBandwidthMB: prof.AllocatedBandwidthMB,
		UsedBandwidthMB:      usedBwMB,
		AllowOverselling:     prof.AllowOverselling,
	}, nil
}

func (p *PostgresStore) CreateProvisioningIntegration(ctx context.Context, integration *ProvisioningIntegration) error {
	if integration.ID == uuid.Nil {
		integration.ID = uuid.New()
	}
	now := time.Now().UTC()
	integration.CreatedAt = now
	integration.UpdatedAt = now

	query := `
		INSERT INTO provisioning_integrations (
			id, organization_id, provider, name, secret_key_hash,
			default_server_id, default_plan_id, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := p.db.ExecContext(ctx, query,
		integration.ID, integration.OrganizationID, integration.Provider, integration.Name,
		integration.SecretKeyHash, integration.DefaultServerID, integration.DefaultPlanID,
		integration.IsActive, integration.CreatedAt, integration.UpdatedAt,
	)
	return err
}

func (p *PostgresStore) GetProvisioningIntegrationByID(ctx context.Context, id uuid.UUID) (*ProvisioningIntegration, error) {
	query := `
		SELECT id, organization_id, provider, name, secret_key_hash,
		       default_server_id, default_plan_id, is_active, created_at, updated_at
		FROM provisioning_integrations
		WHERE id = $1
	`
	var item ProvisioningIntegration
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID, &item.OrganizationID, &item.Provider, &item.Name,
		&item.SecretKeyHash, &item.DefaultServerID, &item.DefaultPlanID,
		&item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (p *PostgresStore) ListProvisioningIntegrations(ctx context.Context, orgID uuid.UUID) ([]*ProvisioningIntegration, error) {
	query := `
		SELECT id, organization_id, provider, name, secret_key_hash,
		       default_server_id, default_plan_id, is_active, created_at, updated_at
		FROM provisioning_integrations
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR organization_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ProvisioningIntegration
	for rows.Next() {
		var item ProvisioningIntegration
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Provider, &item.Name,
			&item.SecretKeyHash, &item.DefaultServerID, &item.DefaultPlanID,
			&item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	return list, nil
}

func (p *PostgresStore) UpdateProvisioningIntegration(ctx context.Context, integration *ProvisioningIntegration) error {
	integration.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE provisioning_integrations
		SET name = $2, default_server_id = $3, default_plan_id = $4,
		    is_active = $5, updated_at = $6
		WHERE id = $1
	`
	res, err := p.db.ExecContext(ctx, query,
		integration.ID, integration.Name, integration.DefaultServerID,
		integration.DefaultPlanID, integration.IsActive, integration.UpdatedAt,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) DeleteProvisioningIntegration(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM provisioning_integrations WHERE id = $1`
	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) LogProvisioningWebhook(ctx context.Context, log *ProvisioningWebhookLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	payloadJSON, err := json.Marshal(log.Payload)
	if err != nil {
		payloadJSON = []byte("{}")
	}

	query := `
		INSERT INTO provisioning_webhook_logs (
			id, organization_id, integration_id, provider, action,
			username, domain, payload, status, http_status,
			response_body, error_message, ip_address, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err = p.db.ExecContext(ctx, query,
		log.ID, log.OrganizationID, log.IntegrationID, log.Provider, log.Action,
		log.Username, log.Domain, payloadJSON, log.Status, log.HTTPStatus,
		log.ResponseBody, log.ErrorMessage, log.IPAddress, log.CreatedAt,
	)
	return err
}

func (p *PostgresStore) ListProvisioningWebhookLogs(ctx context.Context, orgID uuid.UUID, limit int) ([]*ProvisioningWebhookLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, organization_id, integration_id, provider, action,
			   username, domain, payload, status, http_status,
			   response_body, error_message, ip_address, created_at
		FROM provisioning_webhook_logs
		WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR organization_id = $1)
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ProvisioningWebhookLog
	for rows.Next() {
		var l ProvisioningWebhookLog
		var payloadJSON []byte
		var username, domain, respBody, errMsg, ip sql.NullString
		if err := rows.Scan(
			&l.ID, &l.OrganizationID, &l.IntegrationID, &l.Provider, &l.Action,
			&username, &domain, &payloadJSON, &l.Status, &l.HTTPStatus,
			&respBody, &errMsg, &ip, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		if username.Valid {
			l.Username = username.String
		}
		if domain.Valid {
			l.Domain = domain.String
		}
		if respBody.Valid {
			l.ResponseBody = respBody.String
		}
		if errMsg.Valid {
			l.ErrorMessage = errMsg.String
		}
		if ip.Valid {
			l.IPAddress = ip.String
		}
		_ = json.Unmarshal(payloadJSON, &l.Payload)
		list = append(list, &l)
	}
	return list, nil
}

func (p *PostgresStore) GetProvisioningWebhookLogByID(ctx context.Context, id uuid.UUID) (*ProvisioningWebhookLog, error) {
	query := `
		SELECT id, organization_id, integration_id, provider, action,
		       username, domain, payload, status, http_status,
		       response_body, error_message, ip_address, created_at
		FROM provisioning_webhook_logs
		WHERE id = $1
	`
	var l ProvisioningWebhookLog
	var payloadJSON []byte
	var username, domain, respBody, errMsg, ip sql.NullString
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.OrganizationID, &l.IntegrationID, &l.Provider, &l.Action,
		&username, &domain, &payloadJSON, &l.Status, &l.HTTPStatus,
		&respBody, &errMsg, &ip, &l.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if username.Valid {
		l.Username = username.String
	}
	if domain.Valid {
		l.Domain = domain.String
	}
	if respBody.Valid {
		l.ResponseBody = respBody.String
	}
	if errMsg.Valid {
		l.ErrorMessage = errMsg.String
	}
	if ip.Valid {
		l.IPAddress = ip.String
	}
	_ = json.Unmarshal(payloadJSON, &l.Payload)
	return &l, nil
}

var _ = pq.Array
