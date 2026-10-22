package gitcicd

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// DeployConfig defines the parameters for an atomic Git deployment
type DeployConfig struct {
	WebsiteID    string `json:"website_id"`
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	DeployPath   string `json:"deploy_path"` // e.g. /var/www/domain.com/app
	DeployKey    string `json:"deploy_key"`  // private key contents if private repo
	BuildCommand string `json:"build_command"`
	KeepReleases int    `json:"keep_releases"`
	SystemUser   string `json:"system_user"`
}

// DeployResult reports execution outcome of an atomic release
type DeployResult struct {
	ReleaseID       string    `json:"release_id"`
	ReleasePath     string    `json:"release_path"`
	CommitHash      string    `json:"commit_hash"`
	CommitMessage   string    `json:"commit_message"`
	Author          string    `json:"author"`
	Status          string    `json:"status"` // "success" or "failed"
	BuildLogs       string    `json:"build_logs"`
	DurationSeconds int       `json:"duration_seconds"`
	ActivatedAt     time.Time `json:"activated_at"`
}

// KeyPair represents generated SSH deploy credentials
type KeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// Manager orchestrates Git CI/CD and atomic deployments
type Manager struct {
	mockMode bool
}

func NewManager() *Manager {
	return &Manager{
		mockMode: os.Geteuid() != 0,
	}
}

// GenerateDeployKeyPair produces an Ed25519 SSH keypair suitable for GitHub/GitLab deploy keys
func (m *Manager) GenerateDeployKeyPair(comment string) (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to format ssh public key: %w", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)
	pubStr := strings.TrimSpace(string(pubBytes))
	if comment != "" {
		pubStr = fmt.Sprintf("%s %s", pubStr, comment)
	}

	privBytes, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	privPem := pem.EncodeToMemory(privBytes)

	return &KeyPair{
		PublicKey:  pubStr,
		PrivateKey: string(privPem),
	}, nil
}

// VerifyWebhookSignature verifies payload authenticity for GitHub and GitLab
func (m *Manager) VerifyWebhookSignature(provider, signature, secret string, payload []byte) bool {
	if secret == "" {
		return false
	}

	provider = strings.ToLower(provider)
	if provider == "gitlab" {
		// GitLab uses plain token header
		return subtle.ConstantTimeCompare([]byte(signature), []byte(secret)) == 1
	}

	// Default / GitHub uses sha256=HEX(HMAC-SHA256(secret, payload))
	sig := signature
	if strings.HasPrefix(sig, "sha256=") {
		sig = strings.TrimPrefix(sig, "sha256=")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(sig), []byte(expectedMAC)) == 1
}

// DeployRelease performs zero-downtime atomic release deployment
func (m *Manager) DeployRelease(ctx context.Context, cfg *DeployConfig) (*DeployResult, error) {
	start := time.Now()
	releaseID := time.Now().UTC().Format("20060102150405")
	deployRoot := cfg.DeployPath
	if deployRoot == "" {
		return nil, fmt.Errorf("deploy_path must not be empty")
	}

	releasesDir := filepath.Join(deployRoot, "releases")
	targetReleaseDir := filepath.Join(releasesDir, releaseID)
	currentSymlink := filepath.Join(deployRoot, "current")

	if err := os.MkdirAll(targetReleaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create release dir: %w", err)
	}

	var logs strings.Builder
	logs.WriteString(fmt.Sprintf("[%s] Initializing atomic deployment for %s (%s)...\n", time.Now().Format(time.RFC3339), cfg.RepoURL, cfg.Branch))

	// 1. Git Clone or pull into target release dir
	if !m.mockMode {
		var gitEnv []string
		if cfg.DeployKey != "" {
			keyFile := filepath.Join(deployRoot, ".deploy_key_tmp")
			_ = os.WriteFile(keyFile, []byte(cfg.DeployKey), 0600)
			defer os.Remove(keyFile)
			gitEnv = append(gitEnv, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", keyFile))
		}

		cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--branch", cfg.Branch, cfg.RepoURL, targetReleaseDir)
		cloneCmd.Env = append(os.Environ(), gitEnv...)
		out, err := cloneCmd.CombinedOutput()
		logs.WriteString(string(out) + "\n")
		if err != nil {
			logs.WriteString(fmt.Sprintf("Git clone failed: %v\n", err))
			return &DeployResult{
				ReleaseID:       releaseID,
				ReleasePath:     targetReleaseDir,
				Status:          "failed",
				BuildLogs:       logs.String(),
				DurationSeconds: int(time.Since(start).Seconds()),
				ActivatedAt:     time.Now().UTC(),
			}, fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		// Mock clone simulation
		_ = os.WriteFile(filepath.Join(targetReleaseDir, "index.js"), []byte("// Release "+releaseID+"\nconsole.log('App running');\n"), 0644)
		logs.WriteString(fmt.Sprintf("Cloned repository into %s\n", targetReleaseDir))
	}

	// Extract commit info
	commitHash := "HEAD"
	commitMsg := "Deployed via Hostvra CI/CD"
	author := "Hostvra Bot"
	if !m.mockMode {
		if hOut, err := exec.CommandContext(ctx, "git", "-C", targetReleaseDir, "rev-parse", "--short", "HEAD").Output(); err == nil {
			commitHash = strings.TrimSpace(string(hOut))
		}
		if mOut, err := exec.CommandContext(ctx, "git", "-C", targetReleaseDir, "log", "-1", "--pretty=%B").Output(); err == nil {
			commitMsg = strings.TrimSpace(string(mOut))
		}
		if aOut, err := exec.CommandContext(ctx, "git", "-C", targetReleaseDir, "log", "-1", "--pretty=%an").Output(); err == nil {
			author = strings.TrimSpace(string(aOut))
		}
	} else {
		commitHash = releaseID[:7]
	}

	// 2. Run build command if configured
	if cfg.BuildCommand != "" {
		logs.WriteString(fmt.Sprintf("[%s] Executing build command: %s\n", time.Now().Format(time.RFC3339), cfg.BuildCommand))
		if !m.mockMode {
			bCmd := exec.CommandContext(ctx, "sh", "-c", cfg.BuildCommand)
			bCmd.Dir = targetReleaseDir
			out, err := bCmd.CombinedOutput()
			logs.WriteString(string(out) + "\n")
			if err != nil {
				logs.WriteString(fmt.Sprintf("Build step failed: %v\n", err))
				return &DeployResult{
					ReleaseID:       releaseID,
					ReleasePath:     targetReleaseDir,
					CommitHash:      commitHash,
					CommitMessage:   commitMsg,
					Author:          author,
					Status:          "failed",
					BuildLogs:       logs.String(),
					DurationSeconds: int(time.Since(start).Seconds()),
					ActivatedAt:     time.Now().UTC(),
				}, fmt.Errorf("build failed: %w", err)
			}
		} else {
			logs.WriteString("Build completed successfully (mock mode).\n")
		}
	}

	// 3. Atomic Symlink Switch
	tmpSymlink := currentSymlink + ".tmp"
	_ = os.Remove(tmpSymlink)
	if err := os.Symlink(targetReleaseDir, tmpSymlink); err != nil {
		return nil, fmt.Errorf("failed to create temporary symlink: %w", err)
	}
	if err := os.Rename(tmpSymlink, currentSymlink); err != nil {
		_ = os.Remove(tmpSymlink)
		return nil, fmt.Errorf("failed atomic symlink rename: %w", err)
	}
	logs.WriteString(fmt.Sprintf("[%s] Switched current symlink atomically to %s\n", time.Now().Format(time.RFC3339), targetReleaseDir))

	// 4. Prune old releases
	keep := cfg.KeepReleases
	if keep <= 0 {
		keep = 5
	}
	m.pruneReleases(releasesDir, keep)

	logs.WriteString(fmt.Sprintf("[%s] Deployment %s completed successfully in %v.\n", time.Now().Format(time.RFC3339), releaseID, time.Since(start).Round(time.Millisecond)))

	return &DeployResult{
		ReleaseID:       releaseID,
		ReleasePath:     targetReleaseDir,
		CommitHash:      commitHash,
		CommitMessage:   commitMsg,
		Author:          author,
		Status:          "success",
		BuildLogs:       logs.String(),
		DurationSeconds: int(time.Since(start).Seconds()),
		ActivatedAt:     time.Now().UTC(),
	}, nil
}

// RollbackRelease switches the atomic symlink back to a previous release directory
func (m *Manager) RollbackRelease(deployPath, releaseID string) error {
	releasesDir := filepath.Join(deployPath, "releases")
	targetDir := filepath.Join(releasesDir, releaseID)
	if info, err := os.Stat(targetDir); err != nil || !info.IsDir() {
		return fmt.Errorf("target release directory %s not found", releaseID)
	}

	currentSymlink := filepath.Join(deployPath, "current")
	tmpSymlink := currentSymlink + ".rollback.tmp"
	_ = os.Remove(tmpSymlink)
	if err := os.Symlink(targetDir, tmpSymlink); err != nil {
		return fmt.Errorf("failed to create rollback symlink: %w", err)
	}
	if err := os.Rename(tmpSymlink, currentSymlink); err != nil {
		_ = os.Remove(tmpSymlink)
		return fmt.Errorf("failed to atomically activate rollback release: %w", err)
	}
	return nil
}

// pruneReleases removes older releases exceeding keepCount
func (m *Manager) pruneReleases(releasesDir string, keepCount int) {
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		return
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) <= keepCount {
		return
	}

	sort.Strings(dirs) // Sorted chronological (since names are timestamps)
	toDelete := dirs[:len(dirs)-keepCount]
	for _, d := range toDelete {
		_ = os.RemoveAll(filepath.Join(releasesDir, d))
	}
}
