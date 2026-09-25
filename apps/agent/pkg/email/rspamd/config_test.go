package rspamd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRspamdConfigs(t *testing.T) {
	opts := Options{
		SpamAddHeader:  6.0,
		SpamReject:     15.0,
		EnableClamAV:   true,
		TotalHostRAMMB: 4096, // 4GB RAM -> ClamAV allowed
	}

	actions := GenerateActionsConf(opts)
	if !strings.Contains(actions, "reject = 15.0;") || !strings.Contains(actions, "add_header = 6.0;") {
		t.Errorf("actions.conf invalid: %s", actions)
	}

	dkim := GenerateDKIMSigningConf(opts)
	if !strings.Contains(dkim, "$domain/$selector.private") {
		t.Errorf("dkim_signing.conf invalid: %s", dkim)
	}

	av := GenerateAntivirusConf(opts)
	if !strings.Contains(av, "symbol = \"CLAM_VIRUS\";") {
		t.Errorf("antivirus.conf should be enabled for 4GB server")
	}

	// Test RAM safeguard: server with only 1GB RAM should have ClamAV disabled
	lowRAMOpts := Options{
		EnableClamAV:   true,
		TotalHostRAMMB: 1024,
	}
	lowRAMAv := GenerateAntivirusConf(lowRAMOpts)
	if !strings.Contains(lowRAMAv, "enabled = false;") {
		t.Errorf("antivirus should be disabled when RAM is below 2GB to prevent OOM")
	}

	tmpDir, err := os.MkdirTemp("", "hostvra-rspamd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ApplyRspamdConfig(tmpDir, opts); err != nil {
		t.Fatalf("ApplyRspamdConfig failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "actions.conf")); err != nil {
		t.Errorf("actions.conf file missing")
	}
}
