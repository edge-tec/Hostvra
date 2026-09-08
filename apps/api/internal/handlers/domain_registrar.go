package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/domains"
	"hostvra/api/internal/domains/resellerclub"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type DomainRegistrarHandler struct {
	cfg       *config.Config
	store     store.Store
	audit     *audit.Logger
	domainSvc *domains.Service
}

func NewDomainRegistrarHandler(cfg *config.Config, s store.Store, a *audit.Logger) *DomainRegistrarHandler {
	// Initialize ResellerClub client as default registrar
	rcClient, _ := resellerclub.NewClient(resellerclub.Config{
		ResellerID: cfg.ResellerClubResellerID,
		APIKey:     cfg.ResellerClubAPIKey,
		Mode:       cfg.ResellerClubMode,
		BaseURL:    cfg.ResellerClubAPIBaseURL,
		Timeout:    cfg.ResellerClubAPITimeout,
	})

	domainService := domains.NewService(s, rcClient, cfg.DomainEncryptionSecret)

	return &DomainRegistrarHandler{
		cfg:       cfg,
		store:     s,
		audit:     a,
		domainSvc: domainService,
	}
}

func (h *DomainRegistrarHandler) SetDomainService(svc *domains.Service) {
	h.domainSvc = svc
}

// ----------------------------------------------------------------------------
// 1. Domain Search & Availability
// ----------------------------------------------------------------------------

type DomainSearchResultItem struct {
	Domain        string  `json:"domain"`
	TLD           string  `json:"tld"`
	Available     bool    `json:"available"`
	Status        string  `json:"status"` // available, unavailable, error
	RegisterPrice float64 `json:"register_price"`
	RenewPrice    float64 `json:"renew_price"`
	TransferPrice float64 `json:"transfer_price"`
	Currency      string  `json:"currency"`
	IsPopular     bool    `json:"is_popular"`
	Message       string  `json:"message,omitempty"`
}

func (h *DomainRegistrarHandler) SearchDomains(w http.ResponseWriter, r *http.Request) {
	rawQuery := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
	if rawQuery == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Search query is required", nil, "")
		return
	}

	results, err := h.domainSvc.Availability.SearchDomain(r.Context(), rawQuery, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SEARCH_FAILED", "Domain availability check failed", err.Error(), "")
		return
	}

	var items []DomainSearchResultItem
	for _, res := range results {
		items = append(items, DomainSearchResultItem{
			Domain:        res.Domain,
			TLD:           res.TLD,
			Available:     res.Available,
			Status:        res.Status,
			RegisterPrice: res.RegisterPrice,
			RenewPrice:    res.RenewPrice,
			TransferPrice: res.TransferPrice,
			Currency:      res.Currency,
			IsPopular:     res.IsPopular,
			Message:       res.Message,
		})
	}

	response.JSON(w, http.StatusOK, items, &response.Meta{Total: len(items)})
}

// ----------------------------------------------------------------------------
// 2. Whois & Live Domain Info
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) WhoisLookup(w http.ResponseWriter, r *http.Request) {
	domainParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if domainParam == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Domain parameter is required", nil, "")
		return
	}

	cleanDomain, tld, err := domains.ValidateDomainName(domainParam)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN", err.Error(), nil, "")
		return
	}

	info, err := h.domainSvc.Registrar.GetDomainInfo(r.Context(), cleanDomain)
	if err != nil {
		// Return basic whois record
		response.JSON(w, http.StatusOK, &store.WhoisRecord{
			Domain:    cleanDomain,
			TLD:       "." + tld,
			Available: false,
			CheckedAt: time.Now().UTC(),
		}, nil)
		return
	}

	var regDateStr, expDateStr string
	if info.RegistrationDate != nil {
		regDateStr = info.RegistrationDate.Format(time.RFC3339)
	}
	if info.ExpiryDate != nil {
		expDateStr = info.ExpiryDate.Format(time.RFC3339)
	}

	rec := &store.WhoisRecord{
		Domain:         cleanDomain,
		TLD:            "." + tld,
		Available:      false,
		Registrar:      "Hostvra Domain Reseller (ResellerClub)",
		CreationDate:   regDateStr,
		ExpirationDate: expDateStr,
		NameServers:    info.Nameservers,
		Status:         []string{info.Status},
		CheckedAt:      time.Now().UTC(),
	}

	response.JSON(w, http.StatusOK, rec, nil)
}

// ----------------------------------------------------------------------------
// 3. Customer Domains Listing & Details
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	list, err := h.store.ListDomainsByUserID(r.Context(), claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve domains", err.Error(), "")
		return
	}

	response.JSON(w, http.StatusOK, list, &response.Meta{Total: len(list)})
}

func (h *DomainRegistrarHandler) authorizeDomain(r *http.Request, domainID uuid.UUID) (*store.Domain, error) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		return nil, errors.New("unauthorized")
	}

	d, err := h.store.GetDomainByID(r.Context(), domainID)
	if err != nil {
		return nil, errors.New("not_found")
	}

	if claims.Role != "admin" && claims.Role != "owner" && d.UserID != claims.UserID {
		return nil, errors.New("forbidden")
	}

	return d, nil
}

func (h *DomainRegistrarHandler) handleAuthError(w http.ResponseWriter, err error) {
	switch err.Error() {
	case "unauthorized":
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
	case "not_found":
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Domain not found", nil, "")
	case "forbidden":
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Unauthorized access to this domain", nil, "")
	default:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
	}
}

func (h *DomainRegistrarHandler) GetDomain(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	domainID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, err := h.authorizeDomain(r, domainID)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, d, nil)
}

// ----------------------------------------------------------------------------
// 4. Domain Order & Registration Flow
// ----------------------------------------------------------------------------

type OrderDomainRequest struct {
	Domain        string               `json:"domain"`
	Years         int                  `json:"years"`
	Nameservers   []string             `json:"nameservers"`
	PaymentMethod string               `json:"payment_method"`
	AutoRenew     bool                 `json:"auto_renew"`
	Registrant    *domains.ContactInfo `json:"registrant"`
}

func (h *DomainRegistrarHandler) OrderDomain(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req OrderDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Registrant == nil {
		email := claims.Email
		if email == "" {
			cleanDomain := strings.ToLower(strings.TrimSpace(req.Domain))
			if cleanDomain != "" {
				email = "admin@" + cleanDomain
			} else {
				email = "owner@hostvra.local"
			}
		}
		req.Registrant = &domains.ContactInfo{
			FirstName:  "Account",
			LastName:   "Owner",
			Email:      email,
			Phone:      "15551234567",
			Address1:   "100 Hostvra Way",
			City:       "Wilmington",
			State:      "DE",
			PostalCode: "19801",
			Country:    "US",
		}
	}

	res, err := h.domainSvc.Orders.CreateRegistrationOrder(r.Context(), domains.CreateRegistrationOrderRequest{
		UserID:         claims.UserID,
		OrganizationID: &claims.OrganizationID,
		DomainName:     req.DomainDomainOrEmpty(req.Domain),
		Years:          req.Years,
		Nameservers:    req.Nameservers,
		Registrant:     req.Registrant,
		PaymentMethod:  req.PaymentMethod,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ORDER_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.order.create", "domain_order", res.Order.ID.String(), "success", fmt.Sprintf("Created order for %s ($%.2f)", res.Order.DomainName, res.Order.Amount), nil)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"order":   res.Order,
		"domain":  res.Order.DomainName,
		"years":   res.Order.Years,
		"amount":  res.Order.Amount,
		"invoice": res.Invoice,
		"message": res.Message,
	}, nil)
}

func (r *OrderDomainRequest) DomainDomainOrEmpty(d string) string {
	return strings.ToLower(strings.TrimSpace(d))
}

func (h *DomainRegistrarHandler) GetDomainOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid order ID", nil, "")
		return
	}

	order, err := h.store.GetDomainOrderByID(r.Context(), orderID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Order not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, order, nil)
}

// ----------------------------------------------------------------------------
// 5. Nameservers Management
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) GetNameservers(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	_, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	nsList, err := h.domainSvc.Nameservers.GetNameservers(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "NAMESERVERS_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_id":   domainID,
		"nameservers": nsList,
	}, nil)
}

func (h *DomainRegistrarHandler) UpdateNameservers(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	_, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	var req struct {
		Nameservers []string `json:"nameservers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if err := h.domainSvc.Nameservers.UpdateNameservers(r.Context(), domainID, req.Nameservers); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.nameservers.update", "domain", domainID.String(), "success", "Updated nameservers", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"nameservers": req.Nameservers,
		"message":     "Nameservers updated successfully",
	}, nil)
}

// ----------------------------------------------------------------------------
// 6. DNS Records Management
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) ListDNSRecords(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	_, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	records, err := h.domainSvc.DNS.ListRecords(r.Context(), domainID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DNS_LIST_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, records, &response.Meta{Total: len(records)})
}

func (h *DomainRegistrarHandler) CreateDNSRecord(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	_, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	var req struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Value    string `json:"value"`
		TTL      int    `json:"ttl"`
		Priority int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	rec, err := h.domainSvc.DNS.CreateRecord(r.Context(), domainID, req.Type, req.Name, req.Value, req.TTL, req.Priority)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "DNS_CREATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.dns.create", "domain_dns_record", rec.ID.String(), "success", fmt.Sprintf("Created DNS %s record %s", req.Type, req.Name), nil)

	response.JSON(w, http.StatusCreated, rec, nil)
}

func (h *DomainRegistrarHandler) DeleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	_, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	recordID, err := uuid.Parse(chi.URLParam(r, "recordId"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_RECORD_ID", "Invalid record ID", nil, "")
		return
	}

	if err := h.domainSvc.DNS.DeleteRecord(r.Context(), domainID, recordID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DNS_DELETE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.dns.delete", "domain_dns_record", recordID.String(), "success", "Deleted DNS record", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "DNS record deleted"}, nil)
}

// ----------------------------------------------------------------------------
// 7. Security: Registrar Lock & EPP Auth Code
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) GetLock(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	locked, err := h.domainSvc.Registrar.GetRegistrarLock(r.Context(), d.DomainName)
	if err == nil {
		d.RegistrarLock = locked
		_ = h.store.UpdateDomain(r.Context(), d)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_id":      domainID,
		"domain_name":    d.DomainName,
		"registrar_lock": d.RegistrarLock,
	}, nil)
}

func (h *DomainRegistrarHandler) SetLock(w http.ResponseWriter, r *http.Request) {
	h.toggleLock(w, r, true)
}

func (h *DomainRegistrarHandler) SetUnlock(w http.ResponseWriter, r *http.Request) {
	h.toggleLock(w, r, false)
}

func (h *DomainRegistrarHandler) toggleLock(w http.ResponseWriter, r *http.Request, lock bool) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	if err := h.domainSvc.Registrar.SetRegistrarLock(r.Context(), d.DomainName, lock); err != nil {
		response.Error(w, http.StatusInternalServerError, "LOCK_UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	d.RegistrarLock = lock
	_ = h.store.UpdateDomain(r.Context(), d)

	action := "DOMAIN_LOCK_ENABLED"
	if !lock {
		action = "DOMAIN_LOCK_DISABLED"
	}
	h.audit.Log(r.Context(), r, "domains.lock.toggle", "domain", domainID.String(), "success", action, nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_id":      domainID,
		"registrar_lock": lock,
		"message":        fmt.Sprintf("Registrar theft protection set to %v", lock),
	}, nil)
}

func (h *DomainRegistrarHandler) GetEPPCode(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	epp, err := h.domainSvc.Registrar.GetEPPCode(r.Context(), d.DomainName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "EPP_FAILED", "Failed to retrieve authorization code from registrar", nil, "")
		return
	}

	// Never log actual EPP code
	h.audit.Log(r.Context(), r, "domains.epp.retrieve", "domain", domainID.String(), "success", "Retrieved transfer EPP authorization code", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_name": d.DomainName,
		"epp_code":    epp,
	}, nil)
}

func (h *DomainRegistrarHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	contacts, err := h.domainSvc.Registrar.GetContacts(r.Context(), d.DomainName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CONTACTS_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, contacts, nil)
}

func (h *DomainRegistrarHandler) UpdateContacts(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	var req domains.ContactUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}
	req.DomainName = d.DomainName

	if err := h.domainSvc.Registrar.UpdateContacts(r.Context(), req); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "domains.contacts.update", "domain", domainID.String(), "success", "Updated WHOIS contacts", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Contacts updated successfully"}, nil)
}

// ----------------------------------------------------------------------------
// 8. Renewal & Transfer
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) RenewDomain(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	_, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	var req struct {
		Years int `json:"years"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Years < 1 {
		req.Years = 1
	}

	inv, err := h.domainSvc.Renewals.RequestRenewal(r.Context(), domainID, claims.UserID, req.Years)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "RENEWAL_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"invoice": inv,
		"message": "Renewal order created. Please pay invoice to complete renewal.",
	}, nil)
}

func (h *DomainRegistrarHandler) TransferDomain(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req struct {
		Domain     string               `json:"domain"`
		AuthCode   string               `json:"auth_code"`
		Registrant *domains.ContactInfo `json:"registrant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Registrant == nil {
		email := claims.Email
		if email == "" {
			email = "owner@hostvra.local"
		}
		req.Registrant = &domains.ContactInfo{
			FirstName:  "Transfer",
			LastName:   "Owner",
			Email:      email,
			Phone:      "15551234567",
			Address1:   "100 Hostvra Way",
			City:       "Wilmington",
			State:      "DE",
			PostalCode: "19801",
			Country:    "US",
		}
	}

	record, inv, err := h.domainSvc.Transfers.InitiateTransfer(r.Context(), domains.InitiateTransferRequest{
		UserID:         claims.UserID,
		OrganizationID: &claims.OrganizationID,
		DomainName:     req.Domain,
		AuthCode:       req.AuthCode,
		Registrant:     req.Registrant,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, "TRANSFER_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"transfer": record,
		"invoice":  inv,
		"message":  "Transfer initiated. Complete invoice payment to submit to registrar.",
	}, nil)
}

func (h *DomainRegistrarHandler) ToggleAutoRenew(w http.ResponseWriter, r *http.Request) {
	domainID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid domain ID", nil, "")
		return
	}

	d, authErr := h.authorizeDomain(r, domainID)
	if authErr != nil {
		h.handleAuthError(w, authErr)
		return
	}

	d.AutoRenew = !d.AutoRenew
	_ = h.store.UpdateDomain(r.Context(), d)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"domain_id":  domainID,
		"auto_renew": d.AutoRenew,
	}, nil)
}

func (h *DomainRegistrarHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil || (claims.Role != "admin" && claims.Role != "owner") {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Admin permission required", nil, "")
		return
	}

	result, err := h.domainSvc.Registrar.TestConnection(r.Context())
	if err != nil {
		response.Error(w, http.StatusBadGateway, "REGISTRAR_CONNECTION_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, result, nil)
}

// ----------------------------------------------------------------------------
// 9. Legacy Compatibility Endpoints
// ----------------------------------------------------------------------------

func (h *DomainRegistrarHandler) ListTLDs(w http.ResponseWriter, r *http.Request) {
	tlds, err := h.store.ListTLDPricings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve TLDs", err.Error(), "")
		return
	}
	response.JSON(w, http.StatusOK, tlds, &response.Meta{Total: len(tlds)})
}

func (h *DomainRegistrarHandler) UpdateTLD(w http.ResponseWriter, r *http.Request) {
	tld := chi.URLParam(r, "tld")
	var req store.TLDPricing
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	req.TLD = tld
	if err := h.store.SaveTLDPricing(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, req, nil)
}

func (h *DomainRegistrarHandler) ListRegistrars(w http.ResponseWriter, r *http.Request) {
	configs, err := h.store.ListRegistrarConfigs(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, configs, nil)
}

func (h *DomainRegistrarHandler) UpdateRegistrar(w http.ResponseWriter, r *http.Request) {
	registrar := chi.URLParam(r, "registrar")
	var req store.DomainRegistrarConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}
	req.Registrar = registrar
	_ = h.store.SaveRegistrarConfig(r.Context(), &req)
	response.JSON(w, http.StatusOK, req, nil)
}

// Unused suppressor
var _ = strconv.Itoa
