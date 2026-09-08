package domains

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type OrderService struct {
	store   store.Store
	pricing *PricingEngine
}

func NewOrderService(s store.Store, pe *PricingEngine) *OrderService {
	return &OrderService{
		store:   s,
		pricing: pe,
	}
}

type CreateRegistrationOrderRequest struct {
	UserID         uuid.UUID    `json:"user_id"`
	OrganizationID *uuid.UUID   `json:"organization_id,omitempty"`
	DomainName     string       `json:"domain_name"`
	Years          int          `json:"years"`
	Nameservers    []string     `json:"nameservers"`
	PrivacyEnabled bool         `json:"privacy_enabled"`
	Registrant     *ContactInfo `json:"registrant"`
	PaymentMethod  string       `json:"payment_method"`
}

type OrderCheckoutResult struct {
	Order   *store.DomainOrder `json:"order"`
	Invoice *store.Invoice     `json:"invoice"`
	Message string             `json:"message"`
}

func (s *OrderService) CreateRegistrationOrder(ctx context.Context, req CreateRegistrationOrderRequest) (*OrderCheckoutResult, error) {
	cleanDomain, tld, err := ValidateDomainName(req.DomainName)
	if err != nil {
		return nil, err
	}

	if err := ValidateContact(req.Registrant, "Registrant"); err != nil {
		return nil, err
	}

	if req.Years < 1 {
		req.Years = 1
	}

	// Calculate selling price and cost
	priceInfo, err := s.pricing.CalculateRegistrationPrice(ctx, tld, req.Years)
	if err != nil {
		return nil, err
	}

	// Check if domain is already active in Hostvra
	if existingDomain, err := s.store.GetDomainByName(ctx, cleanDomain); err == nil && existingDomain != nil &&
		existingDomain.Status != "cancelled" && existingDomain.Status != "transferred_out" {
		return nil, fmt.Errorf("domain %s is already registered and active in Hostvra", cleanDomain)
	}

	// Compute Idempotency Key: user + domain + years + date
	dateStr := time.Now().UTC().Format("2006-01-02")
	rawIdem := fmt.Sprintf("reg:%s:%s:%d:%s", req.UserID.String(), cleanDomain, req.Years, dateStr)
	hasher := sha256.New()
	hasher.Write([]byte(rawIdem))
	idemKey := hex.EncodeToString(hasher.Sum(nil))

	// Check if order already exists with this idempotency key
	if existing, err := s.store.GetDomainOrderByIdempotencyKey(ctx, idemKey); err == nil && existing != nil {
		inv, _ := s.store.GetInvoiceByID(ctx, *existing.InvoiceID)
		return &OrderCheckoutResult{
			Order:   existing,
			Invoice: inv,
			Message: "Existing active order found for this domain registration",
		}, nil
	}

	now := time.Now().UTC()
	orderID := uuid.New()
	invoiceID := uuid.New()

	// 1. Create Billing Invoice (Unpaid)
	inv := &store.Invoice{
		ID:            invoiceID,
		InvoiceNumber: fmt.Sprintf("DOM-%d-%05d", now.Year(), time.Now().Nanosecond()%90000+10000),
		UserID:        req.UserID,
		Description:   fmt.Sprintf("Domain Registration: %s (%d Year)", cleanDomain, req.Years),
		Subtotal:      priceInfo.TotalPrice,
		Tax:           0.0,
		Discount:      0.0,
		Total:         priceInfo.TotalPrice,
		Currency:      priceInfo.Currency,
		Status:        store.InvoiceStatusUnpaid,
		PaymentMethod: req.PaymentMethod,
		DueDate:       now.AddDate(0, 0, 7),
		CreatedAt:     now,
	}
	if err := s.store.CreateInvoice(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create order invoice: %w", err)
	}

	// 2. Create Domain Order
	order := &store.DomainOrder{
		ID:                 orderID,
		UserID:             req.UserID,
		OrganizationID:     req.OrganizationID,
		DomainName:         cleanDomain,
		OrderType:          "registration",
		Years:              req.Years,
		Amount:             priceInfo.TotalPrice,
		Cost:               priceInfo.WholesaleTotalCost,
		Currency:           priceInfo.Currency,
		PaymentStatus:      "pending",
		ProvisioningStatus: "pending",
		ProviderStatus:     "none",
		IdempotencyKey:     idemKey,
		InvoiceID:          &invoiceID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.store.CreateDomainOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save domain order: %w", err)
	}

	// 3. Log Audit
	_ = s.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainName: cleanDomain,
		UserID:     &req.UserID,
		Action:     "DOMAIN_ORDER_CREATED",
		Details:    fmt.Sprintf("Created registration order %s for domain %s ($%.2f)", orderID.String(), cleanDomain, priceInfo.TotalPrice),
		CreatedAt:  now,
	})

	return &OrderCheckoutResult{
		Order:   order,
		Invoice: inv,
		Message: "Domain order created successfully. Awaiting payment confirmation.",
	}, nil
}
