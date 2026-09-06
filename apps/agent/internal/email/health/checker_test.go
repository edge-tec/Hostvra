package health

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestAuditDomain_Structure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test non-existent domain audit returns structured recommendations
	audit := AuditDomain(ctx, "nonexistent-domain-test-12345.xyz", "default")
	if audit.Domain != "nonexistent-domain-test-12345.xyz" {
		t.Errorf("domain mismatch: %s", audit.Domain)
	}

	if audit.MX.Status != "fail" && audit.MX.Status != "warn" {
		t.Errorf("expected fail/warn for non-existent domain MX")
	}

	if len(audit.Recommendations) == 0 {
		t.Errorf("expected recommendations for unconfigured domain")
	}

	if audit.OpenRelay.Status != "pass" {
		t.Errorf("expected open relay status pass")
	}
}

func TestTestRelayRejection_MockServer(t *testing.T) {
	// Start mock SMTP server that correctly rejects unauthorized relay with 554
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, _ = conn.Write([]byte("220 mail.hostvra.com ESMTP Postfix\r\n"))
		buf := make([]byte, 1024)
		// HELO
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("250 mail.hostvra.com\r\n"))
		// MAIL FROM
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("250 2.1.0 Ok\r\n"))
		// RCPT TO -> Reject relay!
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("554 5.7.1 <victim@target.org>: Relay access denied\r\n"))
	}()

	rejected, err := TestRelayRejection(l.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rejected {
		t.Errorf("expected relay to be reported as safely rejected")
	}
}

func TestTestRelayRejection_OpenRelayDetection(t *testing.T) {
	// Start mock server that dangerously accepts relay (250 Ok)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, _ = conn.Write([]byte("220 vulnerable.relay.com ESMTP\r\n"))
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("250 vulnerable.relay.com\r\n"))
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("250 2.1.0 Ok\r\n"))
		// Dangerous open relay response!
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("250 2.1.5 Ok\r\n"))
	}()

	rejected, err := TestRelayRejection(l.Addr().String(), 2*time.Second)
	if rejected || err == nil || !strings.Contains(err.Error(), "CRITICAL") {
		t.Errorf("expected open relay to be detected and return critical error, got rejected=%v, err=%v", rejected, err)
	}
}
