package email

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestEmailServiceNotifications(t *testing.T) {
	cfg := &config.Config{
		AppURL:       "https://panel.hostvra.com",
		SMTPHost:     "", // simulated
		SMTPPort:     587,
		SMTPFrom:     "noreply@hostvra.com",
		SMTPFromName: "Hostvra Test",
	}

	st := store.NewMemoryStore()
	svc := New(cfg, st)

	ctx := context.Background()

	// 1. Welcome / Account Created
	if err := svc.SendWelcomeAccountCreated(ctx, "user@test.com", "Test User", "user@test.com", "https://panel.hostvra.com/login"); err != nil {
		t.Fatalf("SendWelcomeAccountCreated failed: %v", err)
	}

	// 2. Email Verification
	if err := svc.SendEmailVerification(ctx, "user@test.com", "Test User", "test_token_123"); err != nil {
		t.Fatalf("SendEmailVerification failed: %v", err)
	}

	// 3. Password Reset
	if err := svc.SendPasswordReset(ctx, "user@test.com", "Test User", "reset_token_456"); err != nil {
		t.Fatalf("SendPasswordReset failed: %v", err)
	}

	// 4. Password Changed
	if err := svc.SendPasswordChanged(ctx, "user@test.com", "Test User"); err != nil {
		t.Fatalf("SendPasswordChanged failed: %v", err)
	}

	// 5. Payment Confirmed
	if err := svc.SendPaymentConfirmation(ctx, "user@test.com", "Test User", "INV-2026-001", "29.99", "USD", "Cloud Pro"); err != nil {
		t.Fatalf("SendPaymentConfirmation failed: %v", err)
	}

	// 6. Payment Failed
	if err := svc.SendPaymentFailed(ctx, "user@test.com", "Test User", "INV-2026-002", "49.99", "USD", "Card expired", "https://panel.hostvra.com/billing"); err != nil {
		t.Fatalf("SendPaymentFailed failed: %v", err)
	}

	// 7. 7-Day Expiring Warning
	if err := svc.SendAccountExpiringWarning(ctx, "user@test.com", "Test User", "Cloud Pro", "mysite.com", 7, "https://panel.hostvra.com/billing"); err != nil {
		t.Fatalf("SendAccountExpiringWarning failed: %v", err)
	}

	// 8. Account Expired
	if err := svc.SendAccountExpired(ctx, "user@test.com", "Test User", "Cloud Pro", "mysite.com", "Overdue invoice", "https://panel.hostvra.com/billing"); err != nil {
		t.Fatalf("SendAccountExpired failed: %v", err)
	}

	// 9. System Update Notification
	if err := svc.SendSystemUpdateNotification(ctx, "user@test.com", "Test User", "v2.5.0", "New High-Speed Nginx Engine", "Upgraded Nginx 500M uploads and SMTP notifications.", "https://panel.hostvra.com/settings"); err != nil {
		t.Fatalf("SendSystemUpdateNotification failed: %v", err)
	}
}

func TestEmailServiceDynamicSettings(t *testing.T) {
	cfg := &config.Config{
		AppURL:   "https://panel.hostvra.com",
		SMTPHost: "", // empty in env
	}

	st := store.NewMemoryStore()
	svc := New(cfg, st)

	ctx := context.Background()

	// Initial config should have empty host
	initial := svc.GetResolvedConfig(ctx)
	if initial.Host != "" && initial.Host != "127.0.0.1" {
		t.Fatalf("expected empty or local host, got %s", initial.Host)
	}

	// Update DB SystemSettings with custom SMTP
	sysSettings := &store.SystemSettings{
		SMTPHost:       "smtp.customrelay.com",
		SMTPPort:       465,
		SMTPUser:       "relay_user",
		SMTPPassword:   "relay_secret",
		SMTPFrom:       "alerts@customrelay.com",
		SMTPFromName:   "Custom Relayer",
		SMTPEncryption: "tls",
		UpdatedAt:      time.Now().UTC(),
	}
	_ = st.UpdateSystemSettings(ctx, sysSettings)

	resolved := svc.GetResolvedConfig(ctx)
	if resolved.Host != "smtp.customrelay.com" {
		t.Fatalf("expected smtp.customrelay.com, got %s", resolved.Host)
	}
	if resolved.Port != 465 {
		t.Fatalf("expected port 465, got %d", resolved.Port)
	}
	if resolved.Encryption != "tls" {
		t.Fatalf("expected encryption tls, got %s", resolved.Encryption)
	}
	if resolved.From != "alerts@customrelay.com" {
		t.Fatalf("expected alerts@customrelay.com, got %s", resolved.From)
	}
}

func TestEmailServiceIdempotency(t *testing.T) {
	cfg := &config.Config{
		AppURL: "https://panel.hostvra.com",
	}

	st := store.NewMemoryStore()
	svc := New(cfg, st)

	ctx := context.Background()
	eventKey := "invoice_unique_" + uuid.New().String()

	// First send should be recorded as new
	sent1, err := svc.SendIdempotent(ctx, "test_event", eventKey, "test@test.com", "Test Subject", "<p>Hello</p>")
	if err != nil {
		t.Fatalf("first send failed: %v", err)
	}
	if !sent1 {
		t.Fatalf("expected sent1 to be true")
	}

	// Second send with same eventKey should be deduplicated (idempotent)
	sent2, err := svc.SendIdempotent(ctx, "test_event", eventKey, "test@test.com", "Test Subject", "<p>Hello</p>")
	if err != nil {
		t.Fatalf("second send failed: %v", err)
	}
	if sent2 {
		t.Fatalf("expected sent2 to be false (idempotently skipped)")
	}
}
