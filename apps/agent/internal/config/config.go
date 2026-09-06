package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AgentConfig struct {
	ServerID             string `json:"server_id"`
	AgentKey             string `json:"agent_key"`
	ControlPlaneURL      string `json:"control_plane_url"`
	HeartbeatIntervalSec int    `json:"heartbeat_interval_sec"`
	ConfigFilePath       string `json:"-"`
}

func DefaultConfigPath() string {
	if os.Geteuid() == 0 {
		return "/etc/hostvra/agent.json"
	}
	return "./agent.json"
}

func Load(path string) (*AgentConfig, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.ConfigFilePath = path

	if cfg.HeartbeatIntervalSec <= 0 {
		cfg.HeartbeatIntervalSec = 10
	}

	return &cfg, nil
}

func (c *AgentConfig) Save(path string) error {
	if path == "" {
		path = c.ConfigFilePath
	}
	if path == "" {
		path = DefaultConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
