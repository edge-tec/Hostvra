package billing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

// DunningSummary reports the deterministic outcome of a billing lifecycle pass.
type DunningSummary struct {
	SubscriptionsScanned int `json:"subscriptions_scanned"`
	RenewalsGenerated    int `json:"renewals_generated"`
	AutoPaidFromWallet   int `json:"auto_paid_from_wallet"`
	RemindersSent        int `json:"reminders_sent"`
	SuspendedOverdue     int `json:"suspended_overdue"`
}

// DunningWorker automates recurring subscription invoicing (7d/3d/1d), Credit Wallet Auto-Pay,
// and grace-period suspension with strict idempotency.
type DunningWorker struct {
	store           store.Store
	interval        time.Duration
	gracePeriodDays int
	mu              sync.Mutex
	stopCh          chan struct{}
}

func NewDunningWorker(st store.Store, interval time.Duration, gracePeriodDays int) *DunningWorker {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	if gracePeriodDays <= 0 {
		gracePeriodDays = 3
	}
	return &DunningWorker{
		store:           st,
		interval:        interval,
		gracePeriodDays: gracePeriodDays,
		stopCh:          make(chan struct{}),
	}
}

func (w *DunningWorker) Start(ctx context.Context) {
	go func() {
		log.Printf("[BILLING-DUNNING] Automated subscription renewal & dunning worker started (interval=%v, grace=%dd)", w.interval, w.gracePeriodDays)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				summary, err := w.RunPass(runCtx, time.Now().UTC())
				if err != nil {
					log.Printf("[BILLING-DUNNING-WARN] pass error: %v", err)
				} else if summary.RenewalsGenerated > 0 || summary.AutoPaidFromWallet > 0 || summary.SuspendedOverdue > 0 {
					log.Printf("[BILLING-DUNNING] pass complete: scanned=%d renewals=%d autopaid=%d suspended=%d",
						summary.SubscriptionsScanned, summary.RenewalsGenerated, summary.AutoPaidFromWallet, summary.SuspendedOverdue)
				}
				cancel()
			}
		}
	}()
}

func (w *DunningWorker) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
}

// RunPass executes one idempotent dunning & renewal cycle for all active subscriptions.
func (w *DunningWorker) RunPass(ctx context.Context, now time.Time) (*DunningSummary, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	subs, err := w.store.ListSubscriptions(ctx, uuid.Nil)
	if err != nil {
		return nil, err
	}

	summary := &DunningSummary{SubscriptionsScanned: len(subs)}

	for _, sub := range subs {
		if sub.Status != store.SubStatusActive {
			continue
		}

		cycleKey := sub.NextBillingDate.UTC().Format("2006-01")
		daysUntilDue := sub.NextBillingDate.Sub(now).Hours() / 24.0

		// 1. 7-day advance renewal invoice generation (idempotent per subscription + billing cycle)
		if daysUntilDue <= 7.0 && daysUntilDue > 0 {
			recorded, _ := w.store.RecordBillingDunningEvent(ctx, &store.BillingDunningEvent{
				SubscriptionID: sub.ID,
				UserID:         sub.UserID,
				CycleKey:       cycleKey,
				NoticeStage:    "renewal_7d",
				Details:        fmt.Sprintf("Generated 7-day advance renewal invoice for %s ($%.2f)", sub.PlanName, sub.Amount),
			})
			if recorded {
				subID := sub.ID
				inv := &store.Invoice{
					ID:             uuid.New(),
					InvoiceNumber:  fmt.Sprintf("INV-REN-%s-%s", now.Format("20060102"), sub.ID.String()[:6]),
					UserID:         sub.UserID,
					OrganizationID: sub.OrganizationID,
					SubscriptionID: &subID,
					PlanID:         sub.PlanID,
					Description:    fmt.Sprintf("%s Renewal (%s cycle)", sub.PlanName, sub.BillingCycle),
					Subtotal:       sub.Amount,
					Total:          sub.Amount,
					Currency:       sub.Currency,
					Status:         store.InvoiceStatusUnpaid,
					DueDate:        sub.NextBillingDate,
					CreatedAt:      now,
				}
				if err := w.store.CreateInvoice(ctx, inv); err == nil {
					summary.RenewalsGenerated++
					// Attempt automatic Credit Wallet Auto-Pay if enabled & funded
					if wallet, wErr := w.store.GetOrCreateCustomerWallet(ctx, sub.UserID); wErr == nil && wallet.AutoPayEnabled && wallet.BalanceUSD >= inv.Total {
						idemKey := fmt.Sprintf("autopay:%s:%s", sub.ID.String(), cycleKey)
						if _, dErr := w.store.DebitWalletAtomic(ctx, sub.UserID, inv.Total, "auto_renewal", inv.ID.String(), idemKey, fmt.Sprintf("Auto-Pay renewal for %s", sub.PlanName)); dErr == nil {
							paidAt := now
							inv.Status = store.InvoiceStatusPaid
							inv.PaymentMethod = "credit_wallet_autopay"
							inv.PaidAt = &paidAt
							_ = w.store.UpdateInvoice(ctx, inv)

							if sub.BillingCycle == "yearly" {
								sub.NextBillingDate = sub.NextBillingDate.AddDate(1, 0, 0)
							} else {
								sub.NextBillingDate = sub.NextBillingDate.AddDate(0, 1, 0)
							}
							_ = w.store.UpdateSubscription(ctx, sub)
							_, _ = w.store.RecordBillingDunningEvent(ctx, &store.BillingDunningEvent{
								SubscriptionID: sub.ID,
								UserID:         sub.UserID,
								InvoiceID:      &inv.ID,
								CycleKey:       cycleKey,
								NoticeStage:    "auto_pay_success",
								Details:        fmt.Sprintf("Auto-paid $%.2f from customer credit wallet", inv.Total),
							})
							summary.AutoPaidFromWallet++
							continue
						}
					}
				}
			}
		}

		// 2. 3-day & 1-day renewal reminders
		if daysUntilDue <= 3.0 && daysUntilDue > 1.0 {
			if ok, _ := w.store.RecordBillingDunningEvent(ctx, &store.BillingDunningEvent{
				SubscriptionID: sub.ID,
				UserID:         sub.UserID,
				CycleKey:       cycleKey,
				NoticeStage:    "renewal_3d",
				Details:        "3-day renewal reminder dispatched",
			}); ok {
				summary.RemindersSent++
			}
		}
		if daysUntilDue <= 1.0 && daysUntilDue > 0 {
			if ok, _ := w.store.RecordBillingDunningEvent(ctx, &store.BillingDunningEvent{
				SubscriptionID: sub.ID,
				UserID:         sub.UserID,
				CycleKey:       cycleKey,
				NoticeStage:    "renewal_1d",
				Details:        "Final 24-hour renewal reminder dispatched",
			}); ok {
				summary.RemindersSent++
			}
		}

		// 3. Grace period exceeded -> suspend overdue subscription automatically
		overdueDays := now.Sub(sub.NextBillingDate).Hours() / 24.0
		if overdueDays >= float64(w.gracePeriodDays) {
			if ok, _ := w.store.RecordBillingDunningEvent(ctx, &store.BillingDunningEvent{
				SubscriptionID: sub.ID,
				UserID:         sub.UserID,
				CycleKey:       cycleKey,
				NoticeStage:    "suspended",
				Details:        fmt.Sprintf("Subscription automatically suspended after %d-day grace period", w.gracePeriodDays),
			}); ok {
				sub.Status = store.SubStatusSuspended
				_ = w.store.UpdateSubscription(ctx, sub)
				summary.SuspendedOverdue++
			}
		}
	}

	return summary, nil
}
