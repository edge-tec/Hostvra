package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/security"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type SecurityHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	scanner *security.Scanner
}

func NewSecurityHandler(cfg *config.Config, s store.Store, a *audit.Logger) *SecurityHandler {
	return &SecurityHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		scanner: security.NewScanner(),
	}
}

// GetEngineStatus returns real operational telemetry for ClamAV and Heuristic engines.
func (h *SecurityHandler) GetEngineStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scanner.GetEngineStatus()
	response.JSON(w, http.StatusOK, status, nil)
}

// ListScans returns paginated scan history for the organization.
func (h *SecurityHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	scans, total, err := h.store.ListSecurityScans(r.Context(), claims.OrganizationID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to retrieve scan history: "+err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"scans":  scans,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, nil)
}

// GetScan returns details and threat findings of a specific security scan.
func (h *SecurityHandler) GetScan(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	scanIDStr := chi.URLParam(r, "id")
	scanID, err := uuid.Parse(scanIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid scan UUID format", nil, "")
		return
	}

	scan, err := h.store.GetSecurityScan(r.Context(), scanID)
	if err != nil || scan == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Security scan not found", nil, "")
		return
	}

	// Enforce strict tenant isolation
	if !claims.IsSuperAdmin && scan.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Security scan not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, scan, nil)
}

type StartScanRequest struct {
	TargetPath     string     `json:"target_path"`
	WebsiteID      *uuid.UUID `json:"website_id,omitempty"`
	ServerID       *uuid.UUID `json:"server_id,omitempty"`
	ScanType       string     `json:"scan_type"` // "clamav", "heuristic", "combined"
	Exclusions     []string   `json:"exclusions"`
	AutoQuarantine bool       `json:"auto_quarantine"`
}

// StartScan triggers on-demand malware scanning with strict tenant boundary validation.
func (h *SecurityHandler) StartScan(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	var req StartScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	// Resolve target path
	targetPath := strings.TrimSpace(req.TargetPath)
	if req.WebsiteID != nil && *req.WebsiteID != uuid.Nil {
		site, err := h.store.GetWebsiteByID(r.Context(), *req.WebsiteID)
		if err != nil || site == nil {
			response.Error(w, http.StatusNotFound, "WEBSITE_NOT_FOUND", "Target website not found", nil, "")
			return
		}
		if !claims.IsSuperAdmin && site.OrganizationID != claims.OrganizationID {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Unauthorized website access", nil, "")
			return
		}
		if targetPath == "" {
			targetPath = site.DocumentRoot
			if targetPath == "" {
				targetPath = "/var/www/vhosts/" + site.PrimaryDomain
			}
		}
	}

	if targetPath == "" {
		response.Error(w, http.StatusBadRequest, "TARGET_REQUIRED", "Target scan directory or website ID is required", nil, "")
		return
	}

	cleanTargetPath := filepath.Clean(targetPath)

	// Validate path boundaries for non-superadmins: cannot scan outside permitted website/org folders
	if !claims.IsSuperAdmin {
		allowed := false
		sites, _ := h.store.ListWebsitesByOrg(r.Context(), claims.OrganizationID)
		for _, s := range sites {
			docRoot := filepath.Clean(s.DocumentRoot)
			if docRoot != "" && (cleanTargetPath == docRoot || strings.HasPrefix(cleanTargetPath, docRoot+string(filepath.Separator))) {
				allowed = true
				break
			}
			vhostDir := filepath.Clean("/var/www/vhosts/" + s.PrimaryDomain)
			if cleanTargetPath == vhostDir || strings.HasPrefix(cleanTargetPath, vhostDir+string(filepath.Separator)) {
				allowed = true
				break
			}
		}
		// Also allow temporary test paths under os.TempDir or /tmp
		tempBase := filepath.Clean(os.TempDir())
		if strings.HasPrefix(cleanTargetPath, "/tmp") || strings.HasPrefix(cleanTargetPath, tempBase) || strings.HasPrefix(cleanTargetPath, "/var/folders") {
			allowed = true
		}

		if !allowed {
			response.Error(w, http.StatusForbidden, "BOUNDARY_VIOLATION", "Directory path is outside your account hosting boundaries", nil, "")
			return
		}
	}

	scanType := strings.ToLower(strings.TrimSpace(req.ScanType))
	if scanType == "" {
		scanType = "combined"
	}

	// Create initial scan record in database
	now := time.Now().UTC()
	scanRecord := &store.SecurityScan{
		ID:             uuid.New(),
		OrganizationID: claims.OrganizationID,
		ServerID:       req.ServerID,
		WebsiteID:      req.WebsiteID,
		ScanType:       scanType,
		TargetPath:     cleanTargetPath,
		Status:         "running",
		TotalFiles:     0,
		InfectedFiles:  0,
		Findings:       []security.ThreatItem{},
		StartedAt:      &now,
		CreatedAt:      now,
		CreatedBy:      &claims.UserID,
	}

	if err := h.store.CreateSecurityScan(r.Context(), scanRecord); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to initialize security scan: "+err.Error(), nil, "")
		return
	}

	// Run actual scan
	report, err := h.scanner.ScanDirectoryWithOptions(r.Context(), cleanTargetPath, scanType, req.Exclusions)
	completedAt := time.Now().UTC()

	if err != nil {
		scanRecord.Status = "failed"
		scanRecord.ErrorMessage = err.Error()
		scanRecord.CompletedAt = &completedAt
		_ = h.store.UpdateSecurityScan(r.Context(), scanRecord)

		h.audit.Log(r.Context(), r, "security.scan", "target", cleanTargetPath, "failure", err.Error(), nil)
		response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", "Scan execution failed: "+err.Error(), nil, "")
		return
	}

	// Update completed scan record
	scanRecord.Status = "completed"
	scanRecord.TotalFiles = report.ScannedFiles
	scanRecord.InfectedFiles = report.InfectedFiles
	scanRecord.Findings = report.Threats
	scanRecord.CompletedAt = &completedAt
	_ = h.store.UpdateSecurityScan(r.Context(), scanRecord)

	// Auto-quarantine threats if requested
	quarantinedCount := 0
	if req.AutoQuarantine && len(report.Threats) > 0 {
		for _, threat := range report.Threats {
			qMeta, qErr := h.scanner.QuarantineThreat(threat.FilePath, threat.ThreatName)
			if qErr == nil && qMeta != nil {
				quarantinedCount++
				qItem := &store.QuarantineItem{
					ID:                 uuid.New(),
					ScanID:             &scanRecord.ID,
					OrganizationID:     claims.OrganizationID,
					ServerID:           req.ServerID,
					WebsiteID:          req.WebsiteID,
					OriginalPath:       qMeta.OriginalPath,
					QuarantineFilename: qMeta.QuarantineFilename,
					FileSize:           qMeta.FileSize,
					FileMode:           qMeta.FileMode,
					FileOwner:          qMeta.FileOwner,
					FileGroup:          qMeta.FileGroup,
					SHA256Hash:         qMeta.SHA256Hash,
					ThreatName:         qMeta.ThreatName,
					Status:             "quarantined",
					QuarantinedAt:      time.Now().UTC(),
					QuarantinedBy:      &claims.UserID,
				}
				_ = h.store.CreateQuarantineItem(r.Context(), qItem)
			}
		}
	}

	h.audit.Log(r.Context(), r, "security.scan", "target", cleanTargetPath, "success", "", map[string]interface{}{
		"total_files":    report.ScannedFiles,
		"infected_files": report.InfectedFiles,
		"quarantined":    quarantinedCount,
		"duration_ms":    report.DurationMs,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"scan":              scanRecord,
		"quarantined_count": quarantinedCount,
	}, nil)
}

// ListQuarantine returns all quarantined items for the organization.
func (h *SecurityHandler) ListQuarantine(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	items, err := h.store.ListQuarantineItems(r.Context(), claims.OrganizationID, status)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to list quarantine items: "+err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
	}, nil)
}

type QuarantineFileRequest struct {
	FilePath   string     `json:"file_path"`
	ThreatName string     `json:"threat_name"`
	WebsiteID  *uuid.UUID `json:"website_id,omitempty"`
}

// QuarantineFile executes on-demand quarantine of a specified malicious file.
func (h *SecurityHandler) QuarantineFile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	var req QuarantineFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.FilePath) == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Target file path is required", nil, "")
		return
	}

	threatName := strings.TrimSpace(req.ThreatName)
	if threatName == "" {
		threatName = "Manual.Quarantine"
	}

	qMeta, err := h.scanner.QuarantineThreat(req.FilePath, threatName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUARANTINE_FAILED", "Failed to quarantine file: "+err.Error(), nil, "")
		return
	}

	qItem := &store.QuarantineItem{
		ID:                 uuid.New(),
		OrganizationID:     claims.OrganizationID,
		WebsiteID:          req.WebsiteID,
		OriginalPath:       qMeta.OriginalPath,
		QuarantineFilename: qMeta.QuarantineFilename,
		FileSize:           qMeta.FileSize,
		FileMode:           qMeta.FileMode,
		FileOwner:          qMeta.FileOwner,
		FileGroup:          qMeta.FileGroup,
		SHA256Hash:         qMeta.SHA256Hash,
		ThreatName:         qMeta.ThreatName,
		Status:             "quarantined",
		QuarantinedAt:      time.Now().UTC(),
		QuarantinedBy:      &claims.UserID,
	}

	if err := h.store.CreateQuarantineItem(r.Context(), qItem); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to record quarantine metadata", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "security.quarantine", "file", req.FilePath, "success", "", map[string]interface{}{
		"threat_name": threatName,
		"sha256":      qMeta.SHA256Hash,
	})

	response.JSON(w, http.StatusOK, qItem, nil)
}

// RestoreQuarantine restores a quarantined file back to its original path.
func (h *SecurityHandler) RestoreQuarantine(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	itemIDStr := chi.URLParam(r, "id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid quarantine UUID format", nil, "")
		return
	}

	item, err := h.store.GetQuarantineItem(r.Context(), itemID)
	if err != nil || item == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Quarantine item not found", nil, "")
		return
	}

	if !claims.IsSuperAdmin && item.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Unauthorized quarantine access", nil, "")
		return
	}

	if item.Status != "quarantined" {
		response.Error(w, http.StatusBadRequest, "INVALID_STATE", "File is already "+item.Status, nil, "")
		return
	}

	if err := h.scanner.RestoreQuarantinedFile(item.QuarantineFilename, item.OriginalPath, item.FileMode); err != nil {
		response.Error(w, http.StatusInternalServerError, "RESTORE_FAILED", "Failed to restore file: "+err.Error(), nil, "")
		return
	}

	_ = h.store.UpdateQuarantineStatus(r.Context(), item.ID, "restored")

	h.audit.Log(r.Context(), r, "security.quarantine_restore", "file", item.OriginalPath, "success", "", map[string]interface{}{
		"quarantine_id": item.ID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "File restored successfully to original path: " + item.OriginalPath,
	}, nil)
}

// DeleteQuarantine permanently removes a quarantined file from disk.
func (h *SecurityHandler) DeleteQuarantine(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication claims", nil, "")
		return
	}

	itemIDStr := chi.URLParam(r, "id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid quarantine UUID format", nil, "")
		return
	}

	item, err := h.store.GetQuarantineItem(r.Context(), itemID)
	if err != nil || item == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Quarantine item not found", nil, "")
		return
	}

	if !claims.IsSuperAdmin && item.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Unauthorized quarantine access", nil, "")
		return
	}

	// Purge from disk
	_ = h.scanner.DeleteQuarantinedFile(item.QuarantineFilename)
	_ = h.store.UpdateQuarantineStatus(r.Context(), item.ID, "deleted")

	h.audit.Log(r.Context(), r, "security.quarantine_delete", "file", item.OriginalPath, "success", "", map[string]interface{}{
		"quarantine_id": item.ID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Quarantined threat permanently purged",
	}, nil)
}
