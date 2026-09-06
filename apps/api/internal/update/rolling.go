package update

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RollingUpdatePolicy defines parameters for multi-server fleet updates
type RollingUpdatePolicy struct {
	CanaryNodeID       *uuid.UUID    `json:"canary_node_id,omitempty"`
	CanaryWaitDuration time.Duration `json:"canary_wait_duration"`
	BatchSize          int           `json:"batch_size"` // number of servers per batch
	MaxParallel        int           `json:"max_parallel"`
	AbortOnFailure     bool          `json:"abort_on_failure"`
	AutoRollbackFailed bool          `json:"auto_rollback_failed"`
}

// NodeUpdateState tracks individual server update status in fleet
type NodeUpdateState struct {
	NodeID       uuid.UUID  `json:"node_id"`
	Status       JobStatus  `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// FleetUpdateReport summarizes the multi-server deployment outcome
type FleetUpdateReport struct {
	TargetVersion string                     `json:"target_version"`
	TotalNodes    int                        `json:"total_nodes"`
	Successful    int                        `json:"successful"`
	Failed        int                        `json:"failed"`
	RolledBack    int                        `json:"rolled_back"`
	NodeResults   map[uuid.UUID]*NodeUpdateState `json:"node_results"`
}

// NodeUpdateExecutor defines the function signature for updating a single remote node
type NodeUpdateExecutor func(ctx context.Context, nodeID uuid.UUID, targetVersion string) error

// RollingUpdateCoordinator orchestrates canary-first rolling updates across multi-server fleets
type RollingUpdateCoordinator struct {
	executor NodeUpdateExecutor
}

// NewRollingUpdateCoordinator creates a new multi-server rolling coordinator
func NewRollingUpdateCoordinator(executor NodeUpdateExecutor) *RollingUpdateCoordinator {
	return &RollingUpdateCoordinator{
		executor: executor,
	}
}

// ExecuteRollingUpdate executes rolling deployment across the given node list
func (c *RollingUpdateCoordinator) ExecuteRollingUpdate(
	ctx context.Context,
	nodes []uuid.UUID,
	targetVersion string,
	policy RollingUpdatePolicy,
) (*FleetUpdateReport, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes provided for rolling update")
	}

	report := &FleetUpdateReport{
		TargetVersion: targetVersion,
		TotalNodes:    len(nodes),
		NodeResults:   make(map[uuid.UUID]*NodeUpdateState),
	}

	// 1. Canary Deployment Phase if canary node is specified
	remainingNodes := make([]uuid.UUID, 0, len(nodes))
	if policy.CanaryNodeID != nil {
		canaryID := *policy.CanaryNodeID
		canaryState := &NodeUpdateState{
			NodeID:    canaryID,
			Status:    StatusInstalling,
			StartedAt: time.Now().UTC(),
		}
		report.NodeResults[canaryID] = canaryState

		err := c.executor(ctx, canaryID, targetVersion)
		now := time.Now().UTC()
		canaryState.CompletedAt = &now

		if err != nil {
			canaryState.Status = StatusFailed
			canaryState.ErrorMessage = err.Error()
			report.Failed++

			if policy.AutoRollbackFailed {
				canaryState.Status = StatusRolledBack
				report.RolledBack++
			}
			return report, fmt.Errorf("canary node %s update failed: %w; rolling update aborted", canaryID, err)
		}

		canaryState.Status = StatusCompleted
		report.Successful++

		// Wait observation period
		if policy.CanaryWaitDuration > 0 {
			select {
			case <-ctx.Done():
				return report, ctx.Err()
			case <-time.After(policy.CanaryWaitDuration):
			}
		}

		// Filter out canary node from remaining list
		for _, n := range nodes {
			if n != canaryID {
				remainingNodes = append(remainingNodes, n)
			}
		}
	} else {
		remainingNodes = append(remainingNodes, nodes...)
	}

	// 2. Batch Execution Phase
	batchSize := policy.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	for i := 0; i < len(remainingNodes); i += batchSize {
		end := i + batchSize
		if end > len(remainingNodes) {
			end = len(remainingNodes)
		}
		batch := remainingNodes[i:end]

		var wg sync.WaitGroup
		var batchMu sync.Mutex
		var batchError error

		for _, nodeID := range batch {
			nodeState := &NodeUpdateState{
				NodeID:    nodeID,
				Status:    StatusInstalling,
				StartedAt: time.Now().UTC(),
			}
			report.NodeResults[nodeID] = nodeState

			wg.Add(1)
			go func(nid uuid.UUID, state *NodeUpdateState) {
				defer wg.Done()
				err := c.executor(ctx, nid, targetVersion)
				now := time.Now().UTC()
				state.CompletedAt = &now

				batchMu.Lock()
				defer batchMu.Unlock()

				if err != nil {
					state.Status = StatusFailed
					state.ErrorMessage = err.Error()
					report.Failed++
					if policy.AutoRollbackFailed {
						state.Status = StatusRolledBack
						report.RolledBack++
					}
					if batchError == nil {
						batchError = fmt.Errorf("node %s update failed: %w", nid, err)
					}
				} else {
					state.Status = StatusCompleted
					report.Successful++
				}
			}(nodeID, nodeState)
		}

		wg.Wait()

		if batchError != nil && policy.AbortOnFailure {
			return report, fmt.Errorf("rolling update halted due to batch error: %w", batchError)
		}
	}

	return report, nil
}
