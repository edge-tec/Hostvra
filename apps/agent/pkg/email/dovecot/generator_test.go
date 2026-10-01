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
	if !strings.Contains(mailConf, "mail_location = maildir:/var/mail/vhosts/%d/%n") && !strings.Contains(mailConf, "mail_driver = maildir") {
		t.Errorf("mail_location incorrect: %s", mailConf)
	}

	authConf := GenerateAuthConf()
	if !strings.Contains(authConf, "disable_plaintext_auth = yes") && !strings.Contains(authConf, "auth_allow_cleartext = no") {
		t.Errorf("expected cleartext auth restriction: %s", authConf)
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
	if !strings.Contains(hash, "$") {
		t.Errorf("expected crypt hash with $ prefix, got %s", hash)
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
	if !strings.Contains(content, "info@example.com:{BLF-CRYPT}$2") && !strings.Contains(content, "info@example.com:{SHA512-CRYPT}$6$") {
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

func TestEmptyPasswordHashProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hostvra-empty-hash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := ConfigOptions{
		MailDirBase: "/var/mail/vhosts",
		VmailUID:    5000,
		VmailGID:    5000,
		ConfigDir:   tmpDir,
	}

	// 1. Simulate existing users file on disk with a valid credential
	initialContent := "valid@example.com:{BLF-CRYPT}$2a$10$validblfhashsaltsalt:5000:5000::/var/mail/vhosts/example.com/valid::userdb_quota_rule=*:storage=5120M\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "users"), []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial users file: %v", err)
	}

	// 2. Simulate bad sync input where valid@example.com has empty password hash, plus a new account with empty hash
	accounts := []UserAccount{
		{
			Email:        "valid@example.com",
			PasswordHash: "", // Bad / empty synchronization input
			Domain:       "example.com",
			LocalPart:    "valid",
			QuotaBytes:   5 * 1024 * 1024 * 1024,
		},
		{
			Email:        "new_empty@example.com",
			PasswordHash: "", // Brand new account with empty hash
			Domain:       "example.com",
			LocalPart:    "new_empty",
			QuotaBytes:   5 * 1024 * 1024 * 1024,
		},
	}

	content := GenerateUsersFile(accounts, opts)

	// Invariant: existing valid mailbox + bad/empty sync input = existing password remains intact
	if !strings.Contains(content, "valid@example.com:{BLF-CRYPT}$2a$10$validblfhashsaltsalt") {
		t.Errorf("expected existing valid password hash to be preserved, got:\n%s", content)
	}

	// Invariant: never write empty, corrupted, or {CRYPT}: entry
	if strings.Contains(content, "{CRYPT}:") || strings.Contains(content, "new_empty@example.com") {
		t.Errorf("expected new_empty account with blank hash to be safely skipped without {CRYPT}: entry, got:\n%s", content)
	}
}

