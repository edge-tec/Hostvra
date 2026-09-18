package security

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
	ScannerEngine string       `json:"scanner_engine"` // "clamav" or "heuristic"
	Threats       []ThreatItem `json:"threats"`
	Timestamp     time.Time    `json:"timestamp"`
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
	".php":   true,
	".phtml": true,
	".php3":  true,
	".php4":  true,
	".php5":  true,
	".php7":  true,
	".php8":  true,
	".phps":  true,
	".inc":   true,
	".sus":   true,
	".ico":   true,
	".jpg":   true,
	".png":   true,
	".gif":   true,
	".txt":   true,
	".htaccess": true,
}

// Scanner handles malware and webshell scanning on directories
type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

// ScanDirectory audits a target folder using ClamAV if installed, falling back to heuristic webshell analysis
func (s *Scanner) ScanDirectory(ctx context.Context, targetPath string) (*ScanReport, error) {
	if strings.TrimSpace(targetPath) == "" {
		return nil, errors.New("target directory path is required")
	}

	cleanPath := filepath.Clean(targetPath)
	fi, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("target path not accessible: %w", err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("target path '%s' is not a directory", cleanPath)
	}

	start := time.Now()

	// 1. Try ClamAV CLI (clamdscan or clamscan) if available on the system
	if bin, err := exec.LookPath("clamdscan"); err == nil {
		report, err := s.scanWithClamAV(ctx, bin, cleanPath, start)
		if err == nil {
			return report, nil
		}
	} else if bin, err := exec.LookPath("clamscan"); err == nil {
		report, err := s.scanWithClamAV(ctx, bin, cleanPath, start)
		if err == nil {
			return report, nil
		}
	}

	// 2. High-speed heuristic webshell scanner
	return s.scanWithHeuristics(ctx, cleanPath, start)
}

func (s *Scanner) scanWithClamAV(ctx context.Context, binPath, targetPath string, start time.Time) (*ScanReport, error) {
	cmd := exec.CommandContext(ctx, binPath, "--infected", "--no-summary", "-r", targetPath)
	out, err := cmd.CombinedOutput()

	// Exit code 0: No virus found
	// Exit code 1: Virus found
	// Exit code > 1: Error occurred
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
		ScannedFiles:  len(threats), // ClamAV --infected only outputs infected lines
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

func (s *Scanner) scanWithHeuristics(ctx context.Context, targetPath string, start time.Time) (*ScanReport, error) {
	var threats []ThreatItem
	scannedCount := 0

	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable paths without aborting entire scan
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
				// Record up to 2 distinct threats per file to prevent report explosion
				if len(threats) >= 2 {
					return threats, nil
				}
			}
		}
	}

	return threats, nil
}
