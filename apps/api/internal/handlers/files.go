package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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

		// Security: Prevent symlink escape traversal (resolve target if path exists)
		if resolved, err := filepath.EvalSymlinks(clean); err == nil && resolved != clean {
			for _, rr := range restrictedRoots {
				if resolved == rr || strings.HasPrefix(resolved, rr+"/") {
					return fmt.Errorf("symlink target '%s' is restricted for role '%s'", resolved, claims.Role)
				}
			}
			if !strings.HasPrefix(resolved, "/var/www") && !strings.HasPrefix(resolved, "/home") && !strings.HasPrefix(resolved, "/tmp") {
				return fmt.Errorf("symlink target '%s' escapes authorized user spaces", resolved)
			}
		}

		// Check parent directory symlink resolution for creation mode
		parent := filepath.Dir(clean)
		if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil && resolvedParent != parent {
			for _, rr := range restrictedRoots {
				if resolvedParent == rr || strings.HasPrefix(resolvedParent, rr+"/") {
					return fmt.Errorf("parent directory symlink '%s' is restricted for role '%s'", resolvedParent, claims.Role)
				}
			}
			if !strings.HasPrefix(resolvedParent, "/var/www") && !strings.HasPrefix(resolvedParent, "/home") && !strings.HasPrefix(resolvedParent, "/tmp") {
				return fmt.Errorf("parent directory symlink '%s' escapes authorized user spaces", resolvedParent)
			}
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
	OldPath  string   `json:"old_path"`
	OldPaths []string `json:"old_paths"`
	NewPath  string   `json:"new_path"`
}

type CopyRequest struct {
	SrcPath  string   `json:"src_path"`
	SrcPaths []string `json:"src_paths"`
	DestPath string   `json:"dest_path"`
}

type DeleteRequest struct {
	Path      string   `json:"path"`
	Paths     []string `json:"paths"`
	Permanent bool     `json:"permanent"` // explicit permanent delete flag (default false = move to trash)
	Domain    string   `json:"domain"`    // for trash metadata
}

type ChmodRequest struct {
	Path  string   `json:"path"`
	Paths []string `json:"paths"`
	Mode  string   `json:"mode"` // e.g. "0755", "0644"
	UID   int      `json:"uid,omitempty"`
	GID   int      `json:"gid,omitempty"`
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

// Rename renames a file or moves multiple files to a new destination directory
func (h *FileHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var req RenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if len(req.OldPaths) > 0 {
		if req.NewPath == "" {
			response.Error(w, http.StatusBadRequest, "MISSING_DEST", "Destination directory new_path is required", nil, "")
			return
		}
		if err := h.checkPathAuthorization(r, req.NewPath); err != nil {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
			return
		}

		var movedPaths []string
		var errorMessages []string
		for _, oldP := range req.OldPaths {
			oldP = strings.TrimSpace(oldP)
			if oldP == "" {
				continue
			}
			if err := h.checkPathAuthorization(r, oldP); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", oldP, err.Error()))
				continue
			}
			targetDest := filepath.Join(req.NewPath, filepath.Base(oldP))
			if err := h.fileMgr.Rename(oldP, targetDest); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", oldP, err.Error()))
				continue
			}
			h.audit.Log(r.Context(), r, "file.move", "file", oldP, "success", "", map[string]interface{}{
				"new_path": targetDest,
			})
			movedPaths = append(movedPaths, targetDest)
		}

		if len(movedPaths) == 0 && len(errorMessages) > 0 {
			response.Error(w, http.StatusBadRequest, "MOVE_ERROR", strings.Join(errorMessages, "; "), nil, "")
			return
		}

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"moved":  true,
			"paths":  movedPaths,
			"count":  len(movedPaths),
			"errors": errorMessages,
		}, nil)
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

	targetNew := req.NewPath
	if info, err := os.Stat(req.NewPath); err == nil && info.IsDir() {
		targetNew = filepath.Join(req.NewPath, filepath.Base(req.OldPath))
	}

	if err := h.fileMgr.Rename(req.OldPath, targetNew); err != nil {
		response.Error(w, http.StatusBadRequest, "RENAME_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.rename", "file", req.OldPath, "success", "", map[string]interface{}{
		"new_path": targetNew,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"renamed":  true,
		"old_path": req.OldPath,
		"new_path": targetNew,
	}, nil)
}

// Copy copies a file or directory, or multiple files to a destination directory
func (h *FileHandler) Copy(w http.ResponseWriter, r *http.Request) {
	var req CopyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if len(req.SrcPaths) > 0 {
		if req.DestPath == "" {
			response.Error(w, http.StatusBadRequest, "MISSING_DEST", "Destination dest_path is required", nil, "")
			return
		}
		if err := h.checkPathAuthorization(r, req.DestPath); err != nil {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
			return
		}

		var copiedPaths []string
		var errorMessages []string
		for _, srcP := range req.SrcPaths {
			srcP = strings.TrimSpace(srcP)
			if srcP == "" {
				continue
			}
			if err := h.checkPathAuthorization(r, srcP); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", srcP, err.Error()))
				continue
			}
			targetDest := filepath.Join(req.DestPath, filepath.Base(srcP))
			if err := h.fileMgr.Copy(srcP, targetDest); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", srcP, err.Error()))
				continue
			}
			h.audit.Log(r.Context(), r, "file.copy", "file", srcP, "success", "", map[string]interface{}{
				"dest_path": targetDest,
			})
			copiedPaths = append(copiedPaths, targetDest)
		}

		if len(copiedPaths) == 0 && len(errorMessages) > 0 {
			response.Error(w, http.StatusBadRequest, "COPY_ERROR", strings.Join(errorMessages, "; "), nil, "")
			return
		}

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"copied": true,
			"paths":  copiedPaths,
			"count":  len(copiedPaths),
			"errors": errorMessages,
		}, nil)
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

	targetDest := req.DestPath
	if info, err := os.Stat(req.DestPath); err == nil && info.IsDir() {
		targetDest = filepath.Join(req.DestPath, filepath.Base(req.SrcPath))
	}

	if err := h.fileMgr.Copy(req.SrcPath, targetDest); err != nil {
		response.Error(w, http.StatusBadRequest, "COPY_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.copy", "file", req.SrcPath, "success", "", map[string]interface{}{
		"dest_path": targetDest,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"copied":    true,
		"src_path":  req.SrcPath,
		"dest_path": targetDest,
	}, nil)
}

// Delete moves files to Enterprise Trash by default, or permanently deletes if permanent=true.
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Check for permanent flag from query param or JSON body
	permanent := r.URL.Query().Get("permanent") == "true"

	var pathsToDelete []string
	var domain string

	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath != "" {
		pathsToDelete = append(pathsToDelete, targetPath)
	}

	if r.Body != nil {
		var req DeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Permanent {
				permanent = true
			}
			domain = req.Domain
			if len(req.Paths) > 0 {
				for _, p := range req.Paths {
					p = strings.TrimSpace(p)
					if p != "" {
						pathsToDelete = append(pathsToDelete, p)
					}
				}
			} else if strings.TrimSpace(req.Path) != "" {
				pathsToDelete = append(pathsToDelete, strings.TrimSpace(req.Path))
			}
		}
	}

	if len(pathsToDelete) == 0 {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path or paths required", nil, "")
		return
	}

	if permanent {
		// Permanent delete — only allowed from Trash bin view or by admins
		var deletedPaths []string
		var errorMessages []string

		for _, p := range pathsToDelete {
			cleanPath := filepath.Clean(p)
			if cleanPath == "/" || cleanPath == "." {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: Root directory cannot be deleted", p))
				continue
			}

			if err := h.checkPathAuthorization(r, cleanPath); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", p, err.Error()))
				continue
			}

			if err := h.fileMgr.Delete(cleanPath); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", p, err.Error()))
				continue
			}

			_ = h.store.DeleteFileManagerFavorite(r.Context(), h.getUserID(r), cleanPath)
			_ = h.store.DeleteFileManagerRecent(r.Context(), cleanPath)
			_ = h.store.DeleteFolderLabel(r.Context(), cleanPath)

			h.audit.Log(r.Context(), r, "file.permanent_delete", "file", cleanPath, "success", "", nil)
			deletedPaths = append(deletedPaths, cleanPath)
		}

		if len(deletedPaths) == 0 && len(errorMessages) > 0 {
			response.Error(w, http.StatusBadRequest, "DELETE_ERROR", strings.Join(errorMessages, "; "), nil, "")
			return
		}

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"deleted": true,
			"paths":   deletedPaths,
			"count":   len(deletedPaths),
			"errors":  errorMessages,
		}, nil)
		return
	}

	// Default: Move to Enterprise Trash Bin (soft delete)
	trashBase := h.trashDir()
	var trashedItems []*store.FileManagerTrashItem
	var errorMessages []string

	claims, _ := auth.GetClaims(r.Context())
	deletedBy := "Administrator"
	if claims != nil && claims.Email != "" {
		deletedBy = claims.Email
	}

	for _, p := range pathsToDelete {
		clean := filepath.Clean(p)
		if clean == "/" || clean == "." {
			errorMessages = append(errorMessages, clean+": Root cannot be deleted")
			continue
		}

		if err := h.checkPathAuthorization(r, clean); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", clean, err.Error()))
			continue
		}

		info, err := os.Stat(clean)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", clean, err.Error()))
			continue
		}

		trashID := uuid.New()
		trashFilename := fmt.Sprintf("%s_%s", trashID.String(), filepath.Base(clean))
		physicalTrashPath := filepath.Join(trashBase, trashFilename)

		var totalSize int64
		isDir := info.IsDir()
		if isDir {
			sz, _, _, _ := h.fileMgr.CalculateDirSize(clean)
			totalSize = sz
		} else {
			totalSize = info.Size()
		}

		if err := h.fileMgr.Rename(clean, physicalTrashPath); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", clean, err.Error()))
			continue
		}

		trashItem := &store.FileManagerTrashItem{
			ID:           trashID,
			UserID:       h.getUserID(r),
			Domain:       domain,
			OriginalPath: clean,
			TrashPath:    physicalTrashPath,
			Name:         filepath.Base(clean),
			Size:         totalSize,
			FileType:     filepath.Ext(clean),
			IsDir:        isDir,
			DeletedBy:    deletedBy,
			DeletedAt:    time.Now().UTC(),
		}

		if err := h.store.AddFileManagerTrash(r.Context(), trashItem); err != nil {
			log.Printf("[WARN] Failed to record trash metadata in database (path=%s): %v", clean, err)
		}

		_ = h.store.DeleteFileManagerFavorite(r.Context(), h.getUserID(r), clean)
		_ = h.store.DeleteFileManagerRecent(r.Context(), clean)
		_ = h.store.DeleteFolderLabel(r.Context(), clean)

		h.audit.Log(r.Context(), r, "file.trash", "file", clean, "success", "", map[string]interface{}{
			"trash_id": trashID.String(),
			"size":     totalSize,
		})
		_ = h.store.RecordFileManagerActivityLog(r.Context(), &store.FileManagerActivityLog{
			ID:         uuid.New(),
			UserID:     h.getUserID(r),
			UserEmail:  deletedBy,
			Domain:     domain,
			Action:     "trash",
			SourcePath: clean,
		})

		trashedItems = append(trashedItems, trashItem)
	}

	if len(trashedItems) == 0 && len(errorMessages) > 0 {
		response.Error(w, http.StatusBadRequest, "DELETE_ERROR", strings.Join(errorMessages, "; "), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"trashed": true,
		"items":   trashedItems,
		"count":   len(trashedItems),
		"errors":  errorMessages,
	}, nil)
}

// Permissions changes chmod and optionally chown (single or batch)
func (h *FileHandler) Permissions(w http.ResponseWriter, r *http.Request) {
	var req ChmodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	var targetPaths []string
	if len(req.Paths) > 0 {
		targetPaths = req.Paths
	} else if req.Path != "" {
		targetPaths = []string{req.Path}
	}

	if len(targetPaths) == 0 {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path or paths required", nil, "")
		return
	}

	var parsedMode os.FileMode
	hasMode := false
	if req.Mode != "" {
		cleanMode := strings.TrimPrefix(req.Mode, "0")
		parsed, err := strconv.ParseUint(cleanMode, 8, 32)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_MODE", "Invalid octal permission mode (e.g. 0755 or 0644)", nil, "")
			return
		}
		parsedMode = os.FileMode(parsed)
		hasMode = true
	}

	if req.UID > 0 || req.GID > 0 {
		claims, _ := auth.GetClaims(r.Context())
		if claims != nil && claims.Role != "owner" && claims.Role != "admin" {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", "Only administrators can change file ownership (chown)", nil, "")
			return
		}
	}

	var updatedPaths []string
	var errorMessages []string

	for _, p := range targetPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if err := h.checkPathAuthorization(r, p); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", p, err.Error()))
			continue
		}

		if hasMode {
			if err := h.fileMgr.Chmod(p, parsedMode); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", p, err.Error()))
				continue
			}
		}

		if req.UID > 0 || req.GID > 0 {
			_ = h.fileMgr.Chown(p, req.UID, req.GID)
		}

		h.audit.Log(r.Context(), r, "file.permissions", "file", p, "success", "", map[string]interface{}{
			"mode": req.Mode,
			"uid":  req.UID,
			"gid":  req.GID,
		})
		updatedPaths = append(updatedPaths, p)
	}

	if len(updatedPaths) == 0 && len(errorMessages) > 0 {
		response.Error(w, http.StatusBadRequest, "CHMOD_ERROR", strings.Join(errorMessages, "; "), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"updated": true,
		"paths":   updatedPaths,
		"count":   len(updatedPaths),
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

// ----------------------------------------------------------------------------
// ENTERPRISE FILE MANAGER v3.0 REST HANDLERS
// ----------------------------------------------------------------------------

func (h *FileHandler) trashDir() string {
	p := "/var/lib/hostvra/trash"
	if err := os.MkdirAll(p, 0755); err != nil {
		p = filepath.Join(os.TempDir(), "hostvra_trash")
		_ = os.MkdirAll(p, 0755)
	}
	return p
}

func (h *FileHandler) getUserID(r *http.Request) *uuid.UUID {
	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && claims.UserID != uuid.Nil {
		uid := claims.UserID
		return &uid
	}
	return nil
}

type FavoriteRequest struct {
	Path   string `json:"path"`
	Domain string `json:"domain"`
	Name   string `json:"name"`
	Color  string `json:"color"`
}

type FolderLabelRequest struct {
	Path   string `json:"path"`
	Domain string `json:"domain"`
	Color  string `json:"color"`
	Label  string `json:"label"`
}

type RecentFolderRequest struct {
	Path   string `json:"path"`
	Domain string `json:"domain"`
}

type MoveRequest struct {
	SrcPath          string   `json:"src_path"`
	SrcPaths         []string `json:"src_paths"`
	DestPath         string   `json:"dest_path"`
	DestDomain       string   `json:"dest_domain"`
	ConflictStrategy string   `json:"conflict_strategy"` // "replace", "skip", "rename"
}

type TrashMoveRequest struct {
	Path   string   `json:"path"`
	Paths  []string `json:"paths"`
	Domain string   `json:"domain"`
}

type RestoreRequest struct {
	TrashIDs         []string `json:"trash_ids"`
	TrashID          string   `json:"trash_id"`
	RestoreTo        string   `json:"restore_to"` // "original", "custom_folder", "domain"
	CustomPath       string   `json:"custom_path"`
	TargetDomain     string   `json:"target_domain"`
	ConflictStrategy string   `json:"conflict_strategy"` // "replace", "skip", "rename"
}

type EmptyTrashRequest struct {
	Confirmation string `json:"confirmation"` // must be "DELETE"
	Domain       string `json:"domain"`
}

// QuickAccess returns sidebar favorites, recent folders, domains, roots, and trash stats
func (h *FileHandler) QuickAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserID(r)

	favs, err := h.store.ListFileManagerFavorites(ctx, userID)
	if err != nil {
		favs = []*store.FileManagerFavorite{}
	}
	validFavs := make([]*store.FileManagerFavorite, 0, len(favs))
	for _, f := range favs {
		if _, statErr := os.Stat(f.Path); statErr == nil {
			validFavs = append(validFavs, f)
		} else {
			_ = h.store.DeleteFileManagerFavorite(ctx, userID, f.Path)
		}
	}
	favs = validFavs

	recent, err := h.store.ListFileManagerRecent(ctx, userID, 15)
	if err != nil {
		recent = []*store.FileManagerRecent{}
	}
	validRecent := make([]*store.FileManagerRecent, 0, len(recent))
	for _, r := range recent {
		if info, statErr := os.Stat(r.Path); statErr == nil && info.IsDir() {
			validRecent = append(validRecent, r)
		} else {
			_ = h.store.DeleteFileManagerRecent(ctx, r.Path)
		}
	}
	recent = validRecent

	labels, err := h.store.ListFolderLabels(ctx, "")
	if err != nil {
		labels = []*store.FolderLabel{}
	}
	validLabels := make([]*store.FolderLabel, 0, len(labels))
	for _, l := range labels {
		if _, statErr := os.Stat(l.Path); statErr == nil {
			validLabels = append(validLabels, l)
		} else {
			_ = h.store.DeleteFolderLabel(ctx, l.Path)
		}
	}
	labels = validLabels

	// Fetch websites/domains for My Domains section
	sites, _ := h.store.ListWebsitesByOrg(ctx, uuid.Nil)
	type DomainItem struct {
		ID           string `json:"id"`
		Domain       string `json:"domain"`
		DocumentRoot string `json:"document_root"`
		Status       string `json:"status"`
	}
	domainItems := make([]DomainItem, 0)
	for _, s := range sites {
		if s.DeletedAt == nil {
			domainItems = append(domainItems, DomainItem{
				ID:           s.ID.String(),
				Domain:       s.PrimaryDomain,
				DocumentRoot: s.DocumentRoot,
				Status:       s.Status,
			})
		}
	}

	// Trash stats
	trashItems, _ := h.store.ListFileManagerTrash(ctx, "")
	var trashSize int64
	for _, t := range trashItems {
		trashSize += t.Size
	}

	// Determine existing root directory shortcut
	rootDirectory := "/var/www"
	if _, err := os.Stat("/www/wwwroot"); err == nil {
		rootDirectory = "/www/wwwroot"
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"favorites":      favs,
		"recent":         recent,
		"domains":        domainItems,
		"folder_labels":  labels,
		"root_directory": rootDirectory,
		"shortcuts": []map[string]string{
			{"name": "Root Directory", "path": rootDirectory, "icon": "folder"},
			{"name": "Trash Bin", "path": "/trash", "icon": "trash-2"},
			{"name": "Backups", "path": "/var/backups", "icon": "archive"},
			{"name": "Uploads", "path": filepath.Join(rootDirectory, "uploads"), "icon": "upload"},
			{"name": "Downloads", "path": filepath.Join(rootDirectory, "downloads"), "icon": "download"},
		},
		"trash_stats": map[string]interface{}{
			"count": len(trashItems),
			"bytes": trashSize,
		},
	}, nil)
}

// AddFavorite pins a folder to quick access
func (h *FileHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	cleanPath := filepath.Clean(req.Path)
	if cleanPath == "" || cleanPath == "." {
		response.Error(w, http.StatusBadRequest, "INVALID_PATH", "Valid path required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, cleanPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	name := req.Name
	if name == "" {
		name = filepath.Base(cleanPath)
	}

	fav := &store.FileManagerFavorite{
		ID:     uuid.New(),
		UserID: h.getUserID(r),
		Domain: req.Domain,
		Path:   cleanPath,
		Name:   name,
		Color:  req.Color,
	}

	if err := h.store.AddFileManagerFavorite(r.Context(), fav); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.favorite.add", "folder", cleanPath, "success", "", nil)

	response.JSON(w, http.StatusOK, fav, nil)
}

// DeleteFavorite unpins a folder from quick access
func (h *FileHandler) DeleteFavorite(w http.ResponseWriter, r *http.Request) {
	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath == "" && r.Body != nil {
		var req struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		targetPath = strings.TrimSpace(req.Path)
	}

	if targetPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	cleanPath := filepath.Clean(targetPath)
	userID := h.getUserID(r)

	if err := h.store.DeleteFileManagerFavorite(r.Context(), userID, cleanPath); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.favorite.delete", "folder", cleanPath, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"path":    cleanPath,
	}, nil)
}

// RecordRecent updates the last accessed timestamp for a recently opened folder
func (h *FileHandler) RecordRecent(w http.ResponseWriter, r *http.Request) {
	var req RecentFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	cleanPath := filepath.Clean(req.Path)
	if cleanPath == "" || cleanPath == "." {
		response.Error(w, http.StatusBadRequest, "INVALID_PATH", "Valid path required", nil, "")
		return
	}

	rec := &store.FileManagerRecent{
		ID:     uuid.New(),
		UserID: h.getUserID(r),
		Domain: req.Domain,
		Path:   cleanPath,
	}

	_ = h.store.RecordFileManagerRecent(r.Context(), rec)

	response.JSON(w, http.StatusOK, map[string]interface{}{"recorded": true}, nil)
}

// SetFolderLabel assigns a color and label to a folder
func (h *FileHandler) SetFolderLabel(w http.ResponseWriter, r *http.Request) {
	var req FolderLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	cleanPath := filepath.Clean(req.Path)
	if cleanPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	label := &store.FolderLabel{
		ID:     uuid.New(),
		Domain: req.Domain,
		Path:   cleanPath,
		Color:  req.Color,
		Label:  req.Label,
	}

	if err := h.store.SetFolderLabel(r.Context(), label); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, label, nil)
}

// DeleteFolderLabel removes color label from a folder
func (h *FileHandler) DeleteFolderLabel(w http.ResponseWriter, r *http.Request) {
	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	_ = h.store.DeleteFolderLabel(r.Context(), filepath.Clean(targetPath))
	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true}, nil)
}

// Tree returns lazy subdirectories for the Windows/VS Code folder tree
func (h *FileHandler) Tree(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		targetPath = "/var/www"
		if _, err := os.Stat("/www/wwwroot"); err == nil {
			targetPath = "/www/wwwroot"
		} else if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			targetPath = "/"
		}
	}

	if err := h.checkPathAuthorization(r, targetPath); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	showHidden := r.URL.Query().Get("show_hidden") == "true" || r.URL.Query().Get("show_hidden") == "1"

	nodes, err := h.fileMgr.ListTree(targetPath, showHidden)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "TREE_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"path":  filepath.Clean(targetPath),
		"nodes": nodes,
		"count": len(nodes),
	}, nil)
}

// Move moves files or folders with conflict handling and cross-domain support
func (h *FileHandler) Move(w http.ResponseWriter, r *http.Request) {
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	var sources []string
	if len(req.SrcPaths) > 0 {
		sources = req.SrcPaths
	} else if req.SrcPath != "" {
		sources = []string{req.SrcPath}
	}

	if len(sources) == 0 || req.DestPath == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PATHS", "Sources and dest_path are required", nil, "")
		return
	}

	destDir := filepath.Clean(req.DestPath)
	if err := h.checkPathAuthorization(r, destDir); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	// Auto-create destination folder if missing
	if err := os.MkdirAll(destDir, 0755); err != nil {
		response.Error(w, http.StatusInternalServerError, "MKDIR_ERROR", "Cannot create destination directory: "+err.Error(), nil, "")
		return
	}

	var movedPaths []string
	var errorMessages []string

	for _, src := range sources {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		if err := h.checkPathAuthorization(r, src); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", src, err.Error()))
			continue
		}

		targetDest := filepath.Join(destDir, filepath.Base(src))

		resolvedTarget, skip, err := h.fileMgr.ResolveConflictPath(targetDest, req.ConflictStrategy)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", src, err.Error()))
			continue
		}
		if skip {
			continue
		}

		if err := h.fileMgr.Rename(src, resolvedTarget); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", src, err.Error()))
			continue
		}

		h.audit.Log(r.Context(), r, "file.move", "file", src, "success", "", map[string]interface{}{
			"dest_path":   resolvedTarget,
			"dest_domain": req.DestDomain,
		})
		_ = h.store.RecordFileManagerActivityLog(r.Context(), &store.FileManagerActivityLog{
			ID:              uuid.New(),
			UserID:          h.getUserID(r),
			Domain:          req.DestDomain,
			Action:          "move",
			SourcePath:      src,
			DestinationPath: resolvedTarget,
		})

		movedPaths = append(movedPaths, resolvedTarget)
	}

	if len(movedPaths) == 0 && len(errorMessages) > 0 {
		response.Error(w, http.StatusBadRequest, "MOVE_ERROR", strings.Join(errorMessages, "; "), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"moved":  true,
		"paths":  movedPaths,
		"count":  len(movedPaths),
		"errors": errorMessages,
	}, nil)
}

// MoveToTrash moves files or folders into the Enterprise Trash Bin instead of permanent deletion
func (h *FileHandler) MoveToTrash(w http.ResponseWriter, r *http.Request) {
	var req TrashMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	var targets []string
	if len(req.Paths) > 0 {
		targets = req.Paths
	} else if req.Path != "" {
		targets = []string{req.Path}
	}

	if len(targets) == 0 {
		response.Error(w, http.StatusBadRequest, "MISSING_PATH", "Path required", nil, "")
		return
	}

	trashBase := h.trashDir()
	var trashedItems []*store.FileManagerTrashItem
	var errorMessages []string

	claims, _ := auth.GetClaims(r.Context())
	deletedBy := "Administrator"
	if claims != nil && claims.Email != "" {
		deletedBy = claims.Email
	}

	for _, target := range targets {
		clean := filepath.Clean(target)
		if clean == "/" || clean == "." {
			errorMessages = append(errorMessages, clean+": Root cannot be deleted")
			continue
		}

		if err := h.checkPathAuthorization(r, clean); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", clean, err.Error()))
			continue
		}

		info, err := os.Stat(clean)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", clean, err.Error()))
			continue
		}

		trashID := uuid.New()
		trashFilename := fmt.Sprintf("%s_%s", trashID.String(), filepath.Base(clean))
		physicalTrashPath := filepath.Join(trashBase, trashFilename)

		var totalSize int64
		isDir := info.IsDir()
		if isDir {
			sz, _, _, _ := h.fileMgr.CalculateDirSize(clean)
			totalSize = sz
		} else {
			totalSize = info.Size()
		}

		// Move to physical trash directory
		if err := h.fileMgr.Rename(clean, physicalTrashPath); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", clean, err.Error()))
			continue
		}

		trashItem := &store.FileManagerTrashItem{
			ID:           trashID,
			UserID:       h.getUserID(r),
			Domain:       req.Domain,
			OriginalPath: clean,
			TrashPath:    physicalTrashPath,
			Name:         filepath.Base(clean),
			Size:         totalSize,
			FileType:     filepath.Ext(clean),
			IsDir:        isDir,
			DeletedBy:    deletedBy,
			DeletedAt:    time.Now().UTC(),
		}

		if err := h.store.AddFileManagerTrash(r.Context(), trashItem); err != nil {
			log.Printf("[WARN] Failed to record trash metadata in database (path=%s): %v", clean, err)
		}

		_ = h.store.DeleteFileManagerFavorite(r.Context(), h.getUserID(r), clean)
		_ = h.store.DeleteFileManagerRecent(r.Context(), clean)
		_ = h.store.DeleteFolderLabel(r.Context(), clean)

		h.audit.Log(r.Context(), r, "file.trash", "file", clean, "success", "", map[string]interface{}{
			"trash_id": trashID.String(),
			"size":     totalSize,
		})
		_ = h.store.RecordFileManagerActivityLog(r.Context(), &store.FileManagerActivityLog{
			ID:         uuid.New(),
			UserID:     h.getUserID(r),
			UserEmail:  deletedBy,
			Domain:     req.Domain,
			Action:     "trash",
			SourcePath: clean,
		})

		trashedItems = append(trashedItems, trashItem)
	}

	if len(trashedItems) == 0 && len(errorMessages) > 0 {
		response.Error(w, http.StatusBadRequest, "TRASH_ERROR", strings.Join(errorMessages, "; "), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"trashed": true,
		"items":   trashedItems,
		"count":   len(trashedItems),
		"errors":  errorMessages,
	}, nil)
}

// ListTrash lists all items currently in the Enterprise Trash Bin
func (h *FileHandler) ListTrash(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	items, err := h.store.ListFileManagerTrash(r.Context(), domain)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	var totalFiles int
	var totalFolders int
	var totalSize int64

	for _, it := range items {
		if it.IsDir {
			totalFolders++
		} else {
			totalFiles++
		}
		totalSize += it.Size
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"items":         items,
		"count":         len(items),
		"total_files":   totalFiles,
		"total_folders": totalFolders,
		"total_size":    totalSize,
	}, nil)
}

// RestoreFromTrash restores trashed items to original or custom destination with conflict resolution
func (h *FileHandler) RestoreFromTrash(w http.ResponseWriter, r *http.Request) {
	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	var ids []string
	if len(req.TrashIDs) > 0 {
		ids = req.TrashIDs
	} else if req.TrashID != "" {
		ids = []string{req.TrashID}
	}

	if len(ids) == 0 {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "trash_id or trash_ids required", nil, "")
		return
	}

	var restoredPaths []string
	var errorMessages []string

	for _, rawID := range ids {
		parsedID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: Invalid UUID", rawID))
			continue
		}

		item, err := h.store.GetFileManagerTrashItem(r.Context(), parsedID)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: Trash item not found", rawID))
			continue
		}

		// Determine target path
		targetPath := item.OriginalPath
		if req.RestoreTo == "custom_folder" && req.CustomPath != "" {
			targetPath = filepath.Join(req.CustomPath, item.Name)
		} else if req.RestoreTo == "domain" && req.TargetDomain != "" {
			// Find domain's document root
			targetPath = filepath.Join("/var/www", req.TargetDomain, item.Name)
			if sites, err := h.store.ListWebsitesByOrg(r.Context(), uuid.Nil); err == nil {
				for _, s := range sites {
					if s.PrimaryDomain == req.TargetDomain {
						targetPath = filepath.Join(s.DocumentRoot, item.Name)
						break
					}
				}
			}
		}

		cleanDest := filepath.Clean(targetPath)
		if err := h.checkPathAuthorization(r, cleanDest); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", item.Name, err.Error()))
			continue
		}

		// Automatically recreate destination directory if missing
		parentDir := filepath.Dir(cleanDest)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: Cannot create directory: %s", item.Name, err.Error()))
			continue
		}

		resolvedDest, skip, err := h.fileMgr.ResolveConflictPath(cleanDest, req.ConflictStrategy)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", item.Name, err.Error()))
			continue
		}
		if skip {
			continue
		}

		// Move from physical trash back to target
		if err := h.fileMgr.Rename(item.TrashPath, resolvedDest); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", item.Name, err.Error()))
			continue
		}

		// Delete from database
		_ = h.store.DeleteFileManagerTrashItem(r.Context(), parsedID)

		h.audit.Log(r.Context(), r, "file.restore", "file", resolvedDest, "success", "", map[string]interface{}{
			"original_path": item.OriginalPath,
			"restored_path": resolvedDest,
		})
		_ = h.store.RecordFileManagerActivityLog(r.Context(), &store.FileManagerActivityLog{
			ID:              uuid.New(),
			UserID:          h.getUserID(r),
			Domain:          item.Domain,
			Action:          "restore",
			SourcePath:      item.OriginalPath,
			DestinationPath: resolvedDest,
		})

		restoredPaths = append(restoredPaths, resolvedDest)
	}

	if len(restoredPaths) == 0 && len(errorMessages) > 0 {
		response.Error(w, http.StatusBadRequest, "RESTORE_ERROR", strings.Join(errorMessages, "; "), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"restored": true,
		"paths":    restoredPaths,
		"count":    len(restoredPaths),
		"errors":   errorMessages,
	}, nil)
}

// EmptyTrash permanently purges all files in the Enterprise Trash Bin with path containment verification
func (h *FileHandler) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	var req EmptyTrashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	// Strictly require typing "DELETE" for security
	if strings.TrimSpace(req.Confirmation) != "DELETE" {
		response.Error(w, http.StatusBadRequest, "CONFIRMATION_REQUIRED", "Security check: Type 'DELETE' to confirm permanent purge", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && claims.Role != "" && claims.Role != "owner" && claims.Role != "admin" {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", "Only Administrators can permanently empty the trash bin", nil, "")
		return
	}

	items, err := h.store.ListFileManagerTrash(r.Context(), req.Domain)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	trashBase := filepath.Clean(h.trashDir())
	var purgedCount int
	var freedBytes int64
	var skippedCount int

	for _, item := range items {
		// Security: Verify each item's trash path is canonically inside the trash directory
		cleanPath := filepath.Clean(item.TrashPath)
		if !strings.HasPrefix(cleanPath, trashBase+string(filepath.Separator)) {
			// Path injection — skip this item and log the anomaly
			h.audit.Log(r.Context(), r, "file.trash.path_escape_blocked", "trash", item.TrashPath, "blocked", "", map[string]interface{}{
				"trash_base": trashBase,
				"item_path":  cleanPath,
				"item_id":    item.ID.String(),
			})
			skippedCount++
			continue
		}

		// Permanently remove physical file or directory
		_ = os.RemoveAll(cleanPath)
		purgedCount++
		freedBytes += item.Size
	}

	// Delete from database
	if err := h.store.EmptyFileManagerTrash(r.Context(), req.Domain); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "file.trash.empty", "trash", "", "success", "", map[string]interface{}{
		"files_purged":  purgedCount,
		"bytes_freed":   freedBytes,
		"items_skipped": skippedCount,
		"domain":        req.Domain,
	})
	_ = h.store.RecordFileManagerActivityLog(r.Context(), &store.FileManagerActivityLog{
		ID:         uuid.New(),
		UserID:     h.getUserID(r),
		Domain:     req.Domain,
		Action:     "empty_trash",
		SourcePath: "",
		Details: map[string]interface{}{
			"files_purged":  purgedCount,
			"bytes_freed":   freedBytes,
			"items_skipped": skippedCount,
		},
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"empty":         true,
		"files_purged":  purgedCount,
		"bytes_freed":   freedBytes,
		"items_skipped": skippedCount,
	}, nil)
}

// DeleteTrashItem permanently deletes a single item from trash with path containment verification
func (h *FileHandler) DeleteTrashItem(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	if rawID == "" {
		rawID = r.URL.Query().Get("id")
	}

	parsedID, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Valid UUID required", nil, "")
		return
	}

	item, err := h.store.GetFileManagerTrashItem(r.Context(), parsedID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Trash item not found", nil, "")
		return
	}

	// Security: Verify the trash path is canonically inside the trash directory.
	// This prevents a corrupted or malicious database record from causing
	// arbitrary directory deletion via path injection.
	cleanTrash := filepath.Clean(item.TrashPath)
	trashBase := filepath.Clean(h.trashDir())
	if !strings.HasPrefix(cleanTrash, trashBase+string(filepath.Separator)) {
		response.Error(w, http.StatusForbidden, "PATH_ESCAPE",
			"Trash item path is outside the authorized trash directory", nil, "")
		h.audit.Log(r.Context(), r, "file.trash.path_escape_blocked", "trash", item.TrashPath, "blocked", "", map[string]interface{}{
			"trash_base": trashBase,
			"item_path":  cleanTrash,
		})
		return
	}

	// Verify caller ownership: item must belong to caller or caller must be admin
	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && claims.Role != "" && claims.Role != "owner" && claims.Role != "admin" {
		if item.UserID != nil && claims.UserID != *item.UserID {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED",
				"You can only permanently delete your own trash items", nil, "")
			return
		}
	}

	// Permanently wipe physical file/folder
	_ = os.RemoveAll(cleanTrash)

	_ = h.store.DeleteFileManagerTrashItem(r.Context(), parsedID)

	h.audit.Log(r.Context(), r, "file.trash.delete_item", "trash", item.OriginalPath, "success", "", map[string]interface{}{
		"trash_path": cleanTrash,
		"size":       item.Size,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"id":      rawID,
	}, nil)
}

// Search performs full-text or filter-based global search across directories and domains
func (h *FileHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	targetPath := r.URL.Query().Get("path")
	filterType := r.URL.Query().Get("type")
	allDomains := r.URL.Query().Get("all_domains") == "true" || r.URL.Query().Get("all_domains") == "1"

	var minSize int64
	var maxSize int64
	if s := r.URL.Query().Get("min_size"); s != "" {
		minSize, _ = strconv.ParseInt(s, 10, 64)
	}
	if s := r.URL.Query().Get("max_size"); s != "" {
		maxSize, _ = strconv.ParseInt(s, 10, 64)
	}

	var rootsToSearch []string

	if allDomains {
		if sites, err := h.store.ListWebsitesByOrg(r.Context(), uuid.Nil); err == nil && len(sites) > 0 {
			for _, s := range sites {
				if s.DeletedAt == nil && s.DocumentRoot != "" {
					rootsToSearch = append(rootsToSearch, s.DocumentRoot)
				}
			}
		}
	}

	if len(rootsToSearch) == 0 {
		if targetPath == "" {
			targetPath = "/var/www"
			if _, err := os.Stat("/www/wwwroot"); err == nil {
				targetPath = "/www/wwwroot"
			}
		}
		if err := h.checkPathAuthorization(r, targetPath); err != nil {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
			return
		}
		rootsToSearch = []string{targetPath}
	}

	var allResults []files.FileItem
	for _, root := range rootsToSearch {
		if err := h.checkPathAuthorization(r, root); err != nil {
			continue
		}
		res, err := h.fileMgr.Search(root, query, filterType, minSize, maxSize, 150)
		if err == nil {
			allResults = append(allResults, res...)
		}
		if len(allResults) >= 300 {
			allResults = allResults[:300]
			break
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"results": allResults,
		"count":   len(allResults),
		"query":   query,
	}, nil)
}

// ChunkUpload handles large multipart chunk uploads (>10GB support)
func (h *FileHandler) ChunkUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, "UPLOAD_ERROR", "Failed to parse chunk multipart body", nil, "")
		return
	}

	uploadID := r.FormValue("upload_id")
	filename := filepath.Base(filepath.Clean(r.FormValue("filename")))
	targetDir := r.FormValue("target_dir")
	chunkIndex, _ := strconv.Atoi(r.FormValue("chunk_index"))
	totalChunks, _ := strconv.Atoi(r.FormValue("total_chunks"))

	if uploadID == "" || filename == "" || targetDir == "" || totalChunks <= 0 {
		response.Error(w, http.StatusBadRequest, "MISSING_PARAMS", "upload_id, filename, target_dir, and total_chunks required", nil, "")
		return
	}

	if err := h.checkPathAuthorization(r, targetDir); err != nil {
		response.Error(w, http.StatusForbidden, "ACCESS_DENIED", err.Error(), nil, "")
		return
	}

	file, _, err := r.FormFile("chunk")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "MISSING_CHUNK", "No chunk file uploaded", nil, "")
		return
	}
	defer file.Close()

	chunkDir := filepath.Join(os.TempDir(), "hostvra_chunks", uploadID)
	_ = os.MkdirAll(chunkDir, 0755)

	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%05d", chunkIndex))
	out, err := os.Create(chunkPath)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "IO_ERROR", err.Error(), nil, "")
		return
	}
	_, copyErr := io.Copy(out, file)
	out.Close()
	if copyErr != nil {
		response.Error(w, http.StatusInternalServerError, "IO_ERROR", copyErr.Error(), nil, "")
		return
	}

	// Check if all chunks have arrived
	isComplete := false
	if chunkIndex == totalChunks-1 {
		chunkFiles := make([]string, totalChunks)
		allPresent := true
		for i := 0; i < totalChunks; i++ {
			p := filepath.Join(chunkDir, fmt.Sprintf("chunk_%05d", i))
			if _, err := os.Stat(p); os.IsNotExist(err) {
				allPresent = false
				break
			}
			chunkFiles[i] = p
		}

		if allPresent {
			finalPath := filepath.Join(targetDir, filename)
			if err := h.fileMgr.MergeChunks(finalPath, chunkFiles); err != nil {
				response.Error(w, http.StatusInternalServerError, "MERGE_ERROR", err.Error(), nil, "")
				return
			}
			_ = os.RemoveAll(chunkDir)
			isComplete = true

			h.audit.Log(r.Context(), r, "file.upload.chunked", "file", finalPath, "success", "", nil)
			_ = h.store.RecordFileManagerActivityLog(r.Context(), &store.FileManagerActivityLog{
				ID:              uuid.New(),
				UserID:          h.getUserID(r),
				Action:          "upload",
				DestinationPath: finalPath,
			})
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"chunk_index": chunkIndex,
		"complete":    isComplete,
		"filename":    filename,
	}, nil)
}

// StreamZipDownload archives multiple files/folders into a streaming ZIP download
func (h *FileHandler) StreamZipDownload(w http.ResponseWriter, r *http.Request) {
	rawPaths := r.URL.Query()["paths"]
	if len(rawPaths) == 0 {
		rawPaths = strings.Split(r.URL.Query().Get("paths_csv"), ",")
	}

	var validPaths []string
	for _, p := range rawPaths {
		clean := strings.TrimSpace(p)
		if clean != "" && h.checkPathAuthorization(r, clean) == nil {
			validPaths = append(validPaths, clean)
		}
	}

	if len(validPaths) == 0 {
		http.Error(w, "no authorized paths to archive", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\"download.zip\"")
	w.Header().Set("Content-Type", "application/zip")

	_ = h.fileMgr.StreamZip(validPaths, w)
}

// StorageInfo returns folder size, domain disk usage, free space, and trash size
func (h *FileHandler) StorageInfo(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		targetPath = "/var/www"
		if _, err := os.Stat("/www/wwwroot"); err == nil {
			targetPath = "/www/wwwroot"
		}
	}

	var folderSize int64
	var fileCount int
	var dirCount int
	if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
		folderSize, fileCount, dirCount, _ = h.fileMgr.CalculateDirSize(targetPath)
	}

	// Trash size
	trashItems, _ := h.store.ListFileManagerTrash(r.Context(), "")
	var trashSize int64
	for _, t := range trashItems {
		trashSize += t.Size
	}

	// Disk storage via statfs
	var stat syscall.Statfs_t
	var totalDiskBytes int64
	var freeDiskBytes int64
	var usedDiskBytes int64
	if err := syscall.Statfs(targetPath, &stat); err == nil {
		totalDiskBytes = int64(stat.Blocks) * int64(stat.Bsize)
		freeDiskBytes = int64(stat.Bavail) * int64(stat.Bsize)
		usedDiskBytes = totalDiskBytes - freeDiskBytes
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"folder_size":      folderSize,
		"file_count":       fileCount,
		"dir_count":        dirCount,
		"trash_size":       trashSize,
		"total_disk_bytes": totalDiskBytes,
		"used_disk_bytes":  usedDiskBytes,
		"free_disk_bytes":  freeDiskBytes,
	}, nil)
}

// ActivityLogs returns recent file manager activity audit logs
func (h *FileHandler) ActivityLogs(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	logs, err := h.store.ListFileManagerActivityLogs(r.Context(), domain, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	}, nil)
}

// ListDomains returns website domains for multi-domain directory switching
func (h *FileHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	sites, err := h.store.ListWebsitesByOrg(r.Context(), uuid.Nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	type SimpleDomain struct {
		ID           string `json:"id"`
		Domain       string `json:"domain"`
		DocumentRoot string `json:"document_root"`
		Status       string `json:"status"`
	}

	res := make([]SimpleDomain, 0)
	for _, s := range sites {
		if s.DeletedAt == nil {
			res = append(res, SimpleDomain{
				ID:           s.ID.String(),
				Domain:       s.PrimaryDomain,
				DocumentRoot: s.DocumentRoot,
				Status:       s.Status,
			})
		}
	}

	response.JSON(w, http.StatusOK, res, nil)
}


