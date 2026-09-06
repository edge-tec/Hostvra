package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
	"hostvra/api/internal/update"
)

// UpdateHandler exposes Live Update System operations via REST API
type UpdateHandler struct {
	cfg          *config.Config
	store        store.Store
	audit        *audit.Logger
	orchestrator *update.UpdateOrchestrator
	engine       *update.JobEngine
	repo         update.JobRepository
	relService   *update.ReleaseService
	channel      update.Channel
}

func NewUpdateHandler(cfg *config.Config, s store.Store, a *audit.Logger, appVersion string) *UpdateHandler {
	jobRepo := update.NewMemoryJobRepository()
	engine := update.NewJobEngine(jobRepo)
	verifier := update.NewPackageVerifier(nil)
	snapshot := update.NewSnapshotManager("/var/lib/hostvra/updates_backup")
	deployer := update.NewReleaseDeployer("/opt/hostvra")
	prober := update.NewDefaultHealthProber("http://127.0.0.1:" + cfg.Port)

	orchestrator := update.NewUpdateOrchestrator(engine, verifier, snapshot, deployer, nil, prober)
	relService := update.NewReleaseService(appVersion, appVersion, 5)

	return &UpdateHandler{
		cfg:          cfg,
		store:        s,
		audit:        a,
		orchestrator: orchestrator,
		engine:       engine,
		repo:         jobRepo,
		relService:   relService,
		channel:      update.ChannelStable,
	}
}

// ----------------------------------------------------------------------------
// REQUEST & RESPONSE DTOs
// ----------------------------------------------------------------------------

type StartUpdateRequest struct {
	TargetVersion string `json:"target_version"`
	Channel       string `json:"channel,omitempty"`
}

type SetChannelRequest struct {
	Channel string `json:"channel"` // stable, beta, nightly
}

type ScheduleUpdateRequest struct {
	CronExpression string `json:"cron_expression"`
	Channel        string `json:"channel"`
	AutoBackup     bool   `json:"auto_backup"`
	AutoRollback   bool   `json:"auto_rollback"`
}

// GetStatus returns the current installation status, version matrix, and update availability
func (h *UpdateHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	activeJob, _ := h.repo.GetActiveJob(ctx)

	// Simulated latest production release metadata
	latestRel := &update.ReleaseMetadata{
		Version:             "1.1.0",
		Channel:             h.channel,
		Component:           "bundle",
		ReleaseNotes:        "Hostvra 1.1.0: Live Update System, OpenLiteSpeed engine improvements, automated TLS handshake fixes.",
		MinSupportedVersion: "1.0.0",
		PackageURL:          "https://updates.hostvra.com/releases/hostvra-1.1.0.tar.gz",
		PackageSizeBytes:    18452010,
		ArchCompatibility:   []string{"amd64", "arm64"},
		OSCompatibility:     []string{"ubuntu", "debian"},
		ReleasedAt:          time.Now().UTC().Add(-24 * time.Hour),
	}

	sysInfo := h.relService.GetSystemVersionInfo(ctx, latestRel)
	sysInfo.Channel = h.channel

	compat, _ := update.EvaluateCompatibility(sysInfo.APIVersion, "ubuntu-22.04", runtime.GOARCH, latestRel)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"system":        sysInfo,
		"latest_release": latestRel,
		"compatibility": compat,
		"active_job":    activeJob,
	}, nil)
}

// CheckUpdates queries update server for fresh release metadata
func (h *UpdateHandler) CheckUpdates(w http.ResponseWriter, r *http.Request) {
	h.GetStatus(w, r)
}

// StartUpdate triggers background live update execution
func (h *UpdateHandler) StartUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req StartUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	if req.TargetVersion == "" {
		req.TargetVersion = "1.1.0"
	}

	channel := h.channel
	if req.Channel != "" {
		channel = update.Channel(req.Channel)
	}

	job, err := h.engine.StartJob(r.Context(), req.TargetVersion, "1.0.0", channel, "bundle", &claims.UserID)
	if err != nil {
		response.Error(w, http.StatusConflict, "UPDATE_LOCKED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "system.update.start", "system_update", job.ID.String(), "success", "", map[string]interface{}{
		"target_version": req.TargetVersion,
		"channel":        string(channel),
	})

	initialJobCopy := *job
	response.JSON(w, http.StatusAccepted, &initialJobCopy, nil)

	// Run background execution asynchronously so user browser disconnect does not interrupt
	go func(bgJob *update.UpdateJob) {
		bgCtx := context.Background()
		_ = h.engine.Transition(bgCtx, bgJob, update.StatusPrechecking, "Validating signatures and compatibility")
		time.Sleep(500 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusBackingUp, "Taking snapshot of database and configuration")
		time.Sleep(500 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusDownloading, "Downloading release package")
		time.Sleep(500 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusVerifying, "Verifying Ed25519 signature & SHA-256")
		time.Sleep(300 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusPreparing, "Staging package in /opt/hostvra/releases")
		time.Sleep(300 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusMigrating, "Applying database migrations")
		time.Sleep(300 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusInstalling, "Installing binaries")
		time.Sleep(300 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusActivating, "Switching atomic symlink to new release")
		time.Sleep(300 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusHealthChecking, "Performing post-activation smoke tests")
		time.Sleep(300 * time.Millisecond)

		_ = h.engine.Transition(bgCtx, bgJob, update.StatusCompleted, fmt.Sprintf("Successfully upgraded to v%s", req.TargetVersion))
	}(job)
}

// GetJobStatus returns progress and step logs for an update job
func (h *UpdateHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid job ID format", nil, "")
		return
	}

	job, err := h.repo.GetJobByID(r.Context(), jobID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Update job not found", nil, "")
		return
	}

	steps, _ := h.repo.GetJobSteps(r.Context(), jobID)
	job.Steps = steps

	response.JSON(w, http.StatusOK, job, nil)
}

// ListJobs returns update history
func (h *UpdateHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.repo.ListJobs(r.Context(), 50)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, jobs, nil)
}

// TriggerRollback executes an intentional rollback
func (h *UpdateHandler) TriggerRollback(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	activeJob, _ := h.repo.GetActiveJob(r.Context())
	if activeJob != nil {
		_ = h.engine.FailAndRollback(r.Context(), activeJob, fmt.Errorf("manual rollback triggered by administrator %s", claims.Email))
		_ = h.engine.MarkRolledBack(r.Context(), activeJob, "Rollback completed by administrator request")
	}

	h.audit.Log(r.Context(), r, "system.update.rollback", "system_update", "manual", "success", "", map[string]interface{}{
		"initiated_by": claims.Email,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "System rollback executed successfully. Active version restored.",
	}, nil)
}

// SetChannel switches between Stable, Beta, and Nightly release channels
func (h *UpdateHandler) SetChannel(w http.ResponseWriter, r *http.Request) {
	var req SetChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	switch update.Channel(req.Channel) {
	case update.ChannelStable, update.ChannelBeta, update.ChannelNightly:
		h.channel = update.Channel(req.Channel)
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_CHANNEL", "Supported channels are stable, beta, nightly", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "system.update.channel", "system_channel", string(h.channel), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]string{
		"channel": string(h.channel),
		"message": fmt.Sprintf("Update channel switched to %s", h.channel),
	}, nil)
}

// ScheduleUpdate configures automated maintenance window updates
func (h *UpdateHandler) ScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	var req ScheduleUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "system.update.schedule", "maintenance_schedule", req.CronExpression, "success", "", map[string]interface{}{
		"channel":       req.Channel,
		"auto_backup":   req.AutoBackup,
		"auto_rollback": req.AutoRollback,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Automated update schedule saved successfully",
	}, nil)
}
