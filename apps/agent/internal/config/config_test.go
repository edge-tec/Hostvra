package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentConfigSaveLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-agent.json")

	origCfg := &AgentConfig{
		ServerID:             "srv-12345-67890",
		AgentKey:             "hv_agt_secret_token_abc_xyz",
		ControlPlaneURL:      "http://127.0.0.1:8080",
		HeartbeatIntervalSec: 15,
	}

	if err := origCfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.ServerID != origCfg.ServerID {
		t.Errorf("ServerID mismatch: expected %s, got %s", origCfg.ServerID, loadedCfg.ServerID)
	}
	if loadedCfg.AgentKey != origCfg.AgentKey {
		t.Errorf("AgentKey mismatch: expected %s, got %s", origCfg.AgentKey, loadedCfg.AgentKey)
	}
	if loadedCfg.HeartbeatIntervalSec != 15 {
		t.Errorf("HeartbeatInterval mismatch: expected 15, got %d", loadedCfg.HeartbeatIntervalSec)
	}
}
