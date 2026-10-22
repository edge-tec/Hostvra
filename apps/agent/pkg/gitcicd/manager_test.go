package gitcicd

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitCICD_Manager(t *testing.T) {
	mgr := NewManager()

	// 1. Test Deploy KeyPair Generation
	keypair, err := mgr.GenerateDeployKeyPair("hostvra-deploy-test")
	if err != nil {
		t.Fatalf("failed to generate deploy keypair: %v", err)
	}
	if !strings.HasPrefix(keypair.PublicKey, "ssh-ed25519 ") {
		t.Errorf("expected ssh-ed25519 public key, got %s", keypair.PublicKey)
	}
	if !strings.Contains(keypair.PrivateKey, "BEGIN OPENSSH PRIVATE KEY") {
		t.Errorf("expected OpenSSH private key PEM")
	}

	// 2. Test Webhook Verification
	secret := "super-secure-webhook-secret-12345"
	payload := []byte(`{"ref":"refs/heads/main","after":"c0ffee12345"}`)

	// GitHub style sha256=HEX(HMAC)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	ghSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !mgr.VerifyWebhookSignature("github", ghSig, secret, payload) {
		t.Errorf("expected github webhook verification to pass")
	}
	if mgr.VerifyWebhookSignature("github", "sha256=invalidhash", secret, payload) {
		t.Errorf("expected invalid github webhook to fail")
	}

	// GitLab style token
	if !mgr.VerifyWebhookSignature("gitlab", secret, secret, payload) {
		t.Errorf("expected gitlab webhook verification to pass")
	}

	// 3. Test Atomic Deployment Pipeline
	tempDir := t.TempDir()
	deployRoot := filepath.Join(tempDir, "www", "myapp")

	cfg := &DeployConfig{
		WebsiteID:    "site-12345",
		RepoURL:      "https://github.com/hostvra/sample-app.git",
		Branch:       "main",
		DeployPath:   deployRoot,
		BuildCommand: "echo 'Build successful'",
		KeepReleases: 2,
	}

	// Release 1
	res1, err := mgr.DeployRelease(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed deploy release 1: %v", err)
	}
	if res1.Status != "success" {
		t.Errorf("expected status success, got %s", res1.Status)
	}

	// Verify current symlink exists and points to release 1
	currentSymlink := filepath.Join(deployRoot, "current")
	target1, err := os.Readlink(currentSymlink)
	if err != nil || target1 != res1.ReleasePath {
		t.Fatalf("expected symlink to point to %s, got %s (err: %v)", res1.ReleasePath, target1, err)
	}

	// Wait 1 second and deploy Release 2
	time.Sleep(1 * time.Second)
	res2, err := mgr.DeployRelease(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed deploy release 2: %v", err)
	}

	target2, _ := os.Readlink(currentSymlink)
	if target2 != res2.ReleasePath {
		t.Errorf("expected symlink to point to %s, got %s", res2.ReleasePath, target2)
	}

	// 4. Test Rollback to Release 1
	if err := mgr.RollbackRelease(deployRoot, res1.ReleaseID); err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}
	targetRollback, _ := os.Readlink(currentSymlink)
	if targetRollback != res1.ReleasePath {
		t.Errorf("expected rollback symlink to point to %s, got %s", res1.ReleasePath, targetRollback)
	}
}
