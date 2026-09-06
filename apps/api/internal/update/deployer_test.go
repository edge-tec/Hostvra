package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func createTestArchive(files map[string]string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0755,
			Size: int64(len(content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte(content))
	}

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func TestAtomicReleaseDeploymentAndRollback(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "hostvra-deployer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempBase)

	deployer := NewReleaseDeployer(tempBase)
	ctx := context.Background()

	// 1. Stage Version 1.0.0
	archiveV1 := createTestArchive(map[string]string{
		"bin/hostvra-api":   "echo api v1.0.0",
		"bin/hostvra-agent": "echo agent v1.0.0",
	})
	pathV1, err := deployer.StageRelease(ctx, "1.0.0", bytes.NewReader(archiveV1))
	if err != nil {
		t.Fatalf("failed to stage v1.0.0: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pathV1, "bin/hostvra-api")); err != nil {
		t.Fatalf("expected binary in staged release: %v", err)
	}

	// Activate Version 1.0.0
	if err := deployer.ActivateRelease(ctx, "1.0.0"); err != nil {
		t.Fatalf("failed to activate v1.0.0: %v", err)
	}
	activeV1, err := deployer.GetCurrentActiveVersion(ctx)
	if err != nil || activeV1 != "1.0.0" {
		t.Errorf("expected active version 1.0.0, got %s (err: %v)", activeV1, err)
	}

	// 2. Stage Version 1.1.0 (without destroying v1.0.0!)
	archiveV2 := createTestArchive(map[string]string{
		"bin/hostvra-api":   "echo api v1.1.0",
		"bin/hostvra-agent": "echo agent v1.1.0",
	})
	_, errStage2 := deployer.StageRelease(ctx, "1.1.0", bytes.NewReader(archiveV2))
	if errStage2 != nil {
		t.Fatalf("failed to stage v1.1.0: %v", errStage2)
	}

	// Check that v1.0.0 binary is still completely intact
	if _, err := os.Stat(filepath.Join(pathV1, "bin/hostvra-api")); err != nil {
		t.Fatalf("v1.0.0 binary was damaged during v1.1.0 staging: %v", err)
	}

	// 3. Atomically Activate Version 1.1.0
	if err := deployer.ActivateRelease(ctx, "1.1.0"); err != nil {
		t.Fatalf("failed to activate v1.1.0: %v", err)
	}
	activeV2, err := deployer.GetCurrentActiveVersion(ctx)
	if err != nil || activeV2 != "1.1.0" {
		t.Errorf("expected active version 1.1.0, got %s", activeV2)
	}

	// 4. Instant Zero-Downtime Rollback to Version 1.0.0
	if err := deployer.RollbackRelease(ctx, "1.0.0"); err != nil {
		t.Fatalf("failed to rollback to v1.0.0: %v", err)
	}
	activeRollback, err := deployer.GetCurrentActiveVersion(ctx)
	if err != nil || activeRollback != "1.0.0" {
		t.Errorf("expected active version 1.0.0 after rollback, got %s", activeRollback)
	}
}

func TestPathTraversalRejection(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "hostvra-traversal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempBase)

	deployer := NewReleaseDeployer(tempBase)
	ctx := context.Background()

	// Malicious archive trying to write to ../../etc/shadow
	maliciousArchive := createTestArchive(map[string]string{
		"../../../../../../etc/shadow": "root:passwordhash",
	})

	_, errStage := deployer.StageRelease(ctx, "1.1.0", bytes.NewReader(maliciousArchive))
	if errStage == nil || !errors.Is(errStage, ErrPathTraversalAttack) {
		t.Errorf("expected ErrPathTraversalAttack on malicious archive, got: %v", errStage)
	}
}
