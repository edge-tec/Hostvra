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
