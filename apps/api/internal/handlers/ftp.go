package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/ftp"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type FTPHandler struct {
	cfg      *config.Config
	store    store.Store
	audit    *audit.Logger
	ftpMgr   *ftp.FTPManager
	quotaSvc *quota.Service
}

func NewFTPHandler(cfg *config.Config, s store.Store, a *audit.Logger) *FTPHandler {
	return &FTPHandler{
		cfg:    cfg,
		store:  s,
		audit:  a,
		ftpMgr: ftp.NewFTPManager(),
	}
}

func (h *FTPHandler) SetQuotaService(q *quota.Service) {
	h.quotaSvc = q
}

// Request DTOs
type CreateFTPUserRequest struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	HomeDir           string `json:"home_dir"`
	QuotaMB           int    `json:"quota_mb"`
	UploadBandwidth   int    `json:"upload_bandwidth_kbps"`
	DownloadBandwidth int    `json:"download_bandwidth_kbps"`
}

type UpdateFTPPasswordRequest struct {
	Password string `json:"password"`
}

type UpdateFTPUserRequest struct {
	HomeDir           string `json:"home_dir"`
	QuotaMB           int    `json:"quota_mb"`
	UploadBandwidth   int    `json:"upload_bandwidth_kbps"`
	DownloadBandwidth int    `json:"download_bandwidth_kbps"`
}

// ----------------------------------------------------------------------------
// TENANT ISOLATION HELPERS
// ----------------------------------------------------------------------------

// getTenantAllowedPaths returns the set of filesystem paths owned by the authenticated tenant.
func (h *FTPHandler) getTenantAllowedPaths(r *http.Request) []string {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil || claims.IsSuperAdmin || claims.Role == "admin" || claims.Role == "owner" {
		return nil // nil = unrestricted (superadmin)
	}

	var paths []string
	websites, _ := h.store.ListWebsitesByOrg(r.Context(), claims.OrganizationID)
	for _, w := range websites {
		if w != nil && w.DocumentRoot != "" {
			paths = append(paths, w.DocumentRoot)
		}
		if w != nil && w.PrimaryDomain != "" {
			paths = append(paths, "/var/www/"+w.PrimaryDomain)
		}
	}
	return paths
}

// isTenantPath checks if a path is within any of the tenant's allowed directories.
func (h *FTPHandler) isTenantPath(homeDir string, allowedPaths []string) bool {
	if allowedPaths == nil {
		return true // superadmin has no restrictions
	}
	clean := filepath.Clean(strings.TrimSpace(homeDir))
	if strings.Contains(clean, "/../") || strings.HasPrefix(clean, "../") || clean == ".." {
		return false
	}
	for _, p := range allowedPaths {
		cleanP := filepath.Clean(strings.TrimSpace(p))
		if clean == cleanP || strings.HasPrefix(clean, cleanP+"/") {
			return true
		}
	}
	return false
}

// verifyFTPUserOwnership checks whether the FTP user's home directory belongs to the caller's tenant.
func (h *FTPHandler) verifyFTPUserOwnership(r *http.Request, username string) error {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil || claims.IsSuperAdmin || claims.Role == "admin" || claims.Role == "owner" {
		return nil
	}

	user, err := h.ftpMgr.GetUser(username)
	if err != nil {
		return errors.New("FTP user not found")
	}

	allowedPaths := h.getTenantAllowedPaths(r)
	if !h.isTenantPath(user.HomeDir, allowedPaths) {
		return errors.New("FTP user does not belong to your organization")
	}
	return nil
}

// GetStatus checks daemon operational state
func (h *FTPHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.ftpMgr.GetStatus()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FTP_STATUS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, status, nil)
}

// ListUsers retrieves FTP virtual accounts scoped to the tenant's websites
func (h *FTPHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.ftpMgr.ListUsers()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FTP_LIST_ERROR", err.Error(), nil, "")
		return
	}

	// Filter to only show FTP users within the tenant's allowed paths
	allowedPaths := h.getTenantAllowedPaths(r)
	if allowedPaths != nil {
		var filtered []ftp.FTPUser
		for _, u := range users {
			if h.isTenantPath(u.HomeDir, allowedPaths) {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"count": len(users),
	}, nil)
}

// CreateUser registers a new FTP virtual user with tenant path validation
func (h *FTPHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateFTPUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	if req.Username == "" || req.Password == "" || req.HomeDir == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "Username, password, and home directory are required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && h.quotaSvc != nil {
		unlock := h.quotaSvc.LockUser(claims.UserID)
		defer unlock()
		if err := h.quotaSvc.CheckQuota(r.Context(), claims.UserID, "ftp"); err != nil {
			response.Error(w, http.StatusConflict, "QUOTA_EXCEEDED", err.Error(), nil, "")
			return
		}
	}

	cleanHomeDir := filepath.Clean(strings.TrimSpace(req.HomeDir))

	// Validate that the home directory is within the tenant's allowed paths
	allowedPaths := h.getTenantAllowedPaths(r)
	if !h.isTenantPath(cleanHomeDir, allowedPaths) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Home directory does not belong to your organization's websites", nil, "")
		return
	}

	user, err := h.ftpMgr.CreateUser(
		req.Username,
		req.Password,
		cleanHomeDir,
		req.QuotaMB,
		req.UploadBandwidth,
		req.DownloadBandwidth,
	)
	if err != nil {
		if errors.Is(err, ftp.ErrUserAlreadyExists) {
			response.Error(w, http.StatusConflict, "USER_EXISTS", err.Error(), nil, "")
			return
		}
		if errors.Is(err, ftp.ErrInvalidFTPUsername) || errors.Is(err, ftp.ErrInvalidDirectory) {
			response.Error(w, http.StatusBadRequest, "INVALID_PARAMS", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ftp.user_create", "ftp", user.Username, "success", "", map[string]interface{}{
		"home_dir": user.HomeDir,
		"quota_mb": user.QuotaMB,
	})

	response.JSON(w, http.StatusCreated, user, nil)
}

// ChangePassword updates user password with tenant ownership check
func (h *FTPHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required", nil, "")
		return
	}

	if err := h.verifyFTPUserOwnership(r, username); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	var req UpdateFTPPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	if len(req.Password) < 6 {
		response.Error(w, http.StatusBadRequest, "INVALID_PASSWORD", "Password must be at least 6 characters", nil, "")
		return
	}

	err := h.ftpMgr.ChangePassword(username, req.Password)
	if err != nil {
		if errors.Is(err, ftp.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "FTP user not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "PASSWORD_UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ftp.password_change", "ftp", username, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "FTP password updated successfully",
	}, nil)
}

// UpdateUser modifies home dir, quota, and bandwidth with tenant ownership check
func (h *FTPHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required", nil, "")
		return
	}

	if err := h.verifyFTPUserOwnership(r, username); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	var req UpdateFTPUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	// If updating home dir, ensure it's within tenant paths
	if req.HomeDir != "" {
		allowedPaths := h.getTenantAllowedPaths(r)
		if !h.isTenantPath(req.HomeDir, allowedPaths) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "New home directory does not belong to your organization's websites", nil, "")
			return
		}
	}

	err := h.ftpMgr.UpdateUser(
		username,
		req.HomeDir,
		req.QuotaMB,
		req.UploadBandwidth,
		req.DownloadBandwidth,
	)
	if err != nil {
		if errors.Is(err, ftp.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "FTP user not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ftp.user_update", "ftp", username, "success", "", map[string]interface{}{
		"home_dir": req.HomeDir,
		"quota_mb": req.QuotaMB,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "FTP user configuration updated successfully",
	}, nil)
}

// DeleteUser removes an FTP user with tenant ownership check
func (h *FTPHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required", nil, "")
		return
	}

	if err := h.verifyFTPUserOwnership(r, username); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	err := h.ftpMgr.DeleteUser(username)
	if err != nil {
		if errors.Is(err, ftp.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "FTP user not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ftp.user_delete", "ftp", username, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "FTP user removed successfully",
	}, nil)
}

// ToggleUser enables or disables an FTP user with tenant ownership check
func (h *FTPHandler) ToggleUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required", nil, "")
		return
	}

	if err := h.verifyFTPUserOwnership(r, username); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	target, err := h.ftpMgr.ToggleUser(username)
	if err != nil {
		if errors.Is(err, ftp.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "FTP user not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "TOGGLE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ftp.user_toggle", "ftp", username, "success", "", map[string]interface{}{
		"is_enabled": target.IsEnabled,
	})

	response.JSON(w, http.StatusOK, target, nil)
}

