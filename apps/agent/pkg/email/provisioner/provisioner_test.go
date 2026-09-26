package provisioner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunPreflightChecks(t *testing.T) {
	// Preflight check with a valid FQDN and high ports (to avoid binding privilege issues in test)
	res := RunPreflightChecks("mail.example.org", []int{15870, 15993})

	if res.Hostname != "mail.example.org" {
		t.Errorf("expected mail.example.org, got %s", res.Hostname)
	}

	if len(res.Checks) == 0 {
		t.Fatal("expected preflight checks to contain results")
	}

	foundFQDN := false
	for _, c := range res.Checks {
		if c.Name == "Mail Hostname FQDN" {
			foundFQDN = true
			if c.Status != StatusPassed {
				t.Errorf("expected FQDN check to pass, got %s", c.Status)
			}
		}
	}

	if !foundFQDN {
		t.Error("expected Mail Hostname FQDN check to be present")
	}
}

func TestApplyConfigAtomically(t *testing.T) {
	tmpDir := t.TempDir()
	logCollector := make([]string, 0)
	logger := func(format string, args ...any) {
		logCollector = append(logCollector, format)
	}

	// 1. Initial write
	content1 := "my_config_v1 = true\n"
	err := applyConfigAtomically(tmpDir, "test.cf", content1, "", "", logger)
	if err != nil {
		t.Fatalf("failed to apply config atomically: %v", err)
	}

	read1, err := os.ReadFile(filepath.Join(tmpDir, "test.cf"))
	if err != nil || string(read1) != content1 {
		t.Fatalf("expected content1, got %s", string(read1))
	}

	// 2. Overwrite with backup
	content2 := "my_config_v2 = true\n"
	err = applyConfigAtomically(tmpDir, "test.cf", content2, "", "", logger)
	if err != nil {
		t.Fatalf("failed to overwrite config atomically: %v", err)
	}

	read2, err := os.ReadFile(filepath.Join(tmpDir, "test.cf"))
	if err != nil || string(read2) != content2 {
		t.Fatalf("expected content2, got %s", string(read2))
	}

	bak, err := os.ReadFile(filepath.Join(tmpDir, "test.cf.bak"))
	if err != nil || string(bak) != content1 {
		t.Fatalf("expected backup to contain content1, got %s", string(bak))
	}
}

func TestProvisionMailServer_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ProvisionOptions{
		Hostname:           "mail.testinfra.com",
		PrimaryDomain:      "testinfra.com",
		StorageLocation:    filepath.Join(tmpDir, "vhosts"),
		SMTPPort:           25,
		SMTPSubmissionPort: 587,
		IMAPPort:           143,
		IMAPSPort:          993,
		TLSEnabled:         true,
		SpamFilterEnabled:  true,
		AntivirusEnabled:   false,
		DKIMEnabled:        true,
		InstallPackages:    false, // Do not trigger apt/dnf during unit tests
		ConfigBaseDir:      tmpDir,
	}

	ctx := context.Background()
	res, err := ProvisionMailServer(ctx, opts)
	if err != nil {
		t.Fatalf("ProvisionMailServer returned unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false. Message: %s", res.Message)
	}

	if len(res.Logs) == 0 {
		t.Error("expected provisioning logs to be populated")
	}
}
