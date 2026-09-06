package update

import (
	"context"
	"errors"
	"testing"
)

func TestJobEngineLifecycleAndLocking(t *testing.T) {
	repo := NewMemoryJobRepository()
	engine := NewJobEngine(repo)
	ctx := context.Background()

	// 1. Start Initial Job
	job, err := engine.StartJob(ctx, "1.1.0", "1.0.0", ChannelStable, "bundle", nil)
	if err != nil {
		t.Fatalf("failed to start job: %v", err)
	}

	if job.Status != StatusPending {
		t.Errorf("expected job status PENDING, got %s", job.Status)
	}

	// 2. Concurrency Lock Test: Starting second job while first is active must fail
	_, errConcurrent := engine.StartJob(ctx, "1.2.0", "1.0.0", ChannelStable, "bundle", nil)
	if errConcurrent == nil || !errors.Is(errConcurrent, ErrUpdateInProgress) {
		t.Errorf("expected ErrUpdateInProgress for concurrent update attempt, got: %v", errConcurrent)
	}

	// 3. State Machine Transitions
	transitions := []struct {
		status JobStatus
		msg    string
	}{
		{StatusPrechecking, "Running preflight environment checks"},
		{StatusBackingUp, "Taking snapshot of database and configuration"},
		{StatusDownloading, "Downloading release package from updates server"},
		{StatusVerifying, "Verifying Ed25519 signature and SHA-256 checksum"},
		{StatusPreparing, "Staging package in /opt/hostvra/releases/1.1.0"},
		{StatusMigrating, "Applying database migrations"},
		{StatusInstalling, "Installing updated binaries"},
		{StatusActivating, "Switching symlink to new version"},
		{StatusHealthChecking, "Probing control plane & daemon health"},
		{StatusCompleted, "Update to 1.1.0 successfully completed with zero downtime"},
	}

	for _, tr := range transitions {
		err := engine.Transition(ctx, job, tr.status, tr.msg)
		if err != nil {
			t.Fatalf("failed transition to %s: %v", tr.status, err)
		}
	}

	if job.Status != StatusCompleted {
		t.Errorf("expected final status COMPLETED, got %s", job.Status)
	}
	if job.CompletedAt == nil {
		t.Errorf("expected CompletedAt to be non-nil for terminal status")
	}

	// 4. Once completed, lock is released: new job can be started
	newJob, errNext := engine.StartJob(ctx, "1.2.0", "1.1.0", ChannelStable, "bundle", nil)
	if errNext != nil {
		t.Fatalf("expected to be able to start new job after completion of prior: %v", errNext)
	}

	// 5. Failure and Automatic Rollback Flow
	_ = engine.Transition(ctx, newJob, StatusPrechecking, "Starting checks")
	errFail := engine.FailAndRollback(ctx, newJob, errors.New("simulated migration failure"))
	if errFail != nil {
		t.Fatalf("FailAndRollback failed: %v", errFail)
	}
	if newJob.Status != StatusRollingBack {
		t.Errorf("expected status ROLLING_BACK, got %s", newJob.Status)
	}

	errRolledBack := engine.MarkRolledBack(ctx, newJob, "Restored previous release binaries and configs safely")
	if errRolledBack != nil {
		t.Fatalf("MarkRolledBack failed: %v", errRolledBack)
	}
	if newJob.Status != StatusRolledBack {
		t.Errorf("expected status ROLLED_BACK, got %s", newJob.Status)
	}
}

func TestStepCompletionTiming(t *testing.T) {
	repo := NewMemoryJobRepository()
	engine := NewJobEngine(repo)
	ctx := context.Background()

	job, _ := engine.StartJob(ctx, "1.1.0", "1.0.0", ChannelStable, "bundle", nil)
	_ = engine.Transition(ctx, job, StatusBackingUp, "Backing up")

	steps, _ := repo.GetJobSteps(ctx, job.ID)
	if len(steps) == 0 {
		t.Fatalf("expected step to be created")
	}

	step := steps[0]
	err := engine.CompleteStep(ctx, step, "Database snapshot created successfully", "Snapshot ID: 12345")
	if err != nil {
		t.Fatalf("CompleteStep failed: %v", err)
	}

	if step.Status != StepCompleted || step.CompletedAt == nil {
		t.Errorf("expected step COMPLETED with non-nil completed_at")
	}
}
