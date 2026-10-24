package agenttask

import (
	"testing"

	"github.com/google/uuid"
)

func TestAgentTaskManager_Lifecycle(t *testing.T) {
	mgr := NewManager()
	serverID := uuid.New()

	// 1. Enqueue task
	payload := map[string]interface{}{
		"service": "nginx",
	}
	task := mgr.Enqueue(serverID, "service.restart", payload)
	if task == nil || task.ID == uuid.Nil {
		t.Fatalf("expected valid task, got nil")
	}
	if task.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", task.Status)
	}

	// 2. Fetch pending tasks for server
	pending := mgr.FetchPendingForServer(serverID)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(pending))
	}
	if pending[0].ID != task.ID {
		t.Fatalf("expected task %s, got %s", task.ID, pending[0].ID)
	}

	// Second fetch should return 0 since status is now delivered
	secondFetch := mgr.FetchPendingForServer(serverID)
	if len(secondFetch) != 0 {
		t.Fatalf("expected 0 pending tasks on second fetch, got %d", len(secondFetch))
	}

	// 3. Complete task
	completed, err := mgr.Complete(task.ID, StatusCompleted, 0, "Service nginx restarted successfully")
	if err != nil {
		t.Fatalf("failed to complete task: %v", err)
	}
	if completed.Status != StatusCompleted || completed.ExitCode != 0 {
		t.Fatalf("unexpected completed task: %+v", completed)
	}

	// 4. Get task
	fetched, err := mgr.Get(task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if fetched.Output != "Service nginx restarted successfully" {
		t.Fatalf("unexpected task output: %s", fetched.Output)
	}

	// 5. List by server
	list := mgr.ListByServer(serverID, 10)
	if len(list) != 1 {
		t.Fatalf("expected 1 task in server history, got %d", len(list))
	}
}
