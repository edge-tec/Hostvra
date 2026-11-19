package crypto

import (
	"context"
	"log"
	"time"
)

type BackgroundScanner struct {
	cryptoSvc *CryptoService
	interval  time.Duration
	stopCh    chan struct{}
}

func NewBackgroundScanner(cryptoSvc *CryptoService, interval time.Duration) *BackgroundScanner {
	if interval <= 0 {
		interval = 20 * time.Second
	}
	return &BackgroundScanner{
		cryptoSvc: cryptoSvc,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start launches the background polling worker.
func (b *BackgroundScanner) Start(ctx context.Context) {
	go func() {
		log.Printf("[CRYPTO] Blockchain deposit scanner worker started (interval: %v)", b.interval)
		ticker := time.NewTicker(b.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[CRYPTO] Blockchain deposit scanner worker stopping...")
				return
			case <-b.stopCh:
				log.Println("[CRYPTO] Blockchain deposit scanner worker stopped.")
				return
			case <-ticker.C:
				scanCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				if err := b.cryptoSvc.ScanAllPendingInvoices(scanCtx); err != nil && scanCtx.Err() == nil {
					log.Printf("[CRYPTO-WARN] Error during periodic pending invoice scan: %v", err)
				}
				if b.cryptoSvc.Treasury() != nil {
					if err := b.cryptoSvc.Treasury().ProcessPendingSweeps(scanCtx); err != nil && scanCtx.Err() == nil {
						log.Printf("[CRYPTO-WARN] Error during automated treasury sweep pass: %v", err)
					}
				}
				cancel()
			}
		}
	}()
}

// Stop terminates the scanner worker.
func (b *BackgroundScanner) Stop() {
	select {
	case <-b.stopCh:
	default:
		close(b.stopCh)
	}
}
