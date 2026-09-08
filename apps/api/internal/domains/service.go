package domains

import (
	"hostvra/api/internal/store"
)

// Service aggregates all domain subsystem operations under one cohesive service
type Service struct {
	Registrar      DomainRegistrar
	Pricing        *PricingEngine
	Availability   *AvailabilityService
	Orders         *OrderService
	Provisioning   *ProvisioningWorker
	Nameservers    *NameserverService
	DNS            *DNSService
	Renewals       *RenewalService
	Transfers      *TransferService
	Reconciliation *ReconciliationWorker
	Notifications  *ExpiryNotifier
	Store          store.Store
}

func NewService(s store.Store, registrar DomainRegistrar, encryptionKey string) *Service {
	pricing := NewPricingEngine(s)
	availability := NewAvailabilityService(registrar, pricing)
	orders := NewOrderService(s, pricing)
	provisioning := NewProvisioningWorker(s, registrar)
	nameservers := NewNameserverService(s, registrar)
	dns := NewDNSService(s, registrar)
	renewals := NewRenewalService(s, registrar, pricing)
	transfers := NewTransferService(s, registrar, pricing, encryptionKey)
	reconciliation := NewReconciliationWorker(s, registrar)
	notifications := NewExpiryNotifier(s)

	return &Service{
		Registrar:      registrar,
		Pricing:        pricing,
		Availability:   availability,
		Orders:         orders,
		Provisioning:   provisioning,
		Nameservers:    nameservers,
		DNS:            dns,
		Renewals:       renewals,
		Transfers:      transfers,
		Reconciliation: reconciliation,
		Notifications:  notifications,
		Store:          s,
	}
}
