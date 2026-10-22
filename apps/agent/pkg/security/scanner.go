package security

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrScanBusy            = errors.New("another scan is currently running; please wait for it to finish")
	ErrPathEscapeBlocked   = errors.New("path traversal or unauthorized directory access blocked")
	ErrQuarantineFileNotFound = errors.New("quarantined file not found")
	ErrInvalidQuarantineName  = errors.New("invalid quarantine filename format")
)

type ThreatItem struct {
	FilePath   string `json:"file_path"`
	ThreatName string `json:"threat_name"`
	LineNumber int    `json:"line_number,omitempty"`
	Severity   string `json:"severity"` // "critical", "high", "medium"
}

type ScanReport struct {
	TargetPath    string       `json:"target_path"`
	ScannedFiles  int          `json:"scanned_files"`
	InfectedFiles int          `json:"infected_files"`
	DurationMs    int64        `json:"duration_ms"`
	ScannerEngine string       `json:"scanner_engine"` // "clamav", "heuristic", "combined"
	Threats       []ThreatItem `json:"threats"`
	Timestamp     time.Time    `json:"timestamp"`
}

type QuarantineMetadata struct {
	QuarantineFilename string    `json:"quarantine_filename"`
	OriginalPath       string    `json:"original_path"`
	FileSize           int64     `json:"file_size"`
	FileMode           string    `json:"file_mode"`
	FileOwner          string    `json:"file_owner"`
	FileGroup          string    `json:"file_group"`
	SHA256Hash         string    `json:"sha256_hash"`
	ThreatName         string    `json:"threat_name"`
	QuarantinedAt      time.Time `json:"quarantined_at"`
}

var heuristicSignatures = []struct {
	Name     string
	Pattern  *regexp.Regexp
	Severity string
}{
	{
		Name:     "PHP.Webshell.EvalBase64",
		Pattern:  regexp.MustCompile(`(?i)eval\s*\(\s*(base64_decode|gzinflate|gzuncompress|str_rot13)\s*\(`),
		Severity: "critical",
	},
	{
		Name:     "PHP.Webshell.AssertBase64",
		Pattern:  regexp.MustCompile(`(?i)assert\s*\(\s*(base64_decode|gzinflate|gzuncompress|str_rot13)\s*\(`),
		Severity: "critical",
	},
	{
		Name:     "PHP.Webshell.KnownShellSignature",
		Pattern:  regexp.MustCompile(`(?i)(c99shell|r57shell|b374k|wso_version|fx29_shell|alfa_team|cinjector|filesman)`),
		Severity: "critical",
	},
	{
		Name:     "PHP.Backdoor.RemoteExecution",
		Pattern:  regexp.MustCompile(`(?i)(passthru|shell_exec|system|exec|popen|proc_open)\s*\(\s*\$_(POST|GET|REQUEST|COOKIE|SERVER)\b`),
		Severity: "critical",
	},
	{
		Name:     "PHP.Backdoor.PregReplaceEval",
		Pattern:  regexp.MustCompile(`(?i)preg_replace\s*\(\s*['"]/.*/e['"]`),
		Severity: "high",
	},
	{
		Name:     "PHP.Obfuscation.HexPayload",
		Pattern:  regexp.MustCompile(`(\\x[0-9a-fA-F]{2}){10,}`),
		Severity: "medium",
	},
}

var scannableExtensions = map[string]bool{
	".php":      true,
	".phtml":    true,
	".php3":     true,
	".php4":     true,
	".php5":     true,
	".php7":     true,
	".php8":     true,
	".phps":     true,
	".inc":      true,
	".sus":      true,
	".ico":      true,
	".jpg":      true,
	".png":      true,
	".gif":      true,
	".txt":      true,
	".htaccess": true,
}

// Scanner handles malware and webshell scanning, quarantine, and restoration.
type Scanner struct {
	quarantineDir string
	scanMu        sync.Mutex
}

func NewScanner() *Scanner {
	qDir := os.Getenv("HOSTVRA_QUARANTINE_DIR")
	if qDir == "" {
		qDir = "/var/lib/hostvra/quarantine"
	}
	return &Scanner{
		quarantineDir: qDir,
	}
}

func NewScannerWithQuarantineDir(qDir string) *Scanner {
	return &Scanner{
		quarantineDir: qDir,
	}
}

// GetEngineStatus reports active scanning engines and their versions.
func (s *Scanner) GetEngineStatus() map[string]interface{} {
	status := map[string]interface{}{
		"heuristic_active": true,
		"heuristic_rules":  len(heuristicSignatures),
		"clamav_installed": false,
		"clamav_daemon":    false,
		"quarantine_dir":   s.quarantineDir,
	}

	if bin, err := exec.LookPath("clamdscan"); err == nil {
		status["clamav_installed"] = true
		status["clamav_binary"] = bin
		status["clamav_daemon"] = true
	} else if bin, err := exec.LookPath("clamscan"); err == nil {
		status["clamav_installed"] = true
		status["clamav_binary"] = bin
		status["clamav_daemon"] = false
	}

	// Read freshclam database timestamp if present
	for _, dbPath := range []string{"/var/lib/clamav/daily.cvd", "/var/lib/clamav/daily.cld"} {
		if fi, err := os.Stat(dbPath); err == nil {
			status["clamav_db_updated_at"] = fi.ModTime().UTC()
			break
		}
	}

	return status
}

// ValidateScanPath ensures path exists, is a directory, and does not escape permitted boundaries.
func (s *Scanner) ValidateScanPath(targetPath string, permittedRoots []string) (string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return "", errors.New("target path is required")
	}

	clean := filepath.Clean(targetPath)
	realPath, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", fmt.Errorf("target path cannot be resolved: %w", err)
	}

	fi, err := os.Stat(realPath)
	if err != nil {
		return "", fmt.Errorf("target path not accessible: %w", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("target path '%s' is not a directory", realPath)
	}

	if len(permittedRoots) > 0 {
		allowed := false
		for _, root := range permittedRoots {
			cleanRoot := filepath.Clean(root)
			if realPath == cleanRoot || strings.HasPrefix(realPath, cleanRoot+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", ErrPathEscapeBlocked
		}
	}

	return realPath, nil
}

// ScanDirectory audits a target folder using ClamAV if installed, falling back to heuristic webshell analysis.
func (s *Scanner) ScanDirectory(ctx context.Context, targetPath string) (*ScanReport, error) {
	return s.ScanDirectoryWithOptions(ctx, targetPath, "combined", nil)
}

// ScanDirectoryWithOptions audits with specific engine choice, configurable exclusions, and concurrency protection.
func (s *Scanner) ScanDirectoryWithOptions(ctx context.Context, targetPath string, engine string, exclusions []string) (*ScanReport, error) {
	if !s.scanMu.TryLock() {
		return nil, ErrScanBusy
	}
	defer s.scanMu.Unlock()

	cleanPath, err := s.ValidateScanPath(targetPath, nil)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	engine = strings.ToLower(strings.TrimSpace(engine))
	if engine == "" {
		engine = "combined"
	}

	var clamReport *ScanReport
	var heuristicReport *ScanReport

	// 1. Try ClamAV if requested
	if engine == "clamav" || engine == "combined" {
		if bin, err := exec.LookPath("clamdscan"); err == nil {
			clamReport, _ = s.scanWithClamAV(ctx, bin, cleanPath, start, exclusions)
		} else if bin, err := exec.LookPath("clamscan"); err == nil {
			clamReport, _ = s.scanWithClamAV(ctx, bin, cleanPath, start, exclusions)
		}
	}

	// 2. If engine is strictly clamav, return clamav result
	if engine == "clamav" && clamReport != nil {
		return clamReport, nil
	}

	// 3. Run heuristic analysis
	heuristicReport, err = s.scanWithHeuristics(ctx, cleanPath, start, exclusions)
	if err != nil {
		if clamReport != nil {
			return clamReport, nil
		}
		return nil, err
	}

	// 4. Combine reports if both were executed
	if clamReport != nil && len(clamReport.Threats) > 0 {
		// Deduplicate threats by FilePath
		seenFiles := make(map[string]bool)
		var combinedThreats []ThreatItem
		for _, t := range clamReport.Threats {
			seenFiles[t.FilePath] = true
			combinedThreats = append(combinedThreats, t)
		}
		for _, t := range heuristicReport.Threats {
			if !seenFiles[t.FilePath] {
				combinedThreats = append(combinedThreats, t)
				seenFiles[t.FilePath] = true
			}
		}

		return &ScanReport{
			TargetPath:    cleanPath,
			ScannedFiles:  heuristicReport.ScannedFiles,
			InfectedFiles: len(combinedThreats),
			DurationMs:    time.Since(start).Milliseconds(),
			ScannerEngine: "combined",
			Threats:       combinedThreats,
			Timestamp:     time.Now().UTC(),
		}, nil
	}

	return heuristicReport, nil
}

func (s *Scanner) scanWithClamAV(ctx context.Context, binPath, targetPath string, start time.Time, exclusions []string) (*ScanReport, error) {
	args := []string{"--infected", "--no-summary", "-r"}
	for _, excl := range exclusions {
		cleanExcl := strings.TrimSpace(excl)
		if cleanExcl != "" {
			args = append(args, "--exclude="+cleanExcl)
		}
	}
	args = append(args, targetPath)

	cmd := exec.CommandContext(ctx, binPath, args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() > 1 {
				return nil, fmt.Errorf("clamav scan failed with code %d: %s", exitErr.ExitCode(), string(out))
			}
		} else {
			return nil, err
		}
	}

	threats := parseClamAVOutput(string(out))
	duration := time.Since(start).Milliseconds()

	return &ScanReport{
		TargetPath:    targetPath,
		ScannedFiles:  len(threats),
		InfectedFiles: len(threats),
		DurationMs:    duration,
		ScannerEngine: "clamav",
		Threats:       threats,
		Timestamp:     time.Now().UTC(),
	}, nil
}

func parseClamAVOutput(output string) []ThreatItem {
	var threats []ThreatItem
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ": ") {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 && strings.HasSuffix(parts[1], " FOUND") {
			threatName := strings.TrimSuffix(parts[1], " FOUND")
			threats = append(threats, ThreatItem{
				FilePath:   parts[0],
				ThreatName: threatName,
				Severity:   "critical",
			})
		}
	}
	return threats
}

func (s *Scanner) scanWithHeuristics(ctx context.Context, targetPath string, start time.Time, exclusions []string) (*ScanReport, error) {
	var threats []ThreatItem
	scannedCount := 0

	// Build exclusion patterns
	var exclusionPatterns []*regexp.Regexp
	for _, e := range exclusions {
		trimmed := strings.TrimSpace(e)
		if trimmed != "" {
			if re, err := regexp.Compile(trimmed); err == nil {
				exclusionPatterns = append(exclusionPatterns, re)
			}
		}
	}

	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable paths without aborting entire scan
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, _ := filepath.Rel(targetPath, path)
		for _, re := range exclusionPatterns {
			if re.MatchString(path) || re.MatchString(relPath) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		base := strings.ToLower(d.Name())
		if !scannableExtensions[ext] && base != ".htaccess" {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > 10*1024*1024 { // Skip files > 10MB
			return nil
		}

		scannedCount++
		fileThreats, _ := scanSingleFile(path)
		if len(fileThreats) > 0 {
			threats = append(threats, fileThreats...)
		}

		return nil
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	return &ScanReport{
		TargetPath:    targetPath,
		ScannedFiles:  scannedCount,
		InfectedFiles: len(threats),
		DurationMs:    time.Since(start).Milliseconds(),
		ScannerEngine: "heuristic",
		Threats:       threats,
		Timestamp:     time.Now().UTC(),
	}, nil
}

func scanSingleFile(filePath string) ([]ThreatItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var threats []ThreatItem
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, sig := range heuristicSignatures {
			if sig.Pattern.MatchString(line) {
				threats = append(threats, ThreatItem{
					FilePath:   filePath,
					ThreatName: sig.Name,
					LineNumber: lineNum,
					Severity:   sig.Severity,
				})
				if len(threats) >= 2 {
					return threats, nil
				}
			}
		}
	}

	return threats, nil
}

// ============================================================================
// QUARANTINE & RESTORATION ENGINE
// ============================================================================

// QuarantineThreat moves a malicious file to the restricted quarantine directory.
// The file is stripped of execution permissions (mode 0000) so it cannot be run.
func (s *Scanner) QuarantineThreat(targetFilePath string, threatName string) (*QuarantineMetadata, error) {
	cleanPath := filepath.Clean(targetFilePath)
	fi, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("target file not accessible: %w", err)
	}
	if fi.IsDir() {
		return nil, errors.New("cannot quarantine a directory")
	}

	// Compute SHA-256 hash
	fileBytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file for quarantine: %w", err)
	}
	hash := sha256.Sum256(fileBytes)
	sha256Hex := hex.EncodeToString(hash[:])

	// Ensure quarantine directory exists with 0700 permissions (root only)
	if err := os.MkdirAll(s.quarantineDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create quarantine directory: %w", err)
	}

	// Generate unique quarantine filename
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	quarantineFilename := fmt.Sprintf("%x-%x-%x-%x-%x.quarantine", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	destPath := filepath.Join(s.quarantineDir, quarantineFilename)

	// Write file to quarantine with 0000 permissions (no read/write/exec)
	if err := os.WriteFile(destPath, fileBytes, 0000); err != nil {
		return nil, fmt.Errorf("failed to write quarantined file: %w", err)
	}

	// Remove original infected file
	if err := os.Remove(cleanPath); err != nil {
		// Clean up quarantine copy if deletion failed
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("failed to remove infected file from original location: %w", err)
	}

	modeStr := fmt.Sprintf("%04o", fi.Mode().Perm())

	return &QuarantineMetadata{
		QuarantineFilename: quarantineFilename,
		OriginalPath:       cleanPath,
		FileSize:           fi.Size(),
		FileMode:           modeStr,
		FileOwner:          "root",
		FileGroup:          "root",
		SHA256Hash:         sha256Hex,
		ThreatName:         threatName,
		QuarantinedAt:      time.Now().UTC(),
	}, nil
}

// RestoreQuarantinedFile restores a quarantined file back to its original path with original permissions.
func (s *Scanner) RestoreQuarantinedFile(quarantineFilename string, targetPath string, originalMode string) error {
	// Security check: validate filename to prevent path traversal
	if strings.Contains(quarantineFilename, "/") || strings.Contains(quarantineFilename, "\\") || !strings.HasSuffix(quarantineFilename, ".quarantine") {
		return ErrInvalidQuarantineName
	}

	quarantinePath := filepath.Join(s.quarantineDir, quarantineFilename)
	if _, err := os.Stat(quarantinePath); os.IsNotExist(err) {
		return ErrQuarantineFileNotFound
	}

	cleanTargetPath := filepath.Clean(targetPath)
	if cleanTargetPath == "" || cleanTargetPath == "/" || cleanTargetPath == "." {
		return errors.New("invalid target restore path")
	}

	// Parse file mode
	perm := fs.FileMode(0644)
	if originalMode != "" {
		if parsed, err := strconv.ParseUint(originalMode, 8, 32); err == nil {
			perm = fs.FileMode(parsed)
		}
	}

	// Temporarily adjust quarantine file mode to read it
	_ = os.Chmod(quarantinePath, 0600)
	fileBytes, err := os.ReadFile(quarantinePath)
	if err != nil {
		_ = os.Chmod(quarantinePath, 0000)
		return fmt.Errorf("failed to read quarantined file: %w", err)
	}

	// Ensure destination directory exists
	targetDir := filepath.Dir(cleanTargetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		_ = os.Chmod(quarantinePath, 0000)
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write restored file
	if err := os.WriteFile(cleanTargetPath, fileBytes, perm); err != nil {
		_ = os.Chmod(quarantinePath, 0000)
		return fmt.Errorf("failed to restore file: %w", err)
	}

	// Remove from quarantine directory
	_ = os.Remove(quarantinePath)
	return nil
}

// DeleteQuarantinedFile permanently purges a quarantined file.
func (s *Scanner) DeleteQuarantinedFile(quarantineFilename string) error {
	if strings.Contains(quarantineFilename, "/") || strings.Contains(quarantineFilename, "\\") || !strings.HasSuffix(quarantineFilename, ".quarantine") {
		return ErrInvalidQuarantineName
	}

	quarantinePath := filepath.Join(s.quarantineDir, quarantineFilename)
	if _, err := os.Stat(quarantinePath); os.IsNotExist(err) {
		return ErrQuarantineFileNotFound
	}

	_ = os.Chmod(quarantinePath, 0600)
	return os.Remove(quarantinePath)
}
