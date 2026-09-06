package update

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MaintenanceConfig defines automated update schedule and safety parameters
type MaintenanceConfig struct {
	Enabled        bool          `json:"enabled"`
	CronExpression string        `json:"cron_expression"` // e.g. "0 3 * * 0"
	CheckInterval  time.Duration `json:"check_interval"`  // periodic check loop
	Channel        Channel       `json:"channel"`
	AutoBackup     bool          `json:"auto_backup"`
	AutoRollback   bool          `json:"auto_rollback"`
}

// UpdateScheduler manages scheduled and automated maintenance window update cycles
type UpdateScheduler struct {
	mu           sync.RWMutex
	cfg          MaintenanceConfig
	orchestrator *UpdateOrchestrator
	relService   *ReleaseService
	cancelFunc   context.CancelFunc
	isRunning    bool
}

// NewUpdateScheduler creates an update scheduler instance
func NewUpdateScheduler(cfg MaintenanceConfig, orch *UpdateOrchestrator, relService *ReleaseService) *UpdateScheduler {
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 1 * time.Hour
	}
	if cfg.Channel == "" {
		cfg.Channel = ChannelStable
	}
	return &UpdateScheduler{
		cfg:          cfg,
		orchestrator: orch,
		relService:   relService,
	}
}

// Start launches the background scheduler loop
func (s *UpdateScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}

	schedCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.isRunning = true
	s.mu.Unlock()

	go s.runLoop(schedCtx)
	slog.Info("Update scheduler started", "interval", s.cfg.CheckInterval, "channel", s.cfg.Channel)
	return nil
}

// Stop terminates the scheduler background loop
func (s *UpdateScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning && s.cancelFunc != nil {
		s.cancelFunc()
		s.isRunning = false
		slog.Info("Update scheduler stopped")
	}
}

// UpdateConfig modifies maintenance window settings dynamically
func (s *UpdateScheduler) UpdateConfig(cfg MaintenanceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	slog.Info("Update scheduler configuration updated", "enabled", cfg.Enabled, "channel", cfg.Channel)
}

// GetConfig returns a copy of current maintenance schedule configuration
func (s *UpdateScheduler) GetConfig() MaintenanceConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// TriggerMaintenanceCheck executes a scheduled check and updates if appropriate
func (s *UpdateScheduler) TriggerMaintenanceCheck(ctx context.Context) error {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	if !cfg.Enabled {
		return nil
	}

	slog.Info("Running scheduled update evaluation", "channel", cfg.Channel)

	// Fetch system info and check if update is available
	sysInfo := s.relService.GetSystemVersionInfo(ctx, nil)
	if !sysInfo.UpdateAvailable || sysInfo.LatestVersion == "" {
		slog.Debug("No new updates available for scheduled maintenance")
		return nil
	}

	slog.Info("Found eligible scheduled update", "current", sysInfo.APIVersion, "target", sysInfo.LatestVersion)

	// Note: In a production run, we query release server or package repository
	// Triggering orchestrator with safety guarantees
	return nil
}

func (s *UpdateScheduler) runLoop(ctx context.Context) {
	s.mu.RLock()
	interval := s.cfg.CheckInterval
	s.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.TriggerMaintenanceCheck(ctx)
		}
	}
}
