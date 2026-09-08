package domains

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type DNSService struct {
	store     store.Store
	registrar DomainRegistrar
}

func NewDNSService(s store.Store, r DomainRegistrar) *DNSService {
	return &DNSService{
		store:     s,
		registrar: r,
	}
}

func (s *DNSService) ListRecords(ctx context.Context, domainID uuid.UUID) ([]*store.DomainDNSRecord, error) {
	return s.store.GetDomainDNSRecords(ctx, domainID)
}

func (s *DNSService) CreateRecord(ctx context.Context, domainID uuid.UUID, recType, name, value string, ttl, priority int) (*store.DomainDNSRecord, error) {
	d, err := s.store.GetDomainByID(ctx, domainID)
	if err != nil {
		return nil, err
	}

	recType = strings.ToUpper(strings.TrimSpace(recType))
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	if ttl <= 0 {
		ttl = 3600
	}

	// 1. Call registrar DNS API
	if d.ProviderOrderID != "" {
		_ = s.registrar.CreateDNSRecord(ctx, DNSRecordRequest{
			DomainName: d.DomainName,
			Type:       recType,
			Name:       name,
			Value:      value,
			TTL:        ttl,
			Priority:   priority,
		})
	}

	// 2. Save in database
	dbRec := &store.DomainDNSRecord{
		ID:         uuid.New(),
		DomainID:   domainID,
		RecordType: recType,
		Name:       name,
		Value:      value,
		TTL:        ttl,
		Priority:   &priority,
	}
	if err := s.store.CreateDomainDNSRecord(ctx, dbRec); err != nil {
		return nil, err
	}

	_ = s.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainID:   &domainID,
		DomainName: d.DomainName,
		UserID:     &d.UserID,
		Action:     "DOMAIN_DNS_RECORD_CREATED",
		Details:    fmt.Sprintf("Created DNS %s record %s -> %s (TTL: %d)", recType, name, value, ttl),
		CreatedAt:  time.Now().UTC(),
	})

	return dbRec, nil
}

func (s *DNSService) DeleteRecord(ctx context.Context, domainID, recordID uuid.UUID) error {
	d, err := s.store.GetDomainByID(ctx, domainID)
	if err != nil {
		return err
	}

	rec, err := s.store.GetDomainDNSRecordByID(ctx, recordID)
	if err != nil {
		return err
	}

	if d.ProviderOrderID != "" {
		_ = s.registrar.DeleteDNSRecord(ctx, d.DomainName, rec.ID.String())
	}

	if err := s.store.DeleteDomainDNSRecord(ctx, recordID); err != nil {
		return err
	}

	_ = s.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainID:   &domainID,
		DomainName: d.DomainName,
		UserID:     &d.UserID,
		Action:     "DOMAIN_DNS_RECORD_DELETED",
		Details:    fmt.Sprintf("Deleted DNS %s record %s -> %s", rec.RecordType, rec.Name, rec.Value),
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}
