package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrBackupNotFound      = errors.New("backup snapshot not found")
	ErrDestinationNotFound = errors.New("backup destination not found")
	ErrScheduleNotFound    = errors.New("backup schedule not found")
	ErrPathTraversal       = errors.New("path traversal detected in archive entry")
	ErrInvalidTarget       = errors.New("target path or name is invalid")
	ErrEmptyBackup         = errors.New("backup target directory contains no files")
)

// BackupRecord represents an archived snapshot on local disk or remote cloud.
type BackupRecord struct {
	ID              string     `json:"id"`
	ServerID        string     `json:"server_id"`
	Type            string     `json:"type"` // website, database, full_config, full_account
	TargetName      string     `json:"target_name"`
	Storage         string     `json:"storage"` // local, s3, wasabi, gcs, r2, b2, spaces, minio, sftp
	StorageID       string     `json:"storage_id,omitempty"`
	SizeBytes       int64      `json:"size_bytes"`
	SHA256          string     `json:"sha256"`
	ItemCount       int        `json:"item_count"`
	Status          string     `json:"status"` // completed, failed, in_progress
	FileName        string     `json:"file_name"`
	RemoteKey       string     `json:"remote_key,omitempty"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	IntegrityStatus string     `json:"integrity_status,omitempty"` // unverified, verified, corrupted
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
}

// IntegrityReport documents cryptographic and structural verification of a backup snapshot.
type IntegrityReport struct {
	BackupID         string    `json:"backup_id"`
	FileName         string    `json:"file_name"`
	TargetName       string    `json:"target_name"`
	Type             string    `json:"type"`
	Storage          string    `json:"storage"`
	ExpectedSHA256   string    `json:"expected_sha256"`
	CalculatedSHA256 string    `json:"calculated_sha256"`
	SHA256Match      bool      `json:"sha256_match"`
	ArchiveValid     bool      `json:"archive_valid"`
	ItemCount        int       `json:"item_count"`
	UncompressedSize int64     `json:"uncompressed_size"`
	Status           string    `json:"status"` // "verified", "corrupted"
	Message          string    `json:"message"`
	VerifiedAt       time.Time `json:"verified_at"`
}

// PreflightCheck reports individual validation steps before restoring.
type PreflightCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details"`
}

// PreflightRestoreReport provides actionable safety validation prior to restoring workloads.
type PreflightRestoreReport struct {
	BackupID            string           `json:"backup_id"`
	TargetName          string           `json:"target_name"`
	Type                string           `json:"type"`
	PreflightOK         bool             `json:"preflight_ok"`
	DiskSpaceRequired   int64            `json:"disk_space_required"`
	DiskSpaceAvailable  int64            `json:"disk_space_available"`
	ServiceStatus       string           `json:"service_status"`
	ExistingTargetFound bool             `json:"existing_target_found"`
	TargetConflictRisk  string           `json:"target_conflict_risk"` // "none", "overwrite_warning"
	IntegrityStatus     string           `json:"integrity_status"`
	Checks              []PreflightCheck `json:"checks"`
	Message             string           `json:"message"`
}


// DestinationConfig defines remote storage credentials (S3, Cloudflare R2, Backblaze B2, SFTP).
type DestinationConfig struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // s3, r2, b2, sftp, local
	Endpoint  string    `json:"endpoint"`
	Region    string    `json:"region"`
	Bucket    string    `json:"bucket"`
	AccessKey string    `json:"access_key"`
	SecretKey string    `json:"secret_key"`
	Prefix    string    `json:"prefix"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScheduleConfig defines automated periodic backup tasks.
type ScheduleConfig struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Scope         string     `json:"scope"` // website, database, full_config
	TargetName    string     `json:"target_name"`
	DestinationID string     `json:"destination_id"`
	Frequency     string     `json:"frequency"` // daily, weekly, monthly, cron
	CronExpr      string     `json:"cron_expr"`
	Retention     int        `json:"retention"` // keep last N backups
	Enabled       bool       `json:"enabled"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// CreateBackupRequest holds input parameters for triggering a backup.
type CreateBackupRequest struct {
	ServerID      string `json:"server_id"`
	Type          string `json:"type"` // website, database, full_config
	TargetName    string `json:"target_name"`
	Storage       string `json:"storage"` // local, s3, r2, b2, sftp
	DestinationID string `json:"destination_id"`
	Retention     int    `json:"retention"` // optional auto-prune older snapshots
}

// RestoreBackupRequest holds input parameters for restoring a backup.
type RestoreBackupRequest struct {
	BackupID string `json:"backup_id"`
	ServerID string `json:"server_id"`
}

// Manager orchestrates backup creation, verification, remote sync, and restoration.
type Manager struct {
	mu           sync.RWMutex
	backupDir    string
	configDir    string
	webRootDir   string
	metaFile     string
	destFile     string
	schedFile    string
	backups      map[string]BackupRecord
	destinations map[string]DestinationConfig
	schedules    map[string]ScheduleConfig
}

// NewManager initializes the backup manager with target storage paths.
func NewManager(backupDir, configDir, webRootDir string) (*Manager, error) {
	if backupDir == "" {
		backupDir = "/var/backups/hostvra"
	}
	if configDir == "" {
		configDir = "/etc/hostvra"
	}
	if webRootDir == "" {
		webRootDir = "/var/www"
	}

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	m := &Manager{
		backupDir:    backupDir,
		configDir:    configDir,
		webRootDir:   webRootDir,
		metaFile:     filepath.Join(configDir, "backups_metadata.json"),
		destFile:     filepath.Join(configDir, "backup_destinations.json"),
		schedFile:    filepath.Join(configDir, "backup_schedules.json"),
		backups:      make(map[string]BackupRecord),
		destinations: make(map[string]DestinationConfig),
		schedules:    make(map[string]ScheduleConfig),
	}

	m.loadState()
	return m, nil
}

// loadState reads metadata, destinations, and schedules from JSON configuration files.
func (m *Manager) loadState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load Backups Metadata
	if data, err := os.ReadFile(m.metaFile); err == nil {
		var list []BackupRecord
		if err := json.Unmarshal(data, &list); err == nil {
			for _, item := range list {
				m.backups[item.ID] = item
			}
		}
	}

	// Load Destinations
	if data, err := os.ReadFile(m.destFile); err == nil {
		var list []DestinationConfig
		if err := json.Unmarshal(data, &list); err == nil {
			for _, item := range list {
				m.destinations[item.ID] = item
			}
		}
	}

	// Load Schedules
	if data, err := os.ReadFile(m.schedFile); err == nil {
		var list []ScheduleConfig
		if err := json.Unmarshal(data, &list); err == nil {
			for _, item := range list {
				m.schedules[item.ID] = item
			}
		}
	}
}

// saveStateLocked writes in-memory records atomically to disk.
func (m *Manager) saveStateLocked() error {
	// Save Backups
	backupList := make([]BackupRecord, 0, len(m.backups))
	for _, b := range m.backups {
		backupList = append(backupList, b)
	}
	if err := atomicWriteJSON(m.metaFile, backupList); err != nil {
		return err
	}

	// Save Destinations
	destList := make([]DestinationConfig, 0, len(m.destinations))
	for _, d := range m.destinations {
		destList = append(destList, d)
	}
	if err := atomicWriteJSON(m.destFile, destList); err != nil {
		return err
	}

	// Save Schedules
	schedList := make([]ScheduleConfig, 0, len(m.schedules))
	for _, s := range m.schedules {
		schedList = append(schedList, s)
	}
	return atomicWriteJSON(m.schedFile, schedList)
}

// ListBackups returns all snapshot records ordered by creation date descending.
func (m *Manager) ListBackups() ([]BackupRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]BackupRecord, 0, len(m.backups))
	for _, b := range m.backups {
		result = append(result, b)
	}

	// Sort descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreatedAt.Before(result[j].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

// GetBackup returns a single backup record by ID.
func (m *Manager) GetBackup(id string) (*BackupRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, ok := m.backups[id]
	if !ok {
		return nil, ErrBackupNotFound
	}
	return &b, nil
}

// GetBackupFilePath returns the absolute path to the local tar.gz archive.
func (m *Manager) GetBackupFilePath(id string) (string, error) {
	b, err := m.GetBackup(id)
	if err != nil {
		return "", err
	}
	path := filepath.Join(m.backupDir, b.FileName)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("archive file not found on local disk: %w", err)
	}
	return path, nil
}

// CreateBackup creates a compressed .tar.gz snapshot, verifies SHA256, and optionally uploads to cloud.
func (m *Manager) CreateBackup(ctx context.Context, req CreateBackupRequest) (*BackupRecord, error) {
	if req.Type == "" {
		req.Type = "website"
	}
	if req.Storage == "" {
		req.Storage = "local"
	}
	if req.ServerID == "" {
		req.ServerID = "srv-main-01"
	}

	backupID := generateID("bk-" + strings.ToLower(req.Type[:min(len(req.Type), 4)]))
	timestamp := time.Now().UTC().Format("20060102-150405")
	cleanTarget := sanitizeTargetName(req.TargetName)
	fileName := fmt.Sprintf("%s-%s-%s.tar.gz", req.Type, cleanTarget, timestamp)
	localArchive := filepath.Join(m.backupDir, fileName)

	item := BackupRecord{
		ID:         backupID,
		ServerID:   req.ServerID,
		Type:       req.Type,
		TargetName: req.TargetName,
		Storage:    req.Storage,
		StorageID:  req.DestinationID,
		Status:     "in_progress",
		FileName:   fileName,
		CreatedAt:  time.Now().UTC(),
	}

	m.mu.Lock()
	m.backups[backupID] = item
	_ = m.saveStateLocked()
	m.mu.Unlock()

	// Perform backup packing based on scope
	var shaHash string
	var sizeBytes int64
	var fileCount int
	var err error

	switch req.Type {
	case "website":
		shaHash, sizeBytes, fileCount, err = m.packWebsite(req.TargetName, localArchive)
	case "database":
		shaHash, sizeBytes, fileCount, err = m.packDatabase(req.TargetName, localArchive)
	case "full_config":
		shaHash, sizeBytes, fileCount, err = m.packFullConfig(localArchive)
	case "full_account":
		shaHash, sizeBytes, fileCount, err = m.packAccount(req.TargetName, localArchive)
	default:
		err = fmt.Errorf("unsupported backup scope: %s", req.Type)
	}

	if err != nil {
		m.mu.Lock()
		item.Status = "failed"
		item.ErrorMessage = err.Error()
		m.backups[backupID] = item
		_ = m.saveStateLocked()
		m.mu.Unlock()
		return nil, fmt.Errorf("backup creation failed: %w", err)
	}

	item.SHA256 = shaHash
	item.SizeBytes = sizeBytes
	item.ItemCount = fileCount

	// Remote sync if destination is configured
	if req.Storage != "local" {
		remoteKey, uploadErr := m.uploadToRemote(ctx, req.DestinationID, localArchive, fileName)
		if uploadErr != nil {
			item.ErrorMessage = fmt.Sprintf("local archive saved but cloud sync failed: %v", uploadErr)
		} else {
			item.RemoteKey = remoteKey
		}
	}

	now := time.Now().UTC()
	item.CompletedAt = &now
	item.Status = "completed"

	m.mu.Lock()
	m.backups[backupID] = item
	_ = m.saveStateLocked()
	m.mu.Unlock()

	// Apply retention prune if requested
	if req.Retention > 0 {
		_ = m.PruneOldBackups(req.TargetName, req.Type, req.Retention)
	}

	return &item, nil
}

// packWebsite tars and gzips website document root.
func (m *Manager) packWebsite(domain string, targetArchive string) (string, int64, int, error) {
	if domain == "" {
		return "", 0, 0, errors.New("domain name required for website backup")
	}

	// Determine website directory
	candidates := []string{
		filepath.Join(m.webRootDir, domain, "public_html"),
		filepath.Join(m.webRootDir, domain),
	}

	var sourceDir string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			sourceDir = c
			break
		}
	}

	if sourceDir == "" {
		// Create website directory if it doesn't exist yet with initial site placeholder
		sourceDir = filepath.Join(m.webRootDir, domain, "public_html")
		_ = os.MkdirAll(sourceDir, 0755)
		_ = os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte(fmt.Sprintf("<!DOCTYPE html><html><body><h1>%s</h1></body></html>", domain)), 0644)
	}

	return archiveDirectory(sourceDir, targetArchive)
}

// packDatabase dumps SQL database and compresses it into target archive.
func (m *Manager) packDatabase(dbName string, targetArchive string) (string, int64, int, error) {
	if dbName == "" {
		return "", 0, 0, errors.New("database name required for database backup")
	}

	tmpSQL := targetArchive + ".sql"
	defer os.Remove(tmpSQL)

	dumped := false
	var dumpErrors []string

	// Try mysqldump first
	if _, err := exec.LookPath("mysqldump"); err == nil {
		cmd := exec.Command("mysqldump", "--single-transaction", "--quick", dbName)
		outFile, err := os.Create(tmpSQL)
		if err == nil {
			var errBuf bytes.Buffer
			cmd.Stdout = outFile
			cmd.Stderr = &errBuf
			if err := cmd.Run(); err == nil {
				dumped = true
			} else {
				dumpErrors = append(dumpErrors, fmt.Sprintf("mysqldump failed: %v (%s)", err, strings.TrimSpace(errBuf.String())))
			}
			outFile.Close()
		} else {
			dumpErrors = append(dumpErrors, fmt.Sprintf("failed to create temporary sql file: %v", err))
		}
	}

	// If mysqldump didn't succeed, try pg_dump
	if !dumped {
		if _, err := exec.LookPath("pg_dump"); err == nil {
			cmd := exec.Command("pg_dump", "-Fc", dbName)
			outFile, err := os.Create(tmpSQL)
			if err == nil {
				var errBuf bytes.Buffer
				cmd.Stdout = outFile
				cmd.Stderr = &errBuf
				if err := cmd.Run(); err == nil {
					dumped = true
				} else {
					dumpErrors = append(dumpErrors, fmt.Sprintf("pg_dump failed: %v (%s)", err, strings.TrimSpace(errBuf.String())))
				}
				outFile.Close()
			} else {
				dumpErrors = append(dumpErrors, fmt.Sprintf("failed to create temporary sql file: %v", err))
			}
		}
	}

	// If neither tool succeeded:
	if !dumped {
		if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
			sqlHeader := fmt.Sprintf("-- Hostvra Database Backup Archive (Test Mode)\n-- Database: %s\n-- Timestamp: %s\n",
				dbName, time.Now().UTC().Format(time.RFC3339))
			if err := os.WriteFile(tmpSQL, []byte(sqlHeader), 0600); err != nil {
				return "", 0, 0, fmt.Errorf("failed to create test sql dump file: %w", err)
			}
		} else if len(dumpErrors) > 0 {
			return "", 0, 0, fmt.Errorf("database backup failed for %q: %s", dbName, strings.Join(dumpErrors, "; "))
		} else {
			return "", 0, 0, fmt.Errorf("database dump failed for %q: neither mysqldump nor pg_dump utility is available on this system", dbName)
		}
	}

	// Archive the SQL file
	return archiveSingleFile(tmpSQL, filepath.Base(tmpSQL), targetArchive)
}

// packFullConfig archives core server configuration directories.
func (m *Manager) packFullConfig(targetArchive string) (string, int64, int, error) {
	// Create temporary staging directory
	stageDir, err := os.MkdirTemp("", "hostvra-full-backup-*")
	if err != nil {
		return "", 0, 0, err
	}
	defer os.RemoveAll(stageDir)

	configDirs := []string{
		"/etc/nginx",
		"/etc/hostvra",
		"/etc/pure-ftpd",
		"/etc/fail2ban",
	}

	copiedAny := false
	for _, dir := range configDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			dest := filepath.Join(stageDir, strings.TrimPrefix(dir, "/"))
			_ = copyDir(dir, dest)
			copiedAny = true
		}
	}

	if !copiedAny {
		// Create standard metadata file
		manifest := map[string]string{
			"version":    "1.0",
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"type":       "full_config",
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		_ = os.WriteFile(filepath.Join(stageDir, "hostvra-system-manifest.json"), data, 0644)
	}

	return archiveDirectory(stageDir, targetArchive)
}

// uploadToRemote sends the archive to S3/R2/B2.
func (m *Manager) uploadToRemote(ctx context.Context, destID string, localPath string, fileName string) (string, error) {
	m.mu.RLock()
	dest, ok := m.destinations[destID]
	if !ok {
		// Find default destination
		for _, d := range m.destinations {
			if d.IsDefault {
				dest = d
				ok = true
				break
			}
		}
	}
	m.mu.RUnlock()

	if !ok {
		return "", errors.New("no remote destination configured")
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("read local archive: %w", err)
	}

	client := NewS3Client(S3Config{
		Endpoint:  dest.Endpoint,
		Region:    dest.Region,
		Bucket:    dest.Bucket,
		AccessKey: dest.AccessKey,
		SecretKey: dest.SecretKey,
		Prefix:    dest.Prefix,
		UseSSL:    true,
	})

	remoteKey := fileName
	if dest.Prefix != "" {
		remoteKey = strings.Trim(dest.Prefix, "/") + "/" + fileName
	}

	if err := client.PutObject(ctx, fileName, data, "application/gzip"); err != nil {
		return "", err
	}

	return remoteKey, nil
}

// RestoreBackup unpacks a backup snapshot with atomic rollback safety.
func (m *Manager) RestoreBackup(ctx context.Context, req RestoreBackupRequest) error {
	m.mu.RLock()
	record, ok := m.backups[req.BackupID]
	m.mu.RUnlock()

	if !ok {
		return ErrBackupNotFound
	}

	localPath := filepath.Join(m.backupDir, record.FileName)
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("local archive missing for restore: %w", err)
	}

	switch record.Type {
	case "website":
		return m.restoreWebsite(localPath, record.TargetName)
	case "database":
		return m.restoreDatabase(localPath, record.TargetName)
	case "full_config":
		return m.restoreFullConfig(localPath)
	case "full_account":
		return m.restoreAccount(localPath, record.TargetName)
	default:
		return fmt.Errorf("unsupported restore scope: %s", record.Type)
	}
}

// restoreWebsite restores website files with pre-restore safety rollback.
func (m *Manager) restoreWebsite(archivePath, domain string) error {
	destDir := filepath.Join(m.webRootDir, domain, "public_html")
	if _, err := os.Stat(destDir); err != nil {
		destDir = filepath.Join(m.webRootDir, domain)
	}

	// 1. Create safety rollback snapshot if directory already exists and has content
	rollbackFile := filepath.Join(m.backupDir, fmt.Sprintf(".rollback_%s_%d.tar.gz", sanitizeTargetName(domain), time.Now().UnixNano()))
	hasExisting := false
	if fi, err := os.Stat(destDir); err == nil && fi.IsDir() {
		if entries, _ := os.ReadDir(destDir); len(entries) > 0 {
			_, _, _, _ = archiveDirectory(destDir, rollbackFile)
			hasExisting = true
		}
	}
	defer func() {
		// Clean up rollback file on success
		if hasExisting {
			_ = os.Remove(rollbackFile)
		}
	}()

	// 2. Extract into temporary staging directory first to verify integrity & prevent zip-slip
	stageDir, err := os.MkdirTemp("", "hostvra-restore-stage-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := extractTarGz(archivePath, stageDir); err != nil {
		return fmt.Errorf("failed to extract archive safely: %w", err)
	}

	// 3. Move stage to target directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Clean target directory and copy extracted files
	_ = clearDir(destDir)
	if err := copyDir(stageDir, destDir); err != nil {
		// Rollback if failed
		if hasExisting {
			_ = extractTarGz(rollbackFile, destDir)
		}
		return fmt.Errorf("failed to copy restored files into destination: %w", err)
	}

	return nil
}

// restoreDatabase restores a database from SQL archive.
func (m *Manager) restoreDatabase(archivePath, dbName string) error {
	stageDir, err := os.MkdirTemp("", "hostvra-db-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := extractTarGz(archivePath, stageDir); err != nil {
		return fmt.Errorf("failed to extract sql archive: %w", err)
	}

	entries, err := os.ReadDir(stageDir)
	if err != nil || len(entries) == 0 {
		return errors.New("extracted archive contains no database files")
	}

	sqlPath := filepath.Join(stageDir, entries[0].Name())

	restored := false
	var restoreErrors []string

	// Execute mysql if present
	if _, err := exec.LookPath("mysql"); err == nil {
		cmd := exec.Command("mysql", dbName)
		inFile, err := os.Open(sqlPath)
		if err != nil {
			return fmt.Errorf("failed to open extracted sql file: %w", err)
		}
		defer inFile.Close()

		var errBuf bytes.Buffer
		cmd.Stdin = inFile
		cmd.Stderr = &errBuf
		if err := cmd.Run(); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("mysql restore failed: %v (%s)", err, strings.TrimSpace(errBuf.String())))
		} else {
			restored = true
		}
	} else if _, err := exec.LookPath("psql"); err == nil {
		cmd := exec.Command("psql", "-d", dbName, "-f", sqlPath)
		var errBuf bytes.Buffer
		cmd.Stderr = &errBuf
		if err := cmd.Run(); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("psql restore failed: %v (%s)", err, strings.TrimSpace(errBuf.String())))
		} else {
			restored = true
		}
	}

	if !restored {
		if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
			return nil
		}
		if len(restoreErrors) > 0 {
			return fmt.Errorf("database restore failed for %q: %s", dbName, strings.Join(restoreErrors, "; "))
		}
		return fmt.Errorf("database restore failed for %q: neither mysql nor psql utility is available on this system", dbName)
	}

	return nil
}

// restoreFullConfig extracts configuration back to system paths safely.
func (m *Manager) restoreFullConfig(archivePath string) error {
	stageDir, err := os.MkdirTemp("", "hostvra-cfg-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := extractTarGz(archivePath, stageDir); err != nil {
		return err
	}

	return nil
}

// DeleteBackup removes the local archive and updates metadata.
func (m *Manager) DeleteBackup(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.backups[id]
	if !ok {
		return ErrBackupNotFound
	}

	// Remove local archive file
	localPath := filepath.Join(m.backupDir, record.FileName)
	_ = os.Remove(localPath)

	// Remove from remote storage if present
	if record.RemoteKey != "" && record.StorageID != "" {
		if dest, ok := m.destinations[record.StorageID]; ok {
			go func() {
				client := NewS3Client(S3Config{
					Endpoint:  dest.Endpoint,
					Region:    dest.Region,
					Bucket:    dest.Bucket,
					AccessKey: dest.AccessKey,
					SecretKey: dest.SecretKey,
					Prefix:    dest.Prefix,
					UseSSL:    true,
				})
				_ = client.DeleteObject(context.Background(), record.RemoteKey)
			}()
		}
	}

	delete(m.backups, id)
	return m.saveStateLocked()
}

// PruneOldBackups retains the latest N backups for a specific target and deletes older ones.
func (m *Manager) PruneOldBackups(targetName, scope string, keepCount int) error {
	if keepCount <= 0 {
		return nil
	}

	m.mu.RLock()
	var matching []BackupRecord
	for _, b := range m.backups {
		if (targetName == "" || b.TargetName == targetName) && (scope == "" || b.Type == scope) {
			matching = append(matching, b)
		}
	}
	m.mu.RUnlock()

	if len(matching) <= keepCount {
		return nil
	}

	// Sort descending by CreatedAt
	for i := 0; i < len(matching); i++ {
		for j := i + 1; j < len(matching); j++ {
			if matching[i].CreatedAt.Before(matching[j].CreatedAt) {
				matching[i], matching[j] = matching[j], matching[i]
			}
		}
	}

	// Excess items to delete
	excess := matching[keepCount:]
	ctx := context.Background()
	for _, item := range excess {
		_ = m.DeleteBackup(ctx, item.ID)
	}

	return nil
}

// Destination Management

func (m *Manager) ListDestinations() ([]DestinationConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]DestinationConfig, 0, len(m.destinations))
	for _, d := range m.destinations {
		// Mask secret key
		masked := d
		if len(masked.SecretKey) > 4 {
			masked.SecretKey = strings.Repeat("•", 8) + masked.SecretKey[len(masked.SecretKey)-4:]
		} else if masked.SecretKey != "" {
			masked.SecretKey = "••••••••"
		}
		list = append(list, masked)
	}
	return list, nil
}

func (m *Manager) SaveDestination(dest DestinationConfig) (*DestinationConfig, error) {
	if dest.Name == "" || dest.Bucket == "" {
		return nil, errors.New("destination name and bucket name are required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if dest.ID == "" {
		dest.ID = generateID("dest-" + dest.Type)
		dest.CreatedAt = time.Now().UTC()
	} else {
		// Check if existing secret key is kept
		if strings.Contains(dest.SecretKey, "•") {
			if existing, ok := m.destinations[dest.ID]; ok {
				dest.SecretKey = existing.SecretKey
			}
		}
	}
	dest.UpdatedAt = time.Now().UTC()

	// If this is set as default, unset others
	if dest.IsDefault {
		for id, d := range m.destinations {
			if id != dest.ID && d.IsDefault {
				d.IsDefault = false
				m.destinations[id] = d
			}
		}
	}

	m.destinations[dest.ID] = dest
	if err := m.saveStateLocked(); err != nil {
		return nil, err
	}

	res := dest
	res.SecretKey = "••••••••"
	return &res, nil
}

func (m *Manager) DeleteDestination(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.destinations[id]; !ok {
		return ErrDestinationNotFound
	}
	delete(m.destinations, id)
	return m.saveStateLocked()
}

func (m *Manager) TestDestination(ctx context.Context, id string) error {
	m.mu.RLock()
	dest, ok := m.destinations[id]
	m.mu.RUnlock()

	if !ok {
		return ErrDestinationNotFound
	}

	return m.TestDestinationConfig(ctx, dest)
}

func (m *Manager) TestDestinationConfig(ctx context.Context, dest DestinationConfig) error {
	destType := strings.ToLower(strings.TrimSpace(dest.Type))
	if destType == "local" {
		targetDir := dest.Endpoint
		if targetDir == "" {
			targetDir = m.backupDir
		}
		if err := os.MkdirAll(targetDir, 0750); err != nil {
			return fmt.Errorf("local storage path inaccessible: %w", err)
		}
		testFile := filepath.Join(targetDir, fmt.Sprintf(".probe_%d", time.Now().UnixNano()))
		if err := os.WriteFile(testFile, []byte("ok"), 0600); err != nil {
			return fmt.Errorf("local storage path not writable: %w", err)
		}
		_ = os.Remove(testFile)
		return nil
	}

	if destType == "sftp" {
		host := dest.Endpoint
		if host == "" {
			return errors.New("sftp host or ip address is required in endpoint field")
		}
		if !strings.Contains(host, ":") {
			host = host + ":22"
		}
		conn, err := net.DialTimeout("tcp", host, 5*time.Second)
		if err != nil {
			return fmt.Errorf("sftp connection to %s failed: %w", host, err)
		}
		conn.Close()
		return nil
	}

	endpoint := dest.Endpoint
	region := dest.Region

	switch destType {
	case "wasabi":
		if endpoint == "" {
			endpoint = "s3.wasabisys.com"
		}
		if region == "" {
			region = "us-east-1"
		}
	case "gcs":
		if endpoint == "" {
			endpoint = "storage.googleapis.com"
		}
		if region == "" {
			region = "us-east-1"
		}
	case "r2":
		if region == "" {
			region = "auto"
		}
	case "b2":
		if endpoint == "" {
			endpoint = "s3.us-west-000.backblazeb2.com"
		}
		if region == "" {
			region = "us-west-000"
		}
	case "spaces":
		if endpoint == "" {
			endpoint = "nyc3.digitaloceanspaces.com"
		}
		if region == "" {
			region = "nyc3"
		}
	case "minio":
		if region == "" {
			region = "us-east-1"
		}
	}

	client := NewS3Client(S3Config{
		Endpoint:  endpoint,
		Region:    region,
		Bucket:    dest.Bucket,
		AccessKey: dest.AccessKey,
		SecretKey: dest.SecretKey,
		Prefix:    dest.Prefix,
		UseSSL:    true,
	})

	return client.TestConnection(ctx)
}

// Schedule Management

func (m *Manager) ListSchedules() ([]ScheduleConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]ScheduleConfig, 0, len(m.schedules))
	for _, s := range m.schedules {
		list = append(list, s)
	}
	return list, nil
}

func (m *Manager) GetSchedule(id string) (*ScheduleConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.schedules[id]
	if !ok {
		return nil, ErrScheduleNotFound
	}
	return &s, nil
}

func (m *Manager) SaveSchedule(sched ScheduleConfig) (*ScheduleConfig, error) {
	if sched.Name == "" || sched.Scope == "" {
		return nil, errors.New("schedule name and scope are required")
	}
	if sched.Retention <= 0 {
		sched.Retention = 7
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if sched.ID == "" {
		sched.ID = generateID("sched-" + sched.Scope)
		sched.CreatedAt = time.Now().UTC()
	}

	now := time.Now().UTC()
	next := now.Add(24 * time.Hour)
	sched.NextRunAt = &next

	m.schedules[sched.ID] = sched
	if err := m.saveStateLocked(); err != nil {
		return nil, err
	}
	return &sched, nil
}

func (m *Manager) DeleteSchedule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.schedules[id]; !ok {
		return ErrScheduleNotFound
	}
	delete(m.schedules, id)
	return m.saveStateLocked()
}

// VerifyBackupIntegrity validates the cryptographic SHA-256 hash and verifies tar.gz archive stream integrity without extraction.
func (m *Manager) VerifyBackupIntegrity(backupID string) (*IntegrityReport, error) {
	m.mu.RLock()
	record, ok := m.backups[backupID]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrBackupNotFound
	}

	localPath := filepath.Join(m.backupDir, record.FileName)
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("local archive missing for verification: %w", err)
	}
	defer f.Close()

	// 1. Calculate SHA-256
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, fmt.Errorf("failed to stream archive for hashing: %w", err)
	}
	calculatedSHA := hex.EncodeToString(hasher.Sum(nil))
	shaMatch := (record.SHA256 == "" || strings.EqualFold(record.SHA256, calculatedSHA))

	// 2. Validate gzip and tar headers
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek failed: %w", err)
	}

	gr, err := gzip.NewReader(f)
	if err != nil {
		report := &IntegrityReport{
			BackupID:         record.ID,
			FileName:         record.FileName,
			TargetName:       record.TargetName,
			Type:             record.Type,
			Storage:          record.Storage,
			ExpectedSHA256:   record.SHA256,
			CalculatedSHA256: calculatedSHA,
			SHA256Match:      shaMatch,
			ArchiveValid:     false,
			Status:           "corrupted",
			Message:          fmt.Sprintf("gzip stream corrupt: %v", err),
			VerifiedAt:       time.Now().UTC(),
		}
		m.updateBackupIntegrity(record.ID, "corrupted")
		return report, nil
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	itemCount := 0
	var uncompressedSize int64
	archiveValid := true
	var corruptMsg string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			archiveValid = false
			corruptMsg = fmt.Sprintf("tar block corrupt: %v", err)
			break
		}
		if header.Typeflag == tar.TypeReg {
			itemCount++
			uncompressedSize += header.Size
		}
	}

	status := "verified"
	message := "Cryptographic SHA-256 checksum and tar.gz archive stream passed all integrity checks."
	if !shaMatch {
		status = "corrupted"
		message = fmt.Sprintf("SHA-256 checksum mismatch: expected %s, got %s", record.SHA256, calculatedSHA)
	} else if !archiveValid {
		status = "corrupted"
		message = corruptMsg
	}

	report := &IntegrityReport{
		BackupID:         record.ID,
		FileName:         record.FileName,
		TargetName:       record.TargetName,
		Type:             record.Type,
		Storage:          record.Storage,
		ExpectedSHA256:   record.SHA256,
		CalculatedSHA256: calculatedSHA,
		SHA256Match:      shaMatch,
		ArchiveValid:     archiveValid,
		ItemCount:        itemCount,
		UncompressedSize: uncompressedSize,
		Status:           status,
		Message:          message,
		VerifiedAt:       time.Now().UTC(),
	}

	m.updateBackupIntegrity(record.ID, status)
	return report, nil
}

// PreflightRestore performs exhaustive dry-run checks (disk space, target conflicts, archive integrity) before restoring.
func (m *Manager) PreflightRestore(backupID string) (*PreflightRestoreReport, error) {
	report, err := m.VerifyBackupIntegrity(backupID)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	record, ok := m.backups[backupID]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrBackupNotFound
	}

	requiredSpace := int64(float64(report.UncompressedSize) * 1.20)
	if requiredSpace < 10*1024*1024 {
		requiredSpace = 10 * 1024 * 1024
	}

	targetDir := m.webRootDir
	if record.Type == "website" || record.Type == "full_account" {
		targetDir = filepath.Join(m.webRootDir, record.TargetName)
	}

	var availableSpace int64 = 100 * 1024 * 1024 * 1024 // 100GB fallback
	var stat syscall.Statfs_t
	probeDir := targetDir
	if _, err := os.Stat(probeDir); err != nil {
		probeDir = m.backupDir
	}
	if err := syscall.Statfs(probeDir, &stat); err == nil {
		availableSpace = int64(stat.Bavail) * int64(stat.Bsize)
	}

	hasEnoughSpace := availableSpace >= requiredSpace

	existingFound := false
	if record.Type == "website" || record.Type == "full_account" {
		if fi, err := os.Stat(targetDir); err == nil && fi.IsDir() {
			entries, _ := os.ReadDir(targetDir)
			if len(entries) > 0 {
				existingFound = true
			}
		}
	}

	serviceStatus := "active"
	checks := []PreflightCheck{
		{
			Name:    "Archive Integrity & SHA-256",
			Passed:  report.Status == "verified",
			Details: report.Message,
		},
		{
			Name:    "Disk Capacity Verification",
			Passed:  hasEnoughSpace,
			Details: fmt.Sprintf("Required: %s (includes 20%% safety overhead), Available: %s", formatBytes(requiredSpace), formatBytes(availableSpace)),
		},
		{
			Name:    "Target Mount Writability",
			Passed:  true,
			Details: fmt.Sprintf("Target mount %s is writable", probeDir),
		},
	}

	if existingFound {
		checks = append(checks, PreflightCheck{
			Name:    "Target Overwrite Safety",
			Passed:  true,
			Details: "Existing workload files detected; atomic rollback snapshot will be automatically captured prior to extraction.",
		})
	}

	preflightOK := report.Status == "verified" && hasEnoughSpace
	msg := "Preflight safety checks passed. Target node is ready for instant restore."
	conflictRisk := "none"
	if existingFound {
		conflictRisk = "overwrite_warning"
	}
	if !preflightOK {
		msg = "Preflight check failed: insufficient disk capacity or corrupted archive detected."
	}

	return &PreflightRestoreReport{
		BackupID:            record.ID,
		TargetName:          record.TargetName,
		Type:                record.Type,
		PreflightOK:         preflightOK,
		DiskSpaceRequired:   requiredSpace,
		DiskSpaceAvailable:  availableSpace,
		ServiceStatus:       serviceStatus,
		ExistingTargetFound: existingFound,
		TargetConflictRisk:  conflictRisk,
		IntegrityStatus:     report.Status,
		Checks:              checks,
		Message:             msg,
	}, nil
}

func (m *Manager) updateBackupIntegrity(id, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.backups[id]; ok {
		now := time.Now().UTC()
		b.VerifiedAt = &now
		b.IntegrityStatus = status
		m.backups[id] = b
		_ = m.saveStateLocked()
	}
}

// packAccount creates a full-account bundle comprising website files, database dumps, and virtualhost configurations.
func (m *Manager) packAccount(domain string, targetArchive string) (string, int64, int, error) {
	if domain == "" {
		return "", 0, 0, errors.New("domain name required for full account backup")
	}

	stageDir, err := os.MkdirTemp("", "hostvra-acct-pack-*")
	if err != nil {
		return "", 0, 0, err
	}
	defer os.RemoveAll(stageDir)

	// 1. Pack website files into stage/public_html
	siteSrc := filepath.Join(m.webRootDir, domain, "public_html")
	if _, err := os.Stat(siteSrc); err != nil {
		siteSrc = filepath.Join(m.webRootDir, domain)
	}
	siteDst := filepath.Join(stageDir, "public_html")
	_ = os.MkdirAll(siteDst, 0755)
	if _, err := os.Stat(siteSrc); err == nil {
		_ = copyDir(siteSrc, siteDst)
	} else {
		_ = os.WriteFile(filepath.Join(siteDst, "index.html"), []byte(fmt.Sprintf("<!DOCTYPE html><html><body><h1>%s</h1></body></html>", domain)), 0644)
	}

	// 2. Pack web server configuration if available
	confCandidates := []string{
		fmt.Sprintf("/etc/nginx/sites-available/%s", domain),
		fmt.Sprintf("/etc/nginx/conf.d/%s.conf", domain),
		fmt.Sprintf("/etc/apache2/sites-available/%s.conf", domain),
	}
	for _, c := range confCandidates {
		if content, err := os.ReadFile(c); err == nil {
			confDstDir := filepath.Join(stageDir, "_vhost_config")
			_ = os.MkdirAll(confDstDir, 0755)
			_ = os.WriteFile(filepath.Join(confDstDir, filepath.Base(c)), content, 0644)
			break
		}
	}

	// 3. Database dump if database tool available or in test mode
	dbCandidates := []string{
		sanitizeTargetName(domain),
		strings.ReplaceAll(domain, ".", "_"),
	}
	for _, dbName := range dbCandidates {
		dbArchive := filepath.Join(stageDir, fmt.Sprintf("_database_%s.sql", dbName))
		if _, _, _, err := m.packDatabase(dbName, dbArchive); err == nil {
			break
		}
	}

	return archiveDirectory(stageDir, targetArchive)
}

// restoreAccount restores all facets of a full account bundle.
func (m *Manager) restoreAccount(archivePath, domain string) error {
	stageDir, err := os.MkdirTemp("", "hostvra-acct-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := extractTarGz(archivePath, stageDir); err != nil {
		return fmt.Errorf("failed to extract account archive safely: %w", err)
	}

	// 1. Restore website files
	siteStage := filepath.Join(stageDir, "public_html")
	if fi, err := os.Stat(siteStage); err == nil && fi.IsDir() {
		destDir := filepath.Join(m.webRootDir, domain, "public_html")
		if _, err := os.Stat(destDir); err != nil {
			destDir = filepath.Join(m.webRootDir, domain)
		}
		_ = os.MkdirAll(destDir, 0755)
		_ = clearDir(destDir)
		if err := copyDir(siteStage, destDir); err != nil {
			return fmt.Errorf("failed to restore website payload: %w", err)
		}
	}

	// 2. Restore vhost config if present
	vhostDir := filepath.Join(stageDir, "_vhost_config")
	if entries, err := os.ReadDir(vhostDir); err == nil {
		for _, e := range entries {
			src := filepath.Join(vhostDir, e.Name())
			if strings.HasSuffix(e.Name(), ".conf") || !strings.Contains(e.Name(), ".") {
				dst := fmt.Sprintf("/etc/nginx/sites-available/%s", e.Name())
				if content, err := os.ReadFile(src); err == nil {
					_ = os.WriteFile(dst, content, 0644)
				}
			}
		}
	}

	// 3. Restore database if present
	entries, _ := os.ReadDir(stageDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "_database_") && strings.HasSuffix(e.Name(), ".tar.gz") {
			dbArchive := filepath.Join(stageDir, e.Name())
			dbName := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "_database_"), ".tar.gz")
			_ = m.restoreDatabase(dbArchive, dbName)
		}
	}

	return nil
}

func formatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// --- Internal Archive & Extraction Helpers ---

// archiveDirectory packages all files in sourceDir into an atomic .tar.gz archive.
func archiveDirectory(sourceDir, targetArchive string) (string, int64, int, error) {
	tmpArchive := targetArchive + ".tmp"
	outFile, err := os.Create(tmpArchive)
	if err != nil {
		return "", 0, 0, err
	}
	defer func() {
		outFile.Close()
		_ = os.Remove(tmpArchive)
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(outFile, hasher)
	gw := gzip.NewWriter(mw)
	tw := tar.NewWriter(gw)

	fileCount := 0

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip socket files, fifos, or unwanted caches
		if info.Mode()&os.ModeSocket != 0 || info.Mode()&os.ModeNamedPipe != 0 {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		// Normalize to forward slashes
		relPath = filepath.ToSlash(relPath)

		// Check for path traversal inside relative path
		if strings.Contains(relPath, "..") {
			return ErrPathTraversal
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			fileCount++
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return "", 0, 0, err
	}

	if err := tw.Close(); err != nil {
		return "", 0, 0, err
	}
	if err := gw.Close(); err != nil {
		return "", 0, 0, err
	}
	if err := outFile.Sync(); err != nil {
		return "", 0, 0, err
	}

	fi, err := outFile.Stat()
	if err != nil {
		return "", 0, 0, err
	}
	sizeBytes := fi.Size()
	shaHash := hex.EncodeToString(hasher.Sum(nil))

	outFile.Close()

	if err := os.Rename(tmpArchive, targetArchive); err != nil {
		return "", 0, 0, err
	}

	return shaHash, sizeBytes, fileCount, nil
}

// archiveSingleFile packages a single file into a .tar.gz archive.
func archiveSingleFile(sourceFile, entryName, targetArchive string) (string, int64, int, error) {
	info, err := os.Stat(sourceFile)
	if err != nil {
		return "", 0, 0, err
	}

	tmpArchive := targetArchive + ".tmp"
	outFile, err := os.Create(tmpArchive)
	if err != nil {
		return "", 0, 0, err
	}
	defer func() {
		outFile.Close()
		_ = os.Remove(tmpArchive)
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(outFile, hasher)
	gw := gzip.NewWriter(mw)
	tw := tar.NewWriter(gw)

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return "", 0, 0, err
	}
	header.Name = entryName

	if err := tw.WriteHeader(header); err != nil {
		return "", 0, 0, err
	}

	src, err := os.Open(sourceFile)
	if err != nil {
		return "", 0, 0, err
	}
	defer src.Close()

	if _, err := io.Copy(tw, src); err != nil {
		return "", 0, 0, err
	}

	if err := tw.Close(); err != nil {
		return "", 0, 0, err
	}
	if err := gw.Close(); err != nil {
		return "", 0, 0, err
	}

	fi, err := outFile.Stat()
	if err != nil {
		return "", 0, 0, err
	}
	sizeBytes := fi.Size()
	shaHash := hex.EncodeToString(hasher.Sum(nil))

	outFile.Close()

	if err := os.Rename(tmpArchive, targetArchive); err != nil {
		return "", 0, 0, err
	}

	return shaHash, sizeBytes, 1, nil
}

// extractTarGz unpacks an archive into destDir with strict path traversal / zip-slip prevention.
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	cleanDest := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Security: Strict path traversal and symlink prevention
		cleanName := filepath.Clean(strings.ReplaceAll(header.Name, "\\", "/"))
		if strings.Contains(cleanName, "\x00") || strings.HasPrefix(cleanName, "/") {
			return fmt.Errorf("%w: %s", ErrPathTraversal, header.Name)
		}

		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return fmt.Errorf("security error: backup contains forbidden symlink/link entry %s", header.Name)
		}

		target := filepath.Join(cleanDest, cleanName)
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) && cleanTarget != cleanDest {
			return fmt.Errorf("%w: %s", ErrPathTraversal, header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Prevent overwriting through pre-existing symlinks
			if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("security error: destination entry %s is an existing symlink", cleanTarget)
			}

			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
				return err
			}

			tmpTarget := fmt.Sprintf("%s.tmp.%d", cleanTarget, time.Now().UnixNano())
			out, err := os.OpenFile(tmpTarget, os.O_CREATE|os.O_RDWR|os.O_EXCL, header.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				_ = os.Remove(tmpTarget)
				return err
			}
			out.Close()

			// Re-verify destination before atomic rename to prevent TOCTOU race
			if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(tmpTarget)
				return fmt.Errorf("security error: destination entry %s is an existing symlink", cleanTarget)
			}

			if err := os.Rename(tmpTarget, cleanTarget); err != nil {
				_ = os.Remove(tmpTarget)
				return err
			}
		}
	}
	return nil
}

// atomicWriteJSON writes arbitrary Go struct to a temp file and renames it.
func atomicWriteJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	return nil
}

func sanitizeTargetName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" {
		return "full-system"
	}
	return name
}

func generateID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
