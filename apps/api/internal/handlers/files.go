package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hostvra/agent/pkg/files"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type FileHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	fileMgr *files.FileManager
}

func NewFileHandler(cfg *config.Config, s store.Store, a *audit.Logger) *FileHandler {
	fm := files.NewFileManager()
	// Allow root access for administrative operations in control panel
	fm.AllowRootSystemWide()

	return &FileHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		fileMgr: fm,
	}
}

// checkPathAuthorization enforces multi-tenant and role-based boundaries on file operations.
// Non-admin roles (manager, developer) are strictly confined to /var/www, /home, or /tmp and
// forbidden from reading, writing, or traversing sensitive system files (/etc, /root, /boot, etc.)
func (h *FileHandler) checkPathAuthorization(r *http.Request, targetPath string) error {
	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && claims.Role != "" && claims.Role != "owner" && claims.Role != "admin" {
		clean := filepath.Clean(targetPath)
		restrictedRoots := []string{
			"/etc", "/root", "/boot", "/proc", "/sys", "/dev", "/run", "/var/run",
			"/var/lib/hostvra", "/var/lib/docker", "/usr", "/bin", "/sbin", "/lib", "/lib64",
		}
		if clean == "/" {
			return fmt.Errorf("root directory access forbidden for role '%s'", claims.Role)
		}
		for _, rr := range restrictedRoots {
			if clean == rr || strings.HasPrefix(clean, rr+"/") {
				return fmt.Errorf("system path '%s' is restricted for role '%s'", clean, claims.Role)
			}
		}
		// Confinement to application and user spaces
		if !strings.HasPrefix(clean, "/var/www") && !strings.HasPrefix(clean, "/home") && !strings.HasPrefix(clean, "/tmp") {
			return fmt.Errorf("role '%s' is confined to web and home directories", claims.Role)
		}
	}
	return nil
}

// Request DTOs
type FileContentRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type MkdirRequest struct {
	Path string `json:"path"`
}

type RenameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type CopyRequest struct {
	SrcPath  string `json:"src_path"`
	DestPath string `json:"dest_path"`
}

type DeleteRequest struct {
	Path string `json:"path"`
}

type ChmodRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode"` // e.g. "0755", "0644"
	UID  int    `json:"uid,omitempty"`
	GID  int    `json:"gid,omitempty"`
}

type ArchiveRequest struct {
	Paths    []string `json:"paths"`
	DestPath string   `json:"dest_path"`
	Format   string   `json:"format"` // "zip" or "tar.gz"
}

type ExtractRequest struct {
	ArchivePath string `json:"archive_path"`
	DestDir     string `json:"dest_dir"`
}

// ----------------------------------------------------------------------------
// HANDLERS
// ----------------------------------------------------------------------------

// List returns contents of a directory
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/var/www"
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			dirPath = "/"
		}
	}

	if err := h.checkPathAuthorization(r, dirPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	items, err := h.fileMgr.List(dirPath)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "FILE_LIST_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"current_path": filepath.Clean(dirPath),
		"items":        items,
		"count":        len(items),
	}, nil)
}

// Stat returns metadata for a file or directory
func (h *FileHandler) Stat(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path query parameter required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, targetPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	item, err := h.fileMgr.Stat(targetPath)
	if err != nil {
		response.Error(w, http.StatusNotFound, "FILE_NOT_FOUND", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, item, nil)
}

// GetContent reads text content of a file
func (h *FileHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path query parameter required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, targetPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	data, err := h.fileMgr.ReadFile(targetPath, 10*1024*1024) // 10MB max
	if err != nil {
		response.Error(w, http.StatusBadRequest, "READ_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"path":    targetPath,
		"content": string(data),
		"size":    len(data),
	}, nil)
}

// SaveContent writes text content atomically, creating a .bak copy if modifying existing
func (h *FileHandler) SaveContent(w http.ResponseWriter, r *http.Request) {
	var req FileContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.Path == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, req.Path); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	validatedPath, err := h.fileMgr.ValidatePath(req.Path)
	if err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	// Refuse overwriting through pre-existing symlinks
	if fi, err := os.Lstat(validatedPath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		response.Error(w, http.StatusForbidden, "SYMLINK_OVERWRITE_FORBIDDEN", "Destination is an existing symlink", nil, "")
		return
	}

	if err := h.fileMgr.WriteFile(validatedPath, []byte(req.Content)); err != nil {
		response.Error(w, http.StatusInternalServerError, "WRITE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.save", "file", req.Path, "success", "", map[string]interface{}{
		"size": len(req.Content),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"saved": true,
		"path":  req.Path,
		"size":  len(req.Content),
	}, nil)
}

// Mkdir creates a directory
func (h *FileHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req MkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.Path == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, req.Path); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	if err := h.fileMgr.CreateDirectory(req.Path); err != nil {
		response.Error(w, http.StatusBadRequest, "MKDIR_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.mkdir", "directory", req.Path, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"created": true,
		"path":    req.Path,
	}, nil)
}

// Upload handles multipart file uploads
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// 500MB max memory/temp file parser
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, "UPLOAD_PARSE_ERROR", "Failed to parse multipart upload", nil, "")
		return
	}

	targetDir := r.FormValue("path")
	if targetDir == "" {
		targetDir = "/var/www"
	}

	if err := h.checkPathAuthorization(r, targetDir); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "MISSING_FILE", "No file uploaded", nil, "")
		return
	}
	defer file.Close()

	// Normalize Windows slashes and prevent directory breakout
	normalizedFilename := strings.ReplaceAll(header.Filename, "\\", "/")
	safeFilename := filepath.Base(filepath.Clean(normalizedFilename))
	if safeFilename == "." || safeFilename == "/" || safeFilename == "" || strings.Contains(safeFilename, "\x00") || strings.Contains(safeFilename, "..") {
		response.Error(w, http.StatusBadRequest, "INVALID_FILENAME", "Invalid upload filename", nil, "")
		return
	}

	destPath := filepath.Join(targetDir, safeFilename)
	if err := h.checkPathAuthorization(r, destPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	validatedDest, err := h.fileMgr.ValidatePath(destPath)
	if err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	// Refuse overwriting through pre-existing symlinks
	if fi, err := os.Lstat(validatedDest); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		response.Error(w, http.StatusForbidden, "SYMLINK_OVERWRITE_FORBIDDEN", "Destination is an existing symlink", nil, "")
		return
	}

	// Atomic upload: stream to a private temporary file in the same directory, then rename
	tmpDest := fmt.Sprintf("%s.tmp.%d", validatedDest, time.Now().UnixNano())
	out, err := os.OpenFile(tmpDest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FILE_CREATE_ERROR", err.Error(), nil, "")
		return
	}

	written, err := io.Copy(out, file)
	out.Close()
	if err != nil {
		_ = os.Remove(tmpDest)
		response.Error(w, http.StatusInternalServerError, "STREAM_ERROR", err.Error(), nil, "")
		return
	}

	// Re-verify destination before atomic rename to prevent TOCTOU race
	if fi, err := os.Lstat(validatedDest); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(tmpDest)
		response.Error(w, http.StatusForbidden, "SYMLINK_OVERWRITE_FORBIDDEN", "Destination is an existing symlink", nil, "")
		return
	}

	if err := os.Rename(tmpDest, validatedDest); err != nil {
		_ = os.Remove(tmpDest)
		response.Error(w, http.StatusInternalServerError, "FILE_FINALIZE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.upload", "file", validatedDest, "success", "", map[string]interface{}{
		"filename": header.Filename,
		"bytes":    written,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"uploaded": true,
		"filename": header.Filename,
		"path":     validatedDest,
		"bytes":    written,
	}, nil)
}

// Rename renames or moves a file
func (h *FileHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var req RenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.OldPath == "" || req.NewPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATHS", "Both old_path and new_path are required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, req.OldPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}
	if err := h.checkPathAuthorization(r, req.NewPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	if err := h.fileMgr.Rename(req.OldPath, req.NewPath); err != nil {
		response.Error(w, http.StatusBadRequest, "RENAME_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.rename", "file", req.OldPath, "success", "", map[string]interface{}{
		"new_path": req.NewPath,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"renamed":  true,
		"old_path": req.OldPath,
		"new_path": req.NewPath,
	}, nil)
}

// Copy copies a file or directory
func (h *FileHandler) Copy(w http.ResponseWriter, r *http.Request) {
	var req CopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.SrcPath == "" || req.DestPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATHS", "Both src_path and dest_path are required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, req.SrcPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}
	if err := h.checkPathAuthorization(r, req.DestPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	if err := h.fileMgr.Copy(req.SrcPath, req.DestPath); err != nil {
		response.Error(w, http.StatusBadRequest, "COPY_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.copy", "file", req.SrcPath, "success", "", map[string]interface{}{
		"dest_path": req.DestPath,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"copied":    true,
		"src_path":  req.SrcPath,
		"dest_path": req.DestPath,
	}, nil)
}

// Delete removes a file or directory
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		var req DeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			targetPath = req.Path
		}
	}

	if targetPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, targetPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	if err := h.fileMgr.Delete(targetPath); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.delete", "file", targetPath, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"path":    targetPath,
	}, nil)
}

// Permissions changes chmod and optionally chown
func (h *FileHandler) Permissions(w http.ResponseWriter, r *http.Request) {
	var req ChmodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.Path == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, req.Path); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	// Parse octal mode e.g. "0755"
	if req.Mode != "" {
		cleanMode := strings.TrimPrefix(req.Mode, "0")
		parsed, err := strconv.ParseUint(cleanMode, 8, 32)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_MODE", "Invalid octal permission mode (e.g. 0755 or 0644)", nil, "")
			return
		}

		if err := h.fileMgr.Chmod(req.Path, os.FileMode(parsed)); err != nil {
			response.Error(w, http.StatusBadRequest, "CHMOD_ERROR", err.Error(), nil, "")
			return
		}
	}

	if req.UID > 0 || req.GID > 0 {
		claims, _ := auth.GetClaims(r.Context())
		if claims != nil && claims.Role != "owner" && claims.Role != "admin" {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", "Only administrators can change file ownership (chown)", nil, "")
			return
		}
		_ = h.fileMgr.Chown(req.Path, req.UID, req.GID)
	}

	h.audit.Log(r.Context(), r, "file.permissions", "file", req.Path, "success", "", map[string]interface{}{
		"mode": req.Mode,
		"uid":  req.UID,
		"gid":  req.GID,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"updated": true,
		"path":    req.Path,
		"mode":    req.Mode,
	}, nil)
}

// Archive compresses files into .zip or .tar.gz
func (h *FileHandler) Archive(w http.ResponseWriter, r *http.Request) {
	var req ArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if len(req.Paths) == 0 || req.DestPath == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAMS", "paths and dest_path are required", nil, "")
		return
	}

	for _, p := range req.Paths {
		if err := h.checkPathAuthorization(r, p); err != nil {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
			return
		}
	}
	if err := h.checkPathAuthorization(r, req.DestPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	if err := h.fileMgr.Archive(req.Paths, req.DestPath, req.Format); err != nil {
		response.Error(w, http.StatusBadRequest, "ARCHIVE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.archive", "archive", req.DestPath, "success", "", map[string]interface{}{
		"format": req.Format,
		"count":  len(req.Paths),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"archived":  true,
		"dest_path": req.DestPath,
	}, nil)
}

// Extract extracts a .zip or .tar.gz archive
func (h *FileHandler) Extract(w http.ResponseWriter, r *http.Request) {
	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.ArchivePath == "" || req.DestDir == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAMS", "archive_path and dest_dir are required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, req.ArchivePath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}
	if err := h.checkPathAuthorization(r, req.DestDir); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	if err := h.fileMgr.Extract(req.ArchivePath, req.DestDir); err != nil {
		response.Error(w, http.StatusBadRequest, "EXTRACT_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.extract", "archive", req.ArchivePath, "success", "", map[string]interface{}{
		"dest_dir": req.DestDir,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"extracted": true,
		"dest_dir":  req.DestDir,
	}, nil)
}

// Download streams file content as HTTP attachment
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	if err := h.checkPathAuthorization(r, targetPath); err != nil {
		http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
		return
	}

	valPath, err := h.fileMgr.ValidatePath(targetPath)
	if err != nil {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	file, err := os.Open(valPath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "cannot download directory directly", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(valPath)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	_, _ = io.Copy(w, file)
}

