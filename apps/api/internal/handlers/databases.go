package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type DatabaseHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewDatabaseHandler(cfg *config.Config, s store.Store, a *audit.Logger) *DatabaseHandler {
	return &DatabaseHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

type CreateDatabaseRequest struct {
	ServerID     string `json:"server_id"`
	DBType       string `json:"db_type"` // mysql, mariadb, postgresql
	Name         string `json:"name"`
	CharacterSet string `json:"character_set"`
	Collation    string `json:"collation"`
}

type CreateDatabaseUserRequest struct {
	ServerID  string `json:"server_id"`
	DBType    string `json:"db_type"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	HostAllow string `json:"host_allow"`
}

func (h *DatabaseHandler) List(w http.ResponseWriter, r *http.Request) {
	serverIDStr := r.URL.Query().Get("server_id")
	if serverIDStr == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_SERVER_ID", "server_id query parameter required", nil, "")
		return
	}

	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_ID", "Invalid server UUID", nil, "")
		return
	}

	dbs, err := h.store.ListDatabasesByServer(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve databases", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, dbs, &response.Meta{
		Total: len(dbs),
	})
}

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

	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_ID", "Invalid server UUID", nil, "")
		return
	}

	if req.DBType == "" {
		req.DBType = "mysql"
	}

	db := &store.Database{
		ID:           uuid.New(),
		ServerID:     serverID,
		DBType:       req.DBType,
		Name:         req.Name,
		CharacterSet: req.CharacterSet,
		Collation:    req.Collation,
	}

	if err := h.store.CreateDatabase(r.Context(), db); err != nil {
		response.Error(w, http.StatusConflict, "DATABASE_EXISTS", "Database already exists", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "database.create", "database", db.ID.String(), "success", "", map[string]interface{}{
		"name":      db.Name,
		"db_type":   db.DBType,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusCreated, db, nil)
}

func (h *DatabaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	dbID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid database UUID", nil, "")
		return
	}

	if err := h.store.DeleteDatabase(r.Context(), dbID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete database", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "database.delete", "database", dbID.String(), "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true}, nil)
}

func (h *DatabaseHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateDatabaseUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 8 {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Username and password (min 8 chars) required", nil, "")
		return
	}

	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_ID", "Invalid server UUID", nil, "")
		return
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
