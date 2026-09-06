package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrPathTraversal = errors.New("archive contains unsafe relative path traversal")
	ErrCorruptArchive = errors.New("archive is corrupted or unreadable")
	ErrInvalidTarget = errors.New("target path is outside allowed root")
)

type BackupType string

const (
	TypeWebsite    BackupType = "website"
	TypeDatabase   BackupType = "database"
	TypeFullConfig BackupType = "full_config"
	TypeMailbox    BackupType = "mailbox"
)

type StorageType string

const (
	StorageLocal StorageType = "local"
	StorageS3    StorageType = "s3"
)

type BackupOptions struct {
	Type        BackupType
	SourcePath  string
	TargetDir   string
	Exclusions  []string
	DBType      string
	DBName      string
}

type BackupMetadata struct {
	ID           string     `json:"id"`
	Type         BackupType `json:"type"`
	SourcePath   string     `json:"source_path"`
	ArchivePath  string     `json:"archive_path"`
	FileName     string     `json:"file_name"`
	SizeBytes    int64      `json:"size_bytes"`
	SHA256       string     `json:"sha256"`
	FileCount    int        `json:"file_count"`
	CreatedAt    time.Time  `json:"created_at"`
	Status       string     `json:"status"` // completed, failed
	ErrorMessage string     `json:"error_message,omitempty"`
}

type ArchiveReport struct {
	FileCount        int      `json:"file_count"`
	UncompressedSize int64    `json:"uncompressed_size"`
	Checksum         string   `json:"checksum"`
	Files            []string `json:"files"`
	Valid            bool     `json:"valid"`
}

type RestoreResult struct {
	BackupID       string    `json:"backup_id"`
	TargetDir      string    `json:"target_dir"`
	RestoredFiles  int       `json:"restored_files"`
	RestoredAt     time.Time `json:"restored_at"`
	RolledBack     bool      `json:"rolled_back"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

type Manager struct {
	mu           sync.RWMutex
	backupRoot   string
	defaultExcl  []string
	history      []*BackupMetadata
}

func NewManager(backupRoot string) (*Manager, error) {
	if err := os.MkdirAll(backupRoot, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup root: %w", err)
	}

	return &Manager{
		backupRoot: backupRoot,
		defaultExcl: []string{
			".git",
			"node_modules",
			"cache",
			"tmp",
			".DS_Store",
			"__pycache__",
		},
		history: make([]*BackupMetadata, 0),
	}, nil
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// CreateWebsiteBackup archives a website's document root with gzip compression
func (m *Manager) CreateWebsiteBackup(siteDomain, docRoot string, extraExclusions ...string) (*BackupMetadata, error) {
	if _, err := os.Stat(docRoot); err != nil {
		return nil, fmt.Errorf("document root does not exist: %w", err)
	}

	backupID := newUUID()
	timestamp := time.Now().UTC().Format("20060102-150405")
	cleanDomain := strings.ReplaceAll(siteDomain, "/", "_")
	fileName := fmt.Sprintf("web-%s-%s.tar.gz", cleanDomain, timestamp)
	destPath := filepath.Join(m.backupRoot, fileName)

	meta := &BackupMetadata{
		ID:          backupID,
		Type:        TypeWebsite,
		SourcePath:  docRoot,
		ArchivePath: destPath,
		FileName:    fileName,
		CreatedAt:   time.Now().UTC(),
		Status:      "in_progress",
	}

	allExclusions := append(m.defaultExcl, extraExclusions...)

	fileCount, sizeBytes, checksum, err := m.archiveDirectory(docRoot, destPath, allExclusions)
	if err != nil {
		meta.Status = "failed"
		meta.ErrorMessage = err.Error()
		os.Remove(destPath)
		return meta, err
	}

	meta.Status = "completed"
	meta.FileCount = fileCount
	meta.SizeBytes = sizeBytes
	meta.SHA256 = checksum

	m.mu.Lock()
	m.history = append(m.history, meta)
	m.mu.Unlock()

	return meta, nil
}

// CreateMailboxBackup archives a specific virtual mailbox Maildir with checksum
func (m *Manager) CreateMailboxBackup(domain, localPart, mailDirBase string) (*BackupMetadata, error) {
	if mailDirBase == "" {
		mailDirBase = "/var/mail/vhosts"
	}
	mailboxDir := filepath.Join(mailDirBase, domain, localPart)
	if _, err := os.Stat(mailboxDir); err != nil {
		return nil, fmt.Errorf("mailbox directory does not exist: %w", err)
	}

	backupID := newUUID()
	timestamp := time.Now().UTC().Format("20060102-150405")
	cleanEmail := fmt.Sprintf("%s_%s", localPart, strings.ReplaceAll(domain, "/", "_"))
	fileName := fmt.Sprintf("mail-%s-%s.tar.gz", cleanEmail, timestamp)
	destPath := filepath.Join(m.backupRoot, fileName)

	meta := &BackupMetadata{
		ID:          backupID,
		Type:        TypeMailbox,
		SourcePath:  mailboxDir,
		ArchivePath: destPath,
		FileName:    fileName,
		CreatedAt:   time.Now().UTC(),
		Status:      "in_progress",
	}

	fileCount, sizeBytes, checksum, err := m.archiveDirectory(mailboxDir, destPath, m.defaultExcl)
	if err != nil {
		meta.Status = "failed"
		meta.ErrorMessage = err.Error()
		return meta, err
	}

	meta.Status = "completed"
	meta.FileCount = fileCount
	meta.SizeBytes = sizeBytes
	meta.SHA256 = checksum

	m.mu.Lock()
	m.history = append(m.history, meta)
	m.mu.Unlock()

	return meta, nil
}

// CreateConfigBackup archives Nginx configurations, vhosts, and Hostvra metadata
func (m *Manager) CreateConfigBackup(configDirs []string) (*BackupMetadata, error) {
	backupID := newUUID()
	timestamp := time.Now().UTC().Format("20060102-150405")
	fileName := fmt.Sprintf("config-%s.tar.gz", timestamp)
	destPath := filepath.Join(m.backupRoot, fileName)

	meta := &BackupMetadata{
		ID:          backupID,
		Type:        TypeFullConfig,
		SourcePath:  strings.Join(configDirs, ","),
		ArchivePath: destPath,
		FileName:    fileName,
		CreatedAt:   time.Now().UTC(),
		Status:      "in_progress",
	}

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	hash := sha256.New()
	multiWriter := io.MultiWriter(outFile, hash)

	gw := gzip.NewWriter(multiWriter)
	tw := tar.NewWriter(gw)

	fileCount := 0

	for _, dir := range configDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		basePrefix := filepath.Base(dir)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if relPath == "." {
				return nil
			}

			archivePath := filepath.ToSlash(filepath.Join(basePrefix, relPath))

			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = archivePath

			if err := tw.WriteHeader(hdr); err != nil {
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
			gw.Close()
			tw.Close()
			os.Remove(destPath)
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	stat, err := os.Stat(destPath)
	if err != nil {
		return nil, err
	}

	meta.Status = "completed"
	meta.FileCount = fileCount
	meta.SizeBytes = stat.Size()
	meta.SHA256 = hex.EncodeToString(hash.Sum(nil))

	m.mu.Lock()
	m.history = append(m.history, meta)
	m.mu.Unlock()

	return meta, nil
}

// VerifyArchive inspects an archive's gzip integrity and verifies no tar traversal attacks
func (m *Manager) VerifyArchive(archivePath string) (*ArchiveReport, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	hash := sha256.New()
	trR := io.TeeReader(f, hash)

	gr, err := gzip.NewReader(trR)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptArchive, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	report := &ArchiveReport{
		Files: make([]string, 0),
		Valid: true,
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCorruptArchive, err)
		}

		// Check for malicious paths
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "../") || cleanName == ".." || filepath.IsAbs(cleanName) {
			report.Valid = false
			return report, fmt.Errorf("%w: %s", ErrPathTraversal, hdr.Name)
		}

		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			report.FileCount++
		}
		report.UncompressedSize += hdr.Size
		report.Files = append(report.Files, hdr.Name)
	}

	report.Checksum = hex.EncodeToString(hash.Sum(nil))
	return report, nil
}

// RestoreArchive safely extracts an archive with automatic atomic rollback on failure
func (m *Manager) RestoreArchive(archivePath, targetDir, rollbackDir string) (*RestoreResult, error) {
	// Step 1: Pre-restore verification
	report, err := m.VerifyArchive(archivePath)
	if err != nil {
		return nil, fmt.Errorf("archive verification failed: %w", err)
	}
	if !report.Valid {
		return nil, errors.New("archive is invalid or unsafe")
	}

	res := &RestoreResult{
		TargetDir:  targetDir,
		RestoredAt: time.Now().UTC(),
	}

	// Step 2: Backup current target directory to rollback location if target exists
	targetExists := false
	if _, err := os.Stat(targetDir); err == nil {
		targetExists = true
		if err := os.MkdirAll(rollbackDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create rollback staging directory: %w", err)
		}

		rollbackTarget := filepath.Join(rollbackDir, fmt.Sprintf("target-backup-%d", time.Now().UnixNano()))
		if err := copyDirectory(targetDir, rollbackTarget); err != nil {
			return nil, fmt.Errorf("failed to create pre-restore snapshot: %w", err)
		}

		// Register rollback helper
		defer func() {
			if res.RolledBack {
				_ = os.RemoveAll(targetDir)
				_ = copyDirectory(rollbackTarget, targetDir)
			}
		}()
	}

	// Step 3: Extract archive into staging dir first
	stagingDir := filepath.Join(filepath.Dir(targetDir), fmt.Sprintf(".restore-stage-%s", newUUID()[:8]))
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	restoredCount := 0

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			res.RolledBack = targetExists
			res.ErrorMessage = fmt.Sprintf("tar read error: %v", err)
			return res, err
		}

		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "../") || cleanName == ".." || filepath.IsAbs(cleanName) {
			res.RolledBack = targetExists
			res.ErrorMessage = "path traversal detected during extraction"
			return res, ErrPathTraversal
		}

		extractDest := filepath.Join(stagingDir, cleanName)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(extractDest, 0755); err != nil {
				res.RolledBack = targetExists
				res.ErrorMessage = err.Error()
				return res, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(extractDest), 0755); err != nil {
				res.RolledBack = targetExists
				res.ErrorMessage = err.Error()
				return res, err
			}

			outFile, err := os.OpenFile(extractDest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode&0777))
			if err != nil {
				res.RolledBack = targetExists
				res.ErrorMessage = err.Error()
				return res, err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				res.RolledBack = targetExists
				res.ErrorMessage = err.Error()
				return res, err
			}
			outFile.Close()
			restoredCount++
		}
	}

	// Step 4: Atomically promote staged files to target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		res.RolledBack = targetExists
		res.ErrorMessage = err.Error()
		return res, err
	}

	// Copy/move from staging to target
	if err := copyDirectory(stagingDir, targetDir); err != nil {
		res.RolledBack = targetExists
		res.ErrorMessage = err.Error()
		return res, err
	}

	res.RestoredFiles = restoredCount
	return res, nil
}

// ListBackups returns all created backups
func (m *Manager) ListBackups() []*BackupMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*BackupMetadata, len(m.history))
	copy(out, m.history)
	return out
}

// Internal directory archiving
func (m *Manager) archiveDirectory(srcDir, destArchive string, exclusions []string) (int, int64, string, error) {
	outFile, err := os.OpenFile(destArchive, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return 0, 0, "", err
	}
	defer outFile.Close()

	hash := sha256.New()
	mw := io.MultiWriter(outFile, hash)

	gw := gzip.NewWriter(mw)
	tw := tar.NewWriter(gw)

	fileCount := 0

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Check exclusion patterns
		for _, excl := range exclusions {
			if strings.Contains(relPath, excl) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(hdr); err != nil {
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
		gw.Close()
		tw.Close()
		return 0, 0, "", err
	}

	if err := tw.Close(); err != nil {
		return 0, 0, "", err
	}
	if err := gw.Close(); err != nil {
		return 0, 0, "", err
	}

	stat, err := os.Stat(destArchive)
	if err != nil {
		return 0, 0, "", err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return fileCount, stat.Size(), checksum, nil
}

func copyDirectory(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		if info.Mode().IsRegular() {
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			srcF, err := os.Open(path)
			if err != nil {
				return err
			}
			defer srcF.Close()

			dstF, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
			if err != nil {
				return err
			}
			defer dstF.Close()

			if _, err := io.Copy(dstF, srcF); err != nil {
				return err
			}
		}

		return nil
	})
}
