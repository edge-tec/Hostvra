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

func setupTestFileManagerHandler(t *testing.T) (*FileHandler, string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "hostvra-fm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	logger := audit.NewLogger(memStore, slogger)

	handler := NewFileHandler(cfg, memStore, logger)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return handler, tempDir, cleanup
}

var fixedTestUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func testContextWithAdmin() context.Context {
	claims := &auth.Claims{
		UserID: fixedTestUserID,
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	return context.WithValue(context.Background(), auth.UserContextKey, claims)
}

func TestFileManager_QuickAccessAndFavorites(t *testing.T) {
	handler, tempDir, cleanup := setupTestFileManagerHandler(t)
	defer cleanup()

	folderA := filepath.Join(tempDir, "folderA")
	_ = os.MkdirAll(folderA, 0755)

	// 1. Add Favorite
	favBody, _ := json.Marshal(map[string]interface{}{
		"path":   folderA,
		"domain": "test.com",
		"name":   "Folder A",
		"color":  "emerald",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/filemanager/favorite", bytes.NewReader(favBody))
	req = req.WithContext(testContextWithAdmin())
	w := httptest.NewRecorder()
	handler.AddFavorite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("AddFavorite returned %d: %s", w.Code, w.Body.String())
	}

	// 2. Set Folder Label
	labelBody, _ := json.Marshal(map[string]interface{}{
		"path":   folderA,
		"domain": "test.com",
		"color":  "purple",
		"label":  "Important",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/filemanager/label", bytes.NewReader(labelBody))
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.SetFolderLabel(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("SetFolderLabel returned %d: %s", w.Code, w.Body.String())
	}

	// 3. Get Quick Access
	req = httptest.NewRequest(http.MethodGet, "/api/v1/filemanager/quick-access", nil)
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.QuickAccess(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("QuickAccess returned %d: %s", w.Code, w.Body.String())
	}

	var qaRes map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &qaRes)
	data, ok := qaRes["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data object in quick-access response")
	}
	favs, ok := data["favorites"].([]interface{})
	if !ok || len(favs) == 0 {
		t.Fatalf("Expected at least 1 favorite in response, got %v", data["favorites"])
	}

	// 4. Delete Favorite
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/filemanager/favorite?path="+folderA, nil)
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.DeleteFavorite(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFavorite returned %d: %s", w.Code, w.Body.String())
	}
}

func TestFileManager_TrashAndRestoreWizard(t *testing.T) {
	handler, tempDir, cleanup := setupTestFileManagerHandler(t)
	defer cleanup()

	testFile := filepath.Join(tempDir, "document.txt")
	_ = os.WriteFile(testFile, []byte("Important content for trash test"), 0644)

	// 1. Move to Trash
	trashBody, _ := json.Marshal(map[string]interface{}{
		"path":   testFile,
		"domain": "test.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/filemanager/trash", bytes.NewReader(trashBody))
	req = req.WithContext(testContextWithAdmin())
	w := httptest.NewRecorder()
	handler.MoveToTrash(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("MoveToTrash returned %d: %s", w.Code, w.Body.String())
	}

	// Original file must no longer exist at original path
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatalf("Expected original file to be moved to trash, but still exists")
	}

	// 2. List Trash
	req = httptest.NewRequest(http.MethodGet, "/api/v1/filemanager/trash", nil)
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.ListTrash(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListTrash returned %d: %s", w.Code, w.Body.String())
	}

	var listRes map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &listRes)
	data := listRes["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) == 0 {
		t.Fatalf("Expected at least 1 item in trash")
	}
	firstItem := items[0].(map[string]interface{})
	trashID := firstItem["id"].(string)

	// 3. Restore from Trash to original location
	restoreBody, _ := json.Marshal(map[string]interface{}{
		"trash_id":          trashID,
		"restore_to":        "original",
		"conflict_strategy": "rename",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/filemanager/restore", bytes.NewReader(restoreBody))
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.RestoreFromTrash(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RestoreFromTrash returned %d: %s", w.Code, w.Body.String())
	}

	// File should be restored back to testFile
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("Expected file to be restored at original path: %v", err)
	}

	// 4. Test Empty Trash confirmation requirement
	trashBody2, _ := json.Marshal(map[string]interface{}{
		"path":   testFile,
		"domain": "test.com",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/filemanager/trash", bytes.NewReader(trashBody2))
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.MoveToTrash(w, req)

	// Fail empty trash without "DELETE"
	badEmptyBody, _ := json.Marshal(map[string]interface{}{
		"confirmation": "wrong_confirmation",
	})
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/filemanager/trash/empty", bytes.NewReader(badEmptyBody))
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.EmptyTrash(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("EmptyTrash should reject request when confirmation is not 'DELETE'")
	}

	// Succeed empty trash with "DELETE"
	goodEmptyBody, _ := json.Marshal(map[string]interface{}{
		"confirmation": "DELETE",
	})
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/filemanager/trash/empty", bytes.NewReader(goodEmptyBody))
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.EmptyTrash(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("EmptyTrash with 'DELETE' confirmation failed: %d: %s", w.Code, w.Body.String())
	}
}

func TestFileManager_MoveAndTree(t *testing.T) {
	handler, tempDir, cleanup := setupTestFileManagerHandler(t)
	defer cleanup()

	dir1 := filepath.Join(tempDir, "sourceDir")
	dir2 := filepath.Join(tempDir, "destDir")
	_ = os.MkdirAll(dir1, 0755)
	_ = os.MkdirAll(dir2, 0755)

	file1 := filepath.Join(dir1, "hello.php")
	_ = os.WriteFile(file1, []byte("<?php echo 'hello'; ?>"), 0644)

	// 1. Move file
	moveBody, _ := json.Marshal(map[string]interface{}{
		"src_path":          file1,
		"dest_path":         dir2,
		"conflict_strategy": "rename",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/filemanager/move", bytes.NewReader(moveBody))
	req = req.WithContext(testContextWithAdmin())
	w := httptest.NewRecorder()
	handler.Move(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Move returned %d: %s", w.Code, w.Body.String())
	}

	expectedDest := filepath.Join(dir2, "hello.php")
	if _, err := os.Stat(expectedDest); err != nil {
		t.Fatalf("Moved file does not exist at dest: %v", err)
	}

	// 2. Tree check
	req = httptest.NewRequest(http.MethodGet, "/api/v1/filemanager/tree?path="+tempDir, nil)
	req = req.WithContext(testContextWithAdmin())
	w = httptest.NewRecorder()
	handler.Tree(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Tree returned %d: %s", w.Code, w.Body.String())
	}
}
