package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/appruntime"
	"hostvra/agent/pkg/gitcicd"
	"hostvra/agent/pkg/isolation"
	"hostvra/agent/pkg/wordpress"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

// DeveloperHandler orchestrates Node.js/Python apps, Git CI/CD, and WordPress staging
type DeveloperHandler struct {
	cfg           *config.Config
	store         store.Store
	audit         *audit.Logger
	appRuntimeMgr *appruntime.Manager
	gitMgr        *gitcicd.Manager
	wpMgr         *wordpress.Manager
}

// NewDeveloperHandler initializes the unified DeveloperHandler
func NewDeveloperHandler(cfg *config.Config, s store.Store, a *audit.Logger) *DeveloperHandler {
	return &DeveloperHandler{
		cfg:           cfg,
		store:         s,
		audit:         a,
		appRuntimeMgr: appruntime.NewManager(""),
		gitMgr:        gitcicd.NewManager(),
		wpMgr:         wordpress.NewManager(),
	}
}

// Setters for test mock injection
func (h *DeveloperHandler) SetAppRuntimeManager(m *appruntime.Manager) {
	h.appRuntimeMgr = m
}

func (h *DeveloperHandler) SetGitManager(m *gitcicd.Manager) {
	h.gitMgr = m
}

func (h *DeveloperHandler) SetWordPressManager(m *wordpress.Manager) {
	h.wpMgr = m
}

// verifyWebsiteAccess validates JWT claims and website ownership
func (h *DeveloperHandler) verifyWebsiteAccess(r *http.Request) (*store.Website, *auth.Claims, error) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		return nil, nil, errors.New("unauthorized")
	}
	siteIDStr := chi.URLParam(r, "id")
	if siteIDStr == "" {
		return nil, nil, errors.New("missing website id")
	}
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		return nil, nil, errors.New("invalid website ID")
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site == nil {
		return nil, nil, errors.New("website not found")
	}

	isAdmin := claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin"
	if site.OrganizationID != claims.OrganizationID && !isAdmin {
		return nil, nil, errors.New("forbidden")
	}

	return site, claims, nil
}

// ============================================================================
// 1. MANAGED APPLICATIONS (NODE.JS & PYTHON)
// ============================================================================

type SaveAppRequest struct {
	Name           string            `json:"name"`
	AppType        string            `json:"app_type"`        // "nodejs" or "python"
	RuntimeVersion string            `json:"runtime_version"` // "18", "20", "22", "3.10", "3.11", "3.12"
	AppRoot        string            `json:"app_root"`
	Entrypoint     string            `json:"entrypoint"`
	StartCommand   string            `json:"start_command"`
	Port           int               `json:"port"`
	EnvVars        map[string]string `json:"env_vars"`
}

type ControlAppRequest struct {
	Action string `json:"action"` // "start", "stop", "restart"
}

// GetManagedApp returns the configured Node/Python application for a website
func (h *DeveloperHandler) GetManagedApp(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	app, err := h.store.GetManagedAppByWebsite(r.Context(), site.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.JSON(w, http.StatusOK, nil, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, app, nil)
}

// SaveManagedApp creates or updates a supervised application
func (h *DeveloperHandler) SaveManagedApp(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req SaveAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	if req.AppType != "nodejs" && req.AppType != "python" {
		response.Error(w, http.StatusBadRequest, "INVALID_APP_TYPE", "app_type must be nodejs or python", nil, "")
		return
	}
	if req.AppRoot == "" {
		req.AppRoot = site.DocumentRoot
	}
	if req.Entrypoint == "" {
		if req.AppType == "nodejs" {
			req.Entrypoint = "index.js"
		} else {
			req.Entrypoint = "app.py"
		}
	}
	if req.Name == "" {
		req.Name = site.PrimaryDomain
	}

	// Dynamic port allocation if not provided or out of bounds
	if req.Port < 3000 || req.Port > 9999 {
		port, err := h.appRuntimeMgr.AllocatePort(3000)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "PORT_ERROR", err.Error(), nil, "")
			return
		}
		req.Port = port
	}

	// Fetch or create record
	existing, _ := h.store.GetManagedAppByWebsite(r.Context(), site.ID)
	appID := uuid.New()
	if existing != nil {
		appID = existing.ID
	}

	systemUser := isolation.DeriveUsername(site.PrimaryDomain)

	record := &store.ManagedAppRecord{
		ID:             appID,
		WebsiteID:      site.ID,
		OrganizationID: site.OrganizationID,
		Name:           req.Name,
		AppType:        req.AppType,
		RuntimeVersion: req.RuntimeVersion,
		AppRoot:        req.AppRoot,
		Entrypoint:     req.Entrypoint,
		StartCommand:   req.StartCommand,
		Port:           req.Port,
		SystemUser:     systemUser,
		EnvVars:        req.EnvVars,
		Status:         "stopped",
		UpdatedAt:      time.Now().UTC(),
	}

	// Start or configure supervision
	cfg := &appruntime.ManagedAppConfig{
		ID:             appID.String(),
		WebsiteID:      site.ID.String(),
		Name:           record.Name,
		AppType:        record.AppType,
		RuntimeVersion: record.RuntimeVersion,
		AppRoot:        record.AppRoot,
		Entrypoint:     record.Entrypoint,
		StartCommand:   record.StartCommand,
		Port:           record.Port,
		SystemUser:     systemUser,
		EnvVars:        record.EnvVars,
	}

	// If python, set up virtualenv
	if record.AppType == "python" {
		_ = h.appRuntimeMgr.SetupVirtualEnv(r.Context(), record.AppRoot)
	}

	// Start application process
	status, err := h.appRuntimeMgr.StartApp(r.Context(), cfg)
	if err == nil && status != nil {
		record.Status = status.Status
	}

	if err := h.store.SaveManagedApp(r.Context(), record); err != nil {
		response.Error(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "developer.app.save", "website", site.ID.String(), "success", "", map[string]interface{}{
			"app_type": record.AppType,
			"port":     record.Port,
			"status":   record.Status,
		})
	}

	response.JSON(w, http.StatusOK, record, nil)
}

// ControlManagedApp starts, stops, or restarts the supervised application
func (h *DeveloperHandler) ControlManagedApp(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	app, err := h.store.GetManagedAppByWebsite(r.Context(), site.ID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "No managed application configured for this website", nil, "")
		return
	}

	var req ControlAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	var status *appruntime.AppStatus
	var opErr error

	switch strings.ToLower(req.Action) {
	case "start":
		cfg := &appruntime.ManagedAppConfig{
			ID:             app.ID.String(),
			WebsiteID:      site.ID.String(),
			Name:           app.Name,
			AppType:        app.AppType,
			RuntimeVersion: app.RuntimeVersion,
			AppRoot:        app.AppRoot,
			Entrypoint:     app.Entrypoint,
			StartCommand:   app.StartCommand,
			Port:           app.Port,
			SystemUser:     app.SystemUser,
			EnvVars:        app.EnvVars,
		}
		status, opErr = h.appRuntimeMgr.StartApp(r.Context(), cfg)
	case "stop":
		status, opErr = h.appRuntimeMgr.StopApp(r.Context(), app.ID.String())
	case "restart":
		status, opErr = h.appRuntimeMgr.RestartApp(r.Context(), app.ID.String())
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_ACTION", "action must be start, stop, or restart", nil, "")
		return
	}

	if opErr != nil {
		response.Error(w, http.StatusInternalServerError, "PROCESS_ERROR", opErr.Error(), nil, "")
		return
	}

	if status != nil {
		app.Status = status.Status
		if status.RestartCount > 0 {
			app.RestartCount = status.RestartCount
		}
		_ = h.store.SaveManagedApp(r.Context(), app)
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "developer.app.control", "website", site.ID.String(), "success", "", map[string]interface{}{
			"action": req.Action,
			"status": app.Status,
		})
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"action":  req.Action,
		"status":  app.Status,
		"metrics": status,
	}, nil)
}

// GetAppStatus inspects live runtime status and resource metrics
func (h *DeveloperHandler) GetAppStatus(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	app, err := h.store.GetManagedAppByWebsite(r.Context(), site.ID)
	if err != nil {
		response.JSON(w, http.StatusOK, &appruntime.AppStatus{Status: "not_configured"}, nil)
		return
	}

	status, err := h.appRuntimeMgr.GetStatus(r.Context(), app.ID.String())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STATUS_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, status, nil)
}

// DeleteManagedApp deletes supervision and removes systemd unit
func (h *DeveloperHandler) DeleteManagedApp(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	app, err := h.store.GetManagedAppByWebsite(r.Context(), site.ID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "No managed application found", nil, "")
		return
	}

	_, _ = h.appRuntimeMgr.StopApp(r.Context(), app.ID.String())
	if err := h.store.DeleteManagedApp(r.Context(), app.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "developer.app.delete", "website", site.ID.String(), "success", "", nil)
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

// DetectRuntimes returns discovered Node.js and Python runtimes on the host
func (h *DeveloperHandler) DetectRuntimes(w http.ResponseWriter, r *http.Request) {
	runtimes, err := h.appRuntimeMgr.DetectRuntimes(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DETECT_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, runtimes, nil)
}

// ============================================================================
// 2. GIT-BASED CI/CD PIPELINE
// ============================================================================

type SaveGitRepoRequest struct {
	Provider         string `json:"provider"` // "github", "gitlab", "custom"
	RepoURL          string `json:"repo_url"`
	Branch           string `json:"branch"`
	AutoDeploy       bool   `json:"auto_deploy"`
	BuildCommand     string `json:"build_command"`
	DeployPath       string `json:"deploy_path"`
	ReleaseKeepCount int    `json:"release_keep_count"`
}

type TriggerDeployRequest struct {
	Branch string `json:"branch,omitempty"`
}

// GetGitRepo returns Git CI/CD repository settings and public deploy key
func (h *DeveloperHandler) GetGitRepo(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	repo, err := h.store.GetGitRepository(r.Context(), site.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.JSON(w, http.StatusOK, nil, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	// Hide private key from standard GET
	cleanRepo := *repo
	cleanRepo.DeployKeyPriv = ""
	response.JSON(w, http.StatusOK, cleanRepo, nil)
}

// SaveGitRepo configures Git repository integration and generates deploy keypair
func (h *DeveloperHandler) SaveGitRepo(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req SaveGitRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	if req.RepoURL == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REPO", "repo_url is required", nil, "")
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.Provider == "" {
		if strings.Contains(req.RepoURL, "gitlab") {
			req.Provider = "gitlab"
		} else {
			req.Provider = "github"
		}
	}
	if req.DeployPath == "" {
		req.DeployPath = site.DocumentRoot
	}
	if req.ReleaseKeepCount <= 0 {
		req.ReleaseKeepCount = 5
	}

	existing, _ := h.store.GetGitRepository(r.Context(), site.ID)
	repoID := uuid.New()
	pubKey := ""
	privKey := ""
	secret := ""

	if existing != nil {
		repoID = existing.ID
		pubKey = existing.DeployKeyPub
		privKey = existing.DeployKeyPriv
		secret = existing.WebhookSecret
	}

	// Generate keypair if not already present
	if pubKey == "" || privKey == "" {
		kp, err := h.gitMgr.GenerateDeployKeyPair(fmt.Sprintf("hostvra-deploy-%s", site.PrimaryDomain))
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "KEYGEN_ERROR", err.Error(), nil, "")
			return
		}
		pubKey = kp.PublicKey
		privKey = kp.PrivateKey
	}

	// Generate webhook secret if not present
	if secret == "" {
		secret = uuid.New().String()
	}

	repo := &store.GitRepositoryRecord{
		ID:               repoID,
		WebsiteID:        site.ID,
		OrganizationID:   site.OrganizationID,
		Provider:         req.Provider,
		RepoURL:          req.RepoURL,
		Branch:           req.Branch,
		WebhookSecret:    secret,
		DeployKeyPub:     pubKey,
		DeployKeyPriv:    privKey,
		AutoDeploy:       req.AutoDeploy,
		BuildCommand:     req.BuildCommand,
		DeployPath:       req.DeployPath,
		ReleaseKeepCount: req.ReleaseKeepCount,
		UpdatedAt:        time.Now().UTC(),
	}

	if err := h.store.SaveGitRepository(r.Context(), repo); err != nil {
		response.Error(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "git.repo.save", "website", site.ID.String(), "success", "", map[string]interface{}{
			"repo_url": repo.RepoURL,
			"branch":   repo.Branch,
		})
	}

	cleanRepo := *repo
	cleanRepo.DeployKeyPriv = ""
	response.JSON(w, http.StatusOK, cleanRepo, nil)
}

// DeleteGitRepo removes Git repository integration
func (h *DeveloperHandler) DeleteGitRepo(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	if err := h.store.DeleteGitRepository(r.Context(), site.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "git.repo.delete", "website", site.ID.String(), "success", "", nil)
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

// TriggerDeploy runs manual deployment pipeline
func (h *DeveloperHandler) TriggerDeploy(w http.ResponseWriter, r *http.Request) {
	site, claims, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	repo, err := h.store.GetGitRepository(r.Context(), site.ID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Git repository not configured for this website", nil, "")
		return
	}

	var req TriggerDeployRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetBranch := repo.Branch
	if req.Branch != "" {
		targetBranch = req.Branch
	}

	depID := uuid.New()
	depRecord := &store.GitDeploymentRecord{
		ID:           depID,
		RepositoryID: repo.ID,
		WebsiteID:    site.ID,
		Status:       "building",
		Author:       claims.Email,
		CreatedAt:    time.Now().UTC(),
	}
	_ = h.store.CreateGitDeployment(r.Context(), depRecord)

	deployCfg := &gitcicd.DeployConfig{
		WebsiteID:    site.ID.String(),
		RepoURL:      repo.RepoURL,
		Branch:       targetBranch,
		DeployPath:   repo.DeployPath,
		DeployKey:    repo.DeployKeyPriv,
		BuildCommand: repo.BuildCommand,
		KeepReleases: repo.ReleaseKeepCount,
		SystemUser:   isolation.DeriveUsername(site.PrimaryDomain),
	}

	res, depErr := h.gitMgr.DeployRelease(r.Context(), deployCfg)
	if depErr != nil && res == nil {
		depRecord.Status = "failed"
		depRecord.BuildLogs = depErr.Error()
		_ = h.store.UpdateGitDeployment(r.Context(), depRecord)
		response.Error(w, http.StatusInternalServerError, "DEPLOY_FAILED", depErr.Error(), nil, "")
		return
	}

	depRecord.CommitHash = res.CommitHash
	depRecord.CommitMessage = res.CommitMessage
	depRecord.Author = res.Author
	depRecord.Status = res.Status
	depRecord.BuildLogs = res.BuildLogs
	depRecord.ReleasePath = res.ReleasePath
	depRecord.DurationSeconds = res.DurationSeconds

	_ = h.store.UpdateGitDeployment(r.Context(), depRecord)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "git.deploy", "website", site.ID.String(), "success", "", map[string]interface{}{
			"commit": res.CommitHash,
			"status": res.Status,
		})
	}

	response.JSON(w, http.StatusOK, depRecord, nil)
}

// ListDeployments returns past deployment executions
func (h *DeveloperHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	deps, err := h.store.ListGitDeployments(r.Context(), site.ID, 50)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, deps, nil)
}

// RollbackDeployment reverts the atomic symlink to an earlier release folder
func (h *DeveloperHandler) RollbackDeployment(w http.ResponseWriter, r *http.Request) {
	site, claims, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	depIDStr := chi.URLParam(r, "deploymentId")
	depID, err := uuid.Parse(depIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid deployment ID", nil, "")
		return
	}

	dep, err := h.store.GetGitDeployment(r.Context(), depID)
	if err != nil || dep.WebsiteID != site.ID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Deployment record not found", nil, "")
		return
	}

	repo, err := h.store.GetGitRepository(r.Context(), site.ID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Repository not configured", nil, "")
		return
	}

	releaseID := filepath.Base(dep.ReleasePath)
	if err := h.gitMgr.RollbackRelease(repo.DeployPath, releaseID); err != nil {
		response.Error(w, http.StatusInternalServerError, "ROLLBACK_FAILED", err.Error(), nil, "")
		return
	}

	// Record rollback deployment execution
	rbRecord := &store.GitDeploymentRecord{
		ID:            uuid.New(),
		RepositoryID:  repo.ID,
		WebsiteID:     site.ID,
		CommitHash:    dep.CommitHash,
		CommitMessage: fmt.Sprintf("Rolled back to release %s (%s)", releaseID, dep.CommitHash),
		Author:        claims.Email,
		Status:        "rolled_back",
		BuildLogs:     fmt.Sprintf("Symlink reverted to %s successfully\n", dep.ReleasePath),
		ReleasePath:   dep.ReleasePath,
		CreatedAt:     time.Now().UTC(),
	}
	_ = h.store.CreateGitDeployment(r.Context(), rbRecord)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "git.rollback", "website", site.ID.String(), "success", "", map[string]interface{}{
			"target_release": releaseID,
			"commit":         dep.CommitHash,
		})
	}

	response.JSON(w, http.StatusOK, rbRecord, nil)
}

// HandleWebhook receives signed GitHub / GitLab push events (Public endpoint)
func (h *DeveloperHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SITE", "Invalid website ID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	repo, err := h.store.GetGitRepository(r.Context(), site.ID)
	if err != nil || repo == nil || !repo.AutoDeploy {
		response.JSON(w, http.StatusOK, map[string]string{"message": "Auto-deploy disabled or repository not configured"}, nil)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BODY_READ_ERROR", "Failed to read webhook payload", nil, "")
		return
	}

	// Signature verification
	ghSig := r.Header.Get("X-Hub-Signature-256")
	glToken := r.Header.Get("X-Gitlab-Token")

	sigToVerify := ghSig
	provider := "github"
	if glToken != "" {
		sigToVerify = glToken
		provider = "gitlab"
	}

	if !h.gitMgr.VerifyWebhookSignature(provider, sigToVerify, repo.WebhookSecret, body) {
		response.Error(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "Webhook signature verification failed", nil, "")
		return
	}

	// Parse branch from payload
	var payload struct {
		Ref string `json:"ref"`
	}
	_ = json.Unmarshal(body, &payload)

	expectedRef := fmt.Sprintf("refs/heads/%s", repo.Branch)
	if payload.Ref != "" && payload.Ref != expectedRef && payload.Ref != repo.Branch {
		response.JSON(w, http.StatusOK, map[string]string{"message": "Push to non-target branch ignored"}, nil)
		return
	}

	// Run deployment
	deployCfg := &gitcicd.DeployConfig{
		WebsiteID:    site.ID.String(),
		RepoURL:      repo.RepoURL,
		Branch:       repo.Branch,
		DeployPath:   repo.DeployPath,
		DeployKey:    repo.DeployKeyPriv,
		BuildCommand: repo.BuildCommand,
		KeepReleases: repo.ReleaseKeepCount,
		SystemUser:   isolation.DeriveUsername(site.PrimaryDomain),
	}

	res, depErr := h.gitMgr.DeployRelease(r.Context(), deployCfg)
	status := "success"
	logs := ""
	commit := "HEAD"
	author := "Webhook"
	duration := 0
	releasePath := ""

	if res != nil {
		status = res.Status
		logs = res.BuildLogs
		commit = res.CommitHash
		author = res.Author
		duration = res.DurationSeconds
		releasePath = res.ReleasePath
	} else if depErr != nil {
		status = "failed"
		logs = depErr.Error()
	}

	depRecord := &store.GitDeploymentRecord{
		ID:              uuid.New(),
		RepositoryID:    repo.ID,
		WebsiteID:       site.ID,
		CommitHash:      commit,
		CommitMessage:   "Auto-deploy triggered by webhook",
		Author:          author,
		Status:          status,
		BuildLogs:       logs,
		ReleasePath:     releasePath,
		DurationSeconds: duration,
		CreatedAt:       time.Now().UTC(),
	}
	_ = h.store.CreateGitDeployment(r.Context(), depRecord)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"deployment_id": depRecord.ID,
		"status":        status,
	}, nil)
}

// ============================================================================
// 3. WORDPRESS 1-CLICK STAGING & CLONING ENGINE
// ============================================================================

type CreateStagingApiRequest struct {
	StagingSubdomain string `json:"staging_subdomain"` // e.g. "staging" -> staging.domain.com
	AuthUser         string `json:"auth_user"`
	AuthPassword     string `json:"auth_password"`
}

// GetStaging retrieves the active staging environment for a WordPress site
func (h *DeveloperHandler) GetStaging(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	stg, err := h.store.GetWordPressStaging(r.Context(), site.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.JSON(w, http.StatusOK, nil, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	// Mask auth password
	cleanStg := *stg
	cleanStg.AuthPassword = "••••••••"
	response.JSON(w, http.StatusOK, cleanStg, nil)
}

// CreateStaging provisions an isolated staging environment with serialization-aware DB replacement
func (h *DeveloperHandler) CreateStaging(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req CreateStagingApiRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sub := req.StagingSubdomain
	if sub == "" {
		sub = "staging"
	}
	stagingDomain := fmt.Sprintf("%s.%s", sub, site.PrimaryDomain)
	stagingDocRoot := filepath.Join(filepath.Dir(site.DocumentRoot), fmt.Sprintf("staging_%s", sub))

	authUser := req.AuthUser
	if authUser == "" {
		authUser = "staging_admin"
	}
	authPass := req.AuthPassword
	if authPass == "" {
		authPass = uuid.New().String()[:12]
	}

	liveDBName := fmt.Sprintf("wp_%s", strings.ReplaceAll(site.PrimaryDomain, ".", "_"))
	stagingDBName := fmt.Sprintf("wp_stg_%s", strings.ReplaceAll(site.PrimaryDomain, ".", "_"))

	stgReq := wordpress.CreateStagingRequest{
		LiveDocRoot:    site.DocumentRoot,
		LiveDomain:     site.PrimaryDomain,
		StagingDocRoot: stagingDocRoot,
		StagingDomain:  stagingDomain,
		LiveDBName:     liveDBName,
		StagingDBName:  stagingDBName,
		AuthUser:       authUser,
		AuthPassword:   authPass,
	}

	res, err := h.wpMgr.CreateStaging(r.Context(), stgReq)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STAGING_CREATION_FAILED", err.Error(), nil, "")
		return
	}

	existing, _ := h.store.GetWordPressStaging(r.Context(), site.ID)
	stgID := uuid.New()
	if existing != nil {
		stgID = existing.ID
	}

	stgRecord := &store.WordPressStagingRecord{
		ID:             stgID,
		WebsiteID:      site.ID,
		OrganizationID: site.OrganizationID,
		StagingDomain:  stagingDomain,
		StagingDocRoot: stagingDocRoot,
		StagingDBName:  stagingDBName,
		StagingDBUser:  "u_" + sub,
		AuthEnabled:    true,
		AuthUser:       authUser,
		AuthPassword:   authPass,
		Status:         "active",
		UpdatedAt:      time.Now().UTC(),
	}

	if err := h.store.SaveWordPressStaging(r.Context(), stgRecord); err != nil {
		response.Error(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "wordpress.staging.create", "website", site.ID.String(), "success", "", map[string]interface{}{
			"staging_domain": stagingDomain,
		})
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"staging_domain": stagingDomain,
		"staging_root":   stagingDocRoot,
		"auth_user":      authUser,
		"auth_password":  authPass,
		"message":        res.Message,
	}, nil)
}

// PushToLive promotes staging changes to production with pre-flight safety backup
func (h *DeveloperHandler) PushToLive(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	stg, err := h.store.GetWordPressStaging(r.Context(), site.ID)
	if err != nil || stg.Status != "active" {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Active staging environment not found", nil, "")
		return
	}

	pushReq := wordpress.PushToLiveRequest{
		LiveDocRoot:    site.DocumentRoot,
		LiveDomain:     site.PrimaryDomain,
		StagingDocRoot: stg.StagingDocRoot,
		StagingDomain:  stg.StagingDomain,
		LiveDBName:     stg.StagingDBName,
		StagingDBName:  stg.StagingDBName,
	}

	res, err := h.wpMgr.PushToLive(r.Context(), pushReq)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PUSH_FAILED", err.Error(), nil, "")
		return
	}

	now := time.Now().UTC()
	stg.LastSyncAt = &now
	_ = h.store.SaveWordPressStaging(r.Context(), stg)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "wordpress.staging.push_to_live", "website", site.ID.String(), "success", "", map[string]interface{}{
			"backup_path": res.BackupPath,
		})
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"backup_path":  res.BackupPath,
		"completed_at": res.CompletedAt,
		"message":      res.Message,
	}, nil)
}

// DeleteStaging tears down the staging environment
func (h *DeveloperHandler) DeleteStaging(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	if err := h.store.DeleteWordPressStaging(r.Context(), site.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "wordpress.staging.delete", "website", site.ID.String(), "success", "", nil)
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}
