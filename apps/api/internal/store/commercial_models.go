package store

import (
	"time"

	"github.com/google/uuid"
)

type Coupon struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Description       string     `json:"description"`
	DiscountType      string     `json:"discount_type"` // percentage, fixed
	DiscountValue     float64    `json:"discount_value"`
	MaxUses           int        `json:"max_uses"`
	CurrentUses       int        `json:"current_uses"`
	MaxUsesPerUser    int        `json:"max_uses_per_user"`
	MinOrderUSD       float64    `json:"min_order_usd"`
	FirstOrderOnly    bool       `json:"first_order_only"`
	RecurringDiscount bool       `json:"recurring_discount"`
	ApplicablePlanIDs string     `json:"applicable_plan_ids,omitempty"`
	IsActive          bool       `json:"is_active"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type CouponRedemption struct {
	ID          uuid.UUID `json:"id"`
	CouponID    uuid.UUID `json:"coupon_id"`
	UserID      uuid.UUID `json:"user_id"`
	InvoiceID   uuid.UUID `json:"invoice_id"`
	DiscountUSD float64   `json:"discount_usd"`
	CreatedAt   time.Time `json:"created_at"`
}

type AffiliateAccount struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	ReferralCode       string    `json:"referral_code"`
	CommissionRatePct  float64   `json:"commission_rate_pct"`
	PendingBalanceUSD  float64   `json:"pending_balance_usd"`
	ApprovedBalanceUSD float64   `json:"approved_balance_usd"`
	TotalPaidUSD       float64   `json:"total_paid_usd"`
	TotalReferrals     int       `json:"total_referrals"`
	Status             string    `json:"status"` // active, suspended
	PayoutMethod       string    `json:"payout_method"`
	PayoutDetails      string    `json:"payout_details,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AffiliateReferral struct {
	ID               uuid.UUID `json:"id"`
	AffiliateID      uuid.UUID `json:"affiliate_id"`
	ReferredUserID   uuid.UUID `json:"referred_user_id"`
	AttributionModel string    `json:"attribution_model"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type AffiliateCommission struct {
	ID             uuid.UUID  `json:"id"`
	AffiliateID    uuid.UUID  `json:"affiliate_id"`
	ReferredUserID uuid.UUID  `json:"referred_user_id"`
	InvoiceID      uuid.UUID  `json:"invoice_id"`
	OrderAmountUSD float64    `json:"order_amount_usd"`
	CommissionUSD  float64    `json:"commission_usd"`
	Status         string     `json:"status"` // pending, approved, reversed, paid
	CreatedAt      time.Time  `json:"created_at"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
}

type AffiliateWithdrawal struct {
	ID            uuid.UUID  `json:"id"`
	AffiliateID   uuid.UUID  `json:"affiliate_id"`
	UserID        uuid.UUID  `json:"user_id"`
	AmountUSD     float64    `json:"amount_usd"`
	PayoutMethod  string     `json:"payout_method"`
	PayoutDetails string     `json:"payout_details,omitempty"`
	Status        string     `json:"status"` // pending, approved, rejected, paid
	AdminNotes    string     `json:"admin_notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

type CustomerWallet struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	BalanceUSD     float64   `json:"balance_usd"`
	Currency       string    `json:"currency"`
	AutoPayEnabled bool      `json:"auto_pay_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WalletLedgerEntry struct {
	ID               uuid.UUID `json:"id"`
	WalletID         uuid.UUID `json:"wallet_id"`
	UserID           uuid.UUID `json:"user_id"`
	EntryType        string    `json:"entry_type"` // topup, invoice_payment, auto_renewal, refund, affiliate_payout, admin_adjustment
	Direction        string    `json:"direction"`  // credit, debit
	AmountUSD        float64   `json:"amount_usd"`
	BalanceBeforeUSD float64   `json:"balance_before_usd"`
	BalanceAfterUSD  float64   `json:"balance_after_usd"`
	ReferenceID      string    `json:"reference_id,omitempty"`
	IdempotencyKey   string    `json:"idempotency_key"`
	Notes            string    `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type BillingDunningEvent struct {
	ID             uuid.UUID  `json:"id"`
	SubscriptionID uuid.UUID  `json:"subscription_id"`
	UserID         uuid.UUID  `json:"user_id"`
	InvoiceID      *uuid.UUID `json:"invoice_id,omitempty"`
	CycleKey       string     `json:"cycle_key"`
	NoticeStage    string     `json:"notice_stage"` // renewal_7d, renewal_3d, renewal_1d, auto_pay_success, grace_warning, suspended
	Details        string     `json:"details"`
	CreatedAt      time.Time  `json:"created_at"`
}
