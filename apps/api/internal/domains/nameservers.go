package domains

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type NameserverService struct {
	store     store.Store
	registrar DomainRegistrar
}

func NewNameserverService(s store.Store, r DomainRegistrar) *NameserverService {
	return &NameserverService{
		store:     s,
		registrar: r,
	}
}

// GetNameservers retrieves current nameservers from DB or registrar
func (ns *NameserverService) GetNameservers(ctx context.Context, domainID uuid.UUID) ([]string, error) {
	d, err := ns.store.GetDomainByID(ctx, domainID)
	if err != nil {
		return nil, err
	}

	servers, err := ns.store.GetDomainNameservers(ctx, domainID)
	if err == nil && len(servers) > 0 {
		return servers, nil
	}

	if d.ProviderOrderID != "" {
		remoteNS, err := ns.registrar.GetNameservers(ctx, d.DomainName)
		if err == nil && len(remoteNS) > 0 {
			_ = ns.store.SaveDomainNameservers(ctx, domainID, remoteNS)
			return remoteNS, nil
		}
	}

	return []string{"ns1.hostvra.com", "ns2.hostvra.com"}, nil
}

// UpdateNameservers modifies nameservers at registrar and in database
func (ns *NameserverService) UpdateNameservers(ctx context.Context, domainID uuid.UUID, newServers []string) error {
	d, err := ns.store.GetDomainByID(ctx, domainID)
	if err != nil {
		return err
	}

	var cleaned []string
	for _, s := range newServers {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			cleaned = append(cleaned, strings.ToLower(trimmed))
		}
	}
	if len(cleaned) < 2 {
		return fmt.Errorf("at least 2 valid nameservers are required")
	}

	// 1. Call registrar
	if d.ProviderOrderID != "" {
		if err := ns.registrar.UpdateNameservers(ctx, NameserverUpdateRequest{
			DomainName:      d.DomainName,
			ProviderOrderID: d.ProviderOrderID,
			Nameservers:     cleaned,
		}); err != nil {
			return fmt.Errorf("failed to update nameservers at registrar: %w", err)
		}
	}

	// 2. Save in database
	if err := ns.store.SaveDomainNameservers(ctx, domainID, cleaned); err != nil {
		return fmt.Errorf("failed to update nameservers in database: %w", err)
	}

	// 3. Audit log
	_ = ns.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainID:   &domainID,
		DomainName: d.DomainName,
		UserID:     &d.UserID,
		Action:     "DOMAIN_NAMESERVERS_UPDATED",
		Details:    fmt.Sprintf("Nameservers updated to: %s", strings.Join(cleaned, ", ")),
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}
