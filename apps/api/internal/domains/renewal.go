package domains

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type RenewalService struct {
	store     store.Store
	registrar DomainRegistrar
	pricing   *PricingEngine
}

func NewRenewalService(s store.Store, r DomainRegistrar, pe *PricingEngine) *RenewalService {
	return &RenewalService{
		store:     s,
		registrar: r,
		pricing:   pe,
	}
}

// RequestRenewal creates a renewal invoice and order
func (rs *RenewalService) RequestRenewal(ctx context.Context, domainID uuid.UUID, userID uuid.UUID, years int) (*store.Invoice, error) {
	d, err := rs.store.GetDomainByID(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		return nil, fmt.Errorf("unauthorized to renew this domain")
	}

	if years < 1 {
		years = 1
	}

	priceInfo, err := rs.pricing.CalculateRenewalPrice(ctx, d.TLD, years)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	invID := uuid.New()
	inv := &store.Invoice{
		ID:            invID,
		InvoiceNumber: fmt.Sprintf("REN-%d-%05d", now.Year(), time.Now().Nanosecond()%90000+10000),
		UserID:        userID,
		Description:   fmt.Sprintf("Domain Renewal: %s (%d Year)", d.DomainName, years),
		Subtotal:      priceInfo.TotalPrice,
		Total:         priceInfo.TotalPrice,
		Currency:      priceInfo.Currency,
		Status:        store.InvoiceStatusUnpaid,
		DueDate:       now.AddDate(0, 0, 7),
		CreatedAt:     now,
	}
	if err := rs.store.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}

	renewalOrder := &store.DomainRenewal{
		ID:            uuid.New(),
		DomainID:      domainID,
		UserID:        userID,
		Years:         years,
		Amount:        priceInfo.TotalPrice,
		Currency:      priceInfo.Currency,
		PaymentStatus: "pending",
		Status:        "pending",
		OldExpiryDate: d.ExpiryDate,
		CreatedAt:     now,
	}
	_ = rs.store.CreateDomainRenewal(ctx, renewalOrder)

	return inv, nil
}

// ProcessPaidRenewal executes the renewal at the registrar after payment verification
func (rs *RenewalService) ProcessPaidRenewal(ctx context.Context, domainID uuid.UUID, years int) error {
	d, err := rs.store.GetDomainByID(ctx, domainID)
	if err != nil {
		return err
	}

	result, err := rs.registrar.RenewDomain(ctx, RenewDomainRequest{
		DomainName:      d.DomainName,
		ProviderOrderID: d.ProviderOrderID,
		Years:           years,
		CurrentExpiry:   d.ExpiryDate,
	})
	if err != nil {
		return fmt.Errorf("registrar renewal failed: %w", err)
	}

	// Update local domain expiry date
	d.ExpiryDate = &result.NewExpiryDate
	_ = rs.store.UpdateDomain(ctx, d)

	_ = rs.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainID:   &domainID,
		DomainName: d.DomainName,
		UserID:     &d.UserID,
		Action:     "DOMAIN_RENEWED",
		Details:    fmt.Sprintf("Domain %s renewed for %d year(s). New expiry: %s", d.DomainName, years, result.NewExpiryDate.Format(time.RFC3339)),
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}
