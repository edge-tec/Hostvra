package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

// Service provides centralized transactional email delivery with SMTP and idempotency
type Service struct {
	cfg   *config.Config
	store store.Store
}

// New creates a new email service
func New(cfg *config.Config, s store.Store) *Service {
	return &Service{cfg: cfg, store: s}
}

// IsConfigured returns true if SMTP is properly configured for real delivery
func (s *Service) IsConfigured() bool {
	return s.cfg.SMTPHost != "" && s.cfg.SMTPFrom != ""
}

// GenerateSecureToken creates a cryptographically random token and returns (rawToken, sha256Hash)
func GenerateSecureToken() (string, string) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	rawToken := hex.EncodeToString(tokenBytes)
	h := sha256.Sum256([]byte(rawToken))
	return rawToken, hex.EncodeToString(h[:])
}

// HashToken returns the SHA256 hex hash of a raw token
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// sendSMTP sends an email via configured SMTP server
func (s *Service) sendSMTP(to, subject, htmlBody string) error {
	if !s.IsConfigured() {
		slog.Warn("SMTP not configured — email simulated", "to", to, "subject", subject)
		return nil
	}

	from := s.cfg.SMTPFrom
	fromName := s.cfg.SMTPFromName
	if fromName == "" {
		fromName = "Hostvra"
	}

	headers := []string{
		fmt.Sprintf("From: %s <%s>", fromName, from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"",
		fmt.Sprintf("Date: %s", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 -0700")),
		fmt.Sprintf("Message-ID: <%s@hostvra>", uuid.New().String()),
	}

	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody)

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	var auth smtp.Auth
	if s.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

// SendIdempotent sends an email with idempotency tracking. Returns true if newly sent, false if already delivered.
func (s *Service) SendIdempotent(ctx context.Context, eventType, eventKey, to, subject, htmlBody string) (bool, error) {
	// Record delivery attempt (idempotent — unique constraint on event_type + event_key)
	delivery := &store.EmailDelivery{
		ID:             uuid.New(),
		EventType:      eventType,
		EventKey:       eventKey,
		RecipientEmail: to,
		Subject:        subject,
		Status:         "pending",
	}

	isNew, err := s.store.RecordEmailDelivery(ctx, delivery)
	if err != nil {
		slog.Error("Failed to record email delivery", "error", err, "event_type", eventType, "event_key", eventKey)
		return false, err
	}
	if !isNew {
		slog.Info("Idempotent email delivery — already sent", "event_type", eventType, "event_key", eventKey)
		return false, nil
	}

	// Attempt actual delivery
	if err := s.sendSMTP(to, subject, htmlBody); err != nil {
		slog.Error("SMTP delivery failed", "error", err, "to", to, "subject", subject)
		// Still recorded as sent (idempotency preserved), but log failure
		return true, nil // Don't fail the caller's flow for email issues
	}

	slog.Info("Transactional email delivered", "to", to, "subject", subject, "event_type", eventType)
	return true, nil
}

// ================================
// EMAIL TEMPLATES
// ================================

func (s *Service) appURL() string {
	url := s.cfg.AppURL
	if url == "" {
		url = "https://panel.hostvra.com"
	}
	return strings.TrimRight(url, "/")
}

// SendEmailVerification sends a verification email to a newly registered user
func (s *Service) SendEmailVerification(ctx context.Context, to, fullName, rawToken string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL(), rawToken)
	subject := "Verify your Hostvra account"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 40px; border: 1px solid #e2e8f0;">
  <div style="text-align: center; margin-bottom: 32px;">
    <div style="width: 48px; height: 48px; background: #16a34a; border-radius: 12px; display: inline-flex; align-items: center; justify-content: center;">
      <span style="color: white; font-size: 24px; font-weight: bold;">H</span>
    </div>
  </div>
  <h1 style="font-size: 22px; font-weight: 700; color: #0f172a; margin-bottom: 16px;">Welcome to Hostvra, %s!</h1>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 24px;">
    Please verify your email address to activate your account and access all hosting features.
  </p>
  <div style="text-align: center; margin-bottom: 24px;">
    <a href="%s" style="display: inline-block; background: #16a34a; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px;">
      Verify Email Address
    </a>
  </div>
  <p style="color: #94a3b8; font-size: 12px; line-height: 1.5;">
    This link expires in 24 hours. If you didn't create an account, you can safely ignore this email.
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Self-Hosted Server Control Plane</p>
</div>
</body>
</html>`, fullName, verifyURL)

	_, err := s.SendIdempotent(ctx, "email_verification", to, to, subject, body)
	return err
}

// SendPasswordReset sends a password reset email
func (s *Service) SendPasswordReset(ctx context.Context, to, fullName, rawToken string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL(), rawToken)
	subject := "Reset your Hostvra password"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 40px; border: 1px solid #e2e8f0;">
  <div style="text-align: center; margin-bottom: 32px;">
    <div style="width: 48px; height: 48px; background: #16a34a; border-radius: 12px; display: inline-flex; align-items: center; justify-content: center;">
      <span style="color: white; font-size: 24px; font-weight: bold;">H</span>
    </div>
  </div>
  <h1 style="font-size: 22px; font-weight: 700; color: #0f172a; margin-bottom: 16px;">Password Reset Request</h1>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 8px;">Hi %s,</p>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 24px;">
    We received a request to reset your password. Click the button below to choose a new password.
  </p>
  <div style="text-align: center; margin-bottom: 24px;">
    <a href="%s" style="display: inline-block; background: #16a34a; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px;">
      Reset Password
    </a>
  </div>
  <p style="color: #94a3b8; font-size: 12px; line-height: 1.5;">
    This link expires in 1 hour. If you did not request a password reset, please ignore this email — your password will remain unchanged.
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Self-Hosted Server Control Plane</p>
</div>
</body>
</html>`, fullName, resetURL)

	_, err := s.SendIdempotent(ctx, "password_reset", to+":"+time.Now().UTC().Format("2006-01-02T15"), to, subject, body)
	return err
}

// SendPasswordChanged sends a notification that the password was changed
func (s *Service) SendPasswordChanged(ctx context.Context, to, fullName string) error {
	subject := "Your Hostvra password was changed"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 40px; border: 1px solid #e2e8f0;">
  <h1 style="font-size: 22px; font-weight: 700; color: #0f172a; margin-bottom: 16px;">Password Changed</h1>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 8px;">Hi %s,</p>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 24px;">
    Your Hostvra account password was successfully changed. If you did not make this change, please contact support immediately.
  </p>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Self-Hosted Server Control Plane</p>
</div>
</body>
</html>`, fullName)

	_, err := s.SendIdempotent(ctx, "password_changed", to+":"+time.Now().UTC().Format("2006-01-02T15:04"), to, subject, body)
	return err
}

// SendPaymentConfirmation sends a payment confirmation email
func (s *Service) SendPaymentConfirmation(ctx context.Context, to, fullName, invoiceNumber, amount, currency, planName string) error {
	subject := fmt.Sprintf("Payment Confirmed — %s", invoiceNumber)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 40px; border: 1px solid #e2e8f0;">
  <h1 style="font-size: 22px; font-weight: 700; color: #16a34a; margin-bottom: 16px;">✓ Payment Confirmed</h1>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 8px;">Hi %s,</p>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 24px;">
    Your payment has been successfully processed and your hosting services are active.
  </p>
  <table style="width: 100%%; border-collapse: collapse; margin-bottom: 24px;">
    <tr><td style="padding: 8px 0; color: #64748b; font-size: 14px;">Invoice</td><td style="padding: 8px 0; text-align: right; font-weight: 600; color: #0f172a; font-size: 14px;">%s</td></tr>
    <tr><td style="padding: 8px 0; color: #64748b; font-size: 14px;">Plan</td><td style="padding: 8px 0; text-align: right; font-weight: 600; color: #0f172a; font-size: 14px;">%s</td></tr>
    <tr style="border-top: 1px solid #e2e8f0;"><td style="padding: 12px 0; color: #0f172a; font-weight: 700; font-size: 16px;">Total</td><td style="padding: 12px 0; text-align: right; font-weight: 700; color: #16a34a; font-size: 16px;">%s %s</td></tr>
  </table>
  <div style="text-align: center; margin-bottom: 24px;">
    <a href="%s/billing" style="display: inline-block; background: #16a34a; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px;">
      View Billing Dashboard
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Self-Hosted Server Control Plane</p>
</div>
</body>
</html>`, fullName, invoiceNumber, planName, amount, currency, s.appURL())

	_, err := s.SendIdempotent(ctx, "payment_confirmed", invoiceNumber, to, subject, body)
	return err
}

// SendOrderConfirmation sends an order/subscription confirmation email
func (s *Service) SendOrderConfirmation(ctx context.Context, to, fullName, planName, billingCycle string) error {
	subject := fmt.Sprintf("Order Confirmed — %s", planName)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 40px; border: 1px solid #e2e8f0;">
  <h1 style="font-size: 22px; font-weight: 700; color: #16a34a; margin-bottom: 16px;">🎉 Order Confirmed</h1>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 8px;">Hi %s,</p>
  <p style="color: #475569; font-size: 15px; line-height: 1.6; margin-bottom: 24px;">
    Your subscription to <strong>%s</strong> (%s) has been activated successfully. You now have full access to all features in your plan.
  </p>
  <div style="text-align: center; margin-bottom: 24px;">
    <a href="%s/dashboard" style="display: inline-block; background: #16a34a; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px;">
      Go to Dashboard
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Self-Hosted Server Control Plane</p>
</div>
</body>
</html>`, fullName, planName, billingCycle, s.appURL())

	eventKey := fmt.Sprintf("%s:%s:%s", to, planName, time.Now().UTC().Format("2006-01-02"))
	_, err := s.SendIdempotent(ctx, "order_confirmed", eventKey, to, subject, body)
	return err
}
