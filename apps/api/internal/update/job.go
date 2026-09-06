package update

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the state of an update process
type JobStatus string

const (
	StatusPending        JobStatus = "PENDING"
	StatusPrechecking    JobStatus = "PRECHECKING"
	StatusBackingUp      JobStatus = "BACKING_UP"
	StatusDownloading    JobStatus = "DOWNLOADING"
	StatusVerifying      JobStatus = "VERIFYING"
	StatusPreparing      JobStatus = "PREPARING"
	StatusMigrating      JobStatus = "MIGRATING"
	StatusInstalling     JobStatus = "INSTALLING"
	StatusActivating     JobStatus = "ACTIVATING"
	StatusHealthChecking JobStatus = "HEALTH_CHECKING"
	StatusCompleted      JobStatus = "COMPLETED"
	StatusFailed         JobStatus = "FAILED"
	StatusRollingBack    JobStatus = "ROLLING_BACK"
	StatusRolledBack     JobStatus = "ROLLED_BACK"
)

func (s JobStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusRolledBack
}

func (s JobStatus) IsActive() bool {
	return !s.IsTerminal()
}

// StepStatus represents execution state of an individual update step
type StepStatus string

const (
	StepPending    StepStatus = "PENDING"
	StepRunning    StepStatus = "RUNNING"
	StepCompleted  StepStatus = "COMPLETED"
	StepFailed     StepStatus = "FAILED"
	StepSkipped    StepStatus = "SKIPPED"
	StepRolledBack StepStatus = "ROLLED_BACK"
)

// UpdateJob represents a persistent background update transaction
type UpdateJob struct {
	ID               uuid.UUID      `json:"id"`
	TargetVersion    string         `json:"target_version"`
	Channel          Channel        `json:"channel"`
	Component        string         `json:"component"`
	Status           JobStatus      `json:"status"`
	PreviousVersion  string         `json:"previous_version"`
	NodeID           *uuid.UUID     `json:"node_id,omitempty"`
	BackupSnapshotID *uuid.UUID     `json:"backup_snapshot_id,omitempty"`
	InitiatedBy      *uuid.UUID     `json:"initiated_by,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	Steps            []*UpdateStep  `json:"steps,omitempty"`
}

// UpdateStep represents a discrete milestone within an update job
type UpdateStep struct {
	ID          uuid.UUID  `json:"id"`
	JobID       uuid.UUID  `json:"job_id"`
	StepName    string     `json:"step_name"`
	Status      StepStatus `json:"status"`
	Details     string     `json:"details,omitempty"`
	Logs        string     `json:"logs,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMS  int        `json:"duration_ms"`
}

var (
	ErrUpdateInProgress = errors.New("another update job is currently in progress; concurrent updates are locked")
	ErrInvalidState     = errors.New("invalid state transition")
)

// JobRepository abstracts persistent storage of update jobs and steps
type JobRepository interface {
	CreateJob(ctx context.Context, job *UpdateJob) error
	GetJobByID(ctx context.Context, id uuid.UUID) (*UpdateJob, error)
	GetActiveJob(ctx context.Context) (*UpdateJob, error)
	UpdateJob(ctx context.Context, job *UpdateJob) error
	ListJobs(ctx context.Context, limit int) ([]*UpdateJob, error)
	CreateStep(ctx context.Context, step *UpdateStep) error
	UpdateStep(ctx context.Context, step *UpdateStep) error
	GetJobSteps(ctx context.Context, jobID uuid.UUID) ([]*UpdateStep, error)
}

// JobEngine coordinates persistent background updates and prevents concurrent executions
type JobEngine struct {
	repo  JobRepository
	mu    sync.Mutex
	isBus bool
}

// NewJobEngine creates an initialized JobEngine
func NewJobEngine(repo JobRepository) *JobEngine {
	return &JobEngine{
		repo: repo,
	}
}

// StartJob creates a new update job if no active job is running
func (je *JobEngine) StartJob(
	ctx context.Context,
	targetVersion string,
	previousVersion string,
	channel Channel,
	component string,
	initiatedBy *uuid.UUID,
) (*UpdateJob, error) {
	je.mu.Lock()
	defer je.mu.Unlock()

	active, err := je.repo.GetActiveJob(ctx)
	if err == nil && active != nil {
		return nil, fmt.Errorf("%w: active job ID %s at status %s", ErrUpdateInProgress, active.ID, active.Status)
	}

	now := time.Now().UTC()
	job := &UpdateJob{
		ID:              uuid.New(),
		TargetVersion:   targetVersion,
		PreviousVersion: previousVersion,
		Channel:         channel,
		Component:       component,
		Status:          StatusPending,
		InitiatedBy:     initiatedBy,
		StartedAt:       &now,
		CreatedAt:       now,
		UpdatedAt:       now,
		Steps:           make([]*UpdateStep, 0),
	}

	if err := je.repo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to persist new update job: %w", err)
	}

	return job, nil
}

// Transition advances the job state machine to a new phase
func (je *JobEngine) Transition(ctx context.Context, job *UpdateJob, newStatus JobStatus, stepMessage string) error {
	je.mu.Lock()
	defer je.mu.Unlock()

	now := time.Now().UTC()
	job.Status = newStatus
	job.UpdatedAt = now

	if newStatus.IsTerminal() {
		job.CompletedAt = &now
	}

	step := &UpdateStep{
		ID:         uuid.New(),
		JobID:      job.ID,
		StepName:   string(newStatus),
		Status:     StepRunning,
		Details:    stepMessage,
		StartedAt:  &now,
	}

	if newStatus.IsTerminal() {
		if newStatus == StatusCompleted {
			step.Status = StepCompleted
		} else if newStatus == StatusRolledBack {
			step.Status = StepRolledBack
		} else {
			step.Status = StepFailed
		}
		step.CompletedAt = &now
	}

	if err := je.repo.CreateStep(ctx, step); err != nil {
		return fmt.Errorf("failed to record update step: %w", err)
	}

	if err := je.repo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to persist job transition: %w", err)
	}

	job.Steps = append(job.Steps, step)
	return nil
}

// CompleteStep marks an update step as completed with duration
func (je *JobEngine) CompleteStep(ctx context.Context, step *UpdateStep, details, logs string) error {
	now := time.Now().UTC()
	step.Status = StepCompleted
	step.Details = details
	step.Logs = logs
	step.CompletedAt = &now
	if step.StartedAt != nil {
		step.DurationMS = int(now.Sub(*step.StartedAt).Milliseconds())
	}
	return je.repo.UpdateStep(ctx, step)
}

// FailAndRollback marks the job as ROLLING_BACK and records error
func (je *JobEngine) FailAndRollback(ctx context.Context, job *UpdateJob, reason error) error {
	je.mu.Lock()
	defer je.mu.Unlock()

	now := time.Now().UTC()
	job.Status = StatusRollingBack
	job.ErrorMessage = reason.Error()
	job.UpdatedAt = now

	step := &UpdateStep{
		ID:          uuid.New(),
		JobID:       job.ID,
		StepName:    string(StatusRollingBack),
		Status:      StepRunning,
		Details:     fmt.Sprintf("Critical failure encountered: %v. Initiating automatic rollback.", reason),
		StartedAt:   &now,
	}

	_ = je.repo.CreateStep(ctx, step)
	return je.repo.UpdateJob(ctx, job)
}

// MarkRolledBack marks the job as safely recovered
func (je *JobEngine) MarkRolledBack(ctx context.Context, job *UpdateJob, summary string) error {
	je.mu.Lock()
	defer je.mu.Unlock()

	now := time.Now().UTC()
	job.Status = StatusRolledBack
	job.CompletedAt = &now
	job.UpdatedAt = now

	step := &UpdateStep{
		ID:          uuid.New(),
		JobID:       job.ID,
		StepName:    string(StatusRolledBack),
		Status:      StepCompleted,
		Details:     summary,
		StartedAt:   &now,
		CompletedAt: &now,
	}

	_ = je.repo.CreateStep(ctx, step)
	return je.repo.UpdateJob(ctx, job)
}
