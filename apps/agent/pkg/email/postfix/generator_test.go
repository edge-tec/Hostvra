package postfix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMainCF_AntiOpenRelay(t *testing.T) {
	opts := ConfigOptions{
		Hostname:     "mail.example.com",
		Domain:       "example.com",
		MailDirBase:  "/var/mail/vhosts",
		EnableRspamd: true,
	}

	conf := GenerateMainCF(opts)

	// Verify Open Relay Protections
	if !strings.Contains(conf, "smtpd_relay_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination") {
		t.Errorf("CRITICAL SECURITY FLAW: smtpd_relay_restrictions missing or insecure")
	}

	if !strings.Contains(conf, "smtpd_recipient_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination") {
		t.Errorf("CRITICAL SECURITY FLAW: smtpd_recipient_restrictions missing reject_unauth_destination")
	}

	// Verify TLS and SASL requirements
	if !strings.Contains(conf, "smtpd_sasl_type = dovecot") {
		t.Errorf("expected Dovecot SASL backend")
	}
	if !strings.Contains(conf, "smtpd_tls_protocols = !SSLv2, !SSLv3, !TLSv1, !TLSv1.1") {
		t.Errorf("insecure SSL/TLS protocols allowed")
	}

	// Verify LMTP virtual transport
	if !strings.Contains(conf, "virtual_transport = lmtp:unix:private/dovecot-lmtp") {
		t.Errorf("missing LMTP delivery to Dovecot")
	}
}

func TestGenerateMasterCF_SubmissionAndSMTPS(t *testing.T) {
	master := GenerateMasterCF()

	// Port 587 Submission
	if !strings.Contains(master, "submission inet") {
		t.Errorf("missing submission port 587 configuration")
	}
	if !strings.Contains(master, "smtpd_tls_security_level=encrypt") {
		t.Errorf("submission port must enforce TLS encryption")
	}
	if !strings.Contains(master, "smtpd_sasl_auth_enable=yes") {
		t.Errorf("submission port must enable SASL auth")
	}
	if !strings.Contains(master, "smtpd_relay_restrictions=permit_sasl_authenticated,reject") {
		t.Errorf("submission port must only allow authenticated relay")
	}

	// Port 465 SMTPS
	if !strings.Contains(master, "smtps     inet") {
		t.Errorf("missing SMTPS port 465 configuration")
	}
	if !strings.Contains(master, "smtpd_tls_wrappermode=yes") {
		t.Errorf("SMTPS must use wrappermode")
	}
}

func TestGenerateMapsAndApply(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hostvra-postfix-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	domains := []VirtualDomain{{Domain: "example.com"}, {Domain: "test.org"}}
	mboxes := []VirtualMailbox{
		{Email: "info@example.com", MailPath: "example.com/info/"},
		{Email: "admin@test.org", MailPath: "test.org/admin/"},
	}
	aliases := []VirtualAlias{
		{SourceAddress: "support@example.com", DestinationAddress: "info@example.com"},
	}

	if err := ApplyMaps(tmpDir, domains, mboxes, aliases); err != nil {
		t.Fatalf("ApplyMaps failed: %v", err)
	}

	// Check files created
	vdomains, err := os.ReadFile(filepath.Join(tmpDir, "vdomains"))
	if err != nil || !strings.Contains(string(vdomains), "example.com OK") {
		t.Errorf("vdomains content incorrect: %v", string(vdomains))
	}

	vmailbox, err := os.ReadFile(filepath.Join(tmpDir, "vmailbox"))
	if err != nil || !strings.Contains(string(vmailbox), "info@example.com example.com/info/") {
		t.Errorf("vmailbox content incorrect: %v", string(vmailbox))
	}

	valias, err := os.ReadFile(filepath.Join(tmpDir, "valias"))
	if err != nil || !strings.Contains(string(valias), "support@example.com info@example.com") {
		t.Errorf("valias content incorrect: %v", string(valias))
	}
}
