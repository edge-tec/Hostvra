package domains

import (
	"context"
	"log/slog"
	"time"

	"hostvra/api/internal/store"
)

type ReconciliationWorker struct {
	store     store.Store
	registrar DomainRegistrar
}

func NewReconciliationWorker(s store.Store, r DomainRegistrar) *ReconciliationWorker {
	return &ReconciliationWorker{
		store:     s,
		registrar: r,
	}
}

// ReconcileAll checks domains in non-final states and syncs with registrar
func (rw *ReconciliationWorker) ReconcileAll(ctx context.Context) (int, error) {
	// Distributed lock: ensure only one instance of reconciliation runs cluster-wide
	acquired, unlock, lockErr := rw.store.TryAcquireDomainAdvisoryLock(ctx, "job:domain:reconciliation")
	if lockErr != nil || !acquired {
		slog.Info("Another instance is actively running domain reconciliation; skipping duplicate run")
		return 0, nil
	}
	if unlock != nil {
		defer unlock()
	}

	domains, err := rw.store.ListAllDomains(ctx)
	if err != nil {
		return 0, err
	}

	synced := 0
	for _, d := range domains {
		select {
		case <-ctx.Done():
			return synced, ctx.Err()
		default:
		}

		// Only reconcile domains with a known provider order ID
		if d.ProviderOrderID == "" {
			continue
		}

		// Pacing to respect registrar rate limits (10 requests/second max)
		time.Sleep(100 * time.Millisecond)

		info, err := rw.registrar.GetDomainInfo(ctx, d.DomainName)
		if err != nil {
			slog.Debug("Reconciliation skipped domain due to registrar response", "domain", d.DomainName, "error", err)
			continue
		}

		changed := false
		if info.ExpiryDate != nil && (d.ExpiryDate == nil || !d.ExpiryDate.Equal(*info.ExpiryDate)) {
			d.ExpiryDate = info.ExpiryDate
			changed = true
		}
		if d.RegistrarLock != info.RegistrarLock {
			d.RegistrarLock = info.RegistrarLock
			changed = true
		}
		if info.Status != "" && d.Status != info.Status {
			d.Status = info.Status
			changed = true
		}

		if changed {
			_ = rw.store.UpdateDomain(ctx, d)
			synced++
		}

		if len(info.Nameservers) > 0 {
			_ = rw.store.SaveDomainNameservers(ctx, d.ID, info.Nameservers)
		}
	}

	return synced, nil
}

// StartBackgroundLoop runs periodically to sync domain statuses
func (rw *ReconciliationWorker) StartBackgroundLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := rw.ReconcileAll(ctx)
				if err != nil {
					slog.Warn("Domain reconciliation loop error", "error", err)
				} else if count > 0 {
					slog.Info("Domain reconciliation completed", "synced_domains", count)
				}
			}
		}
	}()
}
