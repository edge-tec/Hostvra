package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/database"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type DatabaseHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
	dbMgr *database.Manager
}

func NewDatabaseHandler(cfg *config.Config, s store.Store, a *audit.Logger) *DatabaseHandler {
	return &DatabaseHandler{
		cfg:   cfg,
		store: s,
		audit: a,
		dbMgr: database.NewManager(),
	}
}

// Request & Response DTOs
type CreateDatabaseRequest struct {
	ServerID     string `json:"server_id"`
	DBType       string `json:"db_type"` // mysql, mariadb, postgresql, sqlserver, mongodb, redis
	Name         string `json:"name"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	CharacterSet string `json:"character_set"`
	Collation    string `json:"collation"`
	HostAllow    string `json:"host_allow"`
	Note         string `json:"note"`
	Quota        string `json:"quota"`
}

type UpdateDatabaseRequest struct {
	Password  string `json:"password"`
	HostAllow string `json:"host_allow"`
	Note      string `json:"note"`
	Quota     string `json:"quota"`
}

type DatabaseToolsRequest struct {
	Action string `json:"action"` // optimize, repair, check, analyze
}

type RootPasswordRequest struct {
	Password string `json:"password"`
}

type AutoBackupRequest struct {
	Enabled bool `json:"enabled"`
}

type BatchOperationRequest struct {
	Action string   `json:"action"` // delete, backup, optimize
	IDs    []string `json:"ids"`
}

type CreateDatabaseUserRequest struct {
	ServerID  string `json:"server_id"`
	DBType    string `json:"db_type"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	HostAllow string `json:"host_allow"`
}

// List returns databases for the given server (or all active databases if server_id omitted)
func (h *DatabaseHandler) List(w http.ResponseWriter, r *http.Request) {
	var serverID uuid.UUID
	serverIDStr := r.URL.Query().Get("server_id")
	if serverIDStr != "" {
		if id, err := uuid.Parse(serverIDStr); err == nil {
			serverID = id
		}
	}

	dbs, err := h.store.ListDatabasesByServer(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve databases", nil, "")
		return
	}

	// Filter out recycle bin items unless requested
	includeTrash := r.URL.Query().Get("include_trash") == "true"
	var filtered []*store.Database
	for _, d := range dbs {
		if d.InRecycleBin && !includeTrash {
			continue
		}
		filtered = append(filtered, d)
	}

	response.JSON(w, http.StatusOK, filtered, &response.Meta{
		Total: len(filtered),
	})
}

// Create provisions a new database and associated user
func (h *DatabaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database name is required", nil, "")
		return
	}

	var serverID uuid.UUID
	if req.ServerID != "" {
		if id, err := uuid.Parse(req.ServerID); err == nil {
			serverID = id
		}
	}

	if req.DBType == "" {
		req.DBType = "mysql"
	}
	if req.CharacterSet == "" {
		req.CharacterSet = "utf8mb4"
	}
	if req.Collation == "" {
		req.Collation = "utf8mb4_unicode_ci"
	}
	if req.HostAllow == "" {
		req.HostAllow = "localhost"
	}
	if req.Username == "" {
		req.Username = req.Name
	}
	if req.Password == "" {
		req.Password = uuid.New().String()[:12]
	}
	if req.Quota == "" {
		req.Quota = "Not set"
	}

	db := &store.Database{
		ID:           uuid.New(),
		ServerID:     serverID,
		DBType:       req.DBType,
		Name:         req.Name,
		Username:     req.Username,
		Password:     req.Password,
		CharacterSet: req.CharacterSet,
		Collation:    req.Collation,
		Quota:        req.Quota,
		BackupStatus: "Not exist",
		BackupCount:  0,
		Location:     "Localhost",
		Note:         req.Note,
		HostAllow:    req.HostAllow,
		InRecycleBin: false,
		CreatedAt:    time.Now().UTC(),
	}

	if err := h.store.CreateDatabase(r.Context(), db); err != nil {
		response.Error(w, http.StatusConflict, "DATABASE_EXISTS", "Database already exists", nil, "")
		return
	}

	// Trigger real-time host provision
	_ = h.dbMgr.ExecuteRealDatabaseCreation(r.Context(), req.Name, req.CharacterSet, req.Collation, req.Username, req.Password, req.HostAllow)

	h.audit.Log(r.Context(), r, "database.create", "database", db.ID.String(), "success", "", map[string]interface{}{
		"name":      db.Name,
		"db_type":   db.DBType,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusCreated, db, nil)
}

// Update updates database password, host allow, note, or quota
func (h *DatabaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	dbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid database UUID", nil, "")
		return
	}

	var req UpdateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	db, err := h.store.GetDatabaseByID(r.Context(), dbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Database not found", nil, "")
		return
	}

	if req.Note != "" {
		db.Note = req.Note
	}
	if req.Quota != "" {
		db.Quota = req.Quota
	}
	if req.Password != "" {
		db.Password = req.Password
		_ = h.dbMgr.ExecuteUpdatePassword(r.Context(), db.Username, db.HostAllow, req.Password)
	}
	if req.HostAllow != "" {
		_ = h.dbMgr.ExecuteUpdatePermission(r.Context(), db.Username, db.HostAllow, req.HostAllow, db.Name)
		db.HostAllow = req.HostAllow
	}

	if err := h.store.UpdateDatabase(r.Context(), db); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update database", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "database.update", "database", dbID.String(), "success", "", nil)
	response.JSON(w, http.StatusOK, db, nil)
}

// Delete drops a database or moves it to the recycle bin
func (h *DatabaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	dbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid database UUID", nil, "")
		return
	}

	recycle := r.URL.Query().Get("recycle_bin") == "true"
	db, err := h.store.GetDatabaseByID(r.Context(), dbID)
	if err == nil && db != nil {
		if !recycle {
			_ = h.dbMgr.ExecuteDropDatabase(r.Context(), db.Name)
		}
	}

	if err := h.store.DeleteDatabase(r.Context(), dbID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete database", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "database.delete", "database", dbID.String(), "success", "", map[string]interface{}{
		"recycle_bin": recycle,
	})
	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "recycle_bin": recycle}, nil)
}

// GetStatus returns the operational status and version of the database server
func (h *DatabaseHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	engine := r.URL.Query().Get("engine")
	if engine == "" {
		engine = "mysql"
	}

	status, err := h.dbMgr.GetStatus(r.Context(), engine)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STATUS_ERROR", "Failed to get database status", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, status, nil)
}

// GetServerDatabases queries live host databases (SHOW DATABASES)
func (h *DatabaseHandler) GetServerDatabases(w http.ResponseWriter, r *http.Request) {
	dbs, err := h.dbMgr.GetServerDatabases(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"databases": dbs}, nil)
}

// Sync pulls databases from the server into Hostvra registry
func (h *DatabaseHandler) Sync(w http.ResponseWriter, r *http.Request) {
	serverDbs, _ := h.dbMgr.GetServerDatabases(r.Context())
	syncedCount := 0

	for _, name := range serverDbs {
		db := &store.Database{
			ID:           uuid.New(),
			DBType:       "mysql",
			Name:         name,
			Username:     name,
			Password:     uuid.New().String()[:10],
			CharacterSet: "utf8mb4",
			Collation:    "utf8mb4_unicode_ci",
			Quota:        "Not set",
			BackupStatus: "Not exist",
			Location:     "Localhost",
			Note:         name,
			HostAllow:    "localhost",
			CreatedAt:    time.Now().UTC(),
		}
		if err := h.store.CreateDatabase(r.Context(), db); err == nil {
			syncedCount++
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"synced":       true,
		"new_imported": syncedCount,
		"total_server": len(serverDbs),
	}, nil)
}

// RunTools executes maintenance actions (optimize, repair, analyze, check)
func (h *DatabaseHandler) RunTools(w http.ResponseWriter, r *http.Request) {
	dbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid database UUID", nil, "")
		return
	}

	var req DatabaseToolsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Action == "" {
		req.Action = "optimize"
	}

	db, err := h.store.GetDatabaseByID(r.Context(), dbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Database not found", nil, "")
		return
	}

	output, err := h.dbMgr.RunDatabaseTools(r.Context(), db.Name, req.Action)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOOL_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"database": db.Name,
		"action":   req.Action,
		"output":   output,
		"success":  true,
	}, nil)
}

// Backup dumps the database and updates status
func (h *DatabaseHandler) Backup(w http.ResponseWriter, r *http.Request) {
	dbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid database UUID", nil, "")
		return
	}

	db, err := h.store.GetDatabaseByID(r.Context(), dbID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Database not found", nil, "")
		return
	}

	dump, err := h.dbMgr.DumpDatabase(r.Context(), db.Name)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DUMP_ERROR", "Failed to dump database", nil, "")
		return
	}

	db.BackupCount++
	db.BackupStatus = "1 Backup"
	_ = h.store.UpdateDatabase(r.Context(), db)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"database":      db.Name,
		"backup_status": db.BackupStatus,
		"backup_count":  db.BackupCount,
		"size_bytes":    len(dump),
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}, nil)
}

// Export dumps the database and streams the actual SQL dump directly as an attachment
func (h *DatabaseHandler) Export(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db")
	if dbName == "" {
		idStr := chi.URLParam(r, "id")
		if idStr != "" {
			if dbID, err := uuid.Parse(idStr); err == nil {
				if db, err := h.store.GetDatabaseByID(r.Context(), dbID); err == nil && db != nil {
					dbName = db.Name
				}
			}
		}
	}
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_NAME", "Database name is required", nil, "")
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "sql"
	}

	var tables []string
	if tblsStr := r.URL.Query().Get("tables"); tblsStr != "" {
		for _, t := range strings.Split(tblsStr, ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				tables = append(tables, trimmed)
			}
		}
	}

	timestamp := time.Now().Format("20060102_150405")
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		tableName := "table"
		if len(tables) > 0 {
			tableName = tables[0]
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_%s_%s.csv\"", dbName, tableName, timestamp))
	case "json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_%s.json\"", dbName, timestamp))
	default:
		w.Header().Set("Content-Type", "application/sql; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_%s.sql\"", dbName, timestamp))
	}

	if err := h.dbMgr.StreamExport(r.Context(), dbName, format, tables, true, true, w); err != nil {
		return
	}
}

// Import restores a SQL dump with buffered parsing and real statement execution metrics
func (h *DatabaseHandler) Import(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		idStr := chi.URLParam(r, "id")
		if id, err := uuid.Parse(idStr); err == nil {
			if db, err := h.store.GetDatabaseByID(r.Context(), id); err == nil && db != nil {
				dbName = db.Name
			}
		}
	}
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_NAME", "Database name is required", nil, "")
		return
	}

	var reader io.Reader
	// Check for multipart form file first
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(128 << 20) // 128 MB max memory
		if err != nil {
			response.Error(w, http.StatusBadRequest, "FORM_ERROR", "Failed to parse form: "+err.Error(), nil, "")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			response.Error(w, http.StatusBadRequest, "FILE_ERROR", "File upload missing 'file' key", nil, "")
			return
		}
		defer file.Close()
		reader = file
	} else {
		reader = r.Body
	}

	successCount, failCount, err := h.dbMgr.ImportSQL(r.Context(), dbName, reader)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "IMPORT_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"successful": successCount,
		"failed":     failCount,
		"total":      successCount + failCount,
	}, nil)
}

// RootPassword gets or sets root database password
func (h *DatabaseHandler) GetRootPassword(w http.ResponseWriter, r *http.Request) {
	pass := h.dbMgr.GetRootPassword()
	response.JSON(w, http.StatusOK, map[string]string{"root_password": pass}, nil)
}

func (h *DatabaseHandler) SetRootPassword(w http.ResponseWriter, r *http.Request) {
	var req RootPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Password) < 6 {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 6 characters", nil, "")
		return
	}

	if err := h.dbMgr.SetRootPassword(req.Password); err != nil {
		response.Error(w, http.StatusInternalServerError, "ERROR", "Failed to update root password", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"success": true}, nil)
}

// AutoBackup gets or sets auto backup setting
func (h *DatabaseHandler) GetAutoBackup(w http.ResponseWriter, r *http.Request) {
	enabled := h.dbMgr.GetAutoBackup()
	response.JSON(w, http.StatusOK, map[string]bool{"enabled": enabled}, nil)
}

func (h *DatabaseHandler) SetAutoBackup(w http.ResponseWriter, r *http.Request) {
	var req AutoBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid payload", nil, "")
		return
	}

	h.dbMgr.SetAutoBackup(req.Enabled)
	response.JSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled}, nil)
}

// AdvancedSetup gets or sets advanced MySQL config
func (h *DatabaseHandler) GetAdvancedSetup(w http.ResponseWriter, r *http.Request) {
	cfg := h.dbMgr.GetAdvancedConfig()
	response.JSON(w, http.StatusOK, cfg, nil)
}

func (h *DatabaseHandler) SetAdvancedSetup(w http.ResponseWriter, r *http.Request) {
	var cfg database.AdvancedConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid payload", nil, "")
		return
	}

	h.dbMgr.SetAdvancedConfig(cfg)
	response.JSON(w, http.StatusOK, cfg, nil)
}

// RecycleBin lists soft-deleted databases or restores them
func (h *DatabaseHandler) ListRecycleBin(w http.ResponseWriter, r *http.Request) {
	dbs, err := h.store.ListDatabasesByServer(r.Context(), uuid.Nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list databases", nil, "")
		return
	}

	var inTrash []*store.Database
	for _, d := range dbs {
		if d.InRecycleBin {
			inTrash = append(inTrash, d)
		}
	}

	response.JSON(w, http.StatusOK, inTrash, &response.Meta{Total: len(inTrash)})
}

func (h *DatabaseHandler) RestoreRecycleBin(w http.ResponseWriter, r *http.Request) {
	dbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid database UUID", nil, "")
		return
	}

	if err := h.store.RestoreDatabase(r.Context(), dbID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to restore database", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"restored": true}, nil)
}

// Batch performs bulk operations
func (h *DatabaseHandler) Batch(w http.ResponseWriter, r *http.Request) {
	var req BatchOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "IDs required", nil, "")
		return
	}

	affected := 0
	for _, idStr := range req.IDs {
		if id, err := uuid.Parse(idStr); err == nil {
			switch req.Action {
			case "delete":
				if err := h.store.DeleteDatabase(r.Context(), id); err == nil {
					affected++
				}
			case "backup":
				if db, err := h.store.GetDatabaseByID(r.Context(), id); err == nil {
					db.BackupCount++
					db.BackupStatus = "1 Backup"
					_ = h.store.UpdateDatabase(r.Context(), db)
					affected++
				}
			case "optimize":
				if db, err := h.store.GetDatabaseByID(r.Context(), id); err == nil {
					_, _ = h.dbMgr.RunDatabaseTools(r.Context(), db.Name, "optimize")
					affected++
				}
			}
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"action":   req.Action,
		"affected": affected,
		"total":    len(req.IDs),
	}, nil)
}

func (h *DatabaseHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateDatabaseUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Username and password (min 6 chars) required", nil, "")
		return
	}

	var serverID uuid.UUID
	if req.ServerID != "" {
		if id, err := uuid.Parse(req.ServerID); err == nil {
			serverID = id
		}
	}

	if req.HostAllow == "" {
		req.HostAllow = "localhost"
	}
	if req.DBType == "" {
		req.DBType = "mysql"
	}

	user := &store.DatabaseUser{
		ID:        uuid.New(),
		ServerID:  serverID,
		DBType:    req.DBType,
		Username:  req.Username,
		HostAllow: req.HostAllow,
	}

	if err := h.store.CreateDatabaseUser(r.Context(), user); err != nil {
		response.Error(w, http.StatusConflict, "USER_EXISTS", "Database user already exists", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "database_user.create", "database_user", user.ID.String(), "success", "", map[string]interface{}{
		"username": user.Username,
	})

	response.JSON(w, http.StatusCreated, user, nil)
}

// GetTables returns the live tables for a given database name
func (h *DatabaseHandler) GetTables(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		idStr := chi.URLParam(r, "id")
		if id, err := uuid.Parse(idStr); err == nil {
			if db, err := h.store.GetDatabaseByID(r.Context(), id); err == nil {
				dbName = db.Name
			}
		}
	}
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database name or ID required", nil, "")
		return
	}

	tables, err := h.dbMgr.GetDatabaseTables(r.Context(), dbName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TABLES_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"database": dbName,
		"tables":   tables,
	}, nil)
}

// ExecuteQuery runs a SQL command on the requested database
func (h *DatabaseHandler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database string `json:"database"`
		Query    string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Query == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database name and SQL query required", nil, "")
		return
	}

	res, err := h.dbMgr.ExecuteQuery(r.Context(), req.Database, req.Query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, res, nil)
}

// GetColumns returns column schema metadata for a database table
func (h *DatabaseHandler) GetColumns(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	tableName := strings.TrimSpace(r.URL.Query().Get("table"))
	if dbName == "" || tableName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database name and table name required", nil, "")
		return
	}

	cols, err := h.dbMgr.GetDatabaseColumns(r.Context(), dbName, tableName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "COLUMNS_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"database": dbName,
		"table":    tableName,
		"columns":  cols,
	}, nil)
}

// GetTree returns real-time database schema tree with tables, views, procedures, functions, events, triggers
func (h *DatabaseHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.dbMgr.GetDatabaseTree(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TREE_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, tree, nil)
}

// GetTableDetails returns live metrics for tables in a database
func (h *DatabaseHandler) GetTableDetails(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		idStr := chi.URLParam(r, "id")
		if id, err := uuid.Parse(idStr); err == nil {
			if db, err := h.store.GetDatabaseByID(r.Context(), id); err == nil && db != nil {
				dbName = db.Name
			}
		}
	}
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database name or ID required", nil, "")
		return
	}

	details, err := h.dbMgr.GetLiveTablesDetails(r.Context(), dbName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DETAILS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"database": dbName,
		"tables":   details,
	}, nil)
}

// GetTableStructure returns columns, indexes, and foreign keys
func (h *DatabaseHandler) GetTableStructure(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	tableName := strings.TrimSpace(r.URL.Query().Get("table"))
	if dbName == "" || tableName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database and table name required", nil, "")
		return
	}

	structure, err := h.dbMgr.GetLiveTableStructure(r.Context(), dbName, tableName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STRUCTURE_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, structure, nil)
}

// BrowseRows returns paginated table rows with primary key detection
func (h *DatabaseHandler) BrowseRows(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	tableName := strings.TrimSpace(r.URL.Query().Get("table"))
	if dbName == "" || tableName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database and table name required", nil, "")
		return
	}

	page := 1
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		fmt.Sscanf(pStr, "%d", &page)
	}
	limit := 25
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}

	sortCol := r.URL.Query().Get("sort_column")
	sortOrder := r.URL.Query().Get("sort_order")
	search := r.URL.Query().Get("search")

	res, err := h.dbMgr.BrowseTableRows(r.Context(), database.BrowseRowsParams{
		Database:   dbName,
		Table:      tableName,
		Page:       page,
		Limit:      limit,
		SortColumn: sortCol,
		SortOrder:  sortOrder,
		SearchWord: search,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "BROWSE_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, res, nil)
}

// InsertRow inserts a new row into the specified table
func (h *DatabaseHandler) InsertRow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database string                 `json:"database"`
		Table    string                 `json:"table"`
		Values   map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Table == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database, table, and values required", nil, "")
		return
	}

	insertID, err := h.dbMgr.InsertTableRow(r.Context(), req.Database, req.Table, req.Values)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INSERT_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"last_insert_id": insertID,
	}, nil)
}

// UpdateRow updates a row matching the primary key
func (h *DatabaseHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database      string                 `json:"database"`
		Table         string                 `json:"table"`
		PrimaryKeyCol string                 `json:"primary_key_col"`
		PrimaryVal    interface{}            `json:"primary_key_val"`
		Values        map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Table == "" || req.PrimaryKeyCol == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database, table, primary_key_col and values required", nil, "")
		return
	}

	affected, err := h.dbMgr.UpdateTableRow(r.Context(), req.Database, req.Table, req.PrimaryKeyCol, req.PrimaryVal, req.Values)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"rows_affected": affected,
	}, nil)
}

// DeleteRows deletes rows matching given primary key values
func (h *DatabaseHandler) DeleteRows(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database      string        `json:"database"`
		Table         string        `json:"table"`
		PrimaryKeyCol string        `json:"primary_key_col"`
		PrimaryVals   []interface{} `json:"primary_key_vals"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Table == "" || req.PrimaryKeyCol == "" || len(req.PrimaryVals) == 0 {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database, table, primary_key_col, and primary_key_vals required", nil, "")
		return
	}

	affected, err := h.dbMgr.DeleteTableRows(r.Context(), req.Database, req.Table, req.PrimaryKeyCol, req.PrimaryVals)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"rows_affected": affected,
	}, nil)
}

// ModifyColumn performs ADD, MODIFY, or DROP column
func (h *DatabaseHandler) ModifyColumn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database  string `json:"database"`
		Table     string `json:"table"`
		Action    string `json:"action"` // add, modify, drop
		Column    string `json:"column"`
		NewName   string `json:"new_name"`
		TypeDef   string `json:"type_def"`
		Collation string `json:"collation"`
		Null      string `json:"null"`
		Default   string `json:"default"`
		Extra     string `json:"extra"`
		Comment   string `json:"comment"`
		AfterCol  string `json:"after_col"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Table == "" || req.Action == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database, table, and action required", nil, "")
		return
	}

	var err error
	switch req.Action {
	case "add":
		err = h.dbMgr.AddColumn(r.Context(), req.Database, req.Table, req.Column, req.TypeDef, req.Collation, req.Null, req.Default, req.Extra, req.AfterCol)
	case "modify":
		err = h.dbMgr.ModifyColumn(r.Context(), req.Database, req.Table, req.Column, req.NewName, req.TypeDef, req.Collation, req.Null, req.Default, req.Extra)
	case "drop":
		err = h.dbMgr.DropColumn(r.Context(), req.Database, req.Table, req.Column)
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be add, modify, or drop", nil, "")
		return
	}

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "COLUMN_OPERATION_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"success": true}, nil)
}

// ModifyIndex performs ADD or DROP index
func (h *DatabaseHandler) ModifyIndex(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database  string   `json:"database"`
		Table     string   `json:"table"`
		Action    string   `json:"action"` // add, drop
		IndexName string   `json:"index_name"`
		IndexType string   `json:"index_type"` // PRIMARY, UNIQUE, INDEX, FULLTEXT
		Columns   []string `json:"columns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Table == "" || req.Action == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database, table, and action required", nil, "")
		return
	}

	var err error
	switch req.Action {
	case "add":
		err = h.dbMgr.AddIndex(r.Context(), req.Database, req.Table, req.IndexName, req.IndexType, req.Columns)
	case "drop":
		err = h.dbMgr.DropIndex(r.Context(), req.Database, req.Table, req.IndexName)
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be add or drop", nil, "")
		return
	}

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INDEX_OPERATION_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"success": true}, nil)
}

// TableOperations performs rename, copy, truncate, drop, maintenance, or collation update
func (h *DatabaseHandler) TableOperations(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Database  string `json:"database"`
		Table     string `json:"table"`
		Action    string `json:"action"` // rename, copy, truncate, drop, optimize, check, analyze, repair, collation
		NewName   string `json:"new_name"`
		CopyData  bool   `json:"copy_data"`
		Collation string `json:"collation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Database == "" || req.Action == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Database and action required", nil, "")
		return
	}

	var err error
	var maintMessage string
	switch req.Action {
	case "rename":
		if req.Table == "" || req.NewName == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Table and new_name required", nil, "")
			return
		}
		err = h.dbMgr.RenameTable(r.Context(), req.Database, req.Table, req.NewName)
	case "copy":
		if req.Table == "" || req.NewName == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Table and new_name required", nil, "")
			return
		}
		err = h.dbMgr.CopyTable(r.Context(), req.Database, req.Table, req.NewName, req.CopyData)
	case "truncate":
		if req.Table == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Table required", nil, "")
			return
		}
		err = h.dbMgr.TruncateTable(r.Context(), req.Database, req.Table)
	case "drop":
		if req.Table == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Table required", nil, "")
			return
		}
		err = h.dbMgr.DropTable(r.Context(), req.Database, req.Table)
	case "optimize", "check", "analyze", "repair":
		if req.Table == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Table required", nil, "")
			return
		}
		maintMessage, err = h.dbMgr.RunTableMaintenance(r.Context(), req.Database, req.Table, req.Action)
	case "collation":
		if req.Collation == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Collation required", nil, "")
			return
		}
		err = h.dbMgr.AlterDatabaseCollation(r.Context(), req.Database, req.Collation)
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_ACTION", "Unknown table operation", nil, "")
		return
	}

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "OPERATION_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"maintenance": maintMessage,
	}, nil)
}

func (h *DatabaseHandler) ListViews(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database required", nil, "")
		return
	}
	views, err := h.dbMgr.GetViews(r.Context(), dbName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "VIEWS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, views, nil)
}

func (h *DatabaseHandler) ListRoutines(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database required", nil, "")
		return
	}
	routines, err := h.dbMgr.GetRoutines(r.Context(), dbName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ROUTINES_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, routines, nil)
}

func (h *DatabaseHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database required", nil, "")
		return
	}
	events, err := h.dbMgr.GetEvents(r.Context(), dbName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "EVENTS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, events, nil)
}

func (h *DatabaseHandler) ListTriggers(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Database required", nil, "")
		return
	}
	triggers, err := h.dbMgr.GetTriggers(r.Context(), dbName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TRIGGERS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, triggers, nil)
}

