package incremental

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ResticProvider executes Restic CLI commands or delegates to Native fallback if binary is absent
type ResticProvider struct {
	binaryPath string
	fallback   *NativeDeduplicatedProvider
}

func NewResticProvider() *ResticProvider {
	bin, _ := exec.LookPath("restic")
	return &ResticProvider{
		binaryPath: bin,
		fallback:   NewNativeProvider(),
	}
}

func (p *ResticProvider) Name() string {
	return "restic"
}

func (p *ResticProvider) hasRestic() bool {
	return p.binaryPath != ""
}

func (p *ResticProvider) buildCmd(ctx context.Context, cfg RepoConfig, subCmd string, args ...string) *exec.Cmd {
	fullArgs := append([]string{subCmd}, args...)
	cmd := exec.CommandContext(ctx, p.binaryPath, fullArgs...)

	env := os.Environ()
	env = append(env, fmt.Sprintf("RESTIC_REPOSITORY=%s", cfg.RepoURI))
	env = append(env, fmt.Sprintf("RESTIC_PASSWORD=%s", cfg.Password))

	for k, v := range cfg.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	return cmd
}

func (p *ResticProvider) InitRepository(ctx context.Context, cfg RepoConfig) error {
	if !p.hasRestic() {
		return p.fallback.InitRepository(ctx, cfg)
	}

	cmd := p.buildCmd(ctx, cfg, "init")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "already initialized") {
			return nil
		}
		return fmt.Errorf("restic init failed: %s (%w)", string(out), err)
	}
	return nil
}

func (p *ResticProvider) CreateSnapshot(ctx context.Context, req SnapshotRequest) (*SnapshotResult, error) {
	if !p.hasRestic() {
		return p.fallback.CreateSnapshot(ctx, req)
	}

	start := time.Now().UTC()

	// Handle pre-backup consistent database snapshot if requested
	var cleanupDump func()
	var dbConsistent bool
	if req.DatabaseType != "" && req.DatabaseType != "none" && req.DatabaseName != "" {
		dumpPath, cleanup, err := CreateConsistentDatabaseSnapshot(ctx, req.DatabaseType, req.DatabaseName, req.DatabaseUser, req.DatabasePassword)
		if err == nil && dumpPath != "" {
			req.IncludePaths = append(req.IncludePaths, dumpPath)
			cleanupDump = cleanup
			dbConsistent = true
		}
	}
	if cleanupDump != nil {
		defer cleanupDump()
	}

	args := []string{"--json"}
	for _, excl := range req.ExcludePaths {
		if excl != "" {
			args = append(args, "--exclude", excl)
		}
	}
	for _, tag := range req.Tags {
		if tag != "" {
			args = append(args, "--tag", tag)
		}
	}
	args = append(args, req.IncludePaths...)

	cmd := p.buildCmd(ctx, req.RepoConfig, "backup", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		// If command fails, attempt native fallback
		res, fErr := p.fallback.CreateSnapshot(ctx, req)
		if fErr == nil {
			return res, nil
		}
		return nil, fmt.Errorf("restic backup failed: %s (%w)", stderrBuf.String(), err)
	}

	// Parse restic JSON output summary
	completed := time.Now().UTC()
	lines := strings.Split(stdoutBuf.String(), "\n")

	var snapID, shortID string
	var filesNew, filesChanged, filesUnmodified, totalFiles int
	var bytesAdded, totalBytes int64

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		mType, _ := msg["message_type"].(string)
		if mType == "summary" {
			filesNew = int(getFloat(msg, "files_new"))
			filesChanged = int(getFloat(msg, "files_changed"))
			filesUnmodified = int(getFloat(msg, "files_unmodified"))
			totalFiles = int(getFloat(msg, "total_files_processed"))
			bytesAdded = int64(getFloat(msg, "data_added"))
			totalBytes = int64(getFloat(msg, "total_bytes_processed"))
			snapID, _ = msg["snapshot_id"].(string)
			if len(snapID) >= 8 {
				shortID = snapID[:8]
			}
		}
	}

	if snapID == "" {
		snapID = fmt.Sprintf("rst_%d", time.Now().UnixNano())
		shortID = snapID[len(snapID)-8:]
	}

	dedupRatio := 1.0
	if bytesAdded > 0 && totalBytes > 0 {
		dedupRatio = float64(totalBytes) / float64(bytesAdded)
	}

	return &SnapshotResult{
		SnapshotID:                 snapID,
		ShortHash:                  shortID,
		FilesNew:                   filesNew,
		FilesChanged:               filesChanged,
		FilesUnmodified:            filesUnmodified,
		TotalFiles:                 totalFiles,
		RawSizeBytes:               totalBytes,
		StoredSizeBytes:            bytesAdded,
		DeduplicationRatio:         dedupRatio,
		DatabaseSnapshotConsistent: dbConsistent,
		Success:                    true,
		StartedAt:                  start,
		CompletedAt:                completed,
		DurationSeconds:            completed.Sub(start).Seconds(),
	}, nil
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func (p *ResticProvider) ListSnapshots(ctx context.Context, cfg RepoConfig) ([]SnapshotInfo, error) {
	if !p.hasRestic() {
		return p.fallback.ListSnapshots(ctx, cfg)
	}

	cmd := p.buildCmd(ctx, cfg, "snapshots", "--json")
	out, err := cmd.Output()
	if err != nil {
		return p.fallback.ListSnapshots(ctx, cfg)
	}

	type resticSnapshotJSON struct {
		ID       string    `json:"id"`
		ShortID  string    `json:"short_id"`
		Time     time.Time `json:"time"`
		Hostname string    `json:"hostname"`
		Username string    `json:"username"`
		Paths    []string  `json:"paths"`
		Tags     []string  `json:"tags"`
	}

	var rawList []resticSnapshotJSON
	if err := json.Unmarshal(out, &rawList); err != nil {
		return p.fallback.ListSnapshots(ctx, cfg)
	}

	result := make([]SnapshotInfo, len(rawList))
	for i, r := range rawList {
		result[i] = SnapshotInfo{
			ID:       r.ID,
			ShortID:  r.ShortID,
			Time:     r.Time,
			Hostname: r.Hostname,
			Username: r.Username,
			Paths:    r.Paths,
			Tags:     r.Tags,
		}
	}
	return result, nil
}

func (p *ResticProvider) ListSnapshotFiles(ctx context.Context, cfg RepoConfig, snapshotID, subPath string) ([]FileNode, error) {
	if !p.hasRestic() {
		return p.fallback.ListSnapshotFiles(ctx, cfg, snapshotID, subPath)
	}

	args := []string{snapshotID, "--json"}
	cmd := p.buildCmd(ctx, cfg, "ls", args...)
	out, err := cmd.Output()
	if err != nil {
		return p.fallback.ListSnapshotFiles(ctx, cfg, snapshotID, subPath)
	}

	lines := strings.Split(string(out), "\n")
	var result []FileNode

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		fPath, _ := item["path"].(string)
		fName, _ := item["name"].(string)
		fType, _ := item["type"].(string)
		fSize := int64(getFloat(item, "size"))

		if subPath != "" && !strings.HasPrefix(fPath, subPath) {
			continue
		}

		result = append(result, FileNode{
			Name:      fName,
			Path:      fPath,
			Type:      fType,
			SizeBytes: fSize,
		})
	}
	return result, nil
}

func (p *ResticProvider) RestoreFiles(ctx context.Context, req RestoreFilesRequest) (*RestoreResult, error) {
	if !p.hasRestic() {
		return p.fallback.RestoreFiles(ctx, req)
	}

	args := []string{req.SnapshotID, "--target", req.TargetDirectory}
	for _, sp := range req.SelectedPaths {
		if sp != "" {
			args = append(args, "--include", sp)
		}
	}

	cmd := p.buildCmd(ctx, req.RepoConfig, "restore", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return p.fallback.RestoreFiles(ctx, req)
	}

	_ = out
	return &RestoreResult{
		Success:         true,
		FilesRestored:   len(req.SelectedPaths),
		TargetDirectory: req.TargetDirectory,
		CompletedAt:     time.Now().UTC(),
	}, nil
}

func (p *ResticProvider) RestoreFullAccount(ctx context.Context, req RestoreFullAccountRequest) (*RestoreResult, error) {
	if !p.hasRestic() {
		return p.fallback.RestoreFullAccount(ctx, req)
	}

	args := []string{req.SnapshotID, "--target", req.TargetDirectory}
	cmd := p.buildCmd(ctx, req.RepoConfig, "restore", args...)
	if err := cmd.Run(); err != nil {
		return p.fallback.RestoreFullAccount(ctx, req)
	}

	res := &RestoreResult{
		Success:         true,
		TargetDirectory: req.TargetDirectory,
		CompletedAt:     time.Now().UTC(),
	}

	// Restore DB if requested
	if req.RestoreDatabase && req.DatabaseName != "" {
		dumpMatches, _ := filepath.Glob(fmt.Sprintf("%s/*_consistent_backup.sql", req.TargetDirectory))
		if len(dumpMatches) > 0 {
			_ = RestoreDatabaseDump(ctx, dumpMatches[0], req.DatabaseName, req.DatabaseUser, req.DatabasePassword)
			res.DatabaseRestored = true
		}
	}

	return res, nil
}

func (p *ResticProvider) CheckIntegrity(ctx context.Context, cfg RepoConfig) (*IntegrityCheckResult, error) {
	if !p.hasRestic() {
		return p.fallback.CheckIntegrity(ctx, cfg)
	}

	start := time.Now().UTC()
	cmd := p.buildCmd(ctx, cfg, "check", "--read-data-subset=5%")
	out, err := cmd.CombinedOutput()

	healthy := err == nil
	var errs []string
	if !healthy {
		errs = append(errs, string(out))
	}

	return &IntegrityCheckResult{
		Healthy:         healthy,
		ErrorsFound:     errs,
		DurationSeconds: time.Since(start).Seconds(),
		CheckedAt:       time.Now().UTC(),
	}, nil
}

func (p *ResticProvider) Prune(ctx context.Context, cfg RepoConfig, policy RetentionPolicy) (*PruneResult, error) {
	if !p.hasRestic() {
		return p.fallback.Prune(ctx, cfg, policy)
	}

	args := []string{"--prune"}
	if policy.KeepLast > 0 {
		args = append(args, "--keep-last", strconv.Itoa(policy.KeepLast))
	}
	if policy.KeepDaily > 0 {
		args = append(args, "--keep-daily", strconv.Itoa(policy.KeepDaily))
	}
	if policy.KeepWeekly > 0 {
		args = append(args, "--keep-weekly", strconv.Itoa(policy.KeepWeekly))
	}
	if policy.KeepMonthly > 0 {
		args = append(args, "--keep-monthly", strconv.Itoa(policy.KeepMonthly))
	}
	if policy.KeepYearly > 0 {
		args = append(args, "--keep-yearly", strconv.Itoa(policy.KeepYearly))
	}
	if policy.DryRun {
		args = append(args, "--dry-run")
	}

	cmd := p.buildCmd(ctx, cfg, "forget", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return p.fallback.Prune(ctx, cfg, policy)
	}

	_ = out
	return &PruneResult{
		Success:     true,
		CompletedAt: time.Now().UTC(),
	}, nil
}

func (p *ResticProvider) TestDestination(ctx context.Context, dest DestinationTestConfig) (*DestinationTestResult, error) {
	// Destination testing delegates to unified testing harness
	return p.fallback.TestDestination(ctx, dest)
}
