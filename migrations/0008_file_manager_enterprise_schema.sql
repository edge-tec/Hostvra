-- ============================================================================
-- Migration 0008: HostVra Enterprise File Manager v3.0 Schema
-- Favorites, Recent Folders, Folder Color Labels, Smart Trash Bin & Audit Logs
-- ============================================================================

-- 1. File Manager Favorites (Pinned folders)
CREATE TABLE IF NOT EXISTS file_manager_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(50) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fm_favorites_user ON file_manager_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_fm_favorites_path ON file_manager_favorites(path);

-- 2. File Manager Recent Folders (Auto-tracked on folder navigation)
CREATE TABLE IF NOT EXISTS file_manager_recent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fm_recent_user_accessed ON file_manager_recent(user_id, last_accessed_at DESC);
CREATE INDEX IF NOT EXISTS idx_fm_recent_path ON file_manager_recent(path);

-- 3. Folder Color Labels (Tags for folders)
CREATE TABLE IF NOT EXISTS folder_labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain VARCHAR(255) NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    color VARCHAR(50) NOT NULL DEFAULT 'blue',
    label VARCHAR(100) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_labels_path ON folder_labels(path);
CREATE INDEX IF NOT EXISTS idx_folder_labels_domain ON folder_labels(domain);

-- 4. Enterprise Trash Bin System
CREATE TABLE IF NOT EXISTS file_manager_trash (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    domain VARCHAR(255) NOT NULL DEFAULT '',
    original_path TEXT NOT NULL,
    trash_path TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    file_type VARCHAR(100) NOT NULL DEFAULT 'file',
    is_dir BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_by VARCHAR(255) NOT NULL DEFAULT '',
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fm_trash_domain ON file_manager_trash(domain);
CREATE INDEX IF NOT EXISTS idx_fm_trash_deleted_at ON file_manager_trash(deleted_at DESC);

-- 5. File Manager Activity & Audit Logs
CREATE TABLE IF NOT EXISTS file_manager_activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    ip_address VARCHAR(100) NOT NULL DEFAULT '',
    browser VARCHAR(255) NOT NULL DEFAULT '',
    domain VARCHAR(255) NOT NULL DEFAULT '',
    action VARCHAR(100) NOT NULL,
    source_path TEXT NOT NULL DEFAULT '',
    destination_path TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fm_activity_domain ON file_manager_activity_logs(domain);
CREATE INDEX IF NOT EXISTS idx_fm_activity_action ON file_manager_activity_logs(action);
CREATE INDEX IF NOT EXISTS idx_fm_activity_created_at ON file_manager_activity_logs(created_at DESC);
