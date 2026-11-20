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

var (
	ErrHDAccountNotFound      = errors.New("hd wallet account not found for network")
	ErrDuplicateDerivationIdx = errors.New("derivation index already allocated for this network")
	ErrDuplicateDepositAddr   = errors.New("deposit address already exists for this network")
	ErrDuplicateSweepJob      = errors.New("sweep job with this idempotency key already exists")
	ErrSweepJobNotFound       = errors.New("treasury sweep job not found")
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION (Thread-safe + Atomic Index Reservation)
// ============================================================================

func (m *MemoryStore) UpsertHDWalletAccount(ctx context.Context, acc *HDWalletAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	now := time.Now().UTC()
	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = now
	}
	acc.UpdatedAt = now
	key := strings.ToLower(acc.NetworkID)
	if existing, ok := m.hdWalletAccounts[key]; ok {
		acc.ID = existing.ID
		acc.NextDerivationIndex = existing.NextDerivationIndex
		acc.CreatedAt = existing.CreatedAt
	}
	cp := *acc
	m.hdWalletAccounts[key] = &cp
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetHDWalletAccountByNetwork(ctx context.Context, networkID string) (*HDWalletAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	acc, ok := m.hdWalletAccounts[strings.ToLower(networkID)]
	if !ok {
		return nil, ErrHDAccountNotFound
	}
	cp := *acc
	return &cp, nil
}

func (m *MemoryStore) ListHDWalletAccounts(ctx context.Context) ([]*HDWalletAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*HDWalletAccount, 0, len(m.hdWalletAccounts))
	for _, a := range m.hdWalletAccounts {
		cp := *a
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NetworkID < out[j].NetworkID })
	return out, nil
}

// ReserveNextDerivationIndex atomically reserves and increments the derivation index for a network.
// Starts at index 1 (skipping index 0, which corresponds to the merchant's primary Trust Wallet address).
// Guarantees concurrent invoice creation can NEVER collide on the same index.
func (m *MemoryStore) ReserveNextDerivationIndex(ctx context.Context, networkID string) (int, *HDWalletAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(networkID)
	acc, ok := m.hdWalletAccounts[key]
	if !ok {
		return 0, nil, ErrHDAccountNotFound
	}
	if acc.NextDerivationIndex < 1 {
		acc.NextDerivationIndex = 1
	}
	reserved := acc.NextDerivationIndex
	acc.NextDerivationIndex++
	acc.UpdatedAt = time.Now().UTC()
	m.saveToDiskLocked()
	cp := *acc
	return reserved, &cp, nil
}

func (m *MemoryStore) CreateHDWalletAddress(ctx context.Context, addr *HDWalletAddress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.hdWalletAddresses {
		if strings.EqualFold(existing.NetworkID, addr.NetworkID) && existing.DerivationIndex == addr.DerivationIndex {
			return ErrDuplicateDerivationIdx
		}
		if strings.EqualFold(existing.NetworkID, addr.NetworkID) && strings.EqualFold(existing.DepositAddress, addr.DepositAddress) {
			return ErrDuplicateDepositAddr
		}
	}
	if addr.ID == uuid.Nil {
		addr.ID = uuid.New()
	}
	if addr.CreatedAt.IsZero() {
		addr.CreatedAt = time.Now().UTC()
	}
	if addr.AddressStatus == "" {
		addr.AddressStatus = "active"
	}
	cp := *addr
	m.hdWalletAddresses[addr.ID] = &cp
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetHDWalletAddressByUserAndNetwork(ctx context.Context, userID uuid.UUID, networkID string) (*HDWalletAddress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.hdWalletAddresses {
		if a.UserID == userID && strings.EqualFold(a.NetworkID, networkID) && a.AddressStatus != "retired" {
			cp := *a
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) GetHDWalletAddressByDepositAddress(ctx context.Context, depositAddr string) (*HDWalletAddress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.hdWalletAddresses {
		if strings.EqualFold(a.DepositAddress, depositAddr) {
			cp := *a
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) ListHDWalletAddresses(ctx context.Context, networkID string) ([]*HDWalletAddress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*HDWalletAddress, 0, len(m.hdWalletAddresses))
	for _, a := range m.hdWalletAddresses {
		if networkID != "" && !strings.EqualFold(a.NetworkID, networkID) {
			continue
		}
		cp := *a
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *MemoryStore) UpdateHDWalletAddressStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.hdWalletAddresses[id]; ok {
		a.AddressStatus = status
		m.saveToDiskLocked()
	}
	return nil
}

func (m *MemoryStore) CreateDepositAddressAllocation(ctx context.Context, alloc *DepositAddressAllocation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if alloc.ID == uuid.Nil {
		alloc.ID = uuid.New()
	}
	if alloc.AllocatedAt.IsZero() {
		alloc.AllocatedAt = time.Now().UTC()
	}
	cp := *alloc
	m.depositAllocations[alloc.InvoiceID] = &cp
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetDepositAddressAllocationByInvoice(ctx context.Context, invoiceID uuid.UUID) (*DepositAddressAllocation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if a, ok := m.depositAllocations[invoiceID]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, nil
}

func (m *MemoryStore) ListDepositAddressAllocations(ctx context.Context, limit int) ([]*DepositAddressAllocation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*DepositAddressAllocation, 0, len(m.depositAllocations))
	for _, a := range m.depositAllocations {
		cp := *a
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AllocatedAt.After(out[j].AllocatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryStore) CreateTreasurySweepJob(ctx context.Context, job *TreasurySweepJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.treasurySweepJobs {
		if existing.IdempotencyKey != "" && existing.IdempotencyKey == job.IdempotencyKey {
			return ErrDuplicateSweepJob
		}
	}
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now
	cp := *job
	m.treasurySweepJobs[job.ID] = &cp
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetTreasurySweepJobByID(ctx context.Context, id uuid.UUID) (*TreasurySweepJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.treasurySweepJobs[id]
	if !ok {
		return nil, ErrSweepJobNotFound
	}
	cp := *j
	return &cp, nil
}

func (m *MemoryStore) GetTreasurySweepJobByIdempotencyKey(ctx context.Context, key string) (*TreasurySweepJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, j := range m.treasurySweepJobs {
		if j.IdempotencyKey == key {
			cp := *j
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) ListTreasurySweepJobs(ctx context.Context, status, networkID string, limit int) ([]*TreasurySweepJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*TreasurySweepJob, 0, len(m.treasurySweepJobs))
	for _, j := range m.treasurySweepJobs {
		if status != "" && !strings.EqualFold(j.Status, status) {
			continue
		}
		if networkID != "" && !strings.EqualFold(j.NetworkID, networkID) {
			continue
		}
		cp := *j
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryStore) UpdateTreasurySweepJob(ctx context.Context, job *TreasurySweepJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job.UpdatedAt = time.Now().UTC()
	cp := *job
	m.treasurySweepJobs[job.ID] = &cp
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetDailySweptUSD(ctx context.Context, since time.Time) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var total float64
	for _, j := range m.treasurySweepJobs {
		if (j.Status == "completed" || j.Status == "broadcast" || j.Status == "confirming") && j.CreatedAt.After(since) {
			total += j.AmountUSD
		}
	}
	return total, nil
}

func (m *MemoryStore) RecordTreasurySweepLog(ctx context.Context, entry *TreasurySweepLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	cp := *entry
	m.treasurySweepLogs = append(m.treasurySweepLogs, &cp)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListTreasurySweepLogs(ctx context.Context, jobID uuid.UUID) ([]*TreasurySweepLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*TreasurySweepLog, 0)
	for _, l := range m.treasurySweepLogs {
		if jobID != uuid.Nil && l.SweepJobID != jobID {
			continue
		}
		cp := *l
		out = append(out, &cp)
	}
	return out, nil
}

func (m *MemoryStore) RecordWalletDerivationAudit(ctx context.Context, entry *WalletDerivationAuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	cp := *entry
	m.walletDerivationAudits = append(m.walletDerivationAudits, &cp)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) ListWalletDerivationAudits(ctx context.Context, limit int) ([]*WalletDerivationAuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*WalletDerivationAuditLog, 0, len(m.walletDerivationAudits))
	for i := len(m.walletDerivationAudits) - 1; i >= 0; i-- {
		cp := *m.walletDerivationAudits[i]
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ============================================================================
// POSTGRESQL DATABASE STORE IMPLEMENTATION
// ============================================================================

func (d *PostgresStore) UpsertHDWalletAccount(ctx context.Context, acc *HDWalletAccount) error {
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO hd_wallet_accounts (id, account_name, wallet_type, network_id, derivation_scheme, derivation_path, public_derivation_reference, next_derivation_index, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (network_id, derivation_path) DO UPDATE SET
			account_name = EXCLUDED.account_name,
			wallet_type = EXCLUDED.wallet_type,
			public_derivation_reference = EXCLUDED.public_derivation_reference,
			status = EXCLUDED.status,
			updated_at = NOW()
	`, acc.ID, acc.AccountName, acc.WalletType, acc.NetworkID, acc.DerivationScheme, acc.DerivationPath, acc.PublicDerivationReference, acc.NextDerivationIndex, acc.Status)
	return err
}

func (d *PostgresStore) GetHDWalletAccountByNetwork(ctx context.Context, networkID string) (*HDWalletAccount, error) {
	var a HDWalletAccount
	err := d.db.QueryRowContext(ctx, `
		SELECT id, account_name, wallet_type, network_id, derivation_scheme, derivation_path, public_derivation_reference, next_derivation_index, status, created_at, updated_at
		FROM hd_wallet_accounts WHERE network_id = $1 AND status = 'active' LIMIT 1
	`, strings.ToLower(networkID)).Scan(
		&a.ID, &a.AccountName, &a.WalletType, &a.NetworkID, &a.DerivationScheme,
		&a.DerivationPath, &a.PublicDerivationReference, &a.NextDerivationIndex,
		&a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHDAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *PostgresStore) ListHDWalletAccounts(ctx context.Context) ([]*HDWalletAccount, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, account_name, wallet_type, network_id, derivation_scheme, derivation_path, public_derivation_reference, next_derivation_index, status, created_at, updated_at
		FROM hd_wallet_accounts ORDER BY network_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*HDWalletAccount, 0)
	for rows.Next() {
		var a HDWalletAccount
		if err := rows.Scan(&a.ID, &a.AccountName, &a.WalletType, &a.NetworkID, &a.DerivationScheme, &a.DerivationPath, &a.PublicDerivationReference, &a.NextDerivationIndex, &a.Status, &a.CreatedAt, &a.UpdatedAt); err == nil {
			out = append(out, &a)
		}
	}
	return out, nil
}

// ReserveNextDerivationIndex uses UPDATE ... RETURNING for atomic row-level locking in PostgreSQL.
// Always starts at >= 1 (skipping index 0 so customer addresses never collide with the merchant's primary Trust Wallet).
func (d *PostgresStore) ReserveNextDerivationIndex(ctx context.Context, networkID string) (int, *HDWalletAccount, error) {
	var a HDWalletAccount
	var reserved int
	err := d.db.QueryRowContext(ctx, `
		UPDATE hd_wallet_accounts
		SET next_derivation_index = GREATEST(next_derivation_index, 1) + 1, updated_at = NOW()
		WHERE network_id = $1 AND status = 'active'
		RETURNING id, account_name, wallet_type, network_id, derivation_scheme, derivation_path, public_derivation_reference, GREATEST(next_derivation_index - 1, 1), next_derivation_index, status, created_at, updated_at
	`, strings.ToLower(networkID)).Scan(
		&a.ID, &a.AccountName, &a.WalletType, &a.NetworkID, &a.DerivationScheme,
		&a.DerivationPath, &a.PublicDerivationReference, &reserved, &a.NextDerivationIndex,
		&a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, ErrHDAccountNotFound
	}
	if err != nil {
		return 0, nil, err
	}
	return reserved, &a, nil
}

func (d *PostgresStore) CreateHDWalletAddress(ctx context.Context, addr *HDWalletAddress) error {
	if addr.ID == uuid.Nil {
		addr.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO hd_wallet_addresses (id, account_id, user_id, network_id, asset_id, derivation_index, derivation_path, deposit_address, address_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, addr.ID, addr.AccountID, addr.UserID, addr.NetworkID, addr.AssetID, addr.DerivationIndex, addr.DerivationPath, addr.DepositAddress, addr.AddressStatus)
	return err
}

func (d *PostgresStore) GetHDWalletAddressByUserAndNetwork(ctx context.Context, userID uuid.UUID, networkID string) (*HDWalletAddress, error) {
	var a HDWalletAddress
	err := d.db.QueryRowContext(ctx, `
		SELECT id, account_id, user_id, network_id, asset_id, derivation_index, derivation_path, deposit_address, address_status, created_at
		FROM hd_wallet_addresses
		WHERE user_id = $1 AND network_id = $2 AND address_status <> 'retired'
		ORDER BY created_at ASC LIMIT 1
	`, userID, strings.ToLower(networkID)).Scan(
		&a.ID, &a.AccountID, &a.UserID, &a.NetworkID, &a.AssetID,
		&a.DerivationIndex, &a.DerivationPath, &a.DepositAddress, &a.AddressStatus, &a.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *PostgresStore) GetHDWalletAddressByDepositAddress(ctx context.Context, depositAddr string) (*HDWalletAddress, error) {
	var a HDWalletAddress
	err := d.db.QueryRowContext(ctx, `
		SELECT id, account_id, user_id, network_id, asset_id, derivation_index, derivation_path, deposit_address, address_status, created_at
		FROM hd_wallet_addresses WHERE LOWER(deposit_address) = LOWER($1) LIMIT 1
	`, depositAddr).Scan(
		&a.ID, &a.AccountID, &a.UserID, &a.NetworkID, &a.AssetID,
		&a.DerivationIndex, &a.DerivationPath, &a.DepositAddress, &a.AddressStatus, &a.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *PostgresStore) ListHDWalletAddresses(ctx context.Context, networkID string) ([]*HDWalletAddress, error) {
	q := `SELECT id, account_id, user_id, network_id, asset_id, derivation_index, derivation_path, deposit_address, address_status, created_at FROM hd_wallet_addresses`
	args := []interface{}{}
	if networkID != "" {
		q += ` WHERE network_id = $1`
		args = append(args, strings.ToLower(networkID))
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := d.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*HDWalletAddress, 0)
	for rows.Next() {
		var a HDWalletAddress
		if err := rows.Scan(&a.ID, &a.AccountID, &a.UserID, &a.NetworkID, &a.AssetID, &a.DerivationIndex, &a.DerivationPath, &a.DepositAddress, &a.AddressStatus, &a.CreatedAt); err == nil {
			out = append(out, &a)
		}
	}
	return out, nil
}

func (d *PostgresStore) UpdateHDWalletAddressStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := d.db.ExecContext(ctx, `UPDATE hd_wallet_addresses SET address_status = $1 WHERE id = $2`, status, id)
	return err
}

func (d *PostgresStore) CreateDepositAddressAllocation(ctx context.Context, alloc *DepositAddressAllocation) error {
	if alloc.ID == uuid.Nil {
		alloc.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO deposit_address_allocations (id, user_id, invoice_id, address_id, network_id, asset_id, allocation_status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (invoice_id) DO NOTHING
	`, alloc.ID, alloc.UserID, alloc.InvoiceID, alloc.AddressID, alloc.NetworkID, alloc.AssetID, alloc.AllocationStatus, alloc.ExpiresAt)
	return err
}

func (d *PostgresStore) GetDepositAddressAllocationByInvoice(ctx context.Context, invoiceID uuid.UUID) (*DepositAddressAllocation, error) {
	var a DepositAddressAllocation
	err := d.db.QueryRowContext(ctx, `
		SELECT id, user_id, invoice_id, address_id, network_id, asset_id, allocation_status, allocated_at, expires_at
		FROM deposit_address_allocations WHERE invoice_id = $1 LIMIT 1
	`, invoiceID).Scan(&a.ID, &a.UserID, &a.InvoiceID, &a.AddressID, &a.NetworkID, &a.AssetID, &a.AllocationStatus, &a.AllocatedAt, &a.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &a, err
}

func (d *PostgresStore) ListDepositAddressAllocations(ctx context.Context, limit int) ([]*DepositAddressAllocation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, user_id, invoice_id, address_id, network_id, asset_id, allocation_status, allocated_at, expires_at
		FROM deposit_address_allocations ORDER BY allocated_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*DepositAddressAllocation, 0)
	for rows.Next() {
		var a DepositAddressAllocation
		if err := rows.Scan(&a.ID, &a.UserID, &a.InvoiceID, &a.AddressID, &a.NetworkID, &a.AssetID, &a.AllocationStatus, &a.AllocatedAt, &a.ExpiresAt); err == nil {
			out = append(out, &a)
		}
	}
	return out, nil
}

func (d *PostgresStore) CreateTreasurySweepJob(ctx context.Context, job *TreasurySweepJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO treasury_sweep_jobs (
			id, invoice_id, source_address_id, source_address, derivation_index,
			destination_wallet_id, destination_address, asset_id, network_id,
			requested_amount, actual_amount, amount_usd, estimated_network_fee, actual_network_fee,
			transaction_hash, status, idempotency_key, retry_count, max_retries, last_error
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
	`, job.ID, job.InvoiceID, job.SourceAddressID, job.SourceAddress, job.DerivationIndex,
		job.DestinationWalletID, job.DestinationAddress, job.AssetID, job.NetworkID,
		job.RequestedAmount, job.ActualAmount, job.AmountUSD, job.EstimatedNetworkFee, job.ActualNetworkFee,
		job.TransactionHash, job.Status, job.IdempotencyKey, job.RetryCount, job.MaxRetries, job.LastError)
	return err
}

func (d *PostgresStore) GetTreasurySweepJobByID(ctx context.Context, id uuid.UUID) (*TreasurySweepJob, error) {
	var j TreasurySweepJob
	err := d.db.QueryRowContext(ctx, `
		SELECT id, invoice_id, source_address_id, source_address, derivation_index,
			destination_wallet_id, destination_address, asset_id, network_id,
			requested_amount, actual_amount, amount_usd, estimated_network_fee, actual_network_fee,
			transaction_hash, status, idempotency_key, retry_count, max_retries, last_error,
			created_at, updated_at, completed_at
		FROM treasury_sweep_jobs WHERE id = $1
	`, id).Scan(
		&j.ID, &j.InvoiceID, &j.SourceAddressID, &j.SourceAddress, &j.DerivationIndex,
		&j.DestinationWalletID, &j.DestinationAddress, &j.AssetID, &j.NetworkID,
		&j.RequestedAmount, &j.ActualAmount, &j.AmountUSD, &j.EstimatedNetworkFee, &j.ActualNetworkFee,
		&j.TransactionHash, &j.Status, &j.IdempotencyKey, &j.RetryCount, &j.MaxRetries, &j.LastError,
		&j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSweepJobNotFound
	}
	return &j, err
}

func (d *PostgresStore) GetTreasurySweepJobByIdempotencyKey(ctx context.Context, key string) (*TreasurySweepJob, error) {
	var j TreasurySweepJob
	err := d.db.QueryRowContext(ctx, `
		SELECT id, invoice_id, source_address_id, source_address, derivation_index,
			destination_wallet_id, destination_address, asset_id, network_id,
			requested_amount, actual_amount, amount_usd, estimated_network_fee, actual_network_fee,
			transaction_hash, status, idempotency_key, retry_count, max_retries, last_error,
			created_at, updated_at, completed_at
		FROM treasury_sweep_jobs WHERE idempotency_key = $1 LIMIT 1
	`, key).Scan(
		&j.ID, &j.InvoiceID, &j.SourceAddressID, &j.SourceAddress, &j.DerivationIndex,
		&j.DestinationWalletID, &j.DestinationAddress, &j.AssetID, &j.NetworkID,
		&j.RequestedAmount, &j.ActualAmount, &j.AmountUSD, &j.EstimatedNetworkFee, &j.ActualNetworkFee,
		&j.TransactionHash, &j.Status, &j.IdempotencyKey, &j.RetryCount, &j.MaxRetries, &j.LastError,
		&j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &j, err
}

func (d *PostgresStore) ListTreasurySweepJobs(ctx context.Context, status, networkID string, limit int) ([]*TreasurySweepJob, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `SELECT id, invoice_id, source_address_id, source_address, derivation_index,
			destination_wallet_id, destination_address, asset_id, network_id,
			requested_amount, actual_amount, amount_usd, estimated_network_fee, actual_network_fee,
			transaction_hash, status, idempotency_key, retry_count, max_retries, last_error,
			created_at, updated_at, completed_at FROM treasury_sweep_jobs WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if status != "" {
		q += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if networkID != "" {
		q += fmt.Sprintf(" AND network_id = $%d", argIdx)
		args = append(args, networkID)
		argIdx++
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := d.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*TreasurySweepJob, 0)
	for rows.Next() {
		var j TreasurySweepJob
		if err := rows.Scan(
			&j.ID, &j.InvoiceID, &j.SourceAddressID, &j.SourceAddress, &j.DerivationIndex,
			&j.DestinationWalletID, &j.DestinationAddress, &j.AssetID, &j.NetworkID,
			&j.RequestedAmount, &j.ActualAmount, &j.AmountUSD, &j.EstimatedNetworkFee, &j.ActualNetworkFee,
			&j.TransactionHash, &j.Status, &j.IdempotencyKey, &j.RetryCount, &j.MaxRetries, &j.LastError,
			&j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
		); err == nil {
			out = append(out, &j)
		}
	}
	return out, nil
}

func (d *PostgresStore) UpdateTreasurySweepJob(ctx context.Context, job *TreasurySweepJob) error {
	_, err := d.db.ExecContext(ctx, `
		UPDATE treasury_sweep_jobs
		SET actual_amount=$1, actual_network_fee=$2, transaction_hash=$3, status=$4, retry_count=$5, last_error=$6, updated_at=NOW(), completed_at=$7
		WHERE id=$8
	`, job.ActualAmount, job.ActualNetworkFee, job.TransactionHash, job.Status, job.RetryCount, job.LastError, job.CompletedAt, job.ID)
	return err
}

func (d *PostgresStore) GetDailySweptUSD(ctx context.Context, since time.Time) (float64, error) {
	var total sql.NullFloat64
	err := d.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount_usd), 0) FROM treasury_sweep_jobs
		WHERE status IN ('completed', 'broadcast', 'confirming') AND created_at >= $1
	`, since).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Float64, nil
}

func (d *PostgresStore) RecordTreasurySweepLog(ctx context.Context, entry *TreasurySweepLog) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO treasury_sweep_logs (id, sweep_job_id, event_type, event_details)
		VALUES ($1, $2, $3, $4)
	`, entry.ID, entry.SweepJobID, entry.EventType, entry.EventDetails)
	return err
}

func (d *PostgresStore) ListTreasurySweepLogs(ctx context.Context, jobID uuid.UUID) ([]*TreasurySweepLog, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, sweep_job_id, event_type, event_details, created_at
		FROM treasury_sweep_logs WHERE sweep_job_id = $1 ORDER BY created_at ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*TreasurySweepLog, 0)
	for rows.Next() {
		var l TreasurySweepLog
		if err := rows.Scan(&l.ID, &l.SweepJobID, &l.EventType, &l.EventDetails, &l.CreatedAt); err == nil {
			out = append(out, &l)
		}
	}
	return out, nil
}

func (d *PostgresStore) RecordWalletDerivationAudit(ctx context.Context, entry *WalletDerivationAuditLog) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO wallet_derivation_audit_logs (id, account_id, address_id, event_type, actor_id, event_details)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, entry.ID, entry.AccountID, entry.AddressID, entry.EventType, entry.ActorID, entry.EventDetails)
	return err
}

func (d *PostgresStore) ListWalletDerivationAudits(ctx context.Context, limit int) ([]*WalletDerivationAuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, account_id, address_id, event_type, actor_id, event_details, created_at
		FROM wallet_derivation_audit_logs ORDER BY created_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*WalletDerivationAuditLog, 0)
	for rows.Next() {
		var l WalletDerivationAuditLog
		if err := rows.Scan(&l.ID, &l.AccountID, &l.AddressID, &l.EventType, &l.ActorID, &l.EventDetails, &l.CreatedAt); err == nil {
			out = append(out, &l)
		}
	}
	return out, nil
}
