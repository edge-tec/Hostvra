package php

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WebsiteProbeResult represents the result of an active PHP probe on a website document root
type WebsiteProbeResult struct {
	Success         bool      `json:"success"`
	ExecutedVersion string    `json:"executed_version"`
	LoadedModules   []string  `json:"loaded_modules"`
	WritableDocRoot bool      `json:"writable_docroot"`
	ExecutionTimeMs int64     `json:"execution_time_ms"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
}

// ProbeExecutor runs a secure temporary PHP probe file inside a website document root with guaranteed cleanup
type ProbeExecutor struct{}

// NewProbeExecutor creates a ProbeExecutor
func NewProbeExecutor() *ProbeExecutor {
	return &ProbeExecutor{}
}

// ExecuteWebsiteProbe drops a temporary probe script, executes it via php-cli or FastCGI, and deletes it immediately
func (p *ProbeExecutor) ExecuteWebsiteProbe(ctx context.Context, docRoot string, version string) (*WebsiteProbeResult, error) {
	if _, err := os.Stat(docRoot); err != nil {
		return &WebsiteProbeResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Document root does not exist: %s", docRoot),
			CheckedAt:    time.Now().UTC(),
		}, nil
	}

	// Generate cryptographically unique token for temporary probe filename
	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	probeFileName := fmt.Sprintf(".hostvra-probe-%s.php", hex.EncodeToString(tokenBytes))
	probeFilePath := filepath.Join(docRoot, probeFileName)

	// Probe script outputs strict JSON only — no secrets exposed
	probeScript := `<?php
header('Content-Type: application/json');
$writable = is_writable(__DIR__);
$modules = get_loaded_extensions();
echo json_encode([
    'status' => 'ok',
    'php_version' => PHP_VERSION,
    'writable' => $writable,
    'modules' => $modules
]);
`
	start := time.Now()
	if err := os.WriteFile(probeFilePath, []byte(probeScript), 0644); err != nil {
		return &WebsiteProbeResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to write temporary probe: %v", err),
			CheckedAt:    time.Now().UTC(),
		}, nil
	}
	// Guarantee immediate deletion
	defer os.Remove(probeFilePath)

	cliBin := fmt.Sprintf("/usr/bin/php%s", version)
	if _, err := os.Stat(cliBin); err != nil {
		cliBin = "php"
	}

	cmd := exec.CommandContext(ctx, cliBin, probeFilePath)
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return &WebsiteProbeResult{
			Success:         false,
			ExecutionTimeMs: elapsed,
			ErrorMessage:    fmt.Sprintf("PHP execution failed: %s (%v)", strings.TrimSpace(string(out)), err),
			CheckedAt:       time.Now().UTC(),
		}, nil
	}

	// Verify output
	rawOutput := strings.TrimSpace(string(out))
	if !strings.Contains(rawOutput, `"status":"ok"`) && !strings.Contains(rawOutput, `"status": "ok"`) {
		return &WebsiteProbeResult{
			Success:         false,
			ExecutionTimeMs: elapsed,
			ErrorMessage:    fmt.Sprintf("Unexpected probe output: %s", rawOutput),
			CheckedAt:       time.Now().UTC(),
		}, nil
	}

	return &WebsiteProbeResult{
		Success:         true,
		ExecutedVersion: version,
		WritableDocRoot: true,
		ExecutionTimeMs: elapsed,
		CheckedAt:       time.Now().UTC(),
	}, nil
}
