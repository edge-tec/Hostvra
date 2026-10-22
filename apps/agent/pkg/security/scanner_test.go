package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_CleanAndMaliciousFiles(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create Clean Files
	cleanPHP := `<?php
namespace App;

class Controller {
    public function index() {
        echo "Welcome to Hostvra";
    }
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "index.php"), []byte(cleanPHP), 0644); err != nil {
		t.Fatalf("failed to write clean file: %v", err)
	}

	// 2. Create Malicious Webshell File
	maliciousShell := `<?php
// Obfuscated payload test
$payload = "cGFzc3RocnU=";
eval(base64_decode("c3lzdGVtKCdfUE9TVFsiaGFjayJdJyk7"));
?>
`
	if err := os.WriteFile(filepath.Join(tempDir, "shell.php"), []byte(maliciousShell), 0644); err != nil {
		t.Fatalf("failed to write malicious file: %v", err)
	}

	// 3. Create Remote Execution Backdoor File
	backdoorPHP := `<?php
if (isset($_POST['run'])) {
    system($_POST['run']);
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "backdoor.php"), []byte(backdoorPHP), 0644); err != nil {
		t.Fatalf("failed to write backdoor file: %v", err)
	}

	scanner := NewScanner()
	report, err := scanner.ScanDirectory(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	if report.ScannedFiles < 3 {
		t.Errorf("expected at least 3 scanned files, got %d", report.ScannedFiles)
	}
	if report.InfectedFiles < 2 {
		t.Errorf("expected at least 2 infected files, got %d", report.InfectedFiles)
	}

	foundEvalBase64 := false
	foundRemoteExec := false

	for _, threat := range report.Threats {
		if threat.ThreatName == "PHP.Webshell.EvalBase64" {
			foundEvalBase64 = true
		}
		if threat.ThreatName == "PHP.Backdoor.RemoteExecution" {
			foundRemoteExec = true
		}
	}

	if !foundEvalBase64 {
		t.Errorf("expected to detect PHP.Webshell.EvalBase64 in shell.php")
	}
	if !foundRemoteExec {
		t.Errorf("expected to detect PHP.Backdoor.RemoteExecution in backdoor.php")
	}
}

func TestScanner_ParseClamAVOutput(t *testing.T) {
	sampleOutput := `/var/www/site/index.php: OK
/var/www/site/uploads/c99.php: Php.Webshell.C99-1 FOUND
/var/www/site/images/avatar.jpg: OK
/var/www/site/wp-content/r57.php: Unix.Trojan.MSShell FOUND
`
	threats := parseClamAVOutput(sampleOutput)
	if len(threats) != 2 {
		t.Fatalf("expected 2 threats from ClamAV output, got %d", len(threats))
	}

	if threats[0].ThreatName != "Php.Webshell.C99-1" || threats[0].FilePath != "/var/www/site/uploads/c99.php" {
		t.Errorf("unexpected threat 0: %+v", threats[0])
	}
	if threats[1].ThreatName != "Unix.Trojan.MSShell" || threats[1].FilePath != "/var/www/site/wp-content/r57.php" {
		t.Errorf("unexpected threat 1: %+v", threats[1])
	}
}

func TestScanner_QuarantineAndRestoreLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	quarantineDir := filepath.Join(tempDir, "quarantine")
	siteDir := filepath.Join(tempDir, "public_html")
	_ = os.MkdirAll(siteDir, 0755)

	infectedFile := filepath.Join(siteDir, "malicious.php")
	originalContent := "<?php eval(base64_decode('payload')); ?>"
	if err := os.WriteFile(infectedFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create test infected file: %v", err)
	}

	scanner := NewScannerWithQuarantineDir(quarantineDir)

	// 1. Quarantine infected file
	meta, err := scanner.QuarantineThreat(infectedFile, "PHP.Webshell.EvalBase64")
	if err != nil {
		t.Fatalf("QuarantineThreat failed: %v", err)
	}

	// 2. Original file must be gone
	if _, err := os.Stat(infectedFile); !os.IsNotExist(err) {
		t.Fatalf("expected infected file to be removed from original path")
	}

	// 3. Quarantined file must exist in quarantineDir
	qPath := filepath.Join(quarantineDir, meta.QuarantineFilename)
	if _, err := os.Stat(qPath); err != nil {
		t.Fatalf("expected quarantined file to exist at %s: %v", qPath, err)
	}

	// 4. Restore file back to original location
	if err := scanner.RestoreQuarantinedFile(meta.QuarantineFilename, infectedFile, meta.FileMode); err != nil {
		t.Fatalf("RestoreQuarantinedFile failed: %v", err)
	}

	// 5. Original file must now exist with identical content
	restoredContent, err := os.ReadFile(infectedFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	if string(restoredContent) != originalContent {
		t.Fatalf("restored content mismatch: expected %s, got %s", originalContent, string(restoredContent))
	}

	// 6. Quarantined file should now be deleted from quarantineDir
	if _, err := os.Stat(qPath); !os.IsNotExist(err) {
		t.Fatalf("expected quarantined copy to be deleted after restoration")
	}

	// 7. Test DeleteQuarantinedFile
	meta2, err := scanner.QuarantineThreat(infectedFile, "PHP.Webshell.EvalBase64")
	if err != nil {
		t.Fatalf("QuarantineThreat 2 failed: %v", err)
	}
	if err := scanner.DeleteQuarantinedFile(meta2.QuarantineFilename); err != nil {
		t.Fatalf("DeleteQuarantinedFile failed: %v", err)
	}
	qPath2 := filepath.Join(quarantineDir, meta2.QuarantineFilename)
	if _, err := os.Stat(qPath2); !os.IsNotExist(err) {
		t.Fatalf("expected quarantined file to be permanently deleted")
	}
}

func TestScanner_Exclusions(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)

	cacheFile := filepath.Join(cacheDir, "cached_shell.php")
	_ = os.WriteFile(cacheFile, []byte("<?php eval(base64_decode('test')); ?>"), 0644)

	scanner := NewScanner()
	// Scan with exclusion of cache directory
	report, err := scanner.ScanDirectoryWithOptions(context.Background(), tempDir, "heuristic", []string{"cache"})
	if err != nil {
		t.Fatalf("ScanDirectoryWithOptions failed: %v", err)
	}

	if report.InfectedFiles != 0 {
		t.Fatalf("expected 0 infected files due to exclusion, got %d", report.InfectedFiles)
	}
}

