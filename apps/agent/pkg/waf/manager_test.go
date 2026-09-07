package waf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWAFManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "modsec")
	logDir := filepath.Join(tempDir, "log")

	mgr, err := NewManager(configDir, logDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Initial Status
	status, err := mgr.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Mode != ModeOn {
		t.Errorf("expected ModeOn, got %s", status.Mode)
	}
	if status.ParanoiaLevel != 1 {
		t.Errorf("expected paranoia level 1, got %d", status.ParanoiaLevel)
	}
	if status.RulesCount <= 0 {
		t.Errorf("expected positive rules count, got %d", status.RulesCount)
	}

	// Verify modsecurity.conf was written
	modsecConf := filepath.Join(configDir, "modsecurity.conf")
	data, err := os.ReadFile(modsecConf)
	if err != nil {
		t.Fatalf("read modsecurity.conf failed: %v", err)
	}
	if !strings.Contains(string(data), "SecRuleEngine On") {
		t.Errorf("modsecurity.conf missing 'SecRuleEngine On'")
	}

	// 2. Update Global Config
	err = mgr.UpdateGlobalConfig(WAFGlobalConfig{
		Mode:             ModeDetectionOnly,
		ParanoiaLevel:    2,
		AnomalyThreshold: 10,
	})
	if err != nil {
		t.Fatalf("UpdateGlobalConfig failed: %v", err)
	}

	status, _ = mgr.GetStatus()
	if status.Mode != ModeDetectionOnly {
		t.Errorf("expected ModeDetectionOnly, got %s", status.Mode)
	}
	if status.ParanoiaLevel != 2 {
		t.Errorf("expected paranoia level 2, got %d", status.ParanoiaLevel)
	}

	// Verify updated file has DetectionOnly
	data, _ = os.ReadFile(modsecConf)
	if !strings.Contains(string(data), "SecRuleEngine DetectionOnly") {
		t.Errorf("expected SecRuleEngine DetectionOnly in file")
	}

	// 3. Test Invalid Config Validations
	if err := mgr.UpdateGlobalConfig(WAFGlobalConfig{Mode: "InvalidMode"}); err != ErrInvalidMode {
		t.Errorf("expected ErrInvalidMode, got %v", err)
	}
	if err := mgr.UpdateGlobalConfig(WAFGlobalConfig{Mode: ModeOn, ParanoiaLevel: 5}); err != ErrInvalidParanoiaLevel {
		t.Errorf("expected ErrInvalidParanoiaLevel, got %v", err)
	}

	// 4. Rule Categories
	cats, err := mgr.ListRuleCategories()
	if err != nil {
		t.Fatalf("ListRuleCategories failed: %v", err)
	}
	if len(cats) < 8 {
		t.Errorf("expected at least 8 rule categories, got %d", len(cats))
	}

	// Toggle a category off
	err = mgr.ToggleRuleCategory("sqli", false)
	if err != nil {
		t.Fatalf("ToggleRuleCategory failed: %v", err)
	}
	cats, _ = mgr.ListRuleCategories()
	for _, c := range cats {
		if c.ID == "sqli" && c.IsEnabled {
			t.Errorf("expected sqli category to be disabled")
		}
	}

	// 5. Per-Website WAF Config with WordPress Preset
	domain := "wp-blog.example.com"
	err = mgr.UpdateWebsiteWAF(domain, WebsiteWAFConfig{
		Enabled:   true,
		Mode:      "On",
		CMSPreset: "wordpress",
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteWAF failed: %v", err)
	}

	siteCfg, err := mgr.GetWebsiteWAF(domain)
	if err != nil {
		t.Fatalf("GetWebsiteWAF failed: %v", err)
	}
	if len(siteCfg.ExcludedRuleIDs) == 0 {
		t.Errorf("expected excluded rule IDs for WordPress preset")
	}

	// Verify per-site file was written
	siteFile := filepath.Join(configDir, "sites", domain+".conf")
	siteData, err := os.ReadFile(siteFile)
	if err != nil {
		t.Fatalf("read site conf failed: %v", err)
	}
	if !strings.Contains(string(siteData), "modsecurity on;") {
		t.Errorf("site conf missing 'modsecurity on;'")
	}
	if !strings.Contains(string(siteData), "SecRuleRemoveById 941100") {
		t.Errorf("site conf missing WordPress rule exclusion")
	}

	// 6. Simulate Attack Probes
	evSQLi, err := mgr.SimulateProbe("sqli", domain)
	if err != nil {
		t.Fatalf("SimulateProbe sqli failed: %v", err)
	}
	if evSQLi.RuleID != 942100 {
		t.Errorf("expected rule 942100, got %d", evSQLi.RuleID)
	}

	evXSS, err := mgr.SimulateProbe("xss", domain)
	if err != nil {
		t.Fatalf("SimulateProbe xss failed: %v", err)
	}
	if evXSS.RuleCategory != "XSS" {
		t.Errorf("expected XSS category, got %s", evXSS.RuleCategory)
	}

	// 7. Get Attack Events
	events, err := mgr.GetAttackEvents(10, "")
	if err != nil {
		t.Fatalf("GetAttackEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 attack events, got %d", len(events))
	}

	// Filter by category
	sqliEvents, _ := mgr.GetAttackEvents(10, "SQLi")
	if len(sqliEvents) != 1 {
		t.Errorf("expected 1 SQLi event, got %d", len(sqliEvents))
	}
}
