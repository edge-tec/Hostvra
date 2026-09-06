package handlers

import (
	"encoding/json"
	"net/http"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/license"
	"hostvra/api/internal/response"
)

type LicenseHandler struct {
	cfg     *config.Config
	manager *license.Manager
	audit   *audit.Logger
}

func NewLicenseHandler(cfg *config.Config, m *license.Manager, a *audit.Logger) *LicenseHandler {
	return &LicenseHandler{
		cfg:     cfg,
		manager: m,
		audit:   a,
	}
}

type ActivateLicenseRequest struct {
	LicenseKey string `json:"license_key"`
}

func (h *LicenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	lic := h.manager.GetActiveLicense()
	response.JSON(w, http.StatusOK, lic, nil)
}

func (h *LicenseHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req ActivateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.LicenseKey == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "License key is required", nil, "")
		return
	}

	payload, err := h.manager.ActivateKey(req.LicenseKey)
	if err != nil {
		response.Error(w, http.StatusPaymentRequired, "INVALID_LICENSE", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "license.activate", "license", payload.LicenseID, "success", "", map[string]interface{}{
		"tier":           payload.Tier,
		"customer_name":  payload.CustomerName,
		"customer_email": payload.CustomerEmail,
	})

	response.JSON(w, http.StatusOK, payload, nil)
}
