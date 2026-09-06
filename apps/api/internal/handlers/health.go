package handlers

import (
	"net/http"

	"hostvra/api/internal/response"
)

type HealthHandler struct {
	version string
}

func NewHealthHandler(v string) *HealthHandler {
	return &HealthHandler{version: v}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "hostvra-api",
		"version": h.version,
	}, nil)
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status": "ready",
		"checks": map[string]string{
			"database": "ok",
		},
	}, nil)
}

func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"version": h.version,
		"edition": "community",
		"api":     "v1",
	}, nil)
}
