package updater

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentUpdaterLifecycleAndRollback(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "hostvra-agent-updater-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempBase)

	updater := NewAgentUpdater(tempBase)
	ctx := context.Background()

	// 1. Stage simulated agent binary v1.0.0
	scriptV1 := "#!/bin/sh\necho 'Hostvra Agent v1.0.0'\n"
	binV1, err := updater.StageAgent(ctx, "1.0.0", bytes.NewReader([]byte(scriptV1)))
	if err != nil {
		t.Fatalf("failed to stage agent v1.0.0: %v", err)
	}
	_ = os.Chmod(binV1, 0755)

	// Probe v1.0.0
	probe1, err := updater.ProbeAgentBinary(ctx, binV1)
	if err != nil || !probe1.IsHealthy {
		t.Fatalf("probe v1.0.0 failed: %v", err)
	}

	// Activate v1.0.0
	if err := updater.ActivateAgent(ctx, "1.0.0"); err != nil {
		t.Fatalf("failed to activate agent v1.0.0: %v", err)
	}

	linkTarget, err := os.Readlink(updater.currentSymlink)
	if err != nil || filepath.Base(linkTarget) != "1.0.0" {
		t.Errorf("expected current to point to 1.0.0, got %s (err: %v)", linkTarget, err)
	}

	// 2. Stage new agent binary v1.1.0
	scriptV2 := "#!/bin/sh\necho 'Hostvra Agent v1.1.0'\n"
	binV2, err := updater.StageAgent(ctx, "1.1.0", bytes.NewReader([]byte(scriptV2)))
	if err != nil {
		t.Fatalf("failed to stage agent v1.1.0: %v", err)
	}
	_ = os.Chmod(binV2, 0755)

	// Activate v1.1.0
	if err := updater.ActivateAgent(ctx, "1.1.0"); err != nil {
		t.Fatalf("failed to activate agent v1.1.0: %v", err)
	}
	linkTargetV2, _ := os.Readlink(updater.currentSymlink)
	if filepath.Base(linkTargetV2) != "1.1.0" {
		t.Errorf("expected current to point to 1.1.0, got %s", linkTargetV2)
	}

	// 3. Rollback to v1.0.0
	if err := updater.RollbackAgent(ctx, "1.0.0"); err != nil {
		t.Fatalf("failed to rollback agent to v1.0.0: %v", err)
	}
	linkTargetRollback, _ := os.Readlink(updater.currentSymlink)
	if filepath.Base(linkTargetRollback) != "1.0.0" {
		t.Errorf("expected current to point back to 1.0.0, got %s", linkTargetRollback)
	}
}

func TestCorruptAgentBinaryProbeRejection(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "hostvra-agent-corrupt-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempBase)

	updater := NewAgentUpdater(tempBase)
	ctx := context.Background()

	// A binary that exits with error code 1
	brokenScript := "#!/bin/sh\nexit 1\n"
	brokenBin, err := updater.StageAgent(ctx, "1.2.0", bytes.NewReader([]byte(brokenScript)))
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	_ = os.Chmod(brokenBin, 0755)

	// Probe should detect non-zero exit and reject
	probeResult, errProbe := updater.ProbeAgentBinary(ctx, brokenBin)
	if errProbe == nil || (probeResult != nil && probeResult.IsHealthy) {
		t.Errorf("expected broken agent probe to be rejected, got healthy")
	}
}
