package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
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

// ResolvedSMTP holds resolved SMTP configuration from DB or env or local Postfix
type ResolvedSMTP struct {
	Host       string
	Port       int
	User       string
	Password   string
	From       string
	FromName   string
	Encryption string // "tls", "starttls", "none"
}

// GetResolvedConfig determines active SMTP configuration with intelligent priority fallback
func (s *Service) GetResolvedConfig(ctx context.Context) ResolvedSMTP {
	res := ResolvedSMTP{
		Port:       587,
		From:       "noreply@hostvra.com",
		FromName:   "Hostvra",
		Encryption: "starttls",
	}

	// 1. Check dynamic DB SystemSettings first (configured via Admin UI)
	if s.store != nil {
		if settings, err := s.store.GetSystemSettings(ctx); err == nil && settings != nil {
			if settings.SMTPHost != "" {
				res.Host = strings.TrimSpace(settings.SMTPHost)
				if settings.SMTPPort > 0 {
					res.Port = settings.SMTPPort
				}
				res.User = strings.TrimSpace(settings.SMTPUser)
				res.Password = settings.SMTPPassword
				if settings.SMTPFrom != "" {
					res.From = strings.TrimSpace(settings.SMTPFrom)
				}
				if settings.SMTPFromName != "" {
					res.FromName = strings.TrimSpace(settings.SMTPFromName)
				}
				if settings.SMTPEncryption != "" {
					res.Encryption = strings.ToLower(strings.TrimSpace(settings.SMTPEncryption))
				} else if res.Port == 465 {
					res.Encryption = "tls"
				}
				return res
			}
		}
	}

	// 2. Check static environment config (.env / system environment variables)
	if s.cfg != nil && s.cfg.SMTPHost != "" {
		res.Host = strings.TrimSpace(s.cfg.SMTPHost)
		if s.cfg.SMTPPort > 0 {
			res.Port = s.cfg.SMTPPort
		}
		res.User = strings.TrimSpace(s.cfg.SMTPUser)
		res.Password = s.cfg.SMTPPassword
		if s.cfg.SMTPFrom != "" {
			res.From = strings.TrimSpace(s.cfg.SMTPFrom)
		}
		if s.cfg.SMTPFromName != "" {
			res.FromName = strings.TrimSpace(s.cfg.SMTPFromName)
		}
		if res.Port == 465 {
			res.Encryption = "tls"
		}
		return res
	}

	// 3. Fallback: check if local Postfix is listening on 127.0.0.1:25
	conn, err := net.DialTimeout("tcp", "127.0.0.1:25", 250*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		res.Host = "127.0.0.1"
		res.Port = 25
		res.Encryption = "none"
		res.User = ""
		res.Password = ""
		return res
	}

	return res
}

// IsConfigured returns true if SMTP is configured or local Postfix is ready for transmission
func (s *Service) IsConfigured() bool {
	cfg := s.GetResolvedConfig(context.Background())
	return cfg.Host != "" && cfg.From != ""
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

// loginAuth implements SMTP LOGIN authentication scheme
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch strings.ToLower(string(fromServer)) {
		case "username:", "user:":
			return []byte(a.username), nil
		case "password:", "pass:":
			return []byte(a.password), nil
		default:
			return []byte(a.password), nil
		}
	}
	return nil, nil
}

// DeliverViaSMTP sends email using precise encryption negotiation (TLS 465, STARTTLS 587, or Plain 25)
func DeliverViaSMTP(ctx context.Context, cfg ResolvedSMTP, to, subject, htmlBody string) error {
	if cfg.Host == "" {
		return errors.New("no SMTP host configured")
	}

	from := cfg.From
	if from == "" {
		from = "noreply@hostvra.com"
	}
	fromName := cfg.FromName
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
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// Mode A: Direct TLS/SSL (Standard for Port 465)
	if cfg.Encryption == "tls" || cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: cfg.Host,
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connection to %s failed: %w", addr, err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf("SMTP client initialization failed: %w", err)
		}
		defer client.Close()

		if cfg.User != "" {
			auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
			if err := client.Auth(auth); err != nil {
				// Retry with LOGIN auth if PLAIN is rejected
				if lErr := client.Auth(LoginAuth(cfg.User, cfg.Password)); lErr != nil {
					return fmt.Errorf("SMTP authentication failed: %w (login auth: %v)", err, lErr)
				}
			}
		}

		if err := client.Mail(from); err != nil {
			return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA command failed: %w", err)
		}
		if _, err := w.Write(msg); err != nil {
			return fmt.Errorf("SMTP message write failed: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("SMTP data close failed: %w", err)
		}
		return client.Quit()
	}

	// Mode B: STARTTLS (Standard for Port 587) or Plain (Port 25)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP connection to %s failed: %w", addr, err)
	}
	defer rawConn.Close()

	client, err := smtp.NewClient(rawConn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP handshake with %s failed: %w", addr, err)
	}
	defer client.Close()

	if cfg.Encryption != "none" && cfg.Port != 25 {
		if hasStartTLS, _ := client.Extension("STARTTLS"); hasStartTLS {
			tlsConfig := &tls.Config{
				ServerName: cfg.Host,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS upgrade failed: %w", err)
			}
		}
	}

	if cfg.User != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			if lErr := client.Auth(LoginAuth(cfg.User, cfg.Password)); lErr != nil {
				return fmt.Errorf("SMTP authentication failed: %w (login auth: %v)", err, lErr)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP message write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP data close failed: %w", err)
	}
	return client.Quit()
}

// sendSMTP sends an email using dynamic resolution
func (s *Service) sendSMTP(ctx context.Context, to, subject, htmlBody string) error {
	resolved := s.GetResolvedConfig(ctx)
	if resolved.Host == "" {
		slog.Warn("SMTP not configured — email simulated", "to", to, "subject", subject)
		return nil
	}

	return DeliverViaSMTP(ctx, resolved, to, subject, htmlBody)
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
	if err := s.sendSMTP(ctx, to, subject, htmlBody); err != nil {
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

// SendWelcomeAccountCreated sends a welcome email when a user registers or is provisioned
func (s *Service) SendWelcomeAccountCreated(ctx context.Context, to, fullName, email, loginURL string) error {
	subject := "🎉 Welcome to Hostvra — Your Account is Ready!"
	if loginURL == "" {
		loginURL = s.appURL() + "/login"
	}
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 580px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #e2e8f0; box-shadow: 0 4px 12px rgba(0,0,0,0.05);">
  <div style="text-align: center; margin-bottom: 24px;">
    <div style="width: 52px; height: 52px; background: #16a34a; border-radius: 14px; display: inline-flex; align-items: center; justify-content: center; margin-bottom: 12px;">
      <span style="color: white; font-size: 26px; font-weight: bold;">H</span>
    </div>
    <h1 style="font-size: 22px; font-weight: 800; color: #0f172a; margin: 0;">Welcome to Hostvra, %s!</h1>
    <p style="color: #64748b; font-size: 14px; margin-top: 6px;">Your high-performance cloud hosting control account has been created.</p>
  </div>

  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 12px; padding: 20px; margin-bottom: 24px;">
    <div style="font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: #64748b; margin-bottom: 12px;">Account Credentials</div>
    <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
      <tr><td style="padding: 6px 0; color: #64748b;">Registered Email:</td><td style="padding: 6px 0; text-align: right; font-weight: 700; color: #0f172a;">%s</td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">Control Panel URL:</td><td style="padding: 6px 0; text-align: right; font-weight: 600; color: #2563eb;"><a href="%s" style="color: #2563eb; text-decoration: none;">%s</a></td></tr>
      <tr><td style="padding: 6px 0; color: #64748b;">Security:</td><td style="padding: 6px 0; text-align: right; font-weight: 600; color: #16a34a;">cgroups v2 &amp; WAF Isolated</td></tr>
    </table>
  </div>

  <div style="text-align: center; margin-bottom: 24px;">
    <a href="%s" style="display: inline-block; background: #16a34a; color: white; padding: 14px 36px; border-radius: 10px; text-decoration: none; font-weight: 700; font-size: 14px; box-shadow: 0 4px 10px rgba(22,163,74,0.25);">
      Log In to Control Panel
    </a>
  </div>

  <div style="background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 12px; padding: 18px; margin-bottom: 20px;">
    <h3 style="font-size: 13px; font-weight: 700; color: #1e40af; margin: 0 0 8px 0;">What you can do with Hostvra:</h3>
    <ul style="margin: 0; padding-left: 20px; font-size: 13px; color: #1e3a8a; line-height: 1.6;">
      <li>Deploy and manage ultra-fast Nginx PHP, Node.js &amp; Python websites</li>
      <li>One-Click SSL certificates with automatic 90-day Let's Encrypt renewal</li>
      <li>Full corporate email hosting with SpamAssassin &amp; FCrDNS deliverability</li>
      <li>Zero 3rd-party fee in-house Crypto &amp; payment gateway management</li>
    </ul>
  </div>

  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
  <p style="color: #94a3b8; font-size: 12px; text-align: center; margin: 0;">Hostvra Cloud Control Plane • support@hostvra.com</p>
</div>
</body>
</html>`, fullName, email, loginURL, loginURL, loginURL)

	eventKey := fmt.Sprintf("welcome:%s:%s", to, time.Now().UTC().Format("2006-01-02"))
	_, err := s.SendIdempotent(ctx, "account_created", eventKey, to, subject, body)
	return err
}

// SendPaymentFailed notifies the customer of a failed or declined payment attempt
func (s *Service) SendPaymentFailed(ctx context.Context, to, fullName, invoiceNumber, amount, currency, reason, retryURL string) error {
	subject := fmt.Sprintf("⚠️ Payment Failed — Invoice %s", invoiceNumber)
	if retryURL == "" {
		retryURL = s.appURL() + "/billing"
	}
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #fee2e2; box-shadow: 0 4px 12px rgba(220,38,38,0.06);">
  <h1 style="font-size: 20px; font-weight: 700; color: #dc2626; margin-bottom: 12px;">⚠️ Payment Processing Failed</h1>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    We were unable to process your payment for Invoice <strong>%s</strong>.
  </p>
  <table style="width: 100%%; border-collapse: collapse; margin: 16px 0; background: #fef2f2; border-radius: 8px; padding: 12px; font-size: 13px;">
    <tr><td style="padding: 6px 12px; color: #991b1b;">Invoice Total:</td><td style="padding: 6px 12px; text-align: right; font-weight: 700; color: #991b1b;">%s %s</td></tr>
    <tr><td style="padding: 6px 12px; color: #991b1b;">Decline Reason:</td><td style="padding: 6px 12px; text-align: right; font-weight: 600; color: #dc2626;">%s</td></tr>
  </table>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    To prevent service disruption or hosting suspension, please retry payment with another payment method (Credit Card, PayPal, or Crypto).
  </p>
  <div style="text-align: center; margin: 24px 0;">
    <a href="%s" style="display: inline-block; background: #dc2626; color: white; padding: 12px 32px; border-radius: 8px; text-decoration: none; font-weight: 700; font-size: 14px;">
      Retry Payment Now
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #fee2e2; margin: 20px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra Automated Billing Core</p>
</div>
</body>
</html>`, fullName, invoiceNumber, amount, currency, reason, retryURL)

	eventKey := fmt.Sprintf("pay_failed:%s:%s:%s", to, invoiceNumber, time.Now().UTC().Format("2006-01-02T15"))
	_, err := s.SendIdempotent(ctx, "payment_failed", eventKey, to, subject, body)
	return err
}

// SendAccountExpiringWarning sends a 7-day expiration reminder before hosting suspension
func (s *Service) SendAccountExpiringWarning(ctx context.Context, to, fullName, planName, domain string, daysRemaining int, renewURL string) error {
	subject := fmt.Sprintf("⚠️ Notice: Your Hosting Plan Expires in %d Days (%s)", daysRemaining, domain)
	if renewURL == "" {
		renewURL = s.appURL() + "/billing"
	}
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 580px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #fed7aa; box-shadow: 0 4px 12px rgba(234,88,12,0.06);">
  <div style="text-align: center; margin-bottom: 20px;">
    <div style="display: inline-block; padding: 8px 16px; background: #fff7ed; border: 1px solid #fdba74; border-radius: 20px; color: #ea580c; font-size: 12px; font-weight: 700; text-transform: uppercase;">
      Expires in %d Days
    </div>
    <h1 style="font-size: 22px; font-weight: 800; color: #0f172a; margin: 12px 0 6px 0;">Upcoming Hosting Expiration</h1>
  </div>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    This is a friendly reminder that your hosting subscription for <strong>%s</strong> (Plan: <em>%s</em>) will expire in <strong>%d days</strong>.
  </p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 12px; padding: 18px; margin: 18px 0;">
    <table style="width: 100%%; border-collapse: collapse; font-size: 13px;">
      <tr><td style="padding: 4px 0; color: #64748b;">Primary Domain:</td><td style="padding: 4px 0; text-align: right; font-weight: 700; color: #0f172a;">%s</td></tr>
      <tr><td style="padding: 4px 0; color: #64748b;">Plan Package:</td><td style="padding: 4px 0; text-align: right; font-weight: 600; color: #0f172a;">%s</td></tr>
      <tr><td style="padding: 4px 0; color: #64748b;">Status:</td><td style="padding: 4px 0; text-align: right; font-weight: 700; color: #ea580c;">Expiring Soon</td></tr>
    </table>
  </div>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    Renew now to maintain uninterrupted 100%% website uptime, SSL certificate validation, and active email inboxes.
  </p>
  <div style="text-align: center; margin: 24px 0;">
    <a href="%s" style="display: inline-block; background: #ea580c; color: white; padding: 14px 36px; border-radius: 10px; text-decoration: none; font-weight: 700; font-size: 14px; box-shadow: 0 4px 10px rgba(234,88,12,0.25);">
      Renew Hosting Subscription
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #fed7aa; margin: 20px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra Automated Billing Core</p>
</div>
</body>
</html>`, daysRemaining, fullName, domain, planName, daysRemaining, domain, planName, renewURL)

	eventKey := fmt.Sprintf("expiring_7d:%s:%s:%s", to, domain, time.Now().UTC().Format("2006-01-02"))
	_, err := s.SendIdempotent(ctx, "account_expiring_warning", eventKey, to, subject, body)
	return err
}

// SendAccountExpired notifies the customer that their account has reached expiration
func (s *Service) SendAccountExpired(ctx context.Context, to, fullName, planName, domain, reason, reactivateURL string) error {
	subject := fmt.Sprintf("❌ Service Expired: Hosting Account for %s Suspended", domain)
	if reactivateURL == "" {
		reactivateURL = s.appURL() + "/billing"
	}
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 580px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #fee2e2; box-shadow: 0 4px 12px rgba(220,38,38,0.06);">
  <div style="text-align: center; margin-bottom: 20px;">
    <div style="display: inline-block; padding: 8px 16px; background: #fef2f2; border: 1px solid #fecaca; border-radius: 20px; color: #dc2626; font-size: 12px; font-weight: 700; text-transform: uppercase;">
      Service Expired
    </div>
    <h1 style="font-size: 22px; font-weight: 800; color: #dc2626; margin: 12px 0 6px 0;">Hosting Account Expired</h1>
  </div>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    Your hosting service for <strong>%s</strong> (Plan: <em>%s</em>) has expired and is now temporarily deactivated.
  </p>
  <div style="background: #fef2f2; border: 1px solid #fecaca; border-radius: 12px; padding: 18px; margin: 18px 0; font-size: 13px; color: #991b1b;">
    <strong>Reason:</strong> %s
  </div>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    Don't worry — all your website files, databases, and emails are safely preserved during the grace period. Simply reactivate your subscription to restore all services immediately.
  </p>
  <div style="text-align: center; margin: 24px 0;">
    <a href="%s" style="display: inline-block; background: #dc2626; color: white; padding: 14px 36px; border-radius: 10px; text-decoration: none; font-weight: 700; font-size: 14px; box-shadow: 0 4px 10px rgba(220,38,38,0.25);">
      Reactivate Hosting Account
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #fee2e2; margin: 20px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra Automated Billing Core</p>
</div>
</body>
</html>`, fullName, domain, planName, reason, reactivateURL)

	eventKey := fmt.Sprintf("expired:%s:%s:%s", to, domain, time.Now().UTC().Format("2006-01-02"))
	_, err := s.SendIdempotent(ctx, "account_expired", eventKey, to, subject, body)
	return err
}

// SendSystemUpdateNotification broadcasts a system update or release announcement to a user
func (s *Service) SendSystemUpdateNotification(ctx context.Context, to, fullName, version, title, summary, changelogURL string) error {
	subject := fmt.Sprintf("🚀 Hostvra System Update: Version %s is Live!", version)
	if changelogURL == "" {
		changelogURL = s.appURL() + "/settings"
	}
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 30px;">
<div style="max-width: 580px; margin: 0 auto; background: #ffffff; border-radius: 16px; padding: 36px; border: 1px solid #bfdbfe; box-shadow: 0 4px 12px rgba(37,99,235,0.06);">
  <div style="text-align: center; margin-bottom: 24px;">
    <div style="width: 52px; height: 52px; background: #2563eb; border-radius: 14px; display: inline-flex; align-items: center; justify-content: center; margin-bottom: 12px;">
      <span style="color: white; font-size: 24px;">⚡</span>
    </div>
    <div style="display: inline-block; padding: 4px 12px; background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 20px; color: #1d4ed8; font-size: 12px; font-weight: 700; margin-bottom: 8px;">
      Hostvra Update %s
    </div>
    <h1 style="font-size: 22px; font-weight: 800; color: #0f172a; margin: 0;">%s</h1>
  </div>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">Hi %s,</p>
  <p style="color: #475569; font-size: 14px; line-height: 1.6;">
    A new update has been successfully deployed to your Hostvra server. Here is what is new:
  </p>
  <div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 12px; padding: 20px; margin: 20px 0; font-size: 14px; color: #334155; line-height: 1.6;">
    %s
  </div>
  <div style="text-align: center; margin: 24px 0;">
    <a href="%s" style="display: inline-block; background: #2563eb; color: white; padding: 14px 36px; border-radius: 10px; text-decoration: none; font-weight: 700; font-size: 14px; box-shadow: 0 4px 10px rgba(37,99,235,0.25);">
      Open Control Panel
    </a>
  </div>
  <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 20px 0;" />
  <p style="color: #94a3b8; font-size: 11px; text-align: center;">Hostvra Cloud Automation &amp; Release System</p>
</div>
</body>
</html>`, version, title, fullName, summary, changelogURL)

	eventKey := fmt.Sprintf("update:%s:%s", to, version)
	_, err := s.SendIdempotent(ctx, "system_update_broadcast", eventKey, to, subject, body)
	return err
}

// TestSMTPConnection tests an outbound SMTP connection and sends a test email
func (s *Service) TestSMTPConnection(ctx context.Context, host string, port int, user, password, from, to, encryption string) error {
	if host == "" {
		return errors.New("SMTP host cannot be empty")
	}
	if port <= 0 {
		port = 587
	}
	if from == "" {
		from = "noreply@hostvra.com"
	}
	if to == "" {
		return errors.New("recipient test email cannot be empty")
	}

	testCfg := ResolvedSMTP{
		Host:       strings.TrimSpace(host),
		Port:       port,
		User:       strings.TrimSpace(user),
		Password:   password,
		From:       strings.TrimSpace(from),
		FromName:   "Hostvra Test Mailer",
		Encryption: strings.ToLower(strings.TrimSpace(encryption)),
	}
	if testCfg.Encryption == "" {
		if testCfg.Port == 465 {
			testCfg.Encryption = "tls"
		} else {
			testCfg.Encryption = "starttls"
		}
	}

	subject := fmt.Sprintf("✅ Hostvra SMTP Test Verification (%s)", time.Now().Format("15:04:05"))
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; padding: 24px; background: #f8fafc;">
<div style="max-width: 500px; margin: auto; background: white; padding: 24px; border-radius: 12px; border: 1px solid #10b981;">
  <h2 style="color: #059669; margin-top: 0;">✓ Outbound SMTP Connection Successful!</h2>
  <p style="color: #334155;">Your Hostvra control panel has successfully authenticated and delivered a test message via:</p>
  <ul>
    <li><strong>SMTP Host:</strong> %s</li>
    <li><strong>Port:</strong> %d</li>
    <li><strong>Encryption:</strong> %s</li>
    <li><strong>Sender:</strong> %s</li>
    <li><strong>Timestamp:</strong> %s</li>
  </ul>
  <p style="color: #64748b; font-size: 12px;">Your email notifications for registrations, password resets, and invoices are now 100%% operational.</p>
</div>
</body>
</html>`, testCfg.Host, testCfg.Port, testCfg.Encryption, testCfg.From, time.Now().UTC().Format(time.RFC1123))

	return DeliverViaSMTP(ctx, testCfg, to, subject, body)
}
