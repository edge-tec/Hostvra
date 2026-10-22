package rspamd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRspamd_ManagerTelemetryAndActions(t *testing.T) {
	// 1. Mock HTTP API server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stat" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"scanned": 15420,
				"clean": 13900,
				"greylist": 320,
				"spam": 1100,
				"reject": 100,
				"learned_ham": 2500,
				"learned_spam": 1800,
				"uptime": 864000
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "rspamd_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	mgr.AdminURL = ts.URL

	// 2. Telemetry polling
	stats, err := mgr.GetTelemetry(context.Background())
	if err != nil {
		t.Fatalf("GetTelemetry failed: %v", err)
	}
	if stats.TotalScanned != 15420 {
		t.Errorf("expected 15420 scanned, got %d", stats.TotalScanned)
	}
	if stats.SpamCount != 1100 {
		t.Errorf("expected 1100 spam, got %d", stats.SpamCount)
	}

	// 3. Thresholds writing
	err = mgr.UpdateThresholds(context.Background(), 14.0, 5.5, 3.5)
	if err != nil {
		t.Fatalf("UpdateThresholds failed: %v", err)
	}

	actionsPath := filepath.Join(tmpDir, "local.d", "actions.conf")
	content, err := os.ReadFile(actionsPath)
	if err != nil {
		t.Fatalf("failed reading actions.conf: %v", err)
	}
	if !strings.Contains(string(content), "reject = 14.0;") {
		t.Errorf("actions.conf missing reject score: %s", string(content))
	}

	// 4. Quarantine Release
	dummyMail := filepath.Join(tmpDir, "quarantine_msg.eml")
	os.WriteFile(dummyMail, []byte("From: sender@test.com\r\nTo: user@test.com\r\nSubject: Test\r\n\r\nHello"), 0644)

	relResult := mgr.ReleaseQuarantinedMessage(context.Background(), dummyMail, "user@test.com")
	if !relResult.Success {
		t.Errorf("expected release success, got error: %s", relResult.Error)
	}
}
