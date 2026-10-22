package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MEMORY STORE: DEVELOPER APPLICATIONS, GIT CI/CD, WORDPRESS STAGING
// ============================================================================

// Managed Applications
func (m *MemoryStore) GetManagedApp(ctx context.Context, id uuid.UUID) (*ManagedAppRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	app, ok := m.managedApps[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyApp := *app
	return &copyApp, nil
}

func (m *MemoryStore) GetManagedAppByWebsite(ctx context.Context, websiteID uuid.UUID) (*ManagedAppRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	appID, ok := m.managedAppsByWebsite[websiteID]
	if !ok {
		return nil, ErrNotFound
	}
	app, ok := m.managedApps[appID]
	if !ok {
		return nil, ErrNotFound
	}
	copyApp := *app
	return &copyApp, nil
}

func (m *MemoryStore) SaveManagedApp(ctx context.Context, app *ManagedAppRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	now := time.Now().UTC()
	if existing, ok := m.managedApps[app.ID]; ok {
		app.CreatedAt = existing.CreatedAt
	} else if app.CreatedAt.IsZero() {
		app.CreatedAt = now
	}
	app.UpdatedAt = now

	copyApp := *app
	m.managedApps[app.ID] = &copyApp
	m.managedAppsByWebsite[app.WebsiteID] = app.ID
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteManagedApp(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	app, ok := m.managedApps[id]
	if !ok {
		return ErrNotFound
	}
	delete(m.managedAppsByWebsite, app.WebsiteID)
	delete(m.managedApps, id)
	m.saveToDiskLocked()
	return nil
}

// Git-Based CI/CD
func (m *MemoryStore) GetGitRepository(ctx context.Context, websiteID uuid.UUID) (*GitRepositoryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	repo, ok := m.gitRepositories[websiteID]
	if !ok {
		return nil, ErrNotFound
	}
	copyRepo := *repo
	return &copyRepo, nil
}

func (m *MemoryStore) SaveGitRepository(ctx context.Context, repo *GitRepositoryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if repo.ID == uuid.Nil {
		repo.ID = uuid.New()
	}
	now := time.Now().UTC()
	if existing, ok := m.gitRepositories[repo.WebsiteID]; ok {
		repo.CreatedAt = existing.CreatedAt
	} else if repo.CreatedAt.IsZero() {
		repo.CreatedAt = now
	}
	repo.UpdatedAt = now

	copyRepo := *repo
	m.gitRepositories[repo.WebsiteID] = &copyRepo
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteGitRepository(ctx context.Context, websiteID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.gitRepositories[websiteID]; !ok {
		return ErrNotFound
	}
	delete(m.gitRepositories, websiteID)
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) CreateGitDeployment(ctx context.Context, dep *GitDeploymentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dep.ID == uuid.Nil {
		dep.ID = uuid.New()
	}
	if dep.CreatedAt.IsZero() {
		dep.CreatedAt = time.Now().UTC()
	}
	copyDep := *dep
	m.gitDeployments[dep.ID] = &copyDep
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) UpdateGitDeployment(ctx context.Context, dep *GitDeploymentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copyDep := *dep
	m.gitDeployments[dep.ID] = &copyDep
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) GetGitDeployment(ctx context.Context, id uuid.UUID) (*GitDeploymentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dep, ok := m.gitDeployments[id]
	if !ok {
		return nil, ErrNotFound
	}
	copyDep := *dep
	return &copyDep, nil
}

func (m *MemoryStore) ListGitDeployments(ctx context.Context, websiteID uuid.UUID, limit int) ([]*GitDeploymentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var matched []*GitDeploymentRecord
	for _, dep := range m.gitDeployments {
		if dep.WebsiteID == websiteID {
			copyDep := *dep
			matched = append(matched, &copyDep)
		}
	}

	// Sort newest first
	for i := 0; i < len(matched)-1; i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].CreatedAt.After(matched[i].CreatedAt) {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	if len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}

// WordPress Staging & Cloning
func (m *MemoryStore) GetWordPressStaging(ctx context.Context, websiteID uuid.UUID) (*WordPressStagingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stg, ok := m.wordpressStagings[websiteID]
	if !ok {
		return nil, ErrNotFound
	}
	copyStg := *stg
	return &copyStg, nil
}

func (m *MemoryStore) SaveWordPressStaging(ctx context.Context, staging *WordPressStagingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if staging.ID == uuid.Nil {
		staging.ID = uuid.New()
	}
	now := time.Now().UTC()
	if existing, ok := m.wordpressStagings[staging.WebsiteID]; ok {
		staging.CreatedAt = existing.CreatedAt
	} else if staging.CreatedAt.IsZero() {
		staging.CreatedAt = now
	}
	staging.UpdatedAt = now

	copyStg := *staging
	m.wordpressStagings[staging.WebsiteID] = &copyStg
	m.saveToDiskLocked()
	return nil
}

func (m *MemoryStore) DeleteWordPressStaging(ctx context.Context, websiteID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.wordpressStagings[websiteID]; !ok {
		return ErrNotFound
	}
	delete(m.wordpressStagings, websiteID)
	m.saveToDiskLocked()
	return nil
}

// ============================================================================
// POSTGRES STORE: DEVELOPER APPLICATIONS, GIT CI/CD, WORDPRESS STAGING
// ============================================================================

func (s *PostgresStore) GetManagedApp(ctx context.Context, id uuid.UUID) (*ManagedAppRecord, error) {
	query := `
		SELECT id, website_id, organization_id, name, app_type, runtime_version,
		       app_root, entrypoint, start_command, port, system_user, env_vars,
		       status, restart_count, created_at, updated_at
		FROM managed_apps
		WHERE id = $1
	`
	var app ManagedAppRecord
	var envData []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID, &app.WebsiteID, &app.OrganizationID, &app.Name, &app.AppType,
		&app.RuntimeVersion, &app.AppRoot, &app.Entrypoint, &app.StartCommand,
		&app.Port, &app.SystemUser, &envData, &app.Status, &app.RestartCount,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(envData) > 0 {
		_ = json.Unmarshal(envData, &app.EnvVars)
	}
	return &app, nil
}

func (s *PostgresStore) GetManagedAppByWebsite(ctx context.Context, websiteID uuid.UUID) (*ManagedAppRecord, error) {
	query := `
		SELECT id, website_id, organization_id, name, app_type, runtime_version,
		       app_root, entrypoint, start_command, port, system_user, env_vars,
		       status, restart_count, created_at, updated_at
		FROM managed_apps
		WHERE website_id = $1
		LIMIT 1
	`
	var app ManagedAppRecord
	var envData []byte
	err := s.db.QueryRowContext(ctx, query, websiteID).Scan(
		&app.ID, &app.WebsiteID, &app.OrganizationID, &app.Name, &app.AppType,
		&app.RuntimeVersion, &app.AppRoot, &app.Entrypoint, &app.StartCommand,
		&app.Port, &app.SystemUser, &envData, &app.Status, &app.RestartCount,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(envData) > 0 {
		_ = json.Unmarshal(envData, &app.EnvVars)
	}
	return &app, nil
}

func (s *PostgresStore) SaveManagedApp(ctx context.Context, app *ManagedAppRecord) error {
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	envData, _ := json.Marshal(app.EnvVars)
	now := time.Now().UTC()
	if app.CreatedAt.IsZero() {
		app.CreatedAt = now
	}
	app.UpdatedAt = now

	query := `
		INSERT INTO managed_apps (
			id, website_id, organization_id, name, app_type, runtime_version,
			app_root, entrypoint, start_command, port, system_user, env_vars,
			status, restart_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			app_type = EXCLUDED.app_type,
			runtime_version = EXCLUDED.runtime_version,
			app_root = EXCLUDED.app_root,
			entrypoint = EXCLUDED.entrypoint,
			start_command = EXCLUDED.start_command,
			port = EXCLUDED.port,
			system_user = EXCLUDED.system_user,
			env_vars = EXCLUDED.env_vars,
			status = EXCLUDED.status,
			restart_count = EXCLUDED.restart_count,
			updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		app.ID, app.WebsiteID, app.OrganizationID, app.Name, app.AppType,
		app.RuntimeVersion, app.AppRoot, app.Entrypoint, app.StartCommand,
		app.Port, app.SystemUser, envData, app.Status, app.RestartCount,
		app.CreatedAt, app.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) DeleteManagedApp(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM managed_apps WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetGitRepository(ctx context.Context, websiteID uuid.UUID) (*GitRepositoryRecord, error) {
	query := `
		SELECT id, website_id, organization_id, provider, repo_url, branch,
		       webhook_secret, deploy_key_pub, deploy_key_priv, auto_deploy,
		       build_command, deploy_path, release_keep_count, created_at, updated_at
		FROM git_repositories
		WHERE website_id = $1
	`
	var repo GitRepositoryRecord
	err := s.db.QueryRowContext(ctx, query, websiteID).Scan(
		&repo.ID, &repo.WebsiteID, &repo.OrganizationID, &repo.Provider, &repo.RepoURL, &repo.Branch,
		&repo.WebhookSecret, &repo.DeployKeyPub, &repo.DeployKeyPriv, &repo.AutoDeploy,
		&repo.BuildCommand, &repo.DeployPath, &repo.ReleaseKeepCount, &repo.CreatedAt, &repo.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &repo, err
}

func (s *PostgresStore) SaveGitRepository(ctx context.Context, repo *GitRepositoryRecord) error {
	if repo.ID == uuid.Nil {
		repo.ID = uuid.New()
	}
	now := time.Now().UTC()
	if repo.CreatedAt.IsZero() {
		repo.CreatedAt = now
	}
	repo.UpdatedAt = now

	query := `
		INSERT INTO git_repositories (
			id, website_id, organization_id, provider, repo_url, branch,
			webhook_secret, deploy_key_pub, deploy_key_priv, auto_deploy,
			build_command, deploy_path, release_keep_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (website_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			repo_url = EXCLUDED.repo_url,
			branch = EXCLUDED.branch,
			webhook_secret = EXCLUDED.webhook_secret,
			deploy_key_pub = EXCLUDED.deploy_key_pub,
			deploy_key_priv = EXCLUDED.deploy_key_priv,
			auto_deploy = EXCLUDED.auto_deploy,
			build_command = EXCLUDED.build_command,
			deploy_path = EXCLUDED.deploy_path,
			release_keep_count = EXCLUDED.release_keep_count,
			updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		repo.ID, repo.WebsiteID, repo.OrganizationID, repo.Provider, repo.RepoURL, repo.Branch,
		repo.WebhookSecret, repo.DeployKeyPub, repo.DeployKeyPriv, repo.AutoDeploy,
		repo.BuildCommand, repo.DeployPath, repo.ReleaseKeepCount, repo.CreatedAt, repo.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) DeleteGitRepository(ctx context.Context, websiteID uuid.UUID) error {
	query := `DELETE FROM git_repositories WHERE website_id = $1`
	res, err := s.db.ExecContext(ctx, query, websiteID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) CreateGitDeployment(ctx context.Context, dep *GitDeploymentRecord) error {
	if dep.ID == uuid.Nil {
		dep.ID = uuid.New()
	}
	if dep.CreatedAt.IsZero() {
		dep.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO git_deployments (
			id, repository_id, website_id, commit_hash, commit_message,
			author, status, build_logs, release_path, duration_seconds, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	_, err := s.db.ExecContext(ctx, query,
		dep.ID, dep.RepositoryID, dep.WebsiteID, dep.CommitHash, dep.CommitMessage,
		dep.Author, dep.Status, dep.BuildLogs, dep.ReleasePath, dep.DurationSeconds, dep.CreatedAt,
	)
	return err
}

func (s *PostgresStore) UpdateGitDeployment(ctx context.Context, dep *GitDeploymentRecord) error {
	query := `
		UPDATE git_deployments SET
			commit_hash = $2,
			commit_message = $3,
			author = $4,
			status = $5,
			build_logs = $6,
			release_path = $7,
			duration_seconds = $8
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		dep.ID, dep.CommitHash, dep.CommitMessage, dep.Author, dep.Status,
		dep.BuildLogs, dep.ReleasePath, dep.DurationSeconds,
	)
	return err
}

func (s *PostgresStore) GetGitDeployment(ctx context.Context, id uuid.UUID) (*GitDeploymentRecord, error) {
	query := `
		SELECT id, repository_id, website_id, commit_hash, commit_message,
		       author, status, build_logs, release_path, duration_seconds, created_at
		FROM git_deployments
		WHERE id = $1
	`
	var dep GitDeploymentRecord
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&dep.ID, &dep.RepositoryID, &dep.WebsiteID, &dep.CommitHash, &dep.CommitMessage,
		&dep.Author, &dep.Status, &dep.BuildLogs, &dep.ReleasePath, &dep.DurationSeconds, &dep.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &dep, err
}

func (s *PostgresStore) ListGitDeployments(ctx context.Context, websiteID uuid.UUID, limit int) ([]*GitDeploymentRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `
		SELECT id, repository_id, website_id, commit_hash, commit_message,
		       author, status, build_logs, release_path, duration_seconds, created_at
		FROM git_deployments
		WHERE website_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, websiteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*GitDeploymentRecord
	for rows.Next() {
		var dep GitDeploymentRecord
		if err := rows.Scan(
			&dep.ID, &dep.RepositoryID, &dep.WebsiteID, &dep.CommitHash, &dep.CommitMessage,
			&dep.Author, &dep.Status, &dep.BuildLogs, &dep.ReleasePath, &dep.DurationSeconds, &dep.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &dep)
	}
	return list, rows.Err()
}

func (s *PostgresStore) GetWordPressStaging(ctx context.Context, websiteID uuid.UUID) (*WordPressStagingRecord, error) {
	query := `
		SELECT id, website_id, organization_id, staging_domain, staging_doc_root,
		       staging_db_name, staging_db_user, auth_enabled, auth_user, auth_password,
		       status, last_sync_at, created_at, updated_at
		FROM wordpress_stagings
		WHERE website_id = $1
	`
	var stg WordPressStagingRecord
	err := s.db.QueryRowContext(ctx, query, websiteID).Scan(
		&stg.ID, &stg.WebsiteID, &stg.OrganizationID, &stg.StagingDomain, &stg.StagingDocRoot,
		&stg.StagingDBName, &stg.StagingDBUser, &stg.AuthEnabled, &stg.AuthUser, &stg.AuthPassword,
		&stg.Status, &stg.LastSyncAt, &stg.CreatedAt, &stg.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &stg, err
}

func (s *PostgresStore) SaveWordPressStaging(ctx context.Context, staging *WordPressStagingRecord) error {
	if staging.ID == uuid.Nil {
		staging.ID = uuid.New()
	}
	now := time.Now().UTC()
	if staging.CreatedAt.IsZero() {
		staging.CreatedAt = now
	}
	staging.UpdatedAt = now

	query := `
		INSERT INTO wordpress_stagings (
			id, website_id, organization_id, staging_domain, staging_doc_root,
			staging_db_name, staging_db_user, auth_enabled, auth_user, auth_password,
			status, last_sync_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (website_id) DO UPDATE SET
			staging_domain = EXCLUDED.staging_domain,
			staging_doc_root = EXCLUDED.staging_doc_root,
			staging_db_name = EXCLUDED.staging_db_name,
			staging_db_user = EXCLUDED.staging_db_user,
			auth_enabled = EXCLUDED.auth_enabled,
			auth_user = EXCLUDED.auth_user,
			auth_password = EXCLUDED.auth_password,
			status = EXCLUDED.status,
			last_sync_at = EXCLUDED.last_sync_at,
			updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		staging.ID, staging.WebsiteID, staging.OrganizationID, staging.StagingDomain, staging.StagingDocRoot,
		staging.StagingDBName, staging.StagingDBUser, staging.AuthEnabled, staging.AuthUser, staging.AuthPassword,
		staging.Status, staging.LastSyncAt, staging.CreatedAt, staging.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) DeleteWordPressStaging(ctx context.Context, websiteID uuid.UUID) error {
	query := `DELETE FROM wordpress_stagings WHERE website_id = $1`
	res, err := s.db.ExecContext(ctx, query, websiteID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
