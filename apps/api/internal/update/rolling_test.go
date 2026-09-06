package update

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRollingUpdateCanarySuccess(t *testing.T) {
	node1 := uuid.New()
	node2 := uuid.New()
	node3 := uuid.New()

	var mu sync.Mutex
	executedNodes := make(map[uuid.UUID]bool)
	executor := func(ctx context.Context, nid uuid.UUID, targetVersion string) error {
		mu.Lock()
		executedNodes[nid] = true
		mu.Unlock()
		return nil
	}

	coordinator := NewRollingUpdateCoordinator(executor)
	policy := RollingUpdatePolicy{
		CanaryNodeID:       &node1,
		CanaryWaitDuration: 10 * time.Millisecond,
		BatchSize:          2,
		AbortOnFailure:     true,
	}

	report, err := coordinator.ExecuteRollingUpdate(context.Background(), []uuid.UUID{node1, node2, node3}, "1.1.0", policy)
	if err != nil {
		t.Fatalf("expected successful rolling update, got error: %v", err)
	}

	if report.Successful != 3 {
		t.Errorf("expected 3 successful nodes, got %d", report.Successful)
	}
	if report.Failed != 0 {
		t.Errorf("expected 0 failed nodes, got %d", report.Failed)
	}
	if len(executedNodes) != 3 {
		t.Errorf("expected 3 nodes executed, got %d", len(executedNodes))
	}
}

func TestRollingUpdateCanaryAbortOnFailure(t *testing.T) {
	canaryNode := uuid.New()
	otherNode := uuid.New()

	executor := func(ctx context.Context, nid uuid.UUID, targetVersion string) error {
		if nid == canaryNode {
			return fmt.Errorf("canary health probe failed")
		}
		return nil
	}

	coordinator := NewRollingUpdateCoordinator(executor)
	policy := RollingUpdatePolicy{
		CanaryNodeID:       &canaryNode,
		CanaryWaitDuration: 10 * time.Millisecond,
		BatchSize:          1,
		AbortOnFailure:     true,
		AutoRollbackFailed: true,
	}

	report, err := coordinator.ExecuteRollingUpdate(context.Background(), []uuid.UUID{canaryNode, otherNode}, "1.1.0", policy)
	if err == nil {
		t.Fatalf("expected error from canary failure, got nil")
	}

	if report.Failed != 1 {
		t.Errorf("expected 1 failed node, got %d", report.Failed)
	}
	if report.RolledBack != 1 {
		t.Errorf("expected 1 rolled back node, got %d", report.RolledBack)
	}

	// Other node should not have been updated
	if _, exists := report.NodeResults[otherNode]; exists {
		t.Errorf("other node should not have been touched after canary failure")
	}
}
