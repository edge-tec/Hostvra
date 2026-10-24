package agenttask

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusDelivered TaskStatus = "delivered"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
)

type AgentTask struct {
	ID         uuid.UUID              `json:"id"`
	ServerID   uuid.UUID              `json:"server_id"`
	Action     string                 `json:"action"` // e.g. "service.reload", "service.restart", "command.exec"
	Payload    map[string]interface{} `json:"payload"`
	Status     TaskStatus             `json:"status"`
	ExitCode   int                    `json:"exit_code"`
	Output     string                 `json:"output"`
	CreatedAt  time.Time              `json:"created_at"`
	ExecutedAt *time.Time             `json:"executed_at,omitempty"`
}

type Manager struct {
	mu    sync.RWMutex
	tasks map[uuid.UUID]*AgentTask
}

func NewManager() *Manager {
	return &Manager{
		tasks: make(map[uuid.UUID]*AgentTask),
	}
}

func (m *Manager) Enqueue(serverID uuid.UUID, action string, payload map[string]interface{}) *AgentTask {
	m.mu.Lock()
	defer m.mu.Unlock()

	if payload == nil {
		payload = make(map[string]interface{})
	}

	task := &AgentTask{
		ID:        uuid.New(),
		ServerID:  serverID,
		Action:    action,
		Payload:   payload,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}
	m.tasks[task.ID] = task
	return task
}

func (m *Manager) FetchPendingForServer(serverID uuid.UUID) []*AgentTask {
	m.mu.Lock()
	defer m.mu.Unlock()

	var pending []*AgentTask
	for _, t := range m.tasks {
		if t.ServerID == serverID && t.Status == StatusPending {
			t.Status = StatusDelivered
			copyT := *t
			pending = append(pending, &copyT)
		}
	}
	return pending
}

func (m *Manager) Complete(taskID uuid.UUID, status TaskStatus, exitCode int, output string) (*AgentTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}

	now := time.Now().UTC()
	task.Status = status
	task.ExitCode = exitCode
	task.Output = output
	task.ExecutedAt = &now

	copyT := *task
	return &copyT, nil
}

func (m *Manager) Get(taskID uuid.UUID) (*AgentTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	copyT := *task
	return &copyT, nil
}

func (m *Manager) ListByServer(serverID uuid.UUID, limit int) []*AgentTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*AgentTask
	for _, t := range m.tasks {
		if t.ServerID == serverID {
			copyT := *t
			list = append(list, &copyT)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list
}
