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
