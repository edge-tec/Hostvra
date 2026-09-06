package updater

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AgentProbeResult holds output of testing an agent binary before activating
type AgentProbeResult struct {
	Version   string `json:"version"`
	IsHealthy bool   `json:"is_healthy"`
	Output    string `json:"output"`
}

// AgentUpdater manages live versioned updates and safe rollbacks of hostvra-agent
type AgentUpdater struct {
	baseDir        string
	releasesDir    string
	currentSymlink string
	systemdService string
}

// NewAgentUpdater creates an AgentUpdater instance
func NewAgentUpdater(baseDir string) *AgentUpdater {
	if baseDir == "" {
		baseDir = "/opt/hostvra-agent"
	}
	return &AgentUpdater{
		baseDir:        baseDir,
		releasesDir:    filepath.Join(baseDir, "releases"),
		currentSymlink: filepath.Join(baseDir, "current"),
		systemdService: "hostvra-agent",
	}
}

// StageAgent writes the new agent binary into /opt/hostvra-agent/releases/<version>/bin/hostvra-agent
func (u *AgentUpdater) StageAgent(ctx context.Context, version string, binaryReader io.Reader) (string, error) {
	relBinDir := filepath.Join(u.releasesDir, version, "bin")
	if err := os.MkdirAll(relBinDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create release bin dir: %w", err)
	}

	targetPath := filepath.Join(relBinDir, "hostvra-agent")
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create agent binary file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, binaryReader); err != nil {
		return "", fmt.Errorf("failed to write agent binary: %w", err)
	}

	return targetPath, nil
}

// ProbeAgentBinary executes the staged binary with --version to verify executable integrity
func (u *AgentUpdater) ProbeAgentBinary(ctx context.Context, binaryPath string) (*AgentProbeResult, error) {
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("staged binary not found at %s: %w", binaryPath, err)
	}

	cmd := exec.CommandContext(ctx, binaryPath, "--version")
	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	if err != nil {
		return &AgentProbeResult{
			IsHealthy: false,
			Output:    outputStr,
		}, fmt.Errorf("agent binary probe execution failed: %s (err: %v)", outputStr, err)
	}

	return &AgentProbeResult{
		Version:   strings.TrimSpace(outputStr),
		IsHealthy: true,
		Output:    outputStr,
	}, nil
}

// ActivateAgent switches current symlink to target version and restarts service
func (u *AgentUpdater) ActivateAgent(ctx context.Context, version string) error {
	targetRelease := filepath.Join(u.releasesDir, version)
	if _, err := os.Stat(targetRelease); err != nil {
		return fmt.Errorf("agent release %s does not exist", version)
	}

	tempLink := fmt.Sprintf("%s.tmp.%d", u.currentSymlink, os.Getpid())
	_ = os.Remove(tempLink)

	if err := os.Symlink(targetRelease, tempLink); err != nil {
		return fmt.Errorf("failed to create temp symlink: %w", err)
	}

	if err := os.Rename(tempLink, u.currentSymlink); err != nil {
		_ = os.Remove(tempLink)
		return fmt.Errorf("failed to atomically activate agent symlink: %w", err)
	}

	// Restart hostvra-agent via systemctl if present
	if _, err := exec.LookPath("systemctl"); err == nil {
		_ = exec.CommandContext(ctx, "systemctl", "restart", u.systemdService).Run()
	}

	return nil
}

// RollbackAgent restores previous agent version
func (u *AgentUpdater) RollbackAgent(ctx context.Context, previousVersion string) error {
	return u.ActivateAgent(ctx, previousVersion)
}
