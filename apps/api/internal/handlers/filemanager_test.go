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
		UserID:       fixedTestUserID,
		Email:        "admin@hostvra.com",
		Role:         "admin",
		IsSuperAdmin: true,
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

func TestFileManager_StrictMultiTenantIsolation(t *testing.T) {
	handler, tempDir, cleanup := setupTestFileManagerHandler(t)
	defer cleanup()

	ctx := context.Background()

	userA := uuid.New()
	orgA := uuid.New()
	userB := uuid.New()
	orgB := uuid.New()

	docRootA := filepath.Join(tempDir, "userA_site")
	docRootB := filepath.Join(tempDir, "userB_site")
	_ = os.MkdirAll(docRootA, 0755)
	_ = os.MkdirAll(docRootB, 0755)
	_ = os.WriteFile(filepath.Join(docRootA, "indexA.html"), []byte("siteA"), 0644)
	_ = os.WriteFile(filepath.Join(docRootB, "indexB.html"), []byte("siteB"), 0644)

	// Register hosting accounts and websites
	_ = handler.store.CreateHostingAccount(ctx, &store.HostingAccount{
		ID:             uuid.New(),
		UserID:         userA,
		OrganizationID: orgA,
		Username:       "usera",
		Domain:         "customera.com",
		Status:         "active",
	})
	_ = handler.store.CreateHostingAccount(ctx, &store.HostingAccount{
		ID:             uuid.New(),
		UserID:         userB,
		OrganizationID: orgB,
		Username:       "userb",
		Domain:         "customerb.com",
		Status:         "active",
	})

	_ = handler.store.CreateWebsite(ctx, &store.Website{
		ID:              uuid.New(),
		OrganizationID:  orgA,
		AccountUsername: "usera",
		PrimaryDomain:   "customera.com",
		DocumentRoot:    docRootA,
		Status:          "running",
	})
	_ = handler.store.CreateWebsite(ctx, &store.Website{
		ID:              uuid.New(),
		OrganizationID:  orgB,
		AccountUsername: "userb",
		PrimaryDomain:   "customerb.com",
		DocumentRoot:    docRootB,
		Status:          "running",
	})

	ctxCustomerA := auth.WithClaims(ctx, &auth.Claims{
		UserID:         userA,
		OrganizationID: orgA,
		Role:           "customer",
	})

	ctxAdmin := auth.WithClaims(ctx, &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.Nil,
		Role:           "admin",
		IsSuperAdmin:   true,
	})

	// 1. Customer A ListDomains must only return customera.com
	reqA := httptest.NewRequest(http.MethodGet, "/api/v1/filemanager/domains", nil).WithContext(ctxCustomerA)
	wA := httptest.NewRecorder()
	handler.ListDomains(wA, reqA)
	if wA.Code != http.StatusOK {
		t.Fatalf("Customer A ListDomains failed: %d: %s", wA.Code, wA.Body.String())
	}
	var domRespA struct {
		Data []struct {
			Domain string `json:"domain"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wA.Body.Bytes(), &domRespA)
	if len(domRespA.Data) != 1 || domRespA.Data[0].Domain != "customera.com" {
		t.Fatalf("Expected customer A to see only customera.com, got: %+v", domRespA.Data)
	}

	// 2. Admin ListDomains returns all domains
	reqAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/filemanager/domains", nil).WithContext(ctxAdmin)
	wAdmin := httptest.NewRecorder()
	handler.ListDomains(wAdmin, reqAdmin)
	if wAdmin.Code != http.StatusOK {
		t.Fatalf("Admin ListDomains failed: %d: %s", wAdmin.Code, wAdmin.Body.String())
	}
	var domRespAdmin struct {
		Data []struct {
			Domain string `json:"domain"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAdmin.Body.Bytes(), &domRespAdmin)
	if len(domRespAdmin.Data) < 2 {
		t.Fatalf("Expected admin to see at least 2 domains, got: %+v", domRespAdmin.Data)
	}

	// 3. Customer A QuickAccess must have only customera.com in domain_items
	reqQA := httptest.NewRequest(http.MethodGet, "/api/v1/filemanager/quick-access", nil).WithContext(ctxCustomerA)
	wQA := httptest.NewRecorder()
	handler.QuickAccess(wQA, reqQA)
	if wQA.Code != http.StatusOK {
		t.Fatalf("Customer A QuickAccess failed: %d: %s", wQA.Code, wQA.Body.String())
	}
	var qaResp struct {
		Data struct {
			Domains []struct {
				Domain string `json:"domain"`
			} `json:"domains"`
			Shortcuts []struct {
				Name string `json:"name"`
			} `json:"shortcuts"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wQA.Body.Bytes(), &qaResp)
	if len(qaResp.Data.Domains) != 1 || qaResp.Data.Domains[0].Domain != "customera.com" {
		t.Fatalf("Expected customer A QuickAccess to see only customera.com, got: %+v", qaResp.Data.Domains)
	}
	for _, sc := range qaResp.Data.Shortcuts {
		if sc.Name == "Backups" {
			t.Fatalf("Customer A must never see Backups shortcut!")
		}
	}

	// 4. Customer A attempting to access Customer B's path must be forbidden
	reqForbidden := httptest.NewRequest(http.MethodGet, "/api/v1/files/list?path="+docRootB, nil).WithContext(ctxCustomerA)
	wForbidden := httptest.NewRecorder()
	handler.List(wForbidden, reqForbidden)
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when Customer A accesses Customer B directory, got: %d (%s)", wForbidden.Code, wForbidden.Body.String())
	}
}
