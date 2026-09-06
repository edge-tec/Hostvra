package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"
)

// MockHealthProber allows controlling health check outcomes in tests
type MockHealthProber struct {
	shouldFail bool
}

func (m *MockHealthProber) ProbeHealth(ctx context.Context) (*SystemHealthReport, error) {
	if m.shouldFail {
		return &SystemHealthReport{
			OverallStatus: HealthCritical,
			FailureReason: "Simulated post-activation crash",
			CheckedAt:     time.Now().UTC(),
		}, errors.New("simulated crash")
	}
	return &SystemHealthReport{
		OverallStatus: HealthOK,
		APIPassed:     true,
		DBPassed:      true,
		AgentPassed:   true,
		SmokeTests:    map[string]bool{"api": true, "web": true},
		CheckedAt:     time.Now().UTC(),
	}, nil
}

func createSignedTestRelease(version string, privKey ed25519.PrivateKey) ([]byte, *ReleaseMetadata) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("binary content for " + version)
	hdr := &tar.Header{
		Name: "bin/hostvra-api",
		Mode: 0755,
		Size: int64(len(content)),
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gw.Close()

	packageData := buf.Bytes()
	hasher := sha256.New()
	hasher.Write(packageData)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	sig := ed25519.Sign(privKey, packageData)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	manifest := &ReleaseMetadata{
		Version:             version,
		Channel:             ChannelStable,
		MinSupportedVersion: "1.0.0",
		SHA256Checksum:      checksum,
		Ed25519Signature:    sigB64,
		ArchCompatibility:   []string{"amd64"},
		OSCompatibility:     []string{"ubuntu"},
		ReleasedAt:          time.Now().UTC(),
	}

	return packageData, manifest
}

func TestUpdateOrchestratorSuccessFlow(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "hostvra-orch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempBase)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	verifier := NewPackageVerifier(pubKey)
	repo := NewMemoryJobRepository()
	engine := NewJobEngine(repo)
	snapshot := NewSnapshotManager(tempBase + "/backups")
	deployer := NewReleaseDeployer(tempBase)
	prober := &MockHealthProber{shouldFail: false}

	orchestrator := NewUpdateOrchestrator(engine, verifier, snapshot, deployer, nil, prober)
	ctx := context.Background()

	// Initial staging of version 1.0.0
	pkgV1, _ := createSignedTestRelease("1.0.0", privKey)
	_, _ = deployer.StageRelease(ctx, "1.0.0", bytes.NewReader(pkgV1))
	_ = deployer.ActivateRelease(ctx, "1.0.0")

	// Start update job to 1.1.0
	job, err := engine.StartJob(ctx, "1.1.0", "1.0.0", ChannelStable, "bundle", nil)
	if err != nil {
		t.Fatalf("failed to start job: %v", err)
	}

	pkgV2, manifestV2 := createSignedTestRelease("1.1.0", privKey)
	configs := map[string][]byte{"/etc/hostvra/api.env": []byte("PORT=8080")}

	err = orchestrator.ExecuteLiveUpdate(ctx, job, manifestV2, pkgV2, configs, "ubuntu-22.04", "amd64")
	if err != nil {
		t.Fatalf("ExecuteLiveUpdate failed: %v", err)
	}

	if job.Status != StatusCompleted {
		t.Errorf("expected final status COMPLETED, got %s", job.Status)
	}

	activeVer, _ := deployer.GetCurrentActiveVersion(ctx)
	if activeVer != "1.1.0" {
		t.Errorf("expected active version 1.1.0, got %s", activeVer)
	}
}

func TestUpdateOrchestratorAutomaticRollbackOnCrash(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "hostvra-orch-rollback-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempBase)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	verifier := NewPackageVerifier(pubKey)
	repo := NewMemoryJobRepository()
	engine := NewJobEngine(repo)
	snapshot := NewSnapshotManager(tempBase + "/backups")
	deployer := NewReleaseDeployer(tempBase)

	// Injected failure during health probe
	failingProber := &MockHealthProber{shouldFail: true}

	orchestrator := NewUpdateOrchestrator(engine, verifier, snapshot, deployer, nil, failingProber)
	ctx := context.Background()

	// Initial staging of working version 1.0.0
	pkgV1, _ := createSignedTestRelease("1.0.0", privKey)
	_, _ = deployer.StageRelease(ctx, "1.0.0", bytes.NewReader(pkgV1))
	_ = deployer.ActivateRelease(ctx, "1.0.0")

	// Start update to 1.1.0 (which will fail health check)
	job, _ := engine.StartJob(ctx, "1.1.0", "1.0.0", ChannelStable, "bundle", nil)
	pkgV2, manifestV2 := createSignedTestRelease("1.1.0", privKey)
	configs := map[string][]byte{"/etc/hostvra/api.env": []byte("PORT=8080")}

	err = orchestrator.ExecuteLiveUpdate(ctx, job, manifestV2, pkgV2, configs, "ubuntu-22.04", "amd64")
	if err == nil {
		t.Errorf("expected error during health check failure, got nil")
	}

	// Verify that job status was marked ROLLED_BACK
	if job.Status != StatusRolledBack {
		t.Errorf("expected job status ROLLED_BACK, got %s", job.Status)
	}

	// Verify that /opt/hostvra/current was automatically restored back to 1.0.0!
	activeVer, _ := deployer.GetCurrentActiveVersion(ctx)
	if activeVer != "1.0.0" {
		t.Errorf("expected active version restored back to 1.0.0, got %s", activeVer)
	}
}
