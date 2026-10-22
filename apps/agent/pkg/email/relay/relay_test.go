package relay

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestRelay_PresetsAndPostmaps(t *testing.T) {
	// 1. Verify preset providers
	for _, p := range []string{"amazon_ses", "sendgrid", "mailgun", "brevo", "postmark"} {
		preset, ok := SupportedProviders[p]
		if !ok {
			t.Errorf("expected preset for %s", p)
		}
		if preset.DefaultHost == "" {
			t.Errorf("expected host for %s", p)
		}
		if preset.SpfInclude == "" {
			t.Errorf("expected spf include for %s", p)
		}
	}

	// 2. Generate sender dependent relayhost
	entries := map[string]string{
		"mycorp.com":  "smtp.sendgrid.net",
		"example.org": "email-smtp.us-east-1.amazonaws.com",
	}
	content := GenerateSenderDependentRelayHost(entries)
	if !strings.Contains(content, "@mycorp.com\t[smtp.sendgrid.net]:587") {
		t.Errorf("unexpected content: %s", content)
	}

	// 3. Generate SASL passwd
	creds := map[string]struct{ User, Pass string }{
		"smtp.sendgrid.net": {User: "apikey", Pass: "SG.secret"},
	}
	sasl := GenerateSaslPasswd(creds)
	if !strings.Contains(sasl, "[smtp.sendgrid.net]:587\tapikey:SG.secret") {
		t.Errorf("unexpected sasl: %s", sasl)
	}
}

func TestRelay_MockSMTPServerProbe(t *testing.T) {
	// Spin up local mock SMTP server to verify TestRelayConnection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		conn.Write([]byte("220 mail.mockrelay.com ESMTP Hostvra Mock\r\n"))

		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}
			req := string(buf[:n])
			if strings.HasPrefix(req, "EHLO") {
				conn.Write([]byte("250-mail.mockrelay.com\r\n250-PIPELINING\r\n250-SIZE 52428800\r\n250 AUTH PLAIN LOGIN\r\n"))
			} else if strings.HasPrefix(req, "AUTH PLAIN") {
				conn.Write([]byte("235 2.7.0 Authentication successful\r\n"))
			} else if strings.HasPrefix(req, "QUIT") {
				conn.Write([]byte("221 2.0.0 Bye\r\n"))
				break
			}
		}
	}()

	res := TestRelayConnection(context.Background(), "127.0.0.1", port, "testuser", "testpass", "none", "plain")
	if !res.Success {
		t.Fatalf("expected probe success, got error: %s", res.Error)
	}
	if !res.AuthSuccess {
		t.Errorf("expected auth success")
	}
	if res.Banner == "" {
		t.Errorf("expected banner")
	}
}
