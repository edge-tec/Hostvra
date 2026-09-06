package dovecot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDovecotConfigurations(t *testing.T) {
	opts := ConfigOptions{
		MailDirBase: "/var/mail/vhosts",
		VmailUID:    5000,
		VmailGID:    5000,
		SSLCertPath: "/etc/ssl/certs/mail.pem",
		SSLKeyPath:  "/etc/ssl/private/mail.key",
	}

	mailConf := GenerateMailConf(opts)
	if !strings.Contains(mailConf, "mail_location = maildir:/var/mail/vhosts/%d/%n") {
		t.Errorf("mail_location incorrect: %s", mailConf)
	}

	authConf := GenerateAuthConf()
	if !strings.Contains(authConf, "disable_plaintext_auth = yes") {
		t.Errorf("expected disable_plaintext_auth = yes")
	}

	sslConf := GenerateSSLConf(opts)
	if !strings.Contains(sslConf, "ssl_min_protocol = TLSv1.2") {
		t.Errorf("expected TLSv1.2 minimum protocol")
	}

	masterConf := GenerateMasterConf()
	if !strings.Contains(masterConf, "/var/spool/postfix/private/auth") {
		t.Errorf("missing Postfix SASL auth unix socket")
	}
	if !strings.Contains(masterConf, "/var/spool/postfix/private/dovecot-lmtp") {
		t.Errorf("missing LMTP unix socket")
	}
}

func TestGenerateUsersFile(t *testing.T) {
	opts := ConfigOptions{
		MailDirBase: "/var/mail/vhosts",
		VmailUID:    5000,
		VmailGID:    5000,
	}

	hash := HashPassword("SecretPass123!")
	if !strings.HasPrefix(hash, "$6$") {
		t.Errorf("expected SHA512-CRYPT format with $6$ prefix, got %s", hash)
	}

	accounts := []UserAccount{
		{
			Email:        "info@example.com",
			PasswordHash: hash,
			Domain:       "example.com",
			LocalPart:    "info",
			QuotaBytes:   5 * 1024 * 1024 * 1024,
		},
	}

	content := GenerateUsersFile(accounts, opts)
	if !strings.Contains(content, "info@example.com:{SHA512-CRYPT}$6$") {
		t.Errorf("users line missing expected auth scheme: %s", content)
	}
	if !strings.Contains(content, "5000:5000::/var/mail/vhosts/example.com/info::userdb_quota_rule=*:storage=5120M") {
		t.Errorf("users line missing quota or uid/gid: %s", content)
	}
}

func TestApplyDovecotConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hostvra-dovecot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := ConfigOptions{
		MailDirBase: "/var/mail/vhosts",
		VmailUID:    5000,
		VmailGID:    5000,
	}
	accounts := []UserAccount{
		{
			Email:        "support@example.com",
			PasswordHash: "$6$rounds=5000$saltsalt$fakehash",
			Domain:       "example.com",
			LocalPart:    "support",
			QuotaBytes:   1024 * 1024 * 1024,
		},
	}

	if err := ApplyDovecotConfig(tmpDir, opts, accounts); err != nil {
		t.Fatalf("ApplyDovecotConfig failed: %v", err)
	}

	// Verify files written
	users, err := os.ReadFile(filepath.Join(tmpDir, "users"))
	if err != nil || !strings.Contains(string(users), "support@example.com") {
		t.Errorf("users file not written correctly: %v", string(users))
	}

	master, err := os.ReadFile(filepath.Join(tmpDir, "conf.d", "10-master.conf"))
	if err != nil || !strings.Contains(string(master), "service imap-login") {
		t.Errorf("10-master.conf not written correctly: %v", string(master))
	}
}
