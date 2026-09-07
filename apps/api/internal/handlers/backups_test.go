package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/backup"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestBackupHandler_API(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_BACKUP_DIR", filepath.Join(tempDir, "backups"))
	t.Setenv("HOSTVRA_CONFIG_DIR", filepath.Join(tempDir, "config"))
	t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))

	// Create dummy web root
	siteDir := filepath.Join(tempDir, "www", "testdomain.com", "public_html")
	_ = os.MkdirAll(siteDir, 0755)
	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php echo 'hostvra'; ?>"), 0644)

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewBackupHandler(cfg, s, auditLogger)

	r := chi.NewRouter()
	r.Route("/api/v1/backups", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/create", h.Create)
		r.Post("/restore", h.Restore)
		r.Delete("/{id}", h.Delete)
		r.Get("/download/{id}", h.Download)
		r.Get("/destinations", h.ListDestinations)
		r.Post("/destinations", h.SaveDestination)
		r.Delete("/destinations/{id}", h.DeleteDestination)
		r.Get("/schedules", h.ListSchedules)
		r.Post("/schedules", h.SaveSchedule)
		r.Delete("/schedules/{id}", h.DeleteSchedule)
	})

	// 1. Initial List should be empty
	req := httptest.NewRequest("GET", "/api/v1/backups", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on initial list, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Create Backup
	createPayload := backup.CreateBackupRequest{
		ServerID:   "srv-test",
		Type:       "website",
		TargetName: "testdomain.com",
		Storage:    "local",
	}
	body, _ := json.Marshal(createPayload)
	req = httptest.NewRequest("POST", "/api/v1/backups/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool                `json:"success"`
		Data    backup.BackupRecord `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Data.ID == "" {
		t.Fatalf("expected created backup to have an ID, got empty")
	}

	// 3. List should now have 1 backup
	req = httptest.NewRequest("GET", "/api/v1/backups", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// 4. Test Restore
	restorePayload := backup.RestoreBackupRequest{
		BackupID: res.Data.ID,
		ServerID: "srv-test",
	}
	body, _ = json.Marshal(restorePayload)
	req = httptest.NewRequest("POST", "/api/v1/backups/restore", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on restore, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Test Cloud Destination
	destPayload := backup.DestinationConfig{
		Name:      "AWS S3 Offsite",
		Type:      "s3",
		Endpoint:  "s3.us-west-2.amazonaws.com",
		Region:    "us-west-2",
		Bucket:    "hostvra-prod-backups",
		AccessKey: "AKIA12345678",
		SecretKey: "secret-key-xyz",
		IsDefault: true,
	}
	body, _ = json.Marshal(destPayload)
	req = httptest.NewRequest("POST", "/api/v1/backups/destinations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on save destination, got %d: %s", rec.Code, rec.Body.String())
	}

	var destRes struct {
		Success bool                     `json:"success"`
		Data    backup.DestinationConfig `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&destRes); err != nil {
		t.Fatalf("failed to decode destination response: %v", err)
	}
	if destRes.Data.ID == "" {
		t.Fatalf("expected destination ID to be set")
	}

	// 6. Test Schedule
	schedPayload := backup.ScheduleConfig{
		Name:          "Nightly DB Snapshots",
		Scope:         "database",
		TargetName:    "prod_db",
		DestinationID: destRes.Data.ID,
		Frequency:     "daily",
		Retention:     7,
		Enabled:       true,
	}
	body, _ = json.Marshal(schedPayload)
	req = httptest.NewRequest("POST", "/api/v1/backups/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on save schedule, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Delete Backup
	req = httptest.NewRequest("DELETE", "/api/v1/backups/"+res.Data.ID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete backup, got %d: %s", rec.Code, rec.Body.String())
	}
}
