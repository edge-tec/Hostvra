package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type SettingsHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewSettingsHandler(cfg *config.Config, s store.Store, a *audit.Logger) *SettingsHandler {
	return &SettingsHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetSystemSettings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to retrieve system settings", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, settings, nil)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var s store.SystemSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if err := h.store.UpdateSystemSettings(r.Context(), &s); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to update system settings", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "system.settings.update", "system_settings", "global", "success", "", map[string]interface{}{
		"panel_port":        s.PanelPort,
		"panel_domain":      s.PanelDomain,
		"security_entrance": s.SecurityEntrance,
	})

	response.JSON(w, http.StatusOK, s, nil)
}

func (h *SettingsHandler) SyncTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	response.JSON(w, http.StatusOK, map[string]string{
		"server_time": now,
		"status":      "synchronized",
	}, nil)
}
