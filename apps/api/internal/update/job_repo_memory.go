package update

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// MemoryJobRepository implements JobRepository in-memory for testing and non-persistent environments
type MemoryJobRepository struct {
	mu    sync.RWMutex
	jobs  map[uuid.UUID]*UpdateJob
	steps map[uuid.UUID][]*UpdateStep
}

func NewMemoryJobRepository() *MemoryJobRepository {
	return &MemoryJobRepository{
		jobs:  make(map[uuid.UUID]*UpdateJob),
		steps: make(map[uuid.UUID][]*UpdateStep),
	}
}

func (m *MemoryJobRepository) CreateJob(ctx context.Context, job *UpdateJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *MemoryJobRepository) GetJobByID(ctx context.Context, id uuid.UUID) (*UpdateJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	jobCopy := *job
	if steps, hasSteps := m.steps[id]; hasSteps {
		jobCopy.Steps = append([]*UpdateStep(nil), steps...)
	}
	return &jobCopy, nil
}

func (m *MemoryJobRepository) GetActiveJob(ctx context.Context) (*UpdateJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, j := range m.jobs {
		if j.Status.IsActive() {
			jobCopy := *j
			if steps, hasSteps := m.steps[j.ID]; hasSteps {
				jobCopy.Steps = append([]*UpdateStep(nil), steps...)
			}
			return &jobCopy, nil
		}
	}
	return nil, nil
}

func (m *MemoryJobRepository) UpdateJob(ctx context.Context, job *UpdateJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	jobCopy := *job
	m.jobs[job.ID] = &jobCopy
	return nil
}

func (m *MemoryJobRepository) ListJobs(ctx context.Context, limit int) ([]*UpdateJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*UpdateJob
	for _, j := range m.jobs {
		jobCopy := *j
		list = append(list, &jobCopy)
		if limit > 0 && len(list) >= limit {
			break
		}
	}
	return list, nil
}

func (m *MemoryJobRepository) CreateStep(ctx context.Context, step *UpdateStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps[step.JobID] = append(m.steps[step.JobID], step)
	return nil
}

func (m *MemoryJobRepository) UpdateStep(ctx context.Context, step *UpdateStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	steps := m.steps[step.JobID]
	for i, s := range steps {
		if s.ID == step.ID {
			steps[i] = step
			return nil
		}
	}
	return nil
}

func (m *MemoryJobRepository) GetJobSteps(ctx context.Context, jobID uuid.UUID) ([]*UpdateStep, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.steps[jobID], nil
}
