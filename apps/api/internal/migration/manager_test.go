package migration

import (
	"context"
	"net"
	"testing"
	"time"

	"hostvra/api/internal/store"
)

func TestMigrationManager(t *testing.T) {
	mgr := NewManager()
	memStore := store.NewMemoryStore()
	mgr.SetStore(memStore)

	// 1. Invalid input validation
	_, err := mgr.CreateJob(context.Background(), CreateMigrationRequest{})
	if err == nil {
		t.Fatalf("expected error on empty request, got nil")
	}

	// 2. Create job with non-listening port (should test TCP, then fail cleanly)
	job, err := mgr.CreateJob(context.Background(), CreateMigrationRequest{
		SourceType: "hostvra",
		SourceHost: "127.0.0.1",
		SourcePort: 54321, // non-listening port
		AuthType:   "api_key",
		APIKey:     "test-key",
	})
	if err != nil {
		t.Fatalf("failed to create migration job: %v", err)
	}

	if job.ID.String() == "" {
		t.Fatalf("expected non-empty job ID")
	}

	// 3. List jobs
	jobs := mgr.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	// 4. Get job
	fetched, err := mgr.GetJob(job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}
	if fetched.ID != job.ID {
		t.Fatalf("expected job ID %s, got %s", job.ID, fetched.ID)
	}

	// Wait for background execution to complete or fail
	time.Sleep(100 * time.Millisecond)

	// 5. Cancel Job
	_, err = mgr.CancelJob(job.ID)
	if err != nil {
		t.Fatalf("failed to cancel job: %v", err)
	}
}

func TestMigrationManager_MockLifecycle(t *testing.T) {
	// Start a local test TCP listener so DialTimeout succeeds
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start local test listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	mgr := NewManager()
	memStore := store.NewMemoryStore()
	mgr.SetStore(memStore)

	job, err := mgr.CreateJob(context.Background(), CreateMigrationRequest{
		SourceType: "cpanel",
		SourceHost: "127.0.0.1",
		SourcePort: port,
		AuthType:   "password",
		Username:   "testuser",
		Password:   "testpass",
		MockMode:   true,
	})
	if err != nil {
		t.Fatalf("failed to start mock migration: %v", err)
	}

	// Wait briefly for lifecycle to progress
	var finalJob *MigrationJob
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		finalJob, _ = mgr.GetJob(job.ID)
		if finalJob != nil && (finalJob.Status == StatusCompleted || finalJob.Status == StatusFailed) {
			break
		}
	}

	if finalJob == nil || finalJob.Status != StatusCompleted {
		t.Fatalf("expected migration to complete, got status %v, err=%v", finalJob.Status, finalJob.ErrorMessage)
	}

	if finalJob.Progress != 100 {
		t.Fatalf("expected progress 100, got %f", finalJob.Progress)
	}

	if len(finalJob.Logs) < 5 {
		t.Fatalf("expected multiple log entries, got %d", len(finalJob.Logs))
	}
}
