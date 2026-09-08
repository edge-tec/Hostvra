package domains

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hostvra/api/internal/store"
)

type ExpiryNotifier struct {
	store store.Store
}

func NewExpiryNotifier(s store.Store) *ExpiryNotifier {
	return &ExpiryNotifier{store: s}
}

// CheckExpiringDomains finds domains expiring within standard reminder intervals
func (en *ExpiryNotifier) CheckExpiringDomains(ctx context.Context) error {
	domains, err := en.store.ListAllDomains(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, d := range domains {
		if d.ExpiryDate == nil {
			continue
		}

		daysLeft := int(d.ExpiryDate.Sub(now).Hours() / 24)
		if daysLeft == 30 || daysLeft == 15 || daysLeft == 7 || daysLeft == 3 || daysLeft == 1 {
			slog.Info("Domain expiring soon", "domain", d.DomainName, "days_remaining", daysLeft)

			_ = en.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
				DomainID:   &d.ID,
				DomainName: d.DomainName,
				UserID:     &d.UserID,
				Action:     "DOMAIN_EXPIRY_NOTIFICATION",
				Details:    fmt.Sprintf("Sent expiry reminder for %s: %d days remaining", d.DomainName, daysLeft),
				CreatedAt:  now,
			})
		}
	}

	return nil
}
