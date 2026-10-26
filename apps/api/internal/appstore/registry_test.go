package appstore

import (
	"strings"
	"testing"
)

func TestAppStore_RegistryDefaults(t *testing.T) {
	r := NewRegistry()
	all := r.GetAll()
	if len(all) == 0 {
		t.Fatal("Expected default packages in registry, got 0")
	}

	foundNginx := false
	for _, p := range all {
		if p.ID == "nginx" {
			foundNginx = true
			break
		}
	}
	if !foundNginx {
		t.Error("Expected nginx package in default registry")
	}
}

func TestAppStore_AdaptScriptForOS(t *testing.T) {
	rawDebianScript := "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y redis-server"
	adapted := adaptScriptForOS(rawDebianScript)
	if adapted == "" {
		t.Fatal("Expected non-empty adapted script")
	}

	// On non-apt hosts, verify adaptation
	if !strings.Contains(adapted, "apt-get") {
		if !strings.Contains(adapted, "install -y redis") {
			t.Errorf("Expected adapted dnf/yum command for redis, got: %s", adapted)
		}
	}
}

func TestAppStore_UpdatesAndSettings(t *testing.T) {
	r := NewRegistry()

	// 1. Check updates
	updates := r.CheckUpdates()
	if len(updates) == 0 {
		t.Fatal("Expected packages when checking updates")
	}

	// 2. Test settings retrieval and modification
	st, err := r.GetSettings("redis")
	if err != nil {
		t.Fatalf("Failed to get settings for redis: %v", err)
	}
	if st.AppID != "redis" {
		t.Errorf("Expected app_id redis, got: %s", st.AppID)
	}

	newPort := 6380
	autoUp := true
	cfgTxt := "# Updated Redis config\nport 6380\n"
	updated, err := r.UpdateSettings("redis", UpdateAppSettingsRequest{
		Port:       &newPort,
		AutoUpdate: &autoUp,
		ConfigText: &cfgTxt,
	})
	if err != nil {
		t.Fatalf("Failed to update settings for redis: %v", err)
	}
	if updated.Port != 6380 {
		t.Errorf("Expected updated port 6380, got %d", updated.Port)
	}
	if !updated.AutoUpdate {
		t.Errorf("Expected auto update to be true")
	}

	// 3. Test ExecuteJob update
	job, err := r.ExecuteJob("redis", "update")
	if err != nil {
		t.Fatalf("Failed to execute update job: %v", err)
	}
	if job.Action != "update" {
		t.Errorf("Expected job action 'update', got %s", job.Action)
	}
}
