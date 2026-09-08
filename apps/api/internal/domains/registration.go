package domains

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type ProvisioningWorker struct {
	store     store.Store
	registrar DomainRegistrar
	mu        sync.Mutex
	locks     map[string]bool // In-flight registration locks by domain name
}

func NewProvisioningWorker(s store.Store, r DomainRegistrar) *ProvisioningWorker {
	return &ProvisioningWorker{
		store:     s,
		registrar: r,
		locks:     make(map[string]bool),
	}
}

// ProcessPaidOrder coordinates the registration of a verified paid domain order
func (pw *ProvisioningWorker) ProcessPaidOrder(ctx context.Context, orderID uuid.UUID, contact *ContactInfo, nameservers []string) error {
	// 1. Load domain order
	order, err := pw.store.GetDomainOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order %s not found: %w", orderID.String(), err)
	}

	// 2. Strict Check: Only paid orders can be provisioned
	if order.PaymentStatus != "paid" {
		return fmt.Errorf("cannot provision domain %s: order is not marked as paid (status: %s)", order.DomainName, order.PaymentStatus)
	}

	// 3. Idempotency Check: Do not re-register completed orders
	if order.ProvisioningStatus == "completed" {
		slog.Info("Domain order already completed; skipping redundant registration", "order_id", orderID, "domain", order.DomainName)
		return nil
	}

	// 4. Acquire domain lock (distributed + local) to prevent concurrent provisioning
	pw.mu.Lock()
	if pw.locks[order.DomainName] {
		pw.mu.Unlock()
		return fmt.Errorf("registration for domain %s is already in-flight", order.DomainName)
	}
	pw.locks[order.DomainName] = true
	pw.mu.Unlock()

	defer func() {
		pw.mu.Lock()
		delete(pw.locks, order.DomainName)
		pw.mu.Unlock()
	}()

	// Acquire DB distributed advisory lock
	acquired, unlock, lockErr := pw.store.TryAcquireDomainAdvisoryLock(ctx, "domain:provision:"+order.DomainName)
	if lockErr != nil || !acquired {
		return fmt.Errorf("registration lock active for domain %s: operation in progress", order.DomainName)
	}
	if unlock != nil {
		defer unlock()
	}

	// 5. Update status to provisioning
	order.ProvisioningStatus = "provisioning"
	_ = pw.store.UpdateDomainOrder(ctx, order)

	// Fallback contacts if not provided in call
	if contact == nil {
		contact = &ContactInfo{
			FirstName:  "Domain",
			LastName:   "Administrator",
			Email:      "admin@" + order.DomainName,
			Phone:      "15551234567",
			Address1:   "100 Hostvra Way",
			City:       "Wilmington",
			State:      "DE",
			PostalCode: "19801",
			Country:    "US",
		}
	}

	if len(nameservers) < 2 {
		nameservers = []string{"ns1.hostvra.com", "ns2.hostvra.com"}
	}

	// 6. Uncertain Registration Check: Check if domain was already registered at provider
	var regResult *DomainRegistrationResult
	existingInfo, infoErr := pw.registrar.GetDomainInfo(ctx, order.DomainName)
	if infoErr == nil && existingInfo != nil && (existingInfo.Status == "Active" || existingInfo.Status == "Registered") && existingInfo.ProviderOrderID != "" {
		slog.Info("Domain already registered at registrar; reconciling provider state without duplicate registration", "domain", order.DomainName, "provider_order_id", existingInfo.ProviderOrderID)
		expDate := time.Now().UTC().AddDate(order.Years, 0, 0)
		if existingInfo.ExpiryDate != nil {
			expDate = *existingInfo.ExpiryDate
		}
		regResult = &DomainRegistrationResult{
			DomainName:       order.DomainName,
			ProviderOrderID:  existingInfo.ProviderOrderID,
			ProviderDomainID: existingInfo.ProviderOrderID,
			Status:           "active",
			RegistrationDate: time.Now().UTC(),
			ExpiryDate:       expDate,
			Nameservers:      existingInfo.Nameservers,
			RawResponse:      "reconciled_from_registrar",
		}
	} else {
		// Call Registrar API
		var regErr error
		regResult, regErr = pw.registrar.RegisterDomain(ctx, RegisterDomainRequest{
			DomainName:     order.DomainName,
			Years:          order.Years,
			Nameservers:    nameservers,
			PrivacyEnabled: false,
			Registrant:     contact,
		})

		if regErr != nil {
			// Network drop or uncertain response recovery: check if registrar actually completed it!
			if recInfo, checkErr := pw.registrar.GetDomainInfo(ctx, order.DomainName); checkErr == nil && recInfo != nil && (recInfo.Status == "Active" || recInfo.Status == "Registered") && recInfo.ProviderOrderID != "" {
				slog.Info("Registrar returned error/timeout but domain is verified Active; safely recovered state", "domain", order.DomainName, "provider_order_id", recInfo.ProviderOrderID)
				expDate := time.Now().UTC().AddDate(order.Years, 0, 0)
				if recInfo.ExpiryDate != nil {
					expDate = *recInfo.ExpiryDate
				}
				regResult = &DomainRegistrationResult{
					DomainName:       order.DomainName,
					ProviderOrderID:  recInfo.ProviderOrderID,
					ProviderDomainID: recInfo.ProviderOrderID,
					Status:           "active",
					RegistrationDate: time.Now().UTC(),
					ExpiryDate:       expDate,
					Nameservers:      recInfo.Nameservers,
					RawResponse:      "recovered_after_timeout",
				}
			} else {
				// Provisioning failed: update order status, retain paid status
				order.ProvisioningStatus = "failed"
				order.FailureReason = regErr.Error()
				order.RetryCount++
				_ = pw.store.UpdateDomainOrder(ctx, order)

				_ = pw.store.RecordDomainTransaction(ctx, &store.DomainTransaction{
					OrderID:      &order.ID,
					Provider:     "resellerclub",
					Operation:    "registration",
					Amount:       order.Amount,
					Cost:         order.Cost,
					Currency:     order.Currency,
					Status:       "failed",
					ErrorMessage: regErr.Error(),
				})

				_ = pw.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
					DomainName: order.DomainName,
					UserID:     &order.UserID,
					Action:     "DOMAIN_REGISTRATION_FAILED",
					Details:    fmt.Sprintf("Registration failed for %s: %s", order.DomainName, regErr.Error()),
					CreatedAt:  time.Now().UTC(),
				})

				return fmt.Errorf("registrar registration failed: %w", regErr)
			}
		}
	}

	// 7. Registration succeeded: create or update Domain record
	_, cleanTLD, _ := ValidateDomainName(order.DomainName)
	tld := cleanTLD
	now := time.Now().UTC()

	var domainID uuid.UUID
	if existingDomain, _ := pw.store.GetDomainByName(ctx, order.DomainName); existingDomain != nil {
		domainID = existingDomain.ID
		existingDomain.Status = "active"
		existingDomain.ProviderOrderID = regResult.ProviderOrderID
		existingDomain.ProviderDomainID = regResult.ProviderDomainID
		existingDomain.RegistrationDate = &regResult.RegistrationDate
		existingDomain.ExpiryDate = &regResult.ExpiryDate
		existingDomain.UpdatedAt = now
		_ = pw.store.UpdateDomain(ctx, existingDomain)
	} else {
		domainID = uuid.New()
		domainRecord := &store.Domain{
			ID:               domainID,
			UserID:           order.UserID,
			OrganizationID:   order.OrganizationID,
			OrderID:          &order.ID,
			DomainName:       order.DomainName,
			TLD:              tld,
			Registrar:        "resellerclub",
			ProviderOrderID:  regResult.ProviderOrderID,
			ProviderDomainID: regResult.ProviderDomainID,
			Status:           "active",
			RegistrationDate: &regResult.RegistrationDate,
			ExpiryDate:       &regResult.ExpiryDate,
			TransferStatus:   "none",
			AutoRenew:        true,
			RegistrarLock:    true,
			PrivacyEnabled:   false,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := pw.store.CreateDomain(ctx, domainRecord); err != nil {
			slog.Error("Failed to save registered domain to store", "error", err)
		}
	}

	// 8. Save Contacts
	_ = pw.store.SaveDomainContacts(ctx, []*store.DomainContact{
		{
			ID:          uuid.New(),
			DomainID:    domainID,
			ContactType: "registrant",
			FirstName:   contact.FirstName,
			LastName:    contact.LastName,
			Email:       contact.Email,
			Phone:       contact.Phone,
			Address1:    contact.Address1,
			City:        contact.City,
			State:       contact.State,
			PostalCode:  contact.PostalCode,
			Country:     contact.Country,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	})

	// 9. Save Nameservers
	_ = pw.store.SaveDomainNameservers(ctx, domainID, nameservers)

	// 10. Update Order to Completed
	order.DomainID = &domainID
	order.ProvisioningStatus = "completed"
	order.ProviderOrderID = regResult.ProviderOrderID
	order.ProviderStatus = "Active"
	order.FailureReason = ""
	_ = pw.store.UpdateDomainOrder(ctx, order)

	// 11. Record Financial Transaction & Audit Log
	_ = pw.store.RecordDomainTransaction(ctx, &store.DomainTransaction{
		DomainID:        &domainID,
		OrderID:         &order.ID,
		Provider:        "resellerclub",
		Operation:       "registration",
		ProviderOrderID: regResult.ProviderOrderID,
		Amount:          order.Amount,
		Cost:            order.Cost,
		Currency:        order.Currency,
		Status:          "success",
	})

	_ = pw.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainID:   &domainID,
		DomainName: order.DomainName,
		UserID:     &order.UserID,
		Action:     "DOMAIN_REGISTERED",
		Details:    fmt.Sprintf("Domain %s successfully registered at ResellerClub (Provider Order: %s)", order.DomainName, regResult.ProviderOrderID),
		CreatedAt:  now,
	})

	slog.Info("Domain registration completed successfully",
		"domain", order.DomainName,
		"provider_order_id", regResult.ProviderOrderID,
		"order_id", order.ID,
	)

	return nil
}
