package incremental

import (
	"context"
	"errors"
	"time"
)

var (
	ErrProviderNotSupported = errors.New("backup provider not supported")
	ErrRepoNotInitialized   = errors.New("backup repository is not initialized")
	ErrSnapshotNotFound     = errors.New("incremental snapshot not found")
	ErrRestorePathInvalid   = errors.New("restore destination path is invalid")
)

// RepoConfig holds connection and encryption details for a repository
type RepoConfig struct {
	ID                  string            `json:"id"`
	Provider            string            `json:"provider"` // "restic", "borg", "native"
	RepoURI             string            `json:"repo_uri"` // e.g. "s3:https://s3.us-east-1.amazonaws.com/bucket/path" or "/var/backups/restic"
	Password            string            `json:"password"`
	EncryptionAlgorithm string            `json:"encryption_algorithm"` // "AES-256", "ChaCha20"
	EnvVars             map[string]string `json:"env_vars,omitempty"`
}

// SnapshotRequest defines targets and pre-backup snapshot parameters
type SnapshotRequest struct {
	RepoConfig                  RepoConfig        `json:"repo_config"`
	TargetName                  string            `json:"target_name"`
	Scope                       string            `json:"scope"` // "website", "database", "mail", "full_account"
	IncludePaths                []string          `json:"include_paths"`
	ExcludePaths                []string          `json:"exclude_paths,omitempty"`
	DatabaseType                string            `json:"database_type,omitempty"` // "mysql", "postgres", "none"
	DatabaseName                string            `json:"database_name,omitempty"`
	DatabaseUser                string            `json:"database_user,omitempty"`
	DatabasePassword            string            `json:"database_password,omitempty"`
	Tags                        []string          `json:"tags,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

// SnapshotResult documents the outcome of an incremental backup
type SnapshotResult struct {
	SnapshotID                 string    `json:"snapshot_id"`
	ShortHash                  string    `json:"short_hash"`
	ParentSnapshotID           string    `json:"parent_snapshot_id,omitempty"`
	FilesNew                   int       `json:"files_new"`
	FilesChanged               int       `json:"files_changed"`
	FilesUnmodified            int       `json:"files_unmodified"`
	TotalFiles                 int       `json:"total_files"`
	RawSizeBytes               int64     `json:"raw_size_bytes"`
	StoredSizeBytes            int64     `json:"stored_size_bytes"`
	DeduplicationRatio         float64   `json:"deduplication_ratio"`
	DatabaseSnapshotConsistent bool      `json:"database_snapshot_consistent"`
	Success                    bool      `json:"success"`
	ErrorMessage               string    `json:"error_message,omitempty"`
	StartedAt                  time.Time `json:"started_at"`
	CompletedAt                time.Time `json:"completed_at"`
	DurationSeconds            float64   `json:"duration_seconds"`
}

// SnapshotInfo provides summary of an existing snapshot in repository
type SnapshotInfo struct {
	ID        string    `json:"id"`
	ShortID   string    `json:"short_id"`
	Time      time.Time `json:"time"`
	Hostname  string    `json:"hostname"`
	Username  string    `json:"username"`
	Paths     []string  `json:"paths"`
	Tags      []string  `json:"tags"`
	SizeBytes int64     `json:"size_bytes"`
}

// FileNode represents an entry in a snapshot's file tree (for restore previews)
type FileNode struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"` // "file", "dir", "symlink"
	SizeBytes  int64     `json:"size_bytes"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
}

// RestoreFilesRequest specifies selective file restoration
type RestoreFilesRequest struct {
	RepoConfig      RepoConfig `json:"repo_config"`
	SnapshotID      string     `json:"snapshot_id"`
	SelectedPaths   []string   `json:"selected_paths"`
	TargetDirectory string     `json:"target_directory"`
	RestoreMode     string     `json:"restore_mode"` // "in_place", "alternate_dir"
}

// RestoreFullAccountRequest specifies comprehensive tenant workload restoration
type RestoreFullAccountRequest struct {
	RepoConfig          RepoConfig `json:"repo_config"`
	SnapshotID          string     `json:"snapshot_id"`
	TargetDirectory     string     `json:"target_directory"`
	RestoreDatabase     bool       `json:"restore_database"`
	DatabaseName        string     `json:"database_name,omitempty"`
	DatabaseUser        string     `json:"database_user,omitempty"`
	DatabasePassword    string     `json:"database_password,omitempty"`
	RestoreMailboxes    bool       `json:"restore_mailboxes"`
	MaildirTargetBase   string     `json:"maildir_target_base,omitempty"`
	IsolatedEnvironment bool       `json:"isolated_environment"` // for disaster recovery drills
}

// RestoreResult returns the result of a restore operation
type RestoreResult struct {
	Success          bool      `json:"success"`
	FilesRestored    int       `json:"files_restored"`
	BytesRestored    int64     `json:"bytes_restored"`
	DatabaseRestored bool      `json:"database_restored"`
	MailboxesRestored bool     `json:"mailboxes_restored"`
	TargetDirectory  string    `json:"target_directory"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	CompletedAt      time.Time `json:"completed_at"`
}

// IntegrityCheckResult reports repository health and index consistency
type IntegrityCheckResult struct {
	Healthy           bool      `json:"healthy"`
	TotalSnapshots    int       `json:"total_snapshots"`
	TotalPacks        int       `json:"total_packs"`
	TotalSizeBytes    int64     `json:"total_size_bytes"`
	ErrorsFound       []string  `json:"errors_found,omitempty"`
	DurationSeconds   float64   `json:"duration_seconds"`
	CheckedAt         time.Time `json:"checked_at"`
}

// RetentionPolicy defines automated snapshot thinning
type RetentionPolicy struct {
	KeepLast    int  `json:"keep_last"`
	KeepHourly  int  `json:"keep_hourly"`
	KeepDaily   int  `json:"keep_daily"`
	KeepWeekly  int  `json:"keep_weekly"`
	KeepMonthly int  `json:"keep_monthly"`
	KeepYearly  int  `json:"keep_yearly"`
	DryRun      bool `json:"dry_run"`
}

// PruneResult details the snapshots and bytes removed by retention
type PruneResult struct {
	SnapshotsKept     int       `json:"snapshots_kept"`
	SnapshotsRemoved  int       `json:"snapshots_removed"`
	BytesReclaimed    int64     `json:"bytes_reclaimed"`
	Success           bool      `json:"success"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	CompletedAt       time.Time `json:"completed_at"`
}

// DestinationTestConfig defines remote destination credentials to test
type DestinationTestConfig struct {
	Type      string `json:"type"` // "s3", "r2", "b2", "wasabi", "gcs", "local", "sftp"
	Endpoint  string `json:"endpoint,omitempty"`
	Region    string `json:"region,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
}

// DestinationTestResult verifies each phase of off-site backup functionality
type DestinationTestResult struct {
	Success          bool      `json:"success"`
	AuthPassed       bool      `json:"auth_passed"`
	UploadPassed     bool      `json:"upload_passed"`
	ListingPassed    bool      `json:"listing_passed"`
	RestorePassed    bool      `json:"restore_passed"`
	DeletionPassed   bool      `json:"deletion_passed"`
	LatencyMs        int64     `json:"latency_ms"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	TestedAt         time.Time `json:"tested_at"`
}

// BackupProvider defines the unified provider interface for Restic, Borg, and Native engines
type BackupProvider interface {
	Name() string
	InitRepository(ctx context.Context, cfg RepoConfig) error
	CreateSnapshot(ctx context.Context, req SnapshotRequest) (*SnapshotResult, error)
	ListSnapshots(ctx context.Context, cfg RepoConfig) ([]SnapshotInfo, error)
	ListSnapshotFiles(ctx context.Context, cfg RepoConfig, snapshotID, subPath string) ([]FileNode, error)
	RestoreFiles(ctx context.Context, req RestoreFilesRequest) (*RestoreResult, error)
	RestoreFullAccount(ctx context.Context, req RestoreFullAccountRequest) (*RestoreResult, error)
	CheckIntegrity(ctx context.Context, cfg RepoConfig) (*IntegrityCheckResult, error)
	Prune(ctx context.Context, cfg RepoConfig, policy RetentionPolicy) (*PruneResult, error)
	TestDestination(ctx context.Context, dest DestinationTestConfig) (*DestinationTestResult, error)
}

// NewIncrementalProvider returns the appropriate BackupProvider implementation based on name
func NewIncrementalProvider(providerName string) BackupProvider {
	switch providerName {
	case "restic":
		return NewResticProvider()
	case "native", "borg":
		fallthrough
	default:
		return NewNativeProvider()
	}
}
