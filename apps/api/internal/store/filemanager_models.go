package store

import (
	"time"

	"github.com/google/uuid"
)

// FileManagerFavorite represents a pinned/starred folder
type FileManagerFavorite struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Domain    string     `json:"domain"`
	Path      string     `json:"path"`
	Name      string     `json:"name"`
	Color     string     `json:"color"`
	CreatedAt time.Time  `json:"created_at"`
}

// FileManagerRecent represents an auto-tracked recently visited folder
type FileManagerRecent struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	Domain         string     `json:"domain"`
	Path           string     `json:"path"`
	LastAccessedAt time.Time  `json:"last_accessed_at"`
}

// FolderLabel represents a custom color label and tag on a folder
type FolderLabel struct {
	ID        uuid.UUID `json:"id"`
	Domain    string    `json:"domain"`
	Path      string    `json:"path"`
	Color     string    `json:"color"` // e.g. blue, emerald, amber, rose, purple, indigo
	Label     string    `json:"label"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FileManagerTrashItem represents a file or folder moved to the Enterprise Trash Bin
type FileManagerTrashItem struct {
	ID           uuid.UUID  `json:"id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Domain       string     `json:"domain"`
	OriginalPath string     `json:"original_path"`
	TrashPath    string     `json:"trash_path"`
	Name         string     `json:"name"`
	Size         int64      `json:"size"`
	FileType     string     `json:"file_type"`
	IsDir        bool       `json:"is_dir"`
	DeletedBy    string     `json:"deleted_by"`
	DeletedAt    time.Time  `json:"deleted_at"`
}

// FileManagerActivityLog represents an audit log entry for file manager actions
type FileManagerActivityLog struct {
	ID              uuid.UUID              `json:"id"`
	UserID          *uuid.UUID             `json:"user_id,omitempty"`
	UserEmail       string                 `json:"user_email"`
	IPAddress       string                 `json:"ip_address"`
	Browser         string                 `json:"browser"`
	Domain          string                 `json:"domain"`
	Action          string                 `json:"action"` // upload, download, move, copy, rename, delete, restore, empty_trash, chmod, archive, extract
	SourcePath      string                 `json:"source_path"`
	DestinationPath string                 `json:"destination_path"`
	Details         map[string]interface{} `json:"details"`
	CreatedAt       time.Time              `json:"created_at"`
}
