package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/response"
)

type DNSHandler struct {
	cfg        *config.Config
	dnsService *dns.Service
	audit      *audit.Logger
}

func NewDNSHandler(cfg *config.Config, dnsService *dns.Service, a *audit.Logger) *DNSHandler {
	return &DNSHandler{
		cfg:        cfg,
		dnsService: dnsService,
		audit:      a,
	}
}

type CreateZoneRequest struct {
	Domain   string `json:"domain"`
	Provider string `json:"provider"`
}

type CreateRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
	Proxied  bool   `json:"proxied"`
}

func (h *DNSHandler) ListZones(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	zones := h.dnsService.ListZones(r.Context(), claims.OrganizationID)
	response.JSON(w, http.StatusOK, zones, &response.Meta{Total: len(zones)})
}

func (h *DNSHandler) CreateZone(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Domain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Domain name is required", nil, "")
		return
	}
	if req.Provider == "" {
		req.Provider = "local"
	}

	zone, err := h.dnsService.CreateZone(r.Context(), claims.OrganizationID, req.Domain, req.Provider)
	if err != nil {
		response.Error(w, http.StatusConflict, "ZONE_EXISTS", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "dns.zone.create", "dns_zone", zone.ID.String(), "success", "", map[string]interface{}{
		"domain":   zone.Domain,
		"provider": zone.Provider,
	})

	response.JSON(w, http.StatusCreated, zone, nil)
}

func (h *DNSHandler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	zoneIDStr := chi.URLParam(r, "zoneID")
	zoneID, err := uuid.Parse(zoneIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid zone ID", nil, "")
		return
	}

	if err := h.dnsService.DeleteZone(r.Context(), zoneID); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "dns.zone.delete", "dns_zone", zoneID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "zone_id": zoneID}, nil)
}

func (h *DNSHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	zoneIDStr := chi.URLParam(r, "zoneID")
	zoneID, err := uuid.Parse(zoneIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid zone ID", nil, "")
		return
	}

	records, err := h.dnsService.ListRecords(r.Context(), zoneID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, records, &response.Meta{Total: len(records)})
}

func (h *DNSHandler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	zoneIDStr := chi.URLParam(r, "zoneID")
	zoneID, err := uuid.Parse(zoneIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid zone ID", nil, "")
		return
	}

	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	rec := &dns.Record{
		ZoneID:   zoneID,
		Type:     dns.RecordType(req.Type),
		Name:     req.Name,
		Content:  req.Content,
		TTL:      req.TTL,
		Priority: req.Priority,
		Proxied:  req.Proxied,
	}

	if err := h.dnsService.AddRecord(r.Context(), rec); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "dns.record.create", "dns_record", rec.ID.String(), "success", "", map[string]interface{}{
		"type":    rec.Type,
		"name":    rec.Name,
		"content": rec.Content,
	})

	response.JSON(w, http.StatusCreated, rec, nil)
}

func (h *DNSHandler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	zoneIDStr := chi.URLParam(r, "zoneID")
	zoneID, err := uuid.Parse(zoneIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid zone ID", nil, "")
		return
	}

	recordIDStr := chi.URLParam(r, "recordID")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid record ID", nil, "")
		return
	}

	if err := h.dnsService.DeleteRecord(r.Context(), zoneID, recordID); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "dns.record.delete", "dns_record", recordID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "record_id": recordID}, nil)
}

func (h *DNSHandler) ExportBindZone(w http.ResponseWriter, r *http.Request) {
	zoneIDStr := chi.URLParam(r, "zoneID")
	zoneID, err := uuid.Parse(zoneIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid zone ID", nil, "")
		return
	}

	content, err := h.dnsService.GenerateBindZoneFile(r.Context(), zoneID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}
