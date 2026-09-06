package webmail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRoundcubeConfig(t *testing.T) {
	opts := ConfigOptions{
		Domain:   "example.com",
		IMAPHost: "ssl://127.0.0.1",
		IMAPPort: 993,
		SMTPHost: "tls://127.0.0.1",
		SMTPPort: 587,
		DesKey:   "123456789012345678901234",
	}

	conf := GenerateRoundcubeConfig(opts)
	if !strings.Contains(conf, "$config['default_port'] = 993;") {
		t.Errorf("expected default_port 993, got: %s", conf)
	}
	if !strings.Contains(conf, "$config['smtp_port'] = 587;") {
		t.Errorf("expected smtp_port 587, got: %s", conf)
	}
	if !strings.Contains(conf, "$config['smtp_user'] = '%u';") {
		t.Errorf("expected smtp_user macro %%u")
	}

	vhost := GenerateWebmailNginxVHost(opts)
	if !strings.Contains(vhost, "server_name webmail.example.com;") {
		t.Errorf("expected server_name webmail.example.com, got: %s", vhost)
	}
	if !strings.Contains(vhost, "location ~ ^/(bin|config|logs|temp)/") {
		t.Errorf("missing protection for sensitive roundcube directories")
	}

	tmpDir, err := os.MkdirTemp("", "hostvra-roundcube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ApplyWebmailConfig(tmpDir, opts); err != nil {
		t.Fatalf("ApplyWebmailConfig failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "config.inc.php")); err != nil {
		t.Errorf("config.inc.php file missing")
	}
}
