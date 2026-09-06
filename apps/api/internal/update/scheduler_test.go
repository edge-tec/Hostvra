package update

import (
	"context"
	"testing"
	"time"
)

func TestUpdateSchedulerLifecycle(t *testing.T) {
	cfg := MaintenanceConfig{
		Enabled:        true,
		CronExpression: "0 3 * * 0",
		CheckInterval:  50 * time.Millisecond,
		Channel:        ChannelStable,
		AutoBackup:     true,
		AutoRollback:   true,
	}

	relService := NewReleaseService("1.0.0", "1.0.0", 5)
	scheduler := NewUpdateScheduler(cfg, nil, relService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Double start should return error
	if err := scheduler.Start(ctx); err == nil {
		t.Errorf("expected error starting already running scheduler, got nil")
	}

	// Update configuration
	cfg.Enabled = false
	scheduler.UpdateConfig(cfg)
	retrieved := scheduler.GetConfig()
	if retrieved.Enabled {
		t.Errorf("expected Enabled=false after update, got true")
	}

	// Trigger manual check
	if err := scheduler.TriggerMaintenanceCheck(ctx); err != nil {
		t.Errorf("unexpected error on manual maintenance check: %v", err)
	}

	scheduler.Stop()
}
