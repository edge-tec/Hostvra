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

type HostingProvisionDetails struct {
	Domain       string
	PlanName     string
	Username     string
	Password     string
	ControlPanel string
	ServerIP     string
	Nameservers  []string
	DiskSpaceMB  int64
	BandwidthMB  int64
	PHPVersion   string
	DBName       string
	DBUser       string
	DBPassword   string
}

// SendHostingProvisioned sends comprehensive hosting welcome email with credentials
func (s *Service) SendHostingProvisioned(ctx context.Context, to, fullName string, p HostingProvisionDetails) error {
	subject := fmt.Sprintf("🚀 Your Hosting is Ready: %s", p.Domain)

	nsRows := ""
	for i, ns := range p.Nameservers {
		nsRows += fmt.Sprintf("<tr><td style=\"padding: 4px 0; color: #64748b; font-size: 13px;\">NS %d</td><td style=\"padding: 4px 0; text-align: right; font-mono: monospace; font-size: 13px; color: #0f172a;\">%s</td></tr>", i+1, ns)
	}
	if nsRows == "" {
		nsRows = `<tr><td style="padding: 4px 0; color: #64748b; font-size: 13px;">NS 1</td><td style="padding: 4px 0; text-align: right; font-mono: monospace; font-size: 13px; color: #0f172a;">ns1.hostvra.com</td></tr>
		<tr><td style="padding: 4px 0; color: #64748b; font-size: 13px;">NS 2</td><td style="padding: 4px 0; text-align: right; font-mono: monospace; font-size: 13px; color: #0f172a;">ns2.hostvra.com</td></tr>`
	}

	dbSection := ""
	if p.DBName != "" {
		dbSection = fmt.Sprintf(`
		<div style="background: #f1f5f9; border-radius: 8px; padding: 16px; margin-top: 16px;">
		  <div style="font-weight: 700; font-size: 14px; color: #0f172a; margin-bottom: 8px;">Database Credentials (MySQL)</div>
		  <table style="width: 100%%; font-size: 13px;">
		    <tr><td style="color: #64748b;">DB Name:</td><td style="font-family: monospace; font-weight: 600; text-align: right;">%s</td></tr>
		    <tr><td style="color: #64748b;">DB User:</td><td style="font-family: monospace; font-weight: 600; text-align: right;">%s</td></tr>
		    <tr><td style="color: #64748b;">Password:</td><td style="font-family: monospace; font-weight: 600; text-align: right;">%s</td></tr>
		  </table>
		</div>`, p.DBName, p.DBUser, p.DBPassword)
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 600px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #e2e8f0; box-shadow: 0 4px 12px rgba(0,0,0,0.05);">
  <div style="text-align: center; margin-bottom: 24px;">
    <h1 style="font-size: 24px; font-weight: 800; color: #0f172a; margin: 0;">🎉 Your Hosting Account is Live!</h1>
    <p style="color: #64748b; font-size: 14px; margin-top: 6px;">Thank you for choosing Hostvra. Your server has been zero-touch provisioned.</p>
  </div>

  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 12px; padding: 20px; margin-bottom: 20px;">
    <div style="font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: #64748b; margin-bottom: 12px;">Account Overview</div>
    <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
      <tr><td style="padding: 6px 0; color: #64748b;">Primary Domain</td><td style="padding: 6px 0; text-align: right; font-weight: 700; color: #0f172a;"><a href="https://%s" style="color: #2563eb; text-decoration: none;">%s</a></td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">Hosting Plan</td><td style="padding: 6px 0; text-align: right; font-weight: 600; color: #0f172a;">%s</td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">Server IP Address</td><td style="padding: 6px 0; text-align: right; font-family: monospace; color: #0f172a;">%s</td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">Storage / Bandwidth</td><td style="padding: 6px 0; text-align: right; color: #0f172a;">%d MB / %d MB</td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">PHP FastCGI Version</td><td style="padding: 6px 0; text-align: right; font-weight: 600; color: #0f172a;">PHP %s</td></tr>
    </table>
  </div>

  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 12px; padding: 20px; margin-bottom: 20px;">
    <div style="font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: #64748b; margin-bottom: 12px;">SFTP / SSH & Panel Access</div>
    <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
      <tr><td style="padding: 6px 0; color: #64748b;">Control Panel URL</td><td style="padding: 6px 0; text-align: right;"><a href="%s" style="color: #2563eb; font-weight: 600; text-decoration: none;">Login to Hostvra</a></td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">System / SFTP User</td><td style="padding: 6px 0; text-align: right; font-family: monospace; font-weight: 700; color: #0f172a;">%s</td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">Temporary Password</td><td style="padding: 6px 0; text-align: right; font-family: monospace; font-weight: 700; color: #0f172a;">%s</td></tr>
    </table>
    %s
  </div>

  <div style="background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 12px; padding: 20px; margin-bottom: 24px;">
    <div style="font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: #1e40af; margin-bottom: 10px;">Nameservers (DNS Configuration)</div>
    <p style="font-size: 13px; color: #1e3a8a; margin: 0 0 10px 0;">Point your domain registrar to the following authoritative nameservers:</p>
    <table style="width: 100%%; border-collapse: collapse;">
      %s
    </table>
  </div>

  <div style="text-align: center; margin-bottom: 24px;">
    <a href="%s/websites" style="display: inline-block; background: #2563eb; color: #ffffff; padding: 14px 36px; border-radius: 10px; text-decoration: none; font-weight: 700; font-size: 14px; box-shadow: 0 4px 10px rgba(37,99,235,0.25);">
      Manage Your Website
    </a>
  </div>

  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 12px; text-align: center; margin: 0;">Hostvra Cloud & Server Management Platform • support@hostvra.com</p>
</div>
</body>
</html>`,
		p.Domain, p.Domain, p.PlanName, p.ServerIP, p.DiskSpaceMB, p.BandwidthMB, p.PHPVersion,
		p.ControlPanel, p.Username, p.Password, dbSection, nsRows, s.appURL(),
	)

	eventKey := fmt.Sprintf("provision:%s:%s", to, p.Domain)
	_, err := s.SendIdempotent(ctx, "hosting_provisioned", eventKey, to, subject, body)
	return err
}

// SendAccountSuspended notifies user of account suspension
func (s *Service) SendAccountSuspended(ctx context.Context, to, fullName, domain, reason string, invoiceID *uuid.UUID) error {
	subject := fmt.Sprintf("⚠️ Notice: Hosting Account Suspended (%s)", domain)
	payURL := s.appURL() + "/billing"
	if invoiceID != nil {
		payURL = fmt.Sprintf("%s/billing?invoice=%s", s.appURL(), invoiceID.String())
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #fee2e2;">
  <h1 style="font-size: 20px; font-weight: 700; color: #dc2626; margin-bottom: 12px;">Account Suspended</h1>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    Your hosting account for <strong>%s</strong> has been temporarily suspended due to: <strong style="color: #dc2626;">%s</strong>.
  </p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    To reactivate your service instantly, please settle the outstanding invoice. Once paid, your website and email will be restored automatically within seconds.
  </p>
  <div style="text-align: center; margin: 24px 0;">
    <a href="%s" style="display: inline-block; background: #dc2626; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px;">
      Pay & Reactivate Account
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 20px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Automated Cloud Billing Engine</p>
</div>
</body>
</html>`, fullName, domain, reason, payURL)

	eventKey := fmt.Sprintf("suspended:%s:%s:%s", to, domain, time.Now().UTC().Format("2006-01-02"))
	_, err := s.SendIdempotent(ctx, "account_suspended", eventKey, to, subject, body)
	return err
}

// SendAccountUnsuspended notifies user that their account was restored
func (s *Service) SendAccountUnsuspended(ctx context.Context, to, fullName, domain string) error {
	subject := fmt.Sprintf("✅ Service Restored: %s is Active", domain)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #dcfce7;">
  <h1 style="font-size: 20px; font-weight: 700; color: #16a34a; margin-bottom: 12px;">✓ Service Reactivated</h1>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    Your payment has been received and your hosting account for <strong>%s</strong> is now completely reactivated. All web, database, and email services are running normally.
  </p>
  <div style="text-align: center; margin: 24px 0;">
    <a href="%s/websites" style="display: inline-block; background: #16a34a; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 14px;">
      View Websites
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 20px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra — Automated Cloud Billing Engine</p>
</div>
</body>
</html>`, fullName, domain, s.appURL())

	eventKey := fmt.Sprintf("unsuspended:%s:%s:%s", to, domain, time.Now().UTC().Format("2006-01-02"))
	_, err := s.SendIdempotent(ctx, "account_unsuspended", eventKey, to, subject, body)
	return err
}
