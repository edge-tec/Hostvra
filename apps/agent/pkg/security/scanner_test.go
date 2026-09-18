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
