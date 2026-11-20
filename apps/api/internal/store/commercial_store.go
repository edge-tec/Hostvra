package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCouponNotFound             = errors.New("coupon code not found")
	ErrCouponInvalid              = errors.New("coupon is inactive, expired, or usage limit reached")
	ErrSelfReferralForbidden      = errors.New("self-referral is prohibited")
	ErrInsufficientWallet         = errors.New("insufficient wallet credit balance")
	ErrWithdrawalAlreadyProcessed = errors.New("withdrawal request has already been processed")
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION (Atomic Mutex + Double-Entry Ledger)
// ============================================================================

func (m *MemoryStore) CreateCoupon(ctx context.Context, c *Coupon) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	code := strings.ToUpper(strings.TrimSpace(c.Code))
	if code == "" {
		return errors.New("coupon code is required")
	}
	if _, exists := m.coupons[code]; exists {
		return fmt.Errorf("coupon code %s already exists", code)
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	c.Code = code
	c.CreatedAt = now
	c.UpdatedAt = now
	cp := *c
	m.coupons[code] = &cp
	return nil
}

func (m *MemoryStore) GetCouponByCode(ctx context.Context, code string) (*Coupon, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.coupons[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return nil, ErrCouponNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *MemoryStore) ListCoupons(ctx context.Context) ([]*Coupon, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Coupon, 0, len(m.coupons))
	for _, c := range m.coupons {
		cp := *c
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *MemoryStore) DeleteCoupon(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, c := range m.coupons {
		if c.ID == id {
			delete(m.coupons, k)
			return nil
		}
	}
	return ErrCouponNotFound
}

func (m *MemoryStore) UpdateCoupon(ctx context.Context, c *Coupon) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, existing := range m.coupons {
		if existing.ID == c.ID {
			newCode := strings.ToUpper(strings.TrimSpace(c.Code))
			if newCode == "" {
				newCode = existing.Code
			}
			if newCode != k {
				if _, conflict := m.coupons[newCode]; conflict {
					return fmt.Errorf("coupon code %s already exists", newCode)
				}
				delete(m.coupons, k)
			}
			c.Code = newCode
			c.CurrentUses = existing.CurrentUses
			c.CreatedAt = existing.CreatedAt
			c.UpdatedAt = time.Now().UTC()
			cp := *c
			m.coupons[newCode] = &cp
			return nil
		}
	}
	return ErrCouponNotFound
}

// RedeemCouponAtomic validates limits, prevents concurrent over-redemption, and records redemption atomically.
func (m *MemoryStore) RedeemCouponAtomic(ctx context.Context, code string, userID, invoiceID uuid.UUID, orderAmountUSD float64, isFirstOrder bool) (*CouponRedemption, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.coupons[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return nil, ErrCouponNotFound
	}
	now := time.Now().UTC()
	if !c.IsActive || (c.ExpiresAt != nil && c.ExpiresAt.Before(now)) {
		return nil, ErrCouponInvalid
	}
	if c.MaxUses > 0 && c.CurrentUses >= c.MaxUses {
		return nil, fmt.Errorf("%w: global redemption limit reached", ErrCouponInvalid)
	}
	if orderAmountUSD < c.MinOrderUSD {
		return nil, fmt.Errorf("minimum order amount for %s is $%.2f", c.Code, c.MinOrderUSD)
	}
	if c.FirstOrderOnly && !isFirstOrder {
		return nil, fmt.Errorf("coupon %s is valid for first-time orders only", c.Code)
	}

	userCount := 0
	for _, r := range m.couponRedemptions {
		if r.CouponID == c.ID && r.InvoiceID == invoiceID {
			cp := *r
			return &cp, nil // Idempotent for same invoice
		}
		if r.CouponID == c.ID && r.UserID == userID {
			userCount++
		}
	}
	if c.MaxUsesPerUser > 0 && userCount >= c.MaxUsesPerUser {
		return nil, fmt.Errorf("%w: per-customer usage limit reached", ErrCouponInvalid)
	}

	var discount float64
	if strings.EqualFold(c.DiscountType, "fixed") {
		discount = math.Min(c.DiscountValue, orderAmountUSD)
	} else {
		discount = math.Round((orderAmountUSD*(c.DiscountValue/100.0))*100) / 100
		if discount > orderAmountUSD {
			discount = orderAmountUSD
		}
	}

	c.CurrentUses++
	c.UpdatedAt = now
	red := &CouponRedemption{
		ID:          uuid.New(),
		CouponID:    c.ID,
		UserID:      userID,
		InvoiceID:   invoiceID,
		DiscountUSD: discount,
		CreatedAt:   now,
	}
	m.couponRedemptions = append(m.couponRedemptions, red)
	cp := *red
	return &cp, nil
}

func (m *MemoryStore) GetOrCreateAffiliateAccount(ctx context.Context, userID uuid.UUID) (*AffiliateAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acc, ok := m.affiliateAccounts[userID]; ok {
		cp := *acc
		return &cp, nil
	}
	now := time.Now().UTC()
	code := "HV-" + strings.ToUpper(userID.String()[:8])
	acc := &AffiliateAccount{
		ID:                uuid.New(),
		UserID:            userID,
		ReferralCode:      code,
		CommissionRatePct: 15.0,
		Status:            "active",
		PayoutMethod:      "credit_wallet",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	m.affiliateAccounts[userID] = acc
	cp := *acc
	return &cp, nil
}

func (m *MemoryStore) GetAffiliateByCode(ctx context.Context, code string) (*AffiliateAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clean := strings.ToUpper(strings.TrimSpace(code))
	for _, a := range m.affiliateAccounts {
		if strings.EqualFold(a.ReferralCode, clean) {
			cp := *a
			return &cp, nil
		}
	}
	return nil, errors.New("affiliate referral code not found")
}

func (m *MemoryStore) ListAffiliateAccounts(ctx context.Context) ([]*AffiliateAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*AffiliateAccount, 0, len(m.affiliateAccounts))
	for _, a := range m.affiliateAccounts {
		cp := *a
		out = append(out, &cp)
	}
	return out, nil
}

func (m *MemoryStore) RecordAffiliateReferral(ctx context.Context, affiliateID, referredUserID uuid.UUID, attributionModel string) (*AffiliateReferral, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var owner *AffiliateAccount
	for _, a := range m.affiliateAccounts {
		if a.ID == affiliateID {
			owner = a
			break
		}
	}
	if owner == nil {
		return nil, errors.New("affiliate account not found")
	}
	if owner.UserID == referredUserID {
		return nil, ErrSelfReferralForbidden
	}
	if existing, ok := m.affiliateReferrals[referredUserID]; ok {
		cp := *existing
		return &cp, nil
	}
	if attributionModel == "" {
		attributionModel = "first_click"
	}
	ref := &AffiliateReferral{
		ID:               uuid.New(),
		AffiliateID:      affiliateID,
		ReferredUserID:   referredUserID,
		AttributionModel: attributionModel,
		Status:           "signed_up",
		CreatedAt:        time.Now().UTC(),
	}
	m.affiliateReferrals[referredUserID] = ref
	owner.TotalReferrals++
	owner.UpdatedAt = time.Now().UTC()
	cp := *ref
	return &cp, nil
}

func (m *MemoryStore) RecordAffiliateCommission(ctx context.Context, referredUserID, invoiceID uuid.UUID, orderAmountUSD float64) (*AffiliateCommission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ref, ok := m.affiliateReferrals[referredUserID]
	if !ok {
		return nil, nil
	}
	if existing, exists := m.affiliateCommissions[invoiceID]; exists {
		cp := *existing
		return &cp, nil
	}
	var owner *AffiliateAccount
	for _, a := range m.affiliateAccounts {
		if a.ID == ref.AffiliateID {
			owner = a
			break
		}
	}
	if owner == nil || owner.Status != "active" {
		return nil, nil
	}
	commUSD := math.Round((orderAmountUSD*(owner.CommissionRatePct/100.0))*100) / 100
	now := time.Now().UTC()
	comm := &AffiliateCommission{
		ID:             uuid.New(),
		AffiliateID:    owner.ID,
		ReferredUserID: referredUserID,
		InvoiceID:      invoiceID,
		OrderAmountUSD: orderAmountUSD,
		CommissionUSD:  commUSD,
		Status:         "approved",
		CreatedAt:      now,
		ApprovedAt:     &now,
	}
	m.affiliateCommissions[invoiceID] = comm
	owner.ApprovedBalanceUSD += commUSD
	owner.UpdatedAt = now
	ref.Status = "converted"
	cp := *comm
	return &cp, nil
}

func (m *MemoryStore) ReverseAffiliateCommissionByInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	comm, ok := m.affiliateCommissions[invoiceID]
	if !ok || comm.Status == "reversed" {
		return nil
	}
	for _, a := range m.affiliateAccounts {
		if a.ID == comm.AffiliateID {
			if comm.Status == "approved" {
				a.ApprovedBalanceUSD = math.Max(0, a.ApprovedBalanceUSD-comm.CommissionUSD)
			} else if comm.Status == "pending" {
				a.PendingBalanceUSD = math.Max(0, a.PendingBalanceUSD-comm.CommissionUSD)
			}
			a.UpdatedAt = time.Now().UTC()
			break
		}
	}
	comm.Status = "reversed"
	return nil
}

func (m *MemoryStore) RequestAffiliateWithdrawal(ctx context.Context, userID uuid.UUID, amountUSD float64, method, details string) (*AffiliateWithdrawal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.affiliateAccounts[userID]
	if !ok {
		return nil, errors.New("affiliate account not found")
	}
	if amountUSD <= 0 || amountUSD > acc.ApprovedBalanceUSD {
		return nil, errors.New("requested withdrawal exceeds approved affiliate balance")
	}
	acc.ApprovedBalanceUSD -= amountUSD
	w := &AffiliateWithdrawal{
		ID:            uuid.New(),
		AffiliateID:   acc.ID,
		UserID:        userID,
		AmountUSD:     amountUSD,
		PayoutMethod:  method,
		PayoutDetails: details,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
	}
	m.affiliateWithdrawals[w.ID] = w
	cp := *w
	return &cp, nil
}

func (m *MemoryStore) ApproveAffiliateWithdrawal(ctx context.Context, withdrawalID uuid.UUID, approve bool, adminNotes string) (*AffiliateWithdrawal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.affiliateWithdrawals[withdrawalID]
	if !ok {
		return nil, errors.New("withdrawal request not found")
	}
	if w.Status != "pending" {
		return nil, ErrWithdrawalAlreadyProcessed
	}
	now := time.Now().UTC()
	w.ProcessedAt = &now
	w.AdminNotes = adminNotes
	if approve {
		w.Status = "paid"
		if acc, ok := m.affiliateAccounts[w.UserID]; ok {
			acc.TotalPaidUSD += w.AmountUSD
		}
	} else {
		w.Status = "rejected"
		if acc, ok := m.affiliateAccounts[w.UserID]; ok {
			acc.ApprovedBalanceUSD += w.AmountUSD
		}
	}
	cp := *w
	return &cp, nil
}

func (m *MemoryStore) ListAffiliateWithdrawals(ctx context.Context, userID uuid.UUID) ([]*AffiliateWithdrawal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*AffiliateWithdrawal, 0)
	for _, w := range m.affiliateWithdrawals {
		if userID != uuid.Nil && w.UserID != userID {
			continue
		}
		cp := *w
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *MemoryStore) GetOrCreateCustomerWallet(ctx context.Context, userID uuid.UUID) (*CustomerWallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w, ok := m.customerWallets[userID]; ok {
		cp := *w
		return &cp, nil
	}
	now := time.Now().UTC()
	w := &CustomerWallet{
		ID:             uuid.New(),
		UserID:         userID,
		BalanceUSD:     0,
		Currency:       "USD",
		AutoPayEnabled: true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.customerWallets[userID] = w
	cp := *w
	return &cp, nil
}

func (m *MemoryStore) SetWalletAutoPay(ctx context.Context, userID uuid.UUID, enabled bool) (*CustomerWallet, error) {
	w, err := m.GetOrCreateCustomerWallet(ctx, userID)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customerWallets[userID].AutoPayEnabled = enabled
	m.customerWallets[userID].UpdatedAt = time.Now().UTC()
	w.AutoPayEnabled = enabled
	return w, nil
}

// CreditWalletAtomic appends an immutable double-entry ledger credit with strict idempotency.
func (m *MemoryStore) CreditWalletAtomic(ctx context.Context, userID uuid.UUID, amountUSD float64, entryType, refID, idempotencyKey, notes string) (*WalletLedgerEntry, error) {
	if amountUSD <= 0 {
		return nil, errors.New("credit amount must be positive")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.walletLedger {
		if e.IdempotencyKey == idempotencyKey {
			cp := *e
			return &cp, nil
		}
	}
	w, ok := m.customerWallets[userID]
	if !ok {
		now := time.Now().UTC()
		w = &CustomerWallet{ID: uuid.New(), UserID: userID, BalanceUSD: 0, Currency: "USD", AutoPayEnabled: true, CreatedAt: now, UpdatedAt: now}
		m.customerWallets[userID] = w
	}
	before := w.BalanceUSD
	after := math.Round((before+amountUSD)*10000) / 10000
	w.BalanceUSD = after
	w.UpdatedAt = time.Now().UTC()
	entry := &WalletLedgerEntry{
		ID:               uuid.New(),
		WalletID:         w.ID,
		UserID:           userID,
		EntryType:        entryType,
		Direction:        "credit",
		AmountUSD:        amountUSD,
		BalanceBeforeUSD: before,
		BalanceAfterUSD:  after,
		ReferenceID:      refID,
		IdempotencyKey:   idempotencyKey,
		Notes:            notes,
		CreatedAt:        time.Now().UTC(),
	}
	m.walletLedger = append(m.walletLedger, entry)
	cp := *entry
	return &cp, nil
}

// DebitWalletAtomic atomically checks funds, prevents double-spend, and appends an immutable debit ledger entry.
func (m *MemoryStore) DebitWalletAtomic(ctx context.Context, userID uuid.UUID, amountUSD float64, entryType, refID, idempotencyKey, notes string) (*WalletLedgerEntry, error) {
	if amountUSD <= 0 {
		return nil, errors.New("debit amount must be positive")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.walletLedger {
		if e.IdempotencyKey == idempotencyKey {
			cp := *e
			return &cp, nil
		}
	}
	w, ok := m.customerWallets[userID]
	if !ok || w.BalanceUSD+1e-9 < amountUSD {
		return nil, ErrInsufficientWallet
	}
	before := w.BalanceUSD
	after := math.Round((before-amountUSD)*10000) / 10000
	if after < 0 {
		after = 0
	}
	w.BalanceUSD = after
	w.UpdatedAt = time.Now().UTC()
	entry := &WalletLedgerEntry{
		ID:               uuid.New(),
		WalletID:         w.ID,
		UserID:           userID,
		EntryType:        entryType,
		Direction:        "debit",
		AmountUSD:        amountUSD,
		BalanceBeforeUSD: before,
		BalanceAfterUSD:  after,
		ReferenceID:      refID,
		IdempotencyKey:   idempotencyKey,
		Notes:            notes,
		CreatedAt:        time.Now().UTC(),
	}
	m.walletLedger = append(m.walletLedger, entry)
	cp := *entry
	return &cp, nil
}

func (m *MemoryStore) ListWalletLedger(ctx context.Context, userID uuid.UUID, limit int) ([]*WalletLedgerEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*WalletLedgerEntry, 0)
	for i := len(m.walletLedger) - 1; i >= 0; i-- {
		e := m.walletLedger[i]
		if userID != uuid.Nil && e.UserID != userID {
			continue
		}
		cp := *e
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// RecordBillingDunningEvent returns (true, nil) if newly recorded, or (false, nil) if already sent for this cycle/stage.
func (m *MemoryStore) RecordBillingDunningEvent(ctx context.Context, ev *BillingDunningEvent) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s:%s", ev.SubscriptionID.String(), ev.CycleKey, ev.NoticeStage)
	if _, exists := m.dunningEvents[key]; exists {
		return false, nil
	}
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	cp := *ev
	m.dunningEvents[key] = &cp
	return true, nil
}

func (m *MemoryStore) ListBillingDunningEvents(ctx context.Context, subID uuid.UUID, limit int) ([]*BillingDunningEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*BillingDunningEvent, 0)
	for _, ev := range m.dunningEvents {
		if subID != uuid.Nil && ev.SubscriptionID != subID {
			continue
		}
		cp := *ev
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ============================================================================
// POSTGRESQL DATABASE STORE IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) CreateCoupon(ctx context.Context, c *Coupon) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO coupons (id, code, description, discount_type, discount_value, max_uses, max_uses_per_user, min_order_usd, first_order_only, recurring_discount, applicable_plan_ids, is_active, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, c.ID, c.Code, c.Description, c.DiscountType, c.DiscountValue, c.MaxUses, c.MaxUsesPerUser, c.MinOrderUSD, c.FirstOrderOnly, c.RecurringDiscount, c.ApplicablePlanIDs, c.IsActive, c.ExpiresAt)
	return err
}

func (p *PostgresStore) GetCouponByCode(ctx context.Context, code string) (*Coupon, error) {
	var c Coupon
	err := p.db.QueryRowContext(ctx, `
		SELECT id, code, description, discount_type, discount_value, max_uses, current_uses, max_uses_per_user, min_order_usd, first_order_only, recurring_discount, applicable_plan_ids, is_active, expires_at, created_at, updated_at
		FROM coupons WHERE UPPER(code) = UPPER($1) LIMIT 1
	`, strings.TrimSpace(code)).Scan(
		&c.ID, &c.Code, &c.Description, &c.DiscountType, &c.DiscountValue, &c.MaxUses, &c.CurrentUses, &c.MaxUsesPerUser,
		&c.MinOrderUSD, &c.FirstOrderOnly, &c.RecurringDiscount, &c.ApplicablePlanIDs, &c.IsActive, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCouponNotFound
	}
	return &c, err
}

func (p *PostgresStore) ListCoupons(ctx context.Context) ([]*Coupon, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, code, description, discount_type, discount_value, max_uses, current_uses, max_uses_per_user, min_order_usd, first_order_only, recurring_discount, applicable_plan_ids, is_active, expires_at, created_at, updated_at
		FROM coupons ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*Coupon, 0)
	for rows.Next() {
		var c Coupon
		if rows.Scan(&c.ID, &c.Code, &c.Description, &c.DiscountType, &c.DiscountValue, &c.MaxUses, &c.CurrentUses, &c.MaxUsesPerUser,
			&c.MinOrderUSD, &c.FirstOrderOnly, &c.RecurringDiscount, &c.ApplicablePlanIDs, &c.IsActive, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt) == nil {
			out = append(out, &c)
		}
	}
	return out, nil
}

func (p *PostgresStore) DeleteCoupon(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.ExecContext(ctx, `DELETE FROM coupons WHERE id = $1`, id)
	return err
}

func (p *PostgresStore) UpdateCoupon(ctx context.Context, c *Coupon) error {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	res, err := p.db.ExecContext(ctx, `
		UPDATE coupons
		SET code = COALESCE(NULLIF($2, ''), code),
		    description = $3,
		    discount_type = $4,
		    discount_value = $5,
		    max_uses = $6,
		    max_uses_per_user = $7,
		    min_order_usd = $8,
		    first_order_only = $9,
		    recurring_discount = $10,
		    applicable_plan_ids = $11,
		    is_active = $12,
		    expires_at = $13,
		    updated_at = NOW()
		WHERE id = $1
	`, c.ID, c.Code, c.Description, c.DiscountType, c.DiscountValue, c.MaxUses, c.MaxUsesPerUser, c.MinOrderUSD, c.FirstOrderOnly, c.RecurringDiscount, c.ApplicablePlanIDs, c.IsActive, c.ExpiresAt)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCouponNotFound
	}
	return nil
}

func (p *PostgresStore) RedeemCouponAtomic(ctx context.Context, code string, userID, invoiceID uuid.UUID, orderAmountUSD float64, isFirstOrder bool) (*CouponRedemption, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var c Coupon
	err = tx.QueryRowContext(ctx, `
		SELECT id, code, discount_type, discount_value, max_uses, current_uses, max_uses_per_user, min_order_usd, first_order_only, is_active, expires_at
		FROM coupons WHERE UPPER(code) = UPPER($1) FOR UPDATE
	`, strings.TrimSpace(code)).Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &c.MaxUses, &c.CurrentUses, &c.MaxUsesPerUser, &c.MinOrderUSD, &c.FirstOrderOnly, &c.IsActive, &c.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCouponNotFound
	}
	if err != nil {
		return nil, err
	}
	if !c.IsActive || (c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now().UTC())) {
		return nil, ErrCouponInvalid
	}
	if c.MaxUses > 0 && c.CurrentUses >= c.MaxUses {
		return nil, fmt.Errorf("%w: global limit reached", ErrCouponInvalid)
	}
	if orderAmountUSD < c.MinOrderUSD {
		return nil, fmt.Errorf("minimum order amount for %s is $%.2f", c.Code, c.MinOrderUSD)
	}
	if c.FirstOrderOnly && !isFirstOrder {
		return nil, fmt.Errorf("coupon %s is valid for first-time orders only", c.Code)
	}

	var userUses int
	_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_redemptions WHERE coupon_id=$1 AND user_id=$2`, c.ID, userID).Scan(&userUses)
	if c.MaxUsesPerUser > 0 && userUses >= c.MaxUsesPerUser {
		return nil, fmt.Errorf("%w: per-customer limit reached", ErrCouponInvalid)
	}

	var discount float64
	if strings.EqualFold(c.DiscountType, "fixed") {
		discount = math.Min(c.DiscountValue, orderAmountUSD)
	} else {
		discount = math.Round((orderAmountUSD*(c.DiscountValue/100.0))*100) / 100
		if discount > orderAmountUSD {
			discount = orderAmountUSD
		}
	}

	red := &CouponRedemption{ID: uuid.New(), CouponID: c.ID, UserID: userID, InvoiceID: invoiceID, DiscountUSD: discount, CreatedAt: time.Now().UTC()}
	if _, err := tx.ExecContext(ctx, `INSERT INTO coupon_redemptions (id, coupon_id, user_id, invoice_id, discount_usd) VALUES ($1,$2,$3,$4,$5)`, red.ID, red.CouponID, red.UserID, red.InvoiceID, red.DiscountUSD); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupons SET current_uses = current_uses + 1, updated_at = NOW() WHERE id = $1`, c.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return red, nil
}

func (p *PostgresStore) GetOrCreateAffiliateAccount(ctx context.Context, userID uuid.UUID) (*AffiliateAccount, error) {
	code := "HV-" + strings.ToUpper(userID.String()[:8])
	_, _ = p.db.ExecContext(ctx, `
		INSERT INTO affiliate_accounts (id, user_id, referral_code, commission_rate_pct, status, payout_method)
		VALUES ($1,$2,$3,15.0,'active','credit_wallet') ON CONFLICT (user_id) DO NOTHING
	`, uuid.New(), userID, code)
	var a AffiliateAccount
	err := p.db.QueryRowContext(ctx, `
		SELECT id, user_id, referral_code, commission_rate_pct, pending_balance_usd, approved_balance_usd, total_paid_usd, total_referrals, status, payout_method, payout_details, created_at, updated_at
		FROM affiliate_accounts WHERE user_id = $1
	`, userID).Scan(&a.ID, &a.UserID, &a.ReferralCode, &a.CommissionRatePct, &a.PendingBalanceUSD, &a.ApprovedBalanceUSD, &a.TotalPaidUSD, &a.TotalReferrals, &a.Status, &a.PayoutMethod, &a.PayoutDetails, &a.CreatedAt, &a.UpdatedAt)
	return &a, err
}

func (p *PostgresStore) GetAffiliateByCode(ctx context.Context, code string) (*AffiliateAccount, error) {
	var a AffiliateAccount
	err := p.db.QueryRowContext(ctx, `
		SELECT id, user_id, referral_code, commission_rate_pct, pending_balance_usd, approved_balance_usd, total_paid_usd, total_referrals, status, payout_method, payout_details, created_at, updated_at
		FROM affiliate_accounts WHERE UPPER(referral_code) = UPPER($1)
	`, strings.TrimSpace(code)).Scan(&a.ID, &a.UserID, &a.ReferralCode, &a.CommissionRatePct, &a.PendingBalanceUSD, &a.ApprovedBalanceUSD, &a.TotalPaidUSD, &a.TotalReferrals, &a.Status, &a.PayoutMethod, &a.PayoutDetails, &a.CreatedAt, &a.UpdatedAt)
	return &a, err
}

func (p *PostgresStore) ListAffiliateAccounts(ctx context.Context) ([]*AffiliateAccount, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, user_id, referral_code, commission_rate_pct, pending_balance_usd, approved_balance_usd, total_paid_usd, total_referrals, status, payout_method, payout_details, created_at, updated_at
		FROM affiliate_accounts ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AffiliateAccount, 0)
	for rows.Next() {
		var a AffiliateAccount
		if rows.Scan(&a.ID, &a.UserID, &a.ReferralCode, &a.CommissionRatePct, &a.PendingBalanceUSD, &a.ApprovedBalanceUSD, &a.TotalPaidUSD, &a.TotalReferrals, &a.Status, &a.PayoutMethod, &a.PayoutDetails, &a.CreatedAt, &a.UpdatedAt) == nil {
			out = append(out, &a)
		}
	}
	return out, nil
}

func (p *PostgresStore) RecordAffiliateReferral(ctx context.Context, affiliateID, referredUserID uuid.UUID, attributionModel string) (*AffiliateReferral, error) {
	var ownerUserID uuid.UUID
	if err := p.db.QueryRowContext(ctx, `SELECT user_id FROM affiliate_accounts WHERE id=$1`, affiliateID).Scan(&ownerUserID); err != nil {
		return nil, err
	}
	if ownerUserID == referredUserID {
		return nil, ErrSelfReferralForbidden
	}
	if attributionModel == "" {
		attributionModel = "first_click"
	}
	ref := &AffiliateReferral{ID: uuid.New(), AffiliateID: affiliateID, ReferredUserID: referredUserID, AttributionModel: attributionModel, Status: "signed_up", CreatedAt: time.Now().UTC()}
	res, err := p.db.ExecContext(ctx, `
		INSERT INTO affiliate_referrals (id, affiliate_id, referred_user_id, attribution_model, status)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (referred_user_id) DO NOTHING
	`, ref.ID, ref.AffiliateID, ref.ReferredUserID, ref.AttributionModel, ref.Status)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			_, _ = p.db.ExecContext(ctx, `UPDATE affiliate_accounts SET total_referrals = total_referrals + 1, updated_at = NOW() WHERE id = $1`, affiliateID)
		}
	}
	return ref, err
}

func (p *PostgresStore) RecordAffiliateCommission(ctx context.Context, referredUserID, invoiceID uuid.UUID, orderAmountUSD float64) (*AffiliateCommission, error) {
	var affID uuid.UUID
	if err := p.db.QueryRowContext(ctx, `SELECT affiliate_id FROM affiliate_referrals WHERE referred_user_id=$1`, referredUserID).Scan(&affID); err != nil {
		return nil, nil
	}
	var rate float64
	if err := p.db.QueryRowContext(ctx, `SELECT commission_rate_pct FROM affiliate_accounts WHERE id=$1 AND status='active'`, affID).Scan(&rate); err != nil {
		return nil, nil
	}
	commUSD := math.Round((orderAmountUSD*(rate/100.0))*100) / 100
	now := time.Now().UTC()
	comm := &AffiliateCommission{ID: uuid.New(), AffiliateID: affID, ReferredUserID: referredUserID, InvoiceID: invoiceID, OrderAmountUSD: orderAmountUSD, CommissionUSD: commUSD, Status: "approved", CreatedAt: now, ApprovedAt: &now}
	res, err := p.db.ExecContext(ctx, `
		INSERT INTO affiliate_commissions (id, affiliate_id, referred_user_id, invoice_id, order_amount_usd, commission_usd, status, approved_at)
		VALUES ($1,$2,$3,$4,$5,$6,'approved',NOW()) ON CONFLICT (invoice_id) DO NOTHING
	`, comm.ID, comm.AffiliateID, comm.ReferredUserID, comm.InvoiceID, comm.OrderAmountUSD, comm.CommissionUSD)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			_, _ = p.db.ExecContext(ctx, `UPDATE affiliate_accounts SET approved_balance_usd = approved_balance_usd + $1, updated_at = NOW() WHERE id = $2`, commUSD, affID)
			_, _ = p.db.ExecContext(ctx, `UPDATE affiliate_referrals SET status='converted' WHERE referred_user_id=$1`, referredUserID)
		}
	}
	return comm, err
}

func (p *PostgresStore) ReverseAffiliateCommissionByInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	var affID uuid.UUID
	var commUSD float64
	var status string
	if err := p.db.QueryRowContext(ctx, `SELECT affiliate_id, commission_usd, status FROM affiliate_commissions WHERE invoice_id=$1`, invoiceID).Scan(&affID, &commUSD, &status); err != nil {
		return nil
	}
	if status == "reversed" {
		return nil
	}
	_, _ = p.db.ExecContext(ctx, `UPDATE affiliate_commissions SET status='reversed' WHERE invoice_id=$1`, invoiceID)
	_, _ = p.db.ExecContext(ctx, `UPDATE affiliate_accounts SET approved_balance_usd = GREATEST(0, approved_balance_usd - $1), updated_at = NOW() WHERE id=$2`, commUSD, affID)
	return nil
}

func (p *PostgresStore) RequestAffiliateWithdrawal(ctx context.Context, userID uuid.UUID, amountUSD float64, method, details string) (*AffiliateWithdrawal, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var affID uuid.UUID
	var approved float64
	if err := tx.QueryRowContext(ctx, `SELECT id, approved_balance_usd FROM affiliate_accounts WHERE user_id=$1 FOR UPDATE`, userID).Scan(&affID, &approved); err != nil {
		return nil, err
	}
	if amountUSD <= 0 || amountUSD > approved {
		return nil, errors.New("requested withdrawal exceeds approved balance")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE affiliate_accounts SET approved_balance_usd = approved_balance_usd - $1, updated_at = NOW() WHERE id=$2`, amountUSD, affID); err != nil {
		return nil, err
	}
	w := &AffiliateWithdrawal{ID: uuid.New(), AffiliateID: affID, UserID: userID, AmountUSD: amountUSD, PayoutMethod: method, PayoutDetails: details, Status: "pending", CreatedAt: time.Now().UTC()}
	if _, err := tx.ExecContext(ctx, `INSERT INTO affiliate_withdrawals (id, affiliate_id, user_id, amount_usd, payout_method, payout_details, status) VALUES ($1,$2,$3,$4,$5,$6,'pending')`, w.ID, w.AffiliateID, w.UserID, w.AmountUSD, w.PayoutMethod, w.PayoutDetails); err != nil {
		return nil, err
	}
	return w, tx.Commit()
}

func (p *PostgresStore) ApproveAffiliateWithdrawal(ctx context.Context, withdrawalID uuid.UUID, approve bool, adminNotes string) (*AffiliateWithdrawal, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var w AffiliateWithdrawal
	if err := tx.QueryRowContext(ctx, `SELECT id, affiliate_id, user_id, amount_usd, payout_method, payout_details, status FROM affiliate_withdrawals WHERE id=$1 FOR UPDATE`, withdrawalID).Scan(&w.ID, &w.AffiliateID, &w.UserID, &w.AmountUSD, &w.PayoutMethod, &w.PayoutDetails, &w.Status); err != nil {
		return nil, err
	}
	if w.Status != "pending" {
		return nil, ErrWithdrawalAlreadyProcessed
	}
	if approve {
		w.Status = "paid"
		_, _ = tx.ExecContext(ctx, `UPDATE affiliate_accounts SET total_paid_usd = total_paid_usd + $1, updated_at = NOW() WHERE id=$2`, w.AmountUSD, w.AffiliateID)
	} else {
		w.Status = "rejected"
		_, _ = tx.ExecContext(ctx, `UPDATE affiliate_accounts SET approved_balance_usd = approved_balance_usd + $1, updated_at = NOW() WHERE id=$2`, w.AmountUSD, w.AffiliateID)
	}
	_, _ = tx.ExecContext(ctx, `UPDATE affiliate_withdrawals SET status=$1, admin_notes=$2, processed_at=NOW() WHERE id=$3`, w.Status, adminNotes, w.ID)
	return &w, tx.Commit()
}

func (p *PostgresStore) ListAffiliateWithdrawals(ctx context.Context, userID uuid.UUID) ([]*AffiliateWithdrawal, error) {
	q := `SELECT id, affiliate_id, user_id, amount_usd, payout_method, payout_details, status, admin_notes, created_at, processed_at FROM affiliate_withdrawals`
	args := []interface{}{}
	if userID != uuid.Nil {
		q += ` WHERE user_id = $1`
		args = append(args, userID)
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AffiliateWithdrawal, 0)
	for rows.Next() {
		var w AffiliateWithdrawal
		if rows.Scan(&w.ID, &w.AffiliateID, &w.UserID, &w.AmountUSD, &w.PayoutMethod, &w.PayoutDetails, &w.Status, &w.AdminNotes, &w.CreatedAt, &w.ProcessedAt) == nil {
			out = append(out, &w)
		}
	}
	return out, nil
}

func (p *PostgresStore) GetOrCreateCustomerWallet(ctx context.Context, userID uuid.UUID) (*CustomerWallet, error) {
	_, _ = p.db.ExecContext(ctx, `INSERT INTO customer_wallets (id, user_id, balance_usd, currency, auto_pay_enabled) VALUES ($1,$2,0,'USD',TRUE) ON CONFLICT (user_id) DO NOTHING`, uuid.New(), userID)
	var w CustomerWallet
	err := p.db.QueryRowContext(ctx, `SELECT id, user_id, balance_usd, currency, auto_pay_enabled, created_at, updated_at FROM customer_wallets WHERE user_id=$1`, userID).Scan(&w.ID, &w.UserID, &w.BalanceUSD, &w.Currency, &w.AutoPayEnabled, &w.CreatedAt, &w.UpdatedAt)
	return &w, err
}

func (p *PostgresStore) SetWalletAutoPay(ctx context.Context, userID uuid.UUID, enabled bool) (*CustomerWallet, error) {
	w, err := p.GetOrCreateCustomerWallet(ctx, userID)
	if err != nil {
		return nil, err
	}
	_, err = p.db.ExecContext(ctx, `UPDATE customer_wallets SET auto_pay_enabled=$1, updated_at=NOW() WHERE user_id=$2`, enabled, userID)
	w.AutoPayEnabled = enabled
	return w, err
}

func (p *PostgresStore) CreditWalletAtomic(ctx context.Context, userID uuid.UUID, amountUSD float64, entryType, refID, idempotencyKey, notes string) (*WalletLedgerEntry, error) {
	if amountUSD <= 0 {
		return nil, errors.New("credit amount must be positive")
	}
	_, _ = p.GetOrCreateCustomerWallet(ctx, userID)
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var existing WalletLedgerEntry
	if err := tx.QueryRowContext(ctx, `SELECT id, wallet_id, user_id, entry_type, direction, amount_usd, balance_before_usd, balance_after_usd, reference_id, idempotency_key, notes, created_at FROM wallet_ledger_entries WHERE idempotency_key=$1`, idempotencyKey).Scan(&existing.ID, &existing.WalletID, &existing.UserID, &existing.EntryType, &existing.Direction, &existing.AmountUSD, &existing.BalanceBeforeUSD, &existing.BalanceAfterUSD, &existing.ReferenceID, &existing.IdempotencyKey, &existing.Notes, &existing.CreatedAt); err == nil {
		return &existing, nil
	}
	var walletID uuid.UUID
	var before float64
	if err := tx.QueryRowContext(ctx, `SELECT id, balance_usd FROM customer_wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&walletID, &before); err != nil {
		return nil, err
	}
	after := math.Round((before+amountUSD)*10000) / 10000
	if _, err := tx.ExecContext(ctx, `UPDATE customer_wallets SET balance_usd=$1, updated_at=NOW() WHERE id=$2`, after, walletID); err != nil {
		return nil, err
	}
	entry := &WalletLedgerEntry{ID: uuid.New(), WalletID: walletID, UserID: userID, EntryType: entryType, Direction: "credit", AmountUSD: amountUSD, BalanceBeforeUSD: before, BalanceAfterUSD: after, ReferenceID: refID, IdempotencyKey: idempotencyKey, Notes: notes, CreatedAt: time.Now().UTC()}
	if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries (id, wallet_id, user_id, entry_type, direction, amount_usd, balance_before_usd, balance_after_usd, reference_id, idempotency_key, notes) VALUES ($1,$2,$3,$4,'credit',$5,$6,$7,$8,$9,$10)`, entry.ID, entry.WalletID, entry.UserID, entry.EntryType, entry.AmountUSD, entry.BalanceBeforeUSD, entry.BalanceAfterUSD, entry.ReferenceID, entry.IdempotencyKey, entry.Notes); err != nil {
		return nil, err
	}
	return entry, tx.Commit()
}

func (p *PostgresStore) DebitWalletAtomic(ctx context.Context, userID uuid.UUID, amountUSD float64, entryType, refID, idempotencyKey, notes string) (*WalletLedgerEntry, error) {
	if amountUSD <= 0 {
		return nil, errors.New("debit amount must be positive")
	}
	_, _ = p.GetOrCreateCustomerWallet(ctx, userID)
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var existing WalletLedgerEntry
	if err := tx.QueryRowContext(ctx, `SELECT id, wallet_id, user_id, entry_type, direction, amount_usd, balance_before_usd, balance_after_usd, reference_id, idempotency_key, notes, created_at FROM wallet_ledger_entries WHERE idempotency_key=$1`, idempotencyKey).Scan(&existing.ID, &existing.WalletID, &existing.UserID, &existing.EntryType, &existing.Direction, &existing.AmountUSD, &existing.BalanceBeforeUSD, &existing.BalanceAfterUSD, &existing.ReferenceID, &existing.IdempotencyKey, &existing.Notes, &existing.CreatedAt); err == nil {
		return &existing, nil
	}
	var walletID uuid.UUID
	var before float64
	if err := tx.QueryRowContext(ctx, `SELECT id, balance_usd FROM customer_wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&walletID, &before); err != nil {
		return nil, err
	}
	if before+1e-9 < amountUSD {
		return nil, ErrInsufficientWallet
	}
	after := math.Round((before-amountUSD)*10000) / 10000
	if after < 0 {
		after = 0
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_wallets SET balance_usd=$1, updated_at=NOW() WHERE id=$2`, after, walletID); err != nil {
		return nil, err
	}
	entry := &WalletLedgerEntry{ID: uuid.New(), WalletID: walletID, UserID: userID, EntryType: entryType, Direction: "debit", AmountUSD: amountUSD, BalanceBeforeUSD: before, BalanceAfterUSD: after, ReferenceID: refID, IdempotencyKey: idempotencyKey, Notes: notes, CreatedAt: time.Now().UTC()}
	if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries (id, wallet_id, user_id, entry_type, direction, amount_usd, balance_before_usd, balance_after_usd, reference_id, idempotency_key, notes) VALUES ($1,$2,$3,$4,'debit',$5,$6,$7,$8,$9,$10)`, entry.ID, entry.WalletID, entry.UserID, entry.EntryType, entry.AmountUSD, entry.BalanceBeforeUSD, entry.BalanceAfterUSD, entry.ReferenceID, entry.IdempotencyKey, entry.Notes); err != nil {
		return nil, err
	}
	return entry, tx.Commit()
}

func (p *PostgresStore) ListWalletLedger(ctx context.Context, userID uuid.UUID, limit int) ([]*WalletLedgerEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id, wallet_id, user_id, entry_type, direction, amount_usd, balance_before_usd, balance_after_usd, reference_id, idempotency_key, notes, created_at FROM wallet_ledger_entries WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*WalletLedgerEntry, 0)
	for rows.Next() {
		var e WalletLedgerEntry
		if rows.Scan(&e.ID, &e.WalletID, &e.UserID, &e.EntryType, &e.Direction, &e.AmountUSD, &e.BalanceBeforeUSD, &e.BalanceAfterUSD, &e.ReferenceID, &e.IdempotencyKey, &e.Notes, &e.CreatedAt) == nil {
			out = append(out, &e)
		}
	}
	return out, nil
}

func (p *PostgresStore) RecordBillingDunningEvent(ctx context.Context, ev *BillingDunningEvent) (bool, error) {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	res, err := p.db.ExecContext(ctx, `
		INSERT INTO billing_dunning_events (id, subscription_id, user_id, invoice_id, cycle_key, notice_stage, details)
		VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (subscription_id, cycle_key, notice_stage) DO NOTHING
	`, ev.ID, ev.SubscriptionID, ev.UserID, ev.InvoiceID, ev.CycleKey, ev.NoticeStage, ev.Details)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (p *PostgresStore) ListBillingDunningEvents(ctx context.Context, subID uuid.UUID, limit int) ([]*BillingDunningEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, subscription_id, user_id, invoice_id, cycle_key, notice_stage, details, created_at FROM billing_dunning_events`
	args := []interface{}{}
	if subID != uuid.Nil {
		q += ` WHERE subscription_id=$1 ORDER BY created_at DESC LIMIT $2`
		args = append(args, subID, limit)
	} else {
		q += ` ORDER BY created_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := p.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*BillingDunningEvent, 0)
	for rows.Next() {
		var e BillingDunningEvent
		if rows.Scan(&e.ID, &e.SubscriptionID, &e.UserID, &e.InvoiceID, &e.CycleKey, &e.NoticeStage, &e.Details, &e.CreatedAt) == nil {
			out = append(out, &e)
		}
	}
	return out, nil
}
