package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// MEMORY STORE FILE MANAGER IMPLEMENTATION
// ============================================================================

func (m *MemoryStore) ListFileManagerFavorites(ctx context.Context, userID *uuid.UUID) ([]*FileManagerFavorite, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	favs := make([]*FileManagerFavorite, 0)
	for _, f := range m.fmFavorites {
		if userID == nil || (f.UserID != nil && *f.UserID == *userID) {
			favs = append(favs, f)
		}
	}
	sort.Slice(favs, func(i, j int) bool {
		return favs[i].CreatedAt.Before(favs[j].CreatedAt)
	})
	return favs, nil
}

func (m *MemoryStore) AddFileManagerFavorite(ctx context.Context, fav *FileManagerFavorite) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if fav.ID == uuid.Nil {
		fav.ID = uuid.New()
	}
	fav.CreatedAt = time.Now().UTC()

	// Replace if existing for same path and user
	for id, f := range m.fmFavorites {
		if f.Path == fav.Path && ((fav.UserID == nil && f.UserID == nil) || (fav.UserID != nil && f.UserID != nil && *fav.UserID == *f.UserID)) {
			delete(m.fmFavorites, id)
			break
		}
	}

	m.fmFavorites[fav.ID] = fav
	return nil
}

func (m *MemoryStore) DeleteFileManagerFavorite(ctx context.Context, userID *uuid.UUID, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanPath := strings.TrimSpace(path)
	for id, f := range m.fmFavorites {
		if f.Path == cleanPath && (userID == nil || (f.UserID != nil && *f.UserID == *userID)) {
			delete(m.fmFavorites, id)
		}
	}
	return nil
}

func (m *MemoryStore) ListFileManagerRecent(ctx context.Context, userID *uuid.UUID, limit int) ([]*FileManagerRecent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	recents := make([]*FileManagerRecent, 0)
	for _, r := range m.fmRecent {
		if userID == nil || (r.UserID != nil && *r.UserID == *userID) {
			recents = append(recents, r)
		}
	}
	sort.Slice(recents, func(i, j int) bool {
		return recents[i].LastAccessedAt.After(recents[j].LastAccessedAt)
	})

	if limit > 0 && len(recents) > limit {
		recents = recents[:limit]
	}
	return recents, nil
}

func (m *MemoryStore) RecordFileManagerRecent(ctx context.Context, rec *FileManagerRecent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanPath := strings.TrimSpace(rec.Path)
	if cleanPath == "" {
		return nil
	}

	// Update timestamp if path already in recent
	for _, existing := range m.fmRecent {
		if existing.Path == cleanPath && ((rec.UserID == nil && existing.UserID == nil) || (rec.UserID != nil && existing.UserID != nil && *rec.UserID == *existing.UserID)) {
			existing.LastAccessedAt = time.Now().UTC()
			existing.Domain = rec.Domain
			return nil
		}
	}

	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	rec.LastAccessedAt = time.Now().UTC()
	m.fmRecent[rec.ID] = rec
	return nil
}

func (m *MemoryStore) ListFolderLabels(ctx context.Context, domain string) ([]*FolderLabel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	labels := make([]*FolderLabel, 0)
	for _, l := range m.fmFolderLabels {
		if domain == "" || l.Domain == "" || l.Domain == domain {
			labels = append(labels, l)
		}
	}
	return labels, nil
}

func (m *MemoryStore) SetFolderLabel(ctx context.Context, label *FolderLabel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if label.ID == uuid.Nil {
		label.ID = uuid.New()
	}
	label.UpdatedAt = time.Now().UTC()
	m.fmFolderLabels[label.Path] = label
	return nil
}

func (m *MemoryStore) DeleteFolderLabel(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.fmFolderLabels, path)
	return nil
}

func (m *MemoryStore) ListFileManagerTrash(ctx context.Context, domain string) ([]*FileManagerTrashItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]*FileManagerTrashItem, 0)
	for _, t := range m.fmTrash {
		if domain == "" || t.Domain == "" || t.Domain == domain {
			items = append(items, t)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].DeletedAt.After(items[j].DeletedAt)
	})
	return items, nil
}

func (m *MemoryStore) AddFileManagerTrash(ctx context.Context, item *FileManagerTrashItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	item.DeletedAt = time.Now().UTC()
	m.fmTrash[item.ID] = item
	return nil
}

func (m *MemoryStore) GetFileManagerTrashItem(ctx context.Context, id uuid.UUID) (*FileManagerTrashItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.fmTrash[id]
	if !exists {
		return nil, ErrNotFound
	}
	return item, nil
}

func (m *MemoryStore) DeleteFileManagerTrashItem(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.fmTrash, id)
	return nil
}

func (m *MemoryStore) EmptyFileManagerTrash(ctx context.Context, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if domain == "" {
		m.fmTrash = make(map[uuid.UUID]*FileManagerTrashItem)
	} else {
		for id, item := range m.fmTrash {
			if item.Domain == domain {
				delete(m.fmTrash, id)
			}
		}
	}
	return nil
}

func (m *MemoryStore) RecordFileManagerActivityLog(ctx context.Context, log *FileManagerActivityLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now().UTC()
	m.fmActivityLogs = append(m.fmActivityLogs, log)

	// Keep last 1,000 logs in memory
	if len(m.fmActivityLogs) > 1000 {
		m.fmActivityLogs = m.fmActivityLogs[len(m.fmActivityLogs)-1000:]
	}
	return nil
}

func (m *MemoryStore) ListFileManagerActivityLogs(ctx context.Context, domain string, limit int) ([]*FileManagerActivityLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := make([]*FileManagerActivityLog, 0)
	for i := len(m.fmActivityLogs) - 1; i >= 0; i-- {
		l := m.fmActivityLogs[i]
		if domain == "" || l.Domain == "" || l.Domain == domain {
			logs = append(logs, l)
		}
		if limit > 0 && len(logs) >= limit {
			break
		}
	}
	return logs, nil
}

// ============================================================================
// POSTGRESQL STORE FILE MANAGER IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) ListFileManagerFavorites(ctx context.Context, userID *uuid.UUID) ([]*FileManagerFavorite, error) {
	query := `
		SELECT id, user_id, domain, path, name, color, created_at
		FROM file_manager_favorites
		WHERE ($1::uuid IS NULL OR user_id = $1)
		ORDER BY created_at ASC
	`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favs []*FileManagerFavorite
	for rows.Next() {
		var f FileManagerFavorite
		if err := rows.Scan(&f.ID, &f.UserID, &f.Domain, &f.Path, &f.Name, &f.Color, &f.CreatedAt); err != nil {
			return nil, err
		}
		favs = append(favs, &f)
	}
	return favs, rows.Err()
}

func (p *PostgresStore) AddFileManagerFavorite(ctx context.Context, fav *FileManagerFavorite) error {
	if fav.ID == uuid.Nil {
		fav.ID = uuid.New()
	}
	query := `
		INSERT INTO file_manager_favorites (id, user_id, domain, path, name, color, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			domain = EXCLUDED.domain,
			name = EXCLUDED.name,
			color = EXCLUDED.color
	`
	_, err := p.db.ExecContext(ctx, query, fav.ID, fav.UserID, fav.Domain, fav.Path, fav.Name, fav.Color)
	return err
}

func (p *PostgresStore) DeleteFileManagerFavorite(ctx context.Context, userID *uuid.UUID, path string) error {
	query := `
		DELETE FROM file_manager_favorites
		WHERE path = $1 AND ($2::uuid IS NULL OR user_id = $2)
	`
	_, err := p.db.ExecContext(ctx, query, path, userID)
	return err
}

func (p *PostgresStore) ListFileManagerRecent(ctx context.Context, userID *uuid.UUID, limit int) ([]*FileManagerRecent, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, user_id, domain, path, last_accessed_at
		FROM file_manager_recent
		WHERE ($1::uuid IS NULL OR user_id = $1)
		ORDER BY last_accessed_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recents []*FileManagerRecent
	for rows.Next() {
		var r FileManagerRecent
		if err := rows.Scan(&r.ID, &r.UserID, &r.Domain, &r.Path, &r.LastAccessedAt); err != nil {
			return nil, err
		}
		recents = append(recents, &r)
	}
	return recents, rows.Err()
}

func (p *PostgresStore) RecordFileManagerRecent(ctx context.Context, rec *FileManagerRecent) error {
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	query := `
		INSERT INTO file_manager_recent (id, user_id, domain, path, last_accessed_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE SET
			domain = EXCLUDED.domain,
			last_accessed_at = NOW()
	`
	_, err := p.db.ExecContext(ctx, query, rec.ID, rec.UserID, rec.Domain, rec.Path)
	return err
}

func (p *PostgresStore) ListFolderLabels(ctx context.Context, domain string) ([]*FolderLabel, error) {
	query := `
		SELECT id, domain, path, color, label, updated_at
		FROM folder_labels
		WHERE ($1 = '' OR domain = '' OR domain = $1)
		ORDER BY updated_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []*FolderLabel
	for rows.Next() {
		var l FolderLabel
		if err := rows.Scan(&l.ID, &l.Domain, &l.Path, &l.Color, &l.Label, &l.UpdatedAt); err != nil {
			return nil, err
		}
		labels = append(labels, &l)
	}
	return labels, rows.Err()
}

func (p *PostgresStore) SetFolderLabel(ctx context.Context, label *FolderLabel) error {
	if label.ID == uuid.Nil {
		label.ID = uuid.New()
	}
	query := `
		INSERT INTO folder_labels (id, domain, path, color, label, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (path) DO UPDATE SET
			domain = EXCLUDED.domain,
			color = EXCLUDED.color,
			label = EXCLUDED.label,
			updated_at = NOW()
	`
	_, err := p.db.ExecContext(ctx, query, label.ID, label.Domain, label.Path, label.Color, label.Label)
	return err
}

func (p *PostgresStore) DeleteFolderLabel(ctx context.Context, path string) error {
	query := `DELETE FROM folder_labels WHERE path = $1`
	_, err := p.db.ExecContext(ctx, query, path)
	return err
}

func (p *PostgresStore) ListFileManagerTrash(ctx context.Context, domain string) ([]*FileManagerTrashItem, error) {
	query := `
		SELECT id, user_id, domain, original_path, trash_path, name, size, file_type, is_dir, deleted_by, deleted_at
		FROM file_manager_trash
		WHERE ($1 = '' OR domain = '' OR domain = $1)
		ORDER BY deleted_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*FileManagerTrashItem
	for rows.Next() {
		var t FileManagerTrashItem
		if err := rows.Scan(&t.ID, &t.UserID, &t.Domain, &t.OriginalPath, &t.TrashPath, &t.Name, &t.Size, &t.FileType, &t.IsDir, &t.DeletedBy, &t.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, &t)
	}
	return items, rows.Err()
}

func (p *PostgresStore) AddFileManagerTrash(ctx context.Context, item *FileManagerTrashItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	query := `
		INSERT INTO file_manager_trash (id, user_id, domain, original_path, trash_path, name, size, file_type, is_dir, deleted_by, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`
	_, err := p.db.ExecContext(ctx, query, item.ID, item.UserID, item.Domain, item.OriginalPath, item.TrashPath, item.Name, item.Size, item.FileType, item.IsDir, item.DeletedBy)
	return err
}

func (p *PostgresStore) GetFileManagerTrashItem(ctx context.Context, id uuid.UUID) (*FileManagerTrashItem, error) {
	query := `
		SELECT id, user_id, domain, original_path, trash_path, name, size, file_type, is_dir, deleted_by, deleted_at
		FROM file_manager_trash
		WHERE id = $1
	`
	var t FileManagerTrashItem
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.UserID, &t.Domain, &t.OriginalPath, &t.TrashPath, &t.Name, &t.Size, &t.FileType, &t.IsDir, &t.DeletedBy, &t.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (p *PostgresStore) DeleteFileManagerTrashItem(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM file_manager_trash WHERE id = $1`
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresStore) EmptyFileManagerTrash(ctx context.Context, domain string) error {
	var err error
	if domain == "" {
		_, err = p.db.ExecContext(ctx, `DELETE FROM file_manager_trash`)
	} else {
		_, err = p.db.ExecContext(ctx, `DELETE FROM file_manager_trash WHERE domain = $1`, domain)
	}
	return err
}

func (p *PostgresStore) RecordFileManagerActivityLog(ctx context.Context, log *FileManagerActivityLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}
	query := `
		INSERT INTO file_manager_activity_logs (id, user_id, user_email, ip_address, browser, domain, action, source_path, destination_path, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`
	_, err = p.db.ExecContext(ctx, query, log.ID, log.UserID, log.UserEmail, log.IPAddress, log.Browser, log.Domain, log.Action, log.SourcePath, log.DestinationPath, detailsJSON)
	return err
}

func (p *PostgresStore) ListFileManagerActivityLogs(ctx context.Context, domain string, limit int) ([]*FileManagerActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, user_id, user_email, ip_address, browser, domain, action, source_path, destination_path, details, created_at
		FROM file_manager_activity_logs
		WHERE ($1 = '' OR domain = '' OR domain = $1)
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, domain, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*FileManagerActivityLog
	for rows.Next() {
		var l FileManagerActivityLog
		var rawDetails []byte
		if err := rows.Scan(&l.ID, &l.UserID, &l.UserEmail, &l.IPAddress, &l.Browser, &l.Domain, &l.Action, &l.SourcePath, &l.DestinationPath, &rawDetails, &l.CreatedAt); err != nil {
			return nil, err
		}
		if len(rawDetails) > 0 {
			_ = json.Unmarshal(rawDetails, &l.Details)
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

// Suppress unused imports
var _ = pq.Array
