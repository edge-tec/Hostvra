package handlers

import (
	"net/http"
	"strconv"

	"hostvra/api/internal/auth"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AuditHandler struct {
	store store.Store
}

func NewAuditHandler(s store.Store) *AuditHandler {
	return &AuditHandler{store: s}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	limit := 50
	if limitQuery := r.URL.Query().Get("limit"); limitQuery != "" {
		if l, err := strconv.Atoi(limitQuery); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	logs, err := h.store.ListAuditLogsByOrg(r.Context(), claims.OrganizationID, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve audit trail", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, logs, &response.Meta{
		Limit: limit,
		Total: len(logs),
	})
}
