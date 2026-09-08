package migration

import (
	"context"
	"testing"
	"time"
)

func TestMigrationManager(t *testing.T) {
	mgr := NewManager()

	// 1. Invalid input validation
	_, err := mgr.CreateJob(context.Background(), CreateMigrationRequest{})
	if err == nil {
		t.Fatalf("expected error on empty request, got nil")
	}

	// 2. Create job with unresolvable host (should connect, then fail cleanly)
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
