package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Default system settings for enterprise panel
func defaultSystemSettings() *SystemSettings {
	return &SystemSettings{
		PanelDomain:         "",
		PanelPort:           "26589",
		SecurityEntrance:    "/hostvra-admin",
		SSLEnabled:          true,
		SSLDaysRemaining:    90,
		DevMode:             false,
		APIEnabled:          true,
		APIKey:              "hv_live_0f9a72b1c4e683d5a892f0e1b3c75d4a",
		PanelUser:           "hostvra_admin",
		BoundAccount:        "admin@hostvra.com",
		MenuBarHidden:       "none",
		ClosePanel:          false,
		IPv6Enabled:         false,
		OfflineMode:         false,
		CDNProxy:            false,
		HomeBulletin:        true,
		SiteMonitor:         true,
		AutoFetchFavicon:    true,
		AutoBackupPanel:     true,
		PanelTheme:          "Dark Slate",
		PanelLanguage:       "English",
		PanelAlias:          "Hostvra Enterprise Cloud Panel",
		SessionTimeout:      "24 Hour(s)",
		DefaultSiteFolder:   "/www/wwwroot",
		DefaultBackupFolder: "/www/backup",
		ServerIP:            "127.0.0.1",
		ServerTime:          time.Now().UTC().Format("2006-01-02 15:04:05 MST"),
		TimezoneRegion:      "Etc",
		TimezoneCity:        "UTC",
		SecurityAlarm:       false,
		BasicAuth:           false,
		GoogleAuth:          false,
		StrongPassword:      true,
		AuthorizedIP:        "",
		NotLoggedInResponse: "404 - Not Found",
		PasswordExpire:      "Never",
		UpdatedAt:           time.Now().UTC(),
	}
}

// ============================================================================
// MEMORY STORE SETTINGS IMPLEMENTATION
// ============================================================================

func (m *MemoryStore) GetSystemSettings(ctx context.Context) (*SystemSettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.systemSettings == nil {
		return defaultSystemSettings(), nil
	}

	// Return a copy
	settingsCopy := *m.systemSettings
	settingsCopy.ServerTime = time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	return &settingsCopy, nil
}

func (m *MemoryStore) UpdateSystemSettings(ctx context.Context, s *SystemSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s.UpdatedAt = time.Now().UTC()
	m.systemSettings = s
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE SETTINGS IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) GetSystemSettings(ctx context.Context) (*SystemSettings, error) {
	query := `SELECT settings FROM system_settings WHERE id = 1`
	var rawJSON []byte
	err := p.db.QueryRowContext(ctx, query).Scan(&rawJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			defaults := defaultSystemSettings()
			_ = p.UpdateSystemSettings(ctx, defaults)
			return defaults, nil
		}
		m := NewMemoryStore()
		return m.GetSystemSettings(ctx)
	}

	settings := &SystemSettings{}
	if err := json.Unmarshal(rawJSON, settings); err != nil {
		m := NewMemoryStore()
		return m.GetSystemSettings(ctx)
	}
	settings.ServerTime = time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	return settings, nil
}

func (p *PostgresStore) UpdateSystemSettings(ctx context.Context, s *SystemSettings) error {
	s.UpdatedAt = time.Now().UTC()
	rawJSON, err := json.Marshal(s)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO system_settings (id, settings, updated_at)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET
			settings = EXCLUDED.settings,
			updated_at = EXCLUDED.updated_at
	`
	_, err = p.db.ExecContext(ctx, query, rawJSON, s.UpdatedAt)
	if err != nil {
		m := NewMemoryStore()
		return m.UpdateSystemSettings(ctx, s)
	}
	return nil
}
