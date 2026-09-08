package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestFileHandler_Lifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-api-file-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewFileHandler(cfg, memStore, auditLogger)

	// Context with Claims
	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// 1. Mkdir
	subDir := filepath.Join(tempDir, "mysite")
	mkdirBody, _ := json.Marshal(map[string]string{"path": subDir})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/mkdir", bytes.NewReader(mkdirBody))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec := httptest.NewRecorder()
	h.Mkdir(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for Mkdir, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. SaveContent (Write File)
	testFile := filepath.Join(subDir, "index.html")
	saveBody, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": "<h1>Welcome to Hostvra</h1>",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/files/content", bytes.NewReader(saveBody))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec = httptest.NewRecorder()
	h.SaveContent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SaveContent, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. GetContent
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/content?path="+testFile, nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec = httptest.NewRecorder()
	h.GetContent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetContent, got %d: %s", rec.Code, rec.Body.String())
	}

	var contentRes struct {
		Success bool `json:"success"`
		Data    struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &contentRes)
	if contentRes.Data.Content != "<h1>Welcome to Hostvra</h1>" {
		t.Errorf("unexpected content: %s", contentRes.Data.Content)
	}

	// 4. List Directory
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/list?path="+subDir, nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec = httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for List, got %d: %s", rec.Code, rec.Body.String())
	}

	var listRes struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &listRes)
	if len(listRes.Data.Items) == 0 || listRes.Data.Items[0].Name != "index.html" {
		t.Errorf("expected index.html in list, got %+v", listRes.Data.Items)
	}

	// 5. Chmod Permissions
	chmodBody, _ := json.Marshal(map[string]string{
		"path": testFile,
		"mode": "0755",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/files/permissions", bytes.NewReader(chmodBody))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec = httptest.NewRecorder()
	h.Permissions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for Permissions, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/files/delete?path="+testFile, nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, claims))
	rec = httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for Delete, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted")
	}
}

func TestFileHandler_NonAdminConfinedAndForbiddenFromSystemPaths(t *testing.T) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewFileHandler(cfg, memStore, auditLogger)

	devClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "dev@client.com",
		Role:           "developer", // non-admin
	}

	// 1. Reading /etc/shadow must be forbidden
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/content?path=/etc/shadow", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, devClaims))
	rec := httptest.NewRecorder()
	h.GetContent(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for /etc/shadow access by developer, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Listing /root must be forbidden
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/list?path=/root", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, devClaims))
	rec = httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for /root list by developer, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Stat /etc/passwd must be forbidden
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/stat?path=/etc/passwd", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, devClaims))
	rec = httptest.NewRecorder()
	h.Stat(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for /etc/passwd stat by developer, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Download /etc/hosts must be forbidden
	req = httptest.NewRequest(http.MethodGet, "/api/v1/files/download?path=/etc/hosts", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, devClaims))
	rec = httptest.NewRecorder()
	h.Download(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for /etc/hosts download by developer, got %d", rec.Code)
	}
}

func TestFileHandler_SymlinkOverwriteForbidden(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-api-symlink-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewFileHandler(cfg, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "admin@hostvra.com",
		Role:           "owner",
	}

	// Create an existing symlink pointing to another file
	targetFile := filepath.Join(tempDir, "target.txt")
	_ = os.WriteFile(targetFile, []byte("target initial"), 0644)
	linkPath := filepath.Join(tempDir, "link.txt")
	_ = os.Symlink(targetFile, linkPath)

	// Attempt to SaveContent targeting the existing symlink
	saveBody, _ := json.Marshal(map[string]string{
		"path":    linkPath,
		"content": "attempted overwrite",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/files/content", bytes.NewReader(saveBody))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, adminClaims))
	rec := httptest.NewRecorder()
	h.SaveContent(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for symlink overwrite, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestFileHandler_NonAdminSymlinkEscapeTraversalBlocked(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-api-symlink-escape-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	h := NewFileHandler(cfg, memStore, auditLogger)

	devClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "developer@client.com",
		Role:           "developer", // non-admin
	}

	// Create a fake sensitive target outside user space
	sensitiveFile := filepath.Join(tempDir, "shadow_file")
	_ = os.WriteFile(sensitiveFile, []byte("root:secret_hash:12345"), 0600)

	// Create symlink inside an allowed user space prefix (e.g. /tmp) pointing to the sensitive file
	linkInTmp := filepath.Join(tempDir, "link_to_sensitive")
	_ = os.Symlink(sensitiveFile, linkInTmp)

	// Developer attempts to read via the symlink
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/content?path="+linkInTmp, nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, devClaims))
	rec := httptest.NewRecorder()
	h.GetContent(rec, req)

	// If the sensitive file is outside /var/www, /home, /tmp or in restricted roots, it must be forbidden
	// Let's create a symlink to /etc/hosts (which exists on all Unix systems)
	hostsLink := filepath.Join(tempDir, "hosts_symlink")
	_ = os.Symlink("/etc/hosts", hostsLink)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/files/content?path="+hostsLink, nil)
	req2 = req2.WithContext(context.WithValue(req2.Context(), auth.UserContextKey, devClaims))
	rec2 := httptest.NewRecorder()
	h.GetContent(rec2, req2)

	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for symlink pointing to /etc/hosts by developer, got %d: %s", rec2.Code, rec2.Body.String())
	}
}


