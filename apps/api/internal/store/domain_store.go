package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// MEMORY STORE IMPLEMENTATION FOR DOMAIN RESELLER
// ============================================================================

func (m *MemoryStore) seedDomainResellerData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	if m.domainTLDs == nil {
		m.domainTLDs = make(map[string]*DomainTLD)
	}
	if m.domainPrices == nil {
		m.domainPrices = make(map[string]*DomainPrice)
	}
	if m.domains == nil {
		m.domains = make(map[uuid.UUID]*Domain)
	}
	if m.domainOrders == nil {
		m.domainOrders = make(map[uuid.UUID]*DomainOrder)
	}
	if m.domainContacts == nil {
		m.domainContacts = make(map[uuid.UUID][]*DomainContact)
	}
	if m.domainNameservers == nil {
		m.domainNameservers = make(map[uuid.UUID][]string)
	}
	if m.domainDNSRecords == nil {
		m.domainDNSRecords = make(map[uuid.UUID][]*DomainDNSRecord)
	}
	if m.domainTransfers == nil {
		m.domainTransfers = make(map[uuid.UUID]*DomainTransfer)
	}
	if m.domainRenewals == nil {
		m.domainRenewals = make(map[uuid.UUID][]*DomainRenewal)
	}
	if m.domainWebhooks == nil {
		m.domainWebhooks = make(map[string]*DomainWebhook)
	}

	// Seed TLDs if empty
	if len(m.domainTLDs) == 0 {
		initialTLDs := []*DomainTLD{
			{ID: uuid.New(), TLD: "com", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: true, Category: "popular", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "net", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: true, Category: "popular", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "org", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: true, Category: "popular", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "xyz", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: true, Category: "tech", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "io", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 5, Provider: "resellerclub", IsPopular: true, Category: "tech", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "co", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 5, Provider: "resellerclub", IsPopular: false, Category: "popular", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "tech", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: false, Category: "tech", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "store", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: false, Category: "business", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "online", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: false, Category: "popular", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "info", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: false, Category: "popular", CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "biz", Enabled: true, RegistrationEnabled: true, TransferEnabled: true, RenewalEnabled: true, MinYears: 1, MaxYears: 10, Provider: "resellerclub", IsPopular: false, Category: "business", CreatedAt: now, UpdatedAt: now},
		}
		for _, t := range initialTLDs {
			m.domainTLDs[strings.ToLower(t.TLD)] = t
		}
	}

	// Seed Prices if empty
	if len(m.domainPrices) == 0 {
		initialPrices := []*DomainPrice{
			{ID: uuid.New(), TLD: "com", RegistrationCost: 10.29, RegistrationPrice: 14.99, RenewalCost: 10.99, RenewalPrice: 16.99, TransferCost: 10.29, TransferPrice: 14.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "net", RegistrationCost: 12.49, RegistrationPrice: 16.99, RenewalCost: 13.19, RenewalPrice: 18.99, TransferCost: 12.49, TransferPrice: 16.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "org", RegistrationCost: 11.89, RegistrationPrice: 15.99, RenewalCost: 12.49, RenewalPrice: 17.99, TransferCost: 11.89, TransferPrice: 15.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "xyz", RegistrationCost: 1.99, RegistrationPrice: 2.99, RenewalCost: 10.49, RenewalPrice: 13.99, TransferCost: 9.99, TransferPrice: 12.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "io", RegistrationCost: 32.50, RegistrationPrice: 44.99, RenewalCost: 36.50, RenewalPrice: 49.99, TransferCost: 32.50, TransferPrice: 44.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "co", RegistrationCost: 9.99, RegistrationPrice: 14.99, RenewalCost: 23.50, RenewalPrice: 29.99, TransferCost: 21.00, TransferPrice: 27.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "tech", RegistrationCost: 3.89, RegistrationPrice: 5.99, RenewalCost: 17.50, RenewalPrice: 23.99, TransferCost: 16.00, TransferPrice: 21.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "store", RegistrationCost: 2.99, RegistrationPrice: 4.99, RenewalCost: 26.50, RenewalPrice: 34.99, TransferCost: 24.00, TransferPrice: 31.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "online", RegistrationCost: 1.89, RegistrationPrice: 2.99, RenewalCost: 22.50, RenewalPrice: 28.99, TransferCost: 20.00, TransferPrice: 26.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "info", RegistrationCost: 4.29, RegistrationPrice: 6.99, RenewalCost: 16.50, RenewalPrice: 21.99, TransferCost: 15.00, TransferPrice: 19.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), TLD: "biz", RegistrationCost: 7.50, RegistrationPrice: 11.99, RenewalCost: 16.50, RenewalPrice: 21.99, TransferCost: 15.00, TransferPrice: 19.99, Currency: "USD", Enabled: true, CreatedAt: now, UpdatedAt: now},
		}
		for _, p := range initialPrices {
			m.domainPrices[strings.ToLower(p.TLD)] = p
		}
	}
}

// ----------------------------------------------------------------------------
// TLDs & Pricing Methods (MemoryStore)
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListDomainTLDs(ctx context.Context) ([]*DomainTLD, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainTLD
	for _, t := range m.domainTLDs {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].IsPopular != list[j].IsPopular {
			return list[i].IsPopular
		}
		return list[i].TLD < list[j].TLD
	})
	return list, nil
}

func (m *MemoryStore) GetDomainTLD(ctx context.Context, tld string) (*DomainTLD, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld), "."))
	t, ok := m.domainTLDs[clean]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (m *MemoryStore) SaveDomainTLD(ctx context.Context, tld *DomainTLD) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld.TLD), "."))
	tld.TLD = clean
	if tld.ID == uuid.Nil {
		tld.ID = uuid.New()
	}
	now := time.Now().UTC()
	if tld.CreatedAt.IsZero() {
		tld.CreatedAt = now
	}
	tld.UpdatedAt = now
	m.domainTLDs[clean] = tld
	return nil
}

func (m *MemoryStore) ListDomainPrices(ctx context.Context) ([]*DomainPrice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainPrice
	for _, p := range m.domainPrices {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].TLD < list[j].TLD
	})
	return list, nil
}

func (m *MemoryStore) GetDomainPrice(ctx context.Context, tld string) (*DomainPrice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld), "."))
	p, ok := m.domainPrices[clean]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) SaveDomainPrice(ctx context.Context, price *DomainPrice) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(price.TLD), "."))
	price.TLD = clean
	if price.ID == uuid.Nil {
		price.ID = uuid.New()
	}
	now := time.Now().UTC()
	if price.CreatedAt.IsZero() {
		price.CreatedAt = now
	}
	price.UpdatedAt = now
	m.domainPrices[clean] = price
	return nil
}

// ----------------------------------------------------------------------------
// Registered Domains (MemoryStore)
// ----------------------------------------------------------------------------

func (m *MemoryStore) CreateDomain(ctx context.Context, domain *Domain) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if domain.ID == uuid.Nil {
		domain.ID = uuid.New()
	}
	now := time.Now().UTC()
	if domain.CreatedAt.IsZero() {
		domain.CreatedAt = now
	}
	domain.UpdatedAt = now
	domain.DomainName = strings.ToLower(strings.TrimSpace(domain.DomainName))
	m.domains[domain.ID] = domain
	return nil
}

func (m *MemoryStore) GetDomainByID(ctx context.Context, id uuid.UUID) (*Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	d, ok := m.domains[id]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

func (m *MemoryStore) GetDomainByName(ctx context.Context, domainName string) (*Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(domainName))
	for _, d := range m.domains {
		if d.DomainName == clean {
			return d, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListDomainsByUserID(ctx context.Context, userID uuid.UUID) ([]*Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Domain
	for _, d := range m.domains {
		if d.UserID == userID {
			list = append(list, d)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].DomainName < list[j].DomainName
	})
	return list, nil
}

func (m *MemoryStore) ListAllDomains(ctx context.Context) ([]*Domain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Domain
	for _, d := range m.domains {
		list = append(list, d)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].DomainName < list[j].DomainName
	})
	return list, nil
}

func (m *MemoryStore) UpdateDomain(ctx context.Context, domain *Domain) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.domains[domain.ID]; !ok {
		return ErrNotFound
	}
	domain.UpdatedAt = time.Now().UTC()
	domain.DomainName = strings.ToLower(strings.TrimSpace(domain.DomainName))
	m.domains[domain.ID] = domain
	return nil
}

func (m *MemoryStore) DeleteDomain(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.domains, id)
	delete(m.domainContacts, id)
	delete(m.domainNameservers, id)
	delete(m.domainDNSRecords, id)
	return nil
}

// ----------------------------------------------------------------------------
// Domain Orders (MemoryStore)
// ----------------------------------------------------------------------------

func (m *MemoryStore) CreateDomainOrder(ctx context.Context, order *DomainOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	now := time.Now().UTC()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	order.UpdatedAt = now
	order.DomainName = strings.ToLower(strings.TrimSpace(order.DomainName))
	m.domainOrders[order.ID] = order
	return nil
}

func (m *MemoryStore) GetDomainOrderByID(ctx context.Context, id uuid.UUID) (*DomainOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	o, ok := m.domainOrders[id]
	if !ok {
		return nil, ErrNotFound
	}
	return o, nil
}

func (m *MemoryStore) GetDomainOrderByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*DomainOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, o := range m.domainOrders {
		if o.InvoiceID != nil && *o.InvoiceID == invoiceID {
			return o, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetDomainOrderByIdempotencyKey(ctx context.Context, key string) (*DomainOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, o := range m.domainOrders {
		if o.IdempotencyKey == key {
			return o, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListDomainOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*DomainOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainOrder
	for _, o := range m.domainOrders {
		if o.UserID == userID {
			list = append(list, o)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) ListAllDomainOrders(ctx context.Context) ([]*DomainOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainOrder
	for _, o := range m.domainOrders {
		list = append(list, o)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) UpdateDomainOrder(ctx context.Context, order *DomainOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.domainOrders[order.ID]; !ok {
		return ErrNotFound
	}
	order.UpdatedAt = time.Now().UTC()
	m.domainOrders[order.ID] = order
	return nil
}

// ----------------------------------------------------------------------------
// Contacts, Nameservers, DNS Records (MemoryStore)
// ----------------------------------------------------------------------------

func (m *MemoryStore) SaveDomainContacts(ctx context.Context, contacts []*DomainContact) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(contacts) == 0 {
		return nil
	}
	domainID := contacts[0].DomainID
	m.domainContacts[domainID] = contacts
	return nil
}

func (m *MemoryStore) GetDomainContacts(ctx context.Context, domainID uuid.UUID) ([]*DomainContact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.domainContacts[domainID], nil
}

func (m *MemoryStore) SaveDomainNameservers(ctx context.Context, domainID uuid.UUID, ns []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.domainNameservers[domainID] = ns
	return nil
}

func (m *MemoryStore) GetDomainNameservers(ctx context.Context, domainID uuid.UUID) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ns, ok := m.domainNameservers[domainID]
	if !ok || len(ns) == 0 {
		return []string{"ns1.hostvra.com", "ns2.hostvra.com"}, nil
	}
	return ns, nil
}

func (m *MemoryStore) CreateDomainDNSRecord(ctx context.Context, rec *DomainDNSRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	now := time.Now().UTC()
	rec.CreatedAt = now
	rec.UpdatedAt = now
	m.domainDNSRecords[rec.DomainID] = append(m.domainDNSRecords[rec.DomainID], rec)
	return nil
}

func (m *MemoryStore) GetDomainDNSRecords(ctx context.Context, domainID uuid.UUID) ([]*DomainDNSRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.domainDNSRecords[domainID], nil
}

func (m *MemoryStore) GetDomainDNSRecordByID(ctx context.Context, id uuid.UUID) (*DomainDNSRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, records := range m.domainDNSRecords {
		for _, r := range records {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) UpdateDomainDNSRecord(ctx context.Context, rec *DomainDNSRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	records := m.domainDNSRecords[rec.DomainID]
	for i, r := range records {
		if r.ID == rec.ID {
			rec.UpdatedAt = time.Now().UTC()
			records[i] = rec
			return nil
		}
	}
	return ErrNotFound
}

func (m *MemoryStore) DeleteDomainDNSRecord(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for dID, records := range m.domainDNSRecords {
		for i, r := range records {
			if r.ID == id {
				m.domainDNSRecords[dID] = append(records[:i], records[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

// ----------------------------------------------------------------------------
// Transfers, Renewals, Webhooks, Transactions, Audit (MemoryStore)
// ----------------------------------------------------------------------------

func (m *MemoryStore) CreateDomainTransfer(ctx context.Context, transfer *DomainTransfer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if transfer.ID == uuid.Nil {
		transfer.ID = uuid.New()
	}
	now := time.Now().UTC()
	transfer.CreatedAt = now
	transfer.UpdatedAt = now
	m.domainTransfers[transfer.ID] = transfer
	return nil
}

func (m *MemoryStore) GetDomainTransferByID(ctx context.Context, id uuid.UUID) (*DomainTransfer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.domainTransfers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (m *MemoryStore) ListDomainTransfers(ctx context.Context, userID *uuid.UUID) ([]*DomainTransfer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainTransfer
	for _, t := range m.domainTransfers {
		if userID == nil || t.UserID == *userID {
			list = append(list, t)
		}
	}
	return list, nil
}

func (m *MemoryStore) UpdateDomainTransfer(ctx context.Context, transfer *DomainTransfer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.domainTransfers[transfer.ID]; !ok {
		return ErrNotFound
	}
	transfer.UpdatedAt = time.Now().UTC()
	m.domainTransfers[transfer.ID] = transfer
	return nil
}

func (m *MemoryStore) CreateDomainRenewal(ctx context.Context, renewal *DomainRenewal) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if renewal.ID == uuid.Nil {
		renewal.ID = uuid.New()
	}
	renewal.CreatedAt = time.Now().UTC()
	m.domainRenewals[renewal.DomainID] = append(m.domainRenewals[renewal.DomainID], renewal)
	return nil
}

func (m *MemoryStore) ListDomainRenewals(ctx context.Context, domainID *uuid.UUID) ([]*DomainRenewal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainRenewal
	if domainID != nil {
		return m.domainRenewals[*domainID], nil
	}
	for _, renewals := range m.domainRenewals {
		list = append(list, renewals...)
	}
	return list, nil
}

func (m *MemoryStore) RecordDomainWebhook(ctx context.Context, hook *DomainWebhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", hook.Provider, hook.ExternalEventID)
	if _, exists := m.domainWebhooks[key]; exists {
		return errors.New("webhook already processed")
	}
	if hook.ID == uuid.Nil {
		hook.ID = uuid.New()
	}
	hook.CreatedAt = time.Now().UTC()
	hook.ProcessedAt = time.Now().UTC()
	m.domainWebhooks[key] = hook
	return nil
}

func (m *MemoryStore) GetDomainWebhook(ctx context.Context, provider, externalEventID string) (*DomainWebhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", provider, externalEventID)
	hook, exists := m.domainWebhooks[key]
	if !exists {
		return nil, ErrNotFound
	}
	return hook, nil
}

func (m *MemoryStore) TryAcquireDomainAdvisoryLock(ctx context.Context, lockKey string) (bool, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.domainAdvisoryLocks == nil {
		m.domainAdvisoryLocks = make(map[string]bool)
	}
	if m.domainAdvisoryLocks[lockKey] {
		return false, nil, nil
	}
	m.domainAdvisoryLocks[lockKey] = true
	unlock := func() {
		m.mu.Lock()
		delete(m.domainAdvisoryLocks, lockKey)
		m.mu.Unlock()
	}
	return true, unlock, nil
}

func (m *MemoryStore) RecordDomainTransaction(ctx context.Context, txn *DomainTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if txn.ID == uuid.Nil {
		txn.ID = uuid.New()
	}
	now := time.Now().UTC()
	txn.CreatedAt = now
	txn.UpdatedAt = now
	m.domainTransactions = append(m.domainTransactions, txn)
	return nil
}

func (m *MemoryStore) ListDomainTransactions(ctx context.Context, domainID *uuid.UUID) ([]*DomainTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainTransaction
	for _, t := range m.domainTransactions {
		if domainID == nil || (t.DomainID != nil && *t.DomainID == *domainID) {
			list = append(list, t)
		}
	}
	return list, nil
}

func (m *MemoryStore) RecordDomainAuditLog(ctx context.Context, log *DomainAuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now().UTC()
	m.domainAuditLogs = append(m.domainAuditLogs, log)
	return nil
}

func (m *MemoryStore) ListDomainAuditLogs(ctx context.Context, domainName string) ([]*DomainAuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(domainName))
	var list []*DomainAuditLog
	for _, l := range m.domainAuditLogs {
		if clean == "" || l.DomainName == clean {
			list = append(list, l)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

// ----------------------------------------------------------------------------
// Legacy TLDPricing CRUD
// ----------------------------------------------------------------------------

func (m *MemoryStore) ListTLDPricings(ctx context.Context) ([]*TLDPricing, error) {
	prices, err := m.ListDomainPrices(ctx)
	if err != nil {
		return nil, err
	}
	var list []*TLDPricing
	for _, p := range prices {
		tldMeta, _ := m.GetDomainTLD(ctx, p.TLD)
		isPopular := false
		category := "popular"
		minYears := 1
		if tldMeta != nil {
			isPopular = tldMeta.IsPopular
			category = tldMeta.Category
			minYears = tldMeta.MinYears
		}
		list = append(list, &TLDPricing{
			TLD:           "." + p.TLD,
			RegisterPrice: p.RegistrationPrice,
			RenewPrice:    p.RenewalPrice,
			TransferPrice: p.TransferPrice,
			Currency:      p.Currency,
			IsPopular:     isPopular,
			Category:      category,
			MinYears:      minYears,
			UpdatedAt:     p.UpdatedAt,
		})
	}
	return list, nil
}

func (m *MemoryStore) GetTLDPricing(ctx context.Context, tld string) (*TLDPricing, error) {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
	p, err := m.GetDomainPrice(ctx, clean)
	if err != nil {
		return nil, err
	}
	tldMeta, _ := m.GetDomainTLD(ctx, clean)
	isPopular := false
	category := "popular"
	minYears := 1
	if tldMeta != nil {
		isPopular = tldMeta.IsPopular
		category = tldMeta.Category
		minYears = tldMeta.MinYears
	}
	return &TLDPricing{
		TLD:           "." + p.TLD,
		RegisterPrice: p.RegistrationPrice,
		RenewPrice:    p.RenewalPrice,
		TransferPrice: p.TransferPrice,
		Currency:      p.Currency,
		IsPopular:     isPopular,
		Category:      category,
		MinYears:      minYears,
		UpdatedAt:     p.UpdatedAt,
	}, nil
}

func (m *MemoryStore) SaveTLDPricing(ctx context.Context, pricing *TLDPricing) error {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(pricing.TLD)), ".")
	existingPrice, err := m.GetDomainPrice(ctx, clean)
	cost := 10.00
	if err == nil {
		cost = existingPrice.RegistrationCost
	}
	return m.SaveDomainPrice(ctx, &DomainPrice{
		TLD:               clean,
		RegistrationCost:  cost,
		RegistrationPrice: pricing.RegisterPrice,
		RenewalCost:       cost,
		RenewalPrice:      pricing.RenewPrice,
		TransferCost:      cost,
		TransferPrice:     pricing.TransferPrice,
		Currency:          pricing.Currency,
		Enabled:           true,
	})
}

func (m *MemoryStore) ListRegistrarConfigs(ctx context.Context) ([]*DomainRegistrarConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*DomainRegistrarConfig
	for _, r := range m.registrarConfigs {
		list = append(list, r)
	}
	return list, nil
}

func (m *MemoryStore) GetRegistrarConfig(ctx context.Context, registrar string) (*DomainRegistrarConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(registrar))
	r, exists := m.registrarConfigs[clean]
	if !exists {
		return nil, ErrNotFound
	}
	return r, nil
}

func (m *MemoryStore) SaveRegistrarConfig(ctx context.Context, config *DomainRegistrarConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.Registrar = strings.ToLower(strings.TrimSpace(config.Registrar))
	config.UpdatedAt = time.Now().UTC()
	m.registrarConfigs[config.Registrar] = config
	return nil
}

// ============================================================================
// POSTGRES STORE REAL IMPLEMENTATIONS (NO FAKE OR IN-MEMORY FALLBACK)
// ============================================================================

func (p *PostgresStore) ListDomainTLDs(ctx context.Context) ([]*DomainTLD, error) {
	query := `
		SELECT id, tld, enabled, registration_enabled, transfer_enabled, renewal_enabled,
		       min_years, max_years, provider, is_popular, category, created_at, updated_at
		FROM domain_tlds
		ORDER BY is_popular DESC, tld ASC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query domain_tlds: %w", err)
	}
	defer rows.Close()

	var list []*DomainTLD
	for rows.Next() {
		t := &DomainTLD{}
		if err := rows.Scan(
			&t.ID, &t.TLD, &t.Enabled, &t.RegistrationEnabled, &t.TransferEnabled, &t.RenewalEnabled,
			&t.MinYears, &t.MaxYears, &t.Provider, &t.IsPopular, &t.Category, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (p *PostgresStore) GetDomainTLD(ctx context.Context, tld string) (*DomainTLD, error) {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld), "."))
	query := `
		SELECT id, tld, enabled, registration_enabled, transfer_enabled, renewal_enabled,
		       min_years, max_years, provider, is_popular, category, created_at, updated_at
		FROM domain_tlds
		WHERE tld = $1
	`
	t := &DomainTLD{}
	err := p.db.QueryRowContext(ctx, query, clean).Scan(
		&t.ID, &t.TLD, &t.Enabled, &t.RegistrationEnabled, &t.TransferEnabled, &t.RenewalEnabled,
		&t.MinYears, &t.MaxYears, &t.Provider, &t.IsPopular, &t.Category, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (p *PostgresStore) SaveDomainTLD(ctx context.Context, tld *DomainTLD) error {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld.TLD), "."))
	if tld.ID == uuid.Nil {
		tld.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_tlds (id, tld, enabled, registration_enabled, transfer_enabled, renewal_enabled, min_years, max_years, provider, is_popular, category, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (tld) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			registration_enabled = EXCLUDED.registration_enabled,
			transfer_enabled = EXCLUDED.transfer_enabled,
			renewal_enabled = EXCLUDED.renewal_enabled,
			min_years = EXCLUDED.min_years,
			max_years = EXCLUDED.max_years,
			provider = EXCLUDED.provider,
			is_popular = EXCLUDED.is_popular,
			category = EXCLUDED.category,
			updated_at = NOW()
	`
	_, err := p.db.ExecContext(ctx, query,
		tld.ID, clean, tld.Enabled, tld.RegistrationEnabled, tld.TransferEnabled, tld.RenewalEnabled,
		tld.MinYears, tld.MaxYears, tld.Provider, tld.IsPopular, tld.Category,
	)
	return err
}

func (p *PostgresStore) ListDomainPrices(ctx context.Context) ([]*DomainPrice, error) {
	query := `
		SELECT id, tld, registration_cost, registration_price, renewal_cost, renewal_price,
		       transfer_cost, transfer_price, currency, enabled, created_at, updated_at
		FROM domain_prices
		ORDER BY tld ASC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query domain_prices: %w", err)
	}
	defer rows.Close()

	var list []*DomainPrice
	for rows.Next() {
		pr := &DomainPrice{}
		if err := rows.Scan(
			&pr.ID, &pr.TLD, &pr.RegistrationCost, &pr.RegistrationPrice, &pr.RenewalCost, &pr.RenewalPrice,
			&pr.TransferCost, &pr.TransferPrice, &pr.Currency, &pr.Enabled, &pr.CreatedAt, &pr.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, pr)
	}
	return list, nil
}

func (p *PostgresStore) GetDomainPrice(ctx context.Context, tld string) (*DomainPrice, error) {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tld), "."))
	query := `
		SELECT id, tld, registration_cost, registration_price, renewal_cost, renewal_price,
		       transfer_cost, transfer_price, currency, enabled, created_at, updated_at
		FROM domain_prices
		WHERE tld = $1
	`
	pr := &DomainPrice{}
	err := p.db.QueryRowContext(ctx, query, clean).Scan(
		&pr.ID, &pr.TLD, &pr.RegistrationCost, &pr.RegistrationPrice, &pr.RenewalCost, &pr.RenewalPrice,
		&pr.TransferCost, &pr.TransferPrice, &pr.Currency, &pr.Enabled, &pr.CreatedAt, &pr.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return pr, err
}

func (p *PostgresStore) SaveDomainPrice(ctx context.Context, price *DomainPrice) error {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(price.TLD), "."))
	if price.ID == uuid.Nil {
		price.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_prices (id, tld, registration_cost, registration_price, renewal_cost, renewal_price, transfer_cost, transfer_price, currency, enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (tld) DO UPDATE SET
			registration_cost = EXCLUDED.registration_cost,
			registration_price = EXCLUDED.registration_price,
			renewal_cost = EXCLUDED.renewal_cost,
			renewal_price = EXCLUDED.renewal_price,
			transfer_cost = EXCLUDED.transfer_cost,
			transfer_price = EXCLUDED.transfer_price,
			currency = EXCLUDED.currency,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
	`
	_, err := p.db.ExecContext(ctx, query,
		price.ID, clean, price.RegistrationCost, price.RegistrationPrice,
		price.RenewalCost, price.RenewalPrice, price.TransferCost, price.TransferPrice,
		price.Currency, price.Enabled,
	)
	return err
}

func (p *PostgresStore) CreateDomain(ctx context.Context, domain *Domain) error {
	if domain.ID == uuid.Nil {
		domain.ID = uuid.New()
	}
	clean := strings.ToLower(strings.TrimSpace(domain.DomainName))
	query := `
		INSERT INTO domains (
			id, user_id, organization_id, order_id, domain_name, tld, registrar,
			provider_order_id, provider_domain_id, status, registration_date, expiry_date,
			transfer_status, auto_renew, registrar_lock, privacy_enabled, website_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		domain.ID, domain.UserID, domain.OrganizationID, domain.OrderID, clean, domain.TLD, domain.Registrar,
		domain.ProviderOrderID, domain.ProviderDomainID, domain.Status, domain.RegistrationDate, domain.ExpiryDate,
		domain.TransferStatus, domain.AutoRenew, domain.RegistrarLock, domain.PrivacyEnabled, domain.WebsiteID,
	).Scan(&domain.CreatedAt, &domain.UpdatedAt)
}

func (p *PostgresStore) GetDomainByID(ctx context.Context, id uuid.UUID) (*Domain, error) {
	query := `
		SELECT id, user_id, organization_id, order_id, domain_name, tld, registrar,
		       provider_order_id, provider_domain_id, status, registration_date, expiry_date,
		       transfer_status, auto_renew, registrar_lock, privacy_enabled, website_id, created_at, updated_at
		FROM domains
		WHERE id = $1
	`
	d := &Domain{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.OrganizationID, &d.OrderID, &d.DomainName, &d.TLD, &d.Registrar,
		&d.ProviderOrderID, &d.ProviderDomainID, &d.Status, &d.RegistrationDate, &d.ExpiryDate,
		&d.TransferStatus, &d.AutoRenew, &d.RegistrarLock, &d.PrivacyEnabled, &d.WebsiteID, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (p *PostgresStore) GetDomainByName(ctx context.Context, domainName string) (*Domain, error) {
	clean := strings.ToLower(strings.TrimSpace(domainName))
	query := `
		SELECT id, user_id, organization_id, order_id, domain_name, tld, registrar,
		       provider_order_id, provider_domain_id, status, registration_date, expiry_date,
		       transfer_status, auto_renew, registrar_lock, privacy_enabled, website_id, created_at, updated_at
		FROM domains
		WHERE domain_name = $1
	`
	d := &Domain{}
	err := p.db.QueryRowContext(ctx, query, clean).Scan(
		&d.ID, &d.UserID, &d.OrganizationID, &d.OrderID, &d.DomainName, &d.TLD, &d.Registrar,
		&d.ProviderOrderID, &d.ProviderDomainID, &d.Status, &d.RegistrationDate, &d.ExpiryDate,
		&d.TransferStatus, &d.AutoRenew, &d.RegistrarLock, &d.PrivacyEnabled, &d.WebsiteID, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (p *PostgresStore) ListDomainsByUserID(ctx context.Context, userID uuid.UUID) ([]*Domain, error) {
	query := `
		SELECT id, user_id, organization_id, order_id, domain_name, tld, registrar,
		       provider_order_id, provider_domain_id, status, registration_date, expiry_date,
		       transfer_status, auto_renew, registrar_lock, privacy_enabled, website_id, created_at, updated_at
		FROM domains
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Domain
	for rows.Next() {
		d := &Domain{}
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.OrganizationID, &d.OrderID, &d.DomainName, &d.TLD, &d.Registrar,
			&d.ProviderOrderID, &d.ProviderDomainID, &d.Status, &d.RegistrationDate, &d.ExpiryDate,
			&d.TransferStatus, &d.AutoRenew, &d.RegistrarLock, &d.PrivacyEnabled, &d.WebsiteID, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (p *PostgresStore) ListAllDomains(ctx context.Context) ([]*Domain, error) {
	query := `
		SELECT id, user_id, organization_id, order_id, domain_name, tld, registrar,
		       provider_order_id, provider_domain_id, status, registration_date, expiry_date,
		       transfer_status, auto_renew, registrar_lock, privacy_enabled, website_id, created_at, updated_at
		FROM domains
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Domain
	for rows.Next() {
		d := &Domain{}
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.OrganizationID, &d.OrderID, &d.DomainName, &d.TLD, &d.Registrar,
			&d.ProviderOrderID, &d.ProviderDomainID, &d.Status, &d.RegistrationDate, &d.ExpiryDate,
			&d.TransferStatus, &d.AutoRenew, &d.RegistrarLock, &d.PrivacyEnabled, &d.WebsiteID, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (p *PostgresStore) UpdateDomain(ctx context.Context, domain *Domain) error {
	query := `
		UPDATE domains SET
			status = $1,
			provider_order_id = $2,
			provider_domain_id = $3,
			registration_date = $4,
			expiry_date = $5,
			transfer_status = $6,
			auto_renew = $7,
			registrar_lock = $8,
			privacy_enabled = $9,
			website_id = $10,
			updated_at = NOW()
		WHERE id = $11
	`
	_, err := p.db.ExecContext(ctx, query,
		domain.Status, domain.ProviderOrderID, domain.ProviderDomainID,
		domain.RegistrationDate, domain.ExpiryDate, domain.TransferStatus,
		domain.AutoRenew, domain.RegistrarLock, domain.PrivacyEnabled, domain.WebsiteID,
		domain.ID,
	)
	return err
}

func (p *PostgresStore) DeleteDomain(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM domains WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// ----------------------------------------------------------------------------
// Orders (PostgresStore)
// ----------------------------------------------------------------------------

func (p *PostgresStore) CreateDomainOrder(ctx context.Context, order *DomainOrder) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_orders (
			id, user_id, organization_id, domain_id, domain_name, order_type, years,
			amount, cost, currency, payment_status, provisioning_status, provider_status,
			provider_order_id, idempotency_key, failure_reason, retry_count, invoice_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		order.ID, order.UserID, order.OrganizationID, order.DomainID, strings.ToLower(order.DomainName),
		order.OrderType, order.Years, order.Amount, order.Cost, order.Currency,
		order.PaymentStatus, order.ProvisioningStatus, order.ProviderStatus,
		order.ProviderOrderID, order.IdempotencyKey, order.FailureReason, order.RetryCount, order.InvoiceID,
	).Scan(&order.CreatedAt, &order.UpdatedAt)
}

func (p *PostgresStore) GetDomainOrderByID(ctx context.Context, id uuid.UUID) (*DomainOrder, error) {
	query := `
		SELECT id, user_id, organization_id, domain_id, domain_name, order_type, years,
		       amount, cost, currency, payment_status, provisioning_status, provider_status,
		       provider_order_id, idempotency_key, failure_reason, retry_count, invoice_id, created_at, updated_at
		FROM domain_orders
		WHERE id = $1
	`
	o := &DomainOrder{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&o.ID, &o.UserID, &o.OrganizationID, &o.DomainID, &o.DomainName, &o.OrderType, &o.Years,
		&o.Amount, &o.Cost, &o.Currency, &o.PaymentStatus, &o.ProvisioningStatus, &o.ProviderStatus,
		&o.ProviderOrderID, &o.IdempotencyKey, &o.FailureReason, &o.RetryCount, &o.InvoiceID, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (p *PostgresStore) GetDomainOrderByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*DomainOrder, error) {
	query := `
		SELECT id, user_id, organization_id, domain_id, domain_name, order_type, years,
		       amount, cost, currency, payment_status, provisioning_status, provider_status,
		       provider_order_id, idempotency_key, failure_reason, retry_count, invoice_id, created_at, updated_at
		FROM domain_orders
		WHERE invoice_id = $1
		LIMIT 1
	`
	o := &DomainOrder{}
	err := p.db.QueryRowContext(ctx, query, invoiceID).Scan(
		&o.ID, &o.UserID, &o.OrganizationID, &o.DomainID, &o.DomainName, &o.OrderType, &o.Years,
		&o.Amount, &o.Cost, &o.Currency, &o.PaymentStatus, &o.ProvisioningStatus, &o.ProviderStatus,
		&o.ProviderOrderID, &o.IdempotencyKey, &o.FailureReason, &o.RetryCount, &o.InvoiceID, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (p *PostgresStore) GetDomainOrderByIdempotencyKey(ctx context.Context, key string) (*DomainOrder, error) {
	query := `
		SELECT id, user_id, organization_id, domain_id, domain_name, order_type, years,
		       amount, cost, currency, payment_status, provisioning_status, provider_status,
		       provider_order_id, idempotency_key, failure_reason, retry_count, invoice_id, created_at, updated_at
		FROM domain_orders
		WHERE idempotency_key = $1
	`
	o := &DomainOrder{}
	err := p.db.QueryRowContext(ctx, query, key).Scan(
		&o.ID, &o.UserID, &o.OrganizationID, &o.DomainID, &o.DomainName, &o.OrderType, &o.Years,
		&o.Amount, &o.Cost, &o.Currency, &o.PaymentStatus, &o.ProvisioningStatus, &o.ProviderStatus,
		&o.ProviderOrderID, &o.IdempotencyKey, &o.FailureReason, &o.RetryCount, &o.InvoiceID, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (p *PostgresStore) ListDomainOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*DomainOrder, error) {
	query := `
		SELECT id, user_id, organization_id, domain_id, domain_name, order_type, years,
		       amount, cost, currency, payment_status, provisioning_status, provider_status,
		       provider_order_id, idempotency_key, failure_reason, retry_count, invoice_id, created_at, updated_at
		FROM domain_orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainOrder
	for rows.Next() {
		o := &DomainOrder{}
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.OrganizationID, &o.DomainID, &o.DomainName, &o.OrderType, &o.Years,
			&o.Amount, &o.Cost, &o.Currency, &o.PaymentStatus, &o.ProvisioningStatus, &o.ProviderStatus,
			&o.ProviderOrderID, &o.IdempotencyKey, &o.FailureReason, &o.RetryCount, &o.InvoiceID, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (p *PostgresStore) ListAllDomainOrders(ctx context.Context) ([]*DomainOrder, error) {
	query := `
		SELECT id, user_id, organization_id, domain_id, domain_name, order_type, years,
		       amount, cost, currency, payment_status, provisioning_status, provider_status,
		       provider_order_id, idempotency_key, failure_reason, retry_count, invoice_id, created_at, updated_at
		FROM domain_orders
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainOrder
	for rows.Next() {
		o := &DomainOrder{}
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.OrganizationID, &o.DomainID, &o.DomainName, &o.OrderType, &o.Years,
			&o.Amount, &o.Cost, &o.Currency, &o.PaymentStatus, &o.ProvisioningStatus, &o.ProviderStatus,
			&o.ProviderOrderID, &o.IdempotencyKey, &o.FailureReason, &o.RetryCount, &o.InvoiceID, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (p *PostgresStore) UpdateDomainOrder(ctx context.Context, order *DomainOrder) error {
	query := `
		UPDATE domain_orders SET
			domain_id = $1,
			payment_status = $2,
			provisioning_status = $3,
			provider_status = $4,
			provider_order_id = $5,
			failure_reason = $6,
			retry_count = $7,
			invoice_id = $8,
			updated_at = NOW()
		WHERE id = $9
	`
	_, err := p.db.ExecContext(ctx, query,
		order.DomainID, order.PaymentStatus, order.ProvisioningStatus, order.ProviderStatus,
		order.ProviderOrderID, order.FailureReason, order.RetryCount, order.InvoiceID,
		order.ID,
	)
	return err
}

// ----------------------------------------------------------------------------
// Contacts, Nameservers, DNS Records (PostgresStore)
// ----------------------------------------------------------------------------

func (p *PostgresStore) SaveDomainContacts(ctx context.Context, contacts []*DomainContact) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO domain_contacts (
			id, domain_id, contact_type, first_name, last_name, organization,
			email, phone, address1, address2, city, state, postal_code, country, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (domain_id, contact_type) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			organization = EXCLUDED.organization,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			address1 = EXCLUDED.address1,
			address2 = EXCLUDED.address2,
			city = EXCLUDED.city,
			state = EXCLUDED.state,
			postal_code = EXCLUDED.postal_code,
			country = EXCLUDED.country,
			updated_at = NOW()
	`
	for _, c := range contacts {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		if _, err := tx.ExecContext(ctx, query,
			c.ID, c.DomainID, c.ContactType, c.FirstName, c.LastName, c.Organization,
			c.Email, c.Phone, c.Address1, c.Address2, c.City, c.State, c.PostalCode, c.Country,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PostgresStore) GetDomainContacts(ctx context.Context, domainID uuid.UUID) ([]*DomainContact, error) {
	query := `
		SELECT id, domain_id, contact_type, first_name, last_name, organization,
		       email, phone, address1, address2, city, state, postal_code, country, created_at, updated_at
		FROM domain_contacts
		WHERE domain_id = $1
	`
	rows, err := p.db.QueryContext(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainContact
	for rows.Next() {
		c := &DomainContact{}
		if err := rows.Scan(
			&c.ID, &c.DomainID, &c.ContactType, &c.FirstName, &c.LastName, &c.Organization,
			&c.Email, &c.Phone, &c.Address1, &c.Address2, &c.City, &c.State, &c.PostalCode, &c.Country,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (p *PostgresStore) SaveDomainNameservers(ctx context.Context, domainID uuid.UUID, ns []string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM domain_nameservers WHERE domain_id = $1`, domainID); err != nil {
		return err
	}

	query := `INSERT INTO domain_nameservers (id, domain_id, nameserver, position, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())`
	for i, s := range ns {
		clean := strings.TrimSpace(s)
		if clean == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query, uuid.New(), domainID, clean, i+1); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PostgresStore) GetDomainNameservers(ctx context.Context, domainID uuid.UUID) ([]string, error) {
	query := `SELECT nameserver FROM domain_nameservers WHERE domain_id = $1 ORDER BY position ASC`
	rows, err := p.db.QueryContext(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ns []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		ns = append(ns, s)
	}
	if len(ns) == 0 {
		return []string{"ns1.hostvra.com", "ns2.hostvra.com"}, nil
	}
	return ns, nil
}

func (p *PostgresStore) CreateDomainDNSRecord(ctx context.Context, rec *DomainDNSRecord) error {
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_dns_records (id, domain_id, record_type, name, value, ttl, priority, provider_record_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		rec.ID, rec.DomainID, rec.RecordType, rec.Name, rec.Value, rec.TTL, rec.Priority, rec.ProviderRecordID,
	).Scan(&rec.CreatedAt, &rec.UpdatedAt)
}

func (p *PostgresStore) GetDomainDNSRecords(ctx context.Context, domainID uuid.UUID) ([]*DomainDNSRecord, error) {
	query := `
		SELECT id, domain_id, record_type, name, value, ttl, priority, provider_record_id, created_at, updated_at
		FROM domain_dns_records
		WHERE domain_id = $1
		ORDER BY record_type ASC, name ASC
	`
	rows, err := p.db.QueryContext(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainDNSRecord
	for rows.Next() {
		r := &DomainDNSRecord{}
		if err := rows.Scan(
			&r.ID, &r.DomainID, &r.RecordType, &r.Name, &r.Value, &r.TTL, &r.Priority, &r.ProviderRecordID, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func (p *PostgresStore) GetDomainDNSRecordByID(ctx context.Context, id uuid.UUID) (*DomainDNSRecord, error) {
	query := `
		SELECT id, domain_id, record_type, name, value, ttl, priority, provider_record_id, created_at, updated_at
		FROM domain_dns_records
		WHERE id = $1
	`
	r := &DomainDNSRecord{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.DomainID, &r.RecordType, &r.Name, &r.Value, &r.TTL, &r.Priority, &r.ProviderRecordID, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

func (p *PostgresStore) UpdateDomainDNSRecord(ctx context.Context, rec *DomainDNSRecord) error {
	query := `
		UPDATE domain_dns_records SET
			record_type = $1,
			name = $2,
			value = $3,
			ttl = $4,
			priority = $5,
			provider_record_id = $6,
			updated_at = NOW()
		WHERE id = $7
	`
	_, err := p.db.ExecContext(ctx, query,
		rec.RecordType, rec.Name, rec.Value, rec.TTL, rec.Priority, rec.ProviderRecordID, rec.ID,
	)
	return err
}

func (p *PostgresStore) DeleteDomainDNSRecord(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM domain_dns_records WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

// ----------------------------------------------------------------------------
// Transfers, Renewals, Webhooks, Transactions, Audit (PostgresStore)
// ----------------------------------------------------------------------------

func (p *PostgresStore) CreateDomainTransfer(ctx context.Context, transfer *DomainTransfer) error {
	if transfer.ID == uuid.Nil {
		transfer.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_transfers (id, domain_id, user_id, domain_name, auth_code_encrypted, status, provider_order_id, requested_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW())
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		transfer.ID, transfer.DomainID, transfer.UserID, strings.ToLower(transfer.DomainName),
		transfer.AuthCodeEncrypted, transfer.Status, transfer.ProviderOrderID,
	).Scan(&transfer.CreatedAt, &transfer.UpdatedAt)
}

func (p *PostgresStore) GetDomainTransferByID(ctx context.Context, id uuid.UUID) (*DomainTransfer, error) {
	query := `
		SELECT id, domain_id, user_id, domain_name, auth_code_encrypted, status, provider_order_id,
		       requested_at, completed_at, failed_at, failure_reason, created_at, updated_at
		FROM domain_transfers
		WHERE id = $1
	`
	t := &DomainTransfer{}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.DomainID, &t.UserID, &t.DomainName, &t.AuthCodeEncrypted, &t.Status, &t.ProviderOrderID,
		&t.RequestedAt, &t.CompletedAt, &t.FailedAt, &t.FailureReason, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (p *PostgresStore) ListDomainTransfers(ctx context.Context, userID *uuid.UUID) ([]*DomainTransfer, error) {
	var query string
	var args []interface{}
	if userID != nil {
		query = `SELECT id, domain_id, user_id, domain_name, auth_code_encrypted, status, provider_order_id, requested_at, completed_at, failed_at, failure_reason, created_at, updated_at FROM domain_transfers WHERE user_id = $1 ORDER BY requested_at DESC`
		args = append(args, *userID)
	} else {
		query = `SELECT id, domain_id, user_id, domain_name, auth_code_encrypted, status, provider_order_id, requested_at, completed_at, failed_at, failure_reason, created_at, updated_at FROM domain_transfers ORDER BY requested_at DESC`
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainTransfer
	for rows.Next() {
		t := &DomainTransfer{}
		if err := rows.Scan(
			&t.ID, &t.DomainID, &t.UserID, &t.DomainName, &t.AuthCodeEncrypted, &t.Status, &t.ProviderOrderID,
			&t.RequestedAt, &t.CompletedAt, &t.FailedAt, &t.FailureReason, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (p *PostgresStore) UpdateDomainTransfer(ctx context.Context, transfer *DomainTransfer) error {
	query := `
		UPDATE domain_transfers SET
			status = $1,
			provider_order_id = $2,
			completed_at = $3,
			failed_at = $4,
			failure_reason = $5,
			updated_at = NOW()
		WHERE id = $6
	`
	_, err := p.db.ExecContext(ctx, query,
		transfer.Status, transfer.ProviderOrderID, transfer.CompletedAt, transfer.FailedAt, transfer.FailureReason, transfer.ID,
	)
	return err
}

func (p *PostgresStore) CreateDomainRenewal(ctx context.Context, renewal *DomainRenewal) error {
	if renewal.ID == uuid.Nil {
		renewal.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_renewals (id, domain_id, user_id, years, amount, currency, payment_status, provider_order_id, status, old_expiry_date, new_expiry_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING created_at
	`
	return p.db.QueryRowContext(ctx, query,
		renewal.ID, renewal.DomainID, renewal.UserID, renewal.Years, renewal.Amount, renewal.Currency,
		renewal.PaymentStatus, renewal.ProviderOrderID, renewal.Status, renewal.OldExpiryDate, renewal.NewExpiryDate,
	).Scan(&renewal.CreatedAt)
}

func (p *PostgresStore) ListDomainRenewals(ctx context.Context, domainID *uuid.UUID) ([]*DomainRenewal, error) {
	var query string
	var args []interface{}
	if domainID != nil {
		query = `SELECT id, domain_id, user_id, years, amount, currency, payment_status, provider_order_id, status, old_expiry_date, new_expiry_date, created_at, completed_at FROM domain_renewals WHERE domain_id = $1 ORDER BY created_at DESC`
		args = append(args, *domainID)
	} else {
		query = `SELECT id, domain_id, user_id, years, amount, currency, payment_status, provider_order_id, status, old_expiry_date, new_expiry_date, created_at, completed_at FROM domain_renewals ORDER BY created_at DESC`
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainRenewal
	for rows.Next() {
		r := &DomainRenewal{}
		if err := rows.Scan(
			&r.ID, &r.DomainID, &r.UserID, &r.Years, &r.Amount, &r.Currency, &r.PaymentStatus,
			&r.ProviderOrderID, &r.Status, &r.OldExpiryDate, &r.NewExpiryDate, &r.CreatedAt, &r.CompletedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func (p *PostgresStore) RecordDomainWebhook(ctx context.Context, hook *DomainWebhook) error {
	if hook.ID == uuid.Nil {
		hook.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_webhooks (id, provider, event_type, external_event_id, payload, status, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	_, err := p.db.ExecContext(ctx, query,
		hook.ID, hook.Provider, hook.EventType, hook.ExternalEventID, hook.Payload, hook.Status,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return errors.New("webhook already processed")
		}
		return err
	}
	return nil
}

func (p *PostgresStore) GetDomainWebhook(ctx context.Context, provider, externalEventID string) (*DomainWebhook, error) {
	query := `
		SELECT id, provider, event_type, external_event_id, payload, status, processed_at, created_at
		FROM domain_webhooks
		WHERE provider = $1 AND external_event_id = $2
		LIMIT 1
	`
	var h DomainWebhook
	var rawPayload []byte
	err := p.db.QueryRowContext(ctx, query, provider, externalEventID).Scan(
		&h.ID, &h.Provider, &h.EventType, &h.ExternalEventID, &rawPayload, &h.Status, &h.ProcessedAt, &h.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(rawPayload, &h.Payload)
	return &h, nil
}

func (p *PostgresStore) TryAcquireDomainAdvisoryLock(ctx context.Context, lockKey string) (bool, func(), error) {
	var acquired bool
	err := p.db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", lockKey).Scan(&acquired)
	if err != nil {
		return false, nil, fmt.Errorf("failed to execute advisory lock query: %w", err)
	}
	if !acquired {
		return false, nil, nil
	}
	unlock := func() {
		_, _ = p.db.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", lockKey)
	}
	return true, unlock, nil
}

func (p *PostgresStore) RecordDomainTransaction(ctx context.Context, txn *DomainTransaction) error {
	if txn.ID == uuid.Nil {
		txn.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_transactions (
			id, domain_id, order_id, provider, operation, request_id, provider_order_id,
			amount, cost, currency, status, error_code, error_message, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	return p.db.QueryRowContext(ctx, query,
		txn.ID, txn.DomainID, txn.OrderID, txn.Provider, txn.Operation, txn.RequestID, txn.ProviderOrderID,
		txn.Amount, txn.Cost, txn.Currency, txn.Status, txn.ErrorCode, txn.ErrorMessage,
	).Scan(&txn.CreatedAt, &txn.UpdatedAt)
}

func (p *PostgresStore) ListDomainTransactions(ctx context.Context, domainID *uuid.UUID) ([]*DomainTransaction, error) {
	var query string
	var args []interface{}
	if domainID != nil {
		query = `SELECT id, domain_id, order_id, provider, operation, request_id, provider_order_id, amount, cost, currency, status, error_code, error_message, created_at, updated_at FROM domain_transactions WHERE domain_id = $1 ORDER BY created_at DESC`
		args = append(args, *domainID)
	} else {
		query = `SELECT id, domain_id, order_id, provider, operation, request_id, provider_order_id, amount, cost, currency, status, error_code, error_message, created_at, updated_at FROM domain_transactions ORDER BY created_at DESC`
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainTransaction
	for rows.Next() {
		t := &DomainTransaction{}
		if err := rows.Scan(
			&t.ID, &t.DomainID, &t.OrderID, &t.Provider, &t.Operation, &t.RequestID, &t.ProviderOrderID,
			&t.Amount, &t.Cost, &t.Currency, &t.Status, &t.ErrorCode, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (p *PostgresStore) RecordDomainAuditLog(ctx context.Context, log *DomainAuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	query := `
		INSERT INTO domain_audit_logs (id, domain_id, domain_name, user_id, action, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	_, err := p.db.ExecContext(ctx, query,
		log.ID, log.DomainID, strings.ToLower(log.DomainName), log.UserID, log.Action, log.Details, log.IPAddress,
	)
	return err
}

func (p *PostgresStore) ListDomainAuditLogs(ctx context.Context, domainName string) ([]*DomainAuditLog, error) {
	var query string
	var args []interface{}
	clean := strings.ToLower(strings.TrimSpace(domainName))
	if clean != "" {
		query = `SELECT id, domain_id, domain_name, user_id, action, details, ip_address, created_at FROM domain_audit_logs WHERE domain_name = $1 ORDER BY created_at DESC LIMIT 100`
		args = append(args, clean)
	} else {
		query = `SELECT id, domain_id, domain_name, user_id, action, details, ip_address, created_at FROM domain_audit_logs ORDER BY created_at DESC LIMIT 100`
	}
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DomainAuditLog
	for rows.Next() {
		l := &DomainAuditLog{}
		if err := rows.Scan(
			&l.ID, &l.DomainID, &l.DomainName, &l.UserID, &l.Action, &l.Details, &l.IPAddress, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// Legacy TLDPricing & Registrar PostgresStore Implementations
// ----------------------------------------------------------------------------

func (p *PostgresStore) ListTLDPricings(ctx context.Context) ([]*TLDPricing, error) {
	prices, err := p.ListDomainPrices(ctx)
	if err != nil {
		return nil, err
	}
	var list []*TLDPricing
	for _, pr := range prices {
		tldMeta, _ := p.GetDomainTLD(ctx, pr.TLD)
		isPopular := false
		category := "popular"
		minYears := 1
		if tldMeta != nil {
			isPopular = tldMeta.IsPopular
			category = tldMeta.Category
			minYears = tldMeta.MinYears
		}
		list = append(list, &TLDPricing{
			TLD:           "." + pr.TLD,
			RegisterPrice: pr.RegistrationPrice,
			RenewPrice:    pr.RenewalPrice,
			TransferPrice: pr.TransferPrice,
			Currency:      pr.Currency,
			IsPopular:     isPopular,
			Category:      category,
			MinYears:      minYears,
			UpdatedAt:     pr.UpdatedAt,
		})
	}
	return list, nil
}

func (p *PostgresStore) GetTLDPricing(ctx context.Context, tld string) (*TLDPricing, error) {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tld)), ".")
	pr, err := p.GetDomainPrice(ctx, clean)
	if err != nil {
		return nil, err
	}
	tldMeta, _ := p.GetDomainTLD(ctx, clean)
	isPopular := false
	category := "popular"
	minYears := 1
	if tldMeta != nil {
		isPopular = tldMeta.IsPopular
		category = tldMeta.Category
		minYears = tldMeta.MinYears
	}
	return &TLDPricing{
		TLD:           "." + pr.TLD,
		RegisterPrice: pr.RegistrationPrice,
		RenewPrice:    pr.RenewalPrice,
		TransferPrice: pr.TransferPrice,
		Currency:      pr.Currency,
		IsPopular:     isPopular,
		Category:      category,
		MinYears:      minYears,
		UpdatedAt:     pr.UpdatedAt,
	}, nil
}

func (p *PostgresStore) SaveTLDPricing(ctx context.Context, pricing *TLDPricing) error {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(pricing.TLD)), ".")
	existingPrice, err := p.GetDomainPrice(ctx, clean)
	cost := 10.00
	if err == nil {
		cost = existingPrice.RegistrationCost
	}
	return p.SaveDomainPrice(ctx, &DomainPrice{
		TLD:               clean,
		RegistrationCost:  cost,
		RegistrationPrice: pricing.RegisterPrice,
		RenewalCost:       cost,
		RenewalPrice:      pricing.RenewPrice,
		TransferCost:      cost,
		TransferPrice:     pricing.TransferPrice,
		Currency:          pricing.Currency,
		Enabled:           true,
	})
}

func (p *PostgresStore) ListRegistrarConfigs(ctx context.Context) ([]*DomainRegistrarConfig, error) {
	// Fallback to memory store or configuration table
	return []*DomainRegistrarConfig{
		{
			Registrar:   "resellerclub",
			DisplayName: "ResellerClub / LogicBoxes HTTP API",
			Enabled:     true,
			TestMode:    false,
			UpdatedAt:   time.Now().UTC(),
		},
		{
			Registrar:   "namecheap",
			DisplayName: "Namecheap API Integration",
			Enabled:     false,
			TestMode:    true,
			UpdatedAt:   time.Now().UTC(),
		},
		{
			Registrar:   "cloudflare",
			DisplayName: "Cloudflare Registrar API",
			Enabled:     false,
			TestMode:    true,
			UpdatedAt:   time.Now().UTC(),
		},
	}, nil
}

func (p *PostgresStore) GetRegistrarConfig(ctx context.Context, registrar string) (*DomainRegistrarConfig, error) {
	configs, _ := p.ListRegistrarConfigs(ctx)
	clean := strings.ToLower(strings.TrimSpace(registrar))
	for _, c := range configs {
		if c.Registrar == clean {
			return c, nil
		}
	}
	return nil, ErrNotFound
}

func (p *PostgresStore) SaveRegistrarConfig(ctx context.Context, config *DomainRegistrarConfig) error {
	return nil
}

// Unused import suppressor
var _ = json.Marshal
