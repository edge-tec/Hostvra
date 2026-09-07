package store

import (
	"time"

	"github.com/google/uuid"
)

type PlanTier string

const (
	PlanTierStarter    PlanTier = "starter"
	PlanTierBusiness   PlanTier = "business"
	PlanTierEnterprise PlanTier = "enterprise"
	PlanTierReseller   PlanTier = "reseller"
)

type HostingPlan struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	Tier          PlanTier  `json:"tier"`
	PriceMonthly  float64   `json:"price_monthly"`
	PriceYearly   float64   `json:"price_yearly"`
	Currency      string    `json:"currency"` // USD, BDT, EUR
	DiskSpaceMB   int64     `json:"disk_space_mb"`
	BandwidthMB   int64     `json:"bandwidth_mb"`
	MaxWebsites   int       `json:"max_websites"`
	MaxDatabases  int       `json:"max_databases"`
	MaxMailboxes  int       `json:"max_mailboxes"`
	MaxFTP        int       `json:"max_ftp"`
	DedicatedIP   bool      `json:"dedicated_ip"`
	FreeSSL       bool      `json:"free_ssl"`
	Features      []string  `json:"features"`
	IsActive      bool      `json:"is_active"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SubscriptionStatus string

const (
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusPending   SubscriptionStatus = "pending"
	SubStatusSuspended SubscriptionStatus = "suspended"
	SubStatusCancelled SubscriptionStatus = "cancelled"
	SubStatusExpired   SubscriptionStatus = "expired"
)

type Subscription struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"user_id"`
	OrganizationID  uuid.UUID          `json:"organization_id"`
	PlanID          uuid.UUID          `json:"plan_id"`
	PlanName        string             `json:"plan_name"`
	ServerID        *uuid.UUID         `json:"server_id,omitempty"`
	Status          SubscriptionStatus `json:"status"`
	BillingCycle    string             `json:"billing_cycle"` // monthly, yearly
	Amount          float64            `json:"amount"`
	Currency        string             `json:"currency"`
	DiskUsedMB      int64              `json:"disk_used_mb"`
	BandwidthUsedMB int64              `json:"bandwidth_used_mb"`
	WebsitesCount   int                `json:"websites_count"`
	NextBillingDate time.Time          `json:"next_billing_date"`
	AutoRenew       bool               `json:"auto_renew"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type InvoiceStatus string

const (
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusUnpaid    InvoiceStatus = "unpaid"
	InvoiceStatusOverdue   InvoiceStatus = "overdue"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

type Invoice struct {
	ID             uuid.UUID     `json:"id"`
	InvoiceNumber  string        `json:"invoice_number"`
	UserID         uuid.UUID     `json:"user_id"`
	SubscriptionID *uuid.UUID    `json:"subscription_id,omitempty"`
	PlanID         uuid.UUID     `json:"plan_id"`
	Description    string        `json:"description"`
	Subtotal       float64       `json:"subtotal"`
	Tax            float64       `json:"tax"`
	Discount       float64       `json:"discount"`
	Total          float64       `json:"total"`
	Currency       string        `json:"currency"`
	Status         InvoiceStatus `json:"status"`
	PaymentMethod  string        `json:"payment_method,omitempty"` // stripe, bkash, nagad, sslcommerz, paypal
	TransactionID  string        `json:"transaction_id,omitempty"`
	DueDate        time.Time     `json:"due_date"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

type PaymentGatewayConfig struct {
	Gateway     string    `json:"gateway"` // stripe, bkash, nagad, sslcommerz, paypal
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	TestMode    bool      `json:"test_mode"`
	ApiKey      string    `json:"api_key,omitempty"`
	SecretKey   string    `json:"secret_key,omitempty"`
	MerchantID  string    `json:"merchant_id,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}
