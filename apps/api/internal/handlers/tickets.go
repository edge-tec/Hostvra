package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type SupportHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewSupportHandler(cfg *config.Config, s store.Store, a *audit.Logger) *SupportHandler {
	return &SupportHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

// ----------------------------------------------------------------------------
// Support Tickets Handlers
// ----------------------------------------------------------------------------

func (h *SupportHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	status := r.URL.Query().Get("status")
	dept := r.URL.Query().Get("department")

	// Superadmin/Owner can see all org tickets, regular users see org tickets
	tickets, err := h.store.ListTickets(r.Context(), claims.OrganizationID, status, dept)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to list tickets", err.Error(), "")
		return
	}

	if tickets == nil {
		tickets = []store.Ticket{}
	}

	response.JSON(w, http.StatusOK, tickets, nil)
}

func (h *SupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ticket ID format", nil, "")
		return
	}

	ticket, err := h.store.GetTicket(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Ticket not found", nil, "")
		return
	}

	replies, err := h.store.ListTicketReplies(r.Context(), id)
	if err != nil {
		replies = []store.TicketReply{}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"ticket":  ticket,
		"replies": replies,
	}, nil)
}

type CreateTicketRequest struct {
	Subject        string                 `json:"subject"`
	Department     store.TicketDepartment `json:"department"`
	Priority       store.TicketPriority   `json:"priority"`
	RelatedService string                 `json:"related_service,omitempty"`
	Message        string                 `json:"message"`
}

func (h *SupportHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", err.Error(), "")
		return
	}

	if strings.TrimSpace(req.Subject) == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Ticket subject is required", nil, "")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Initial message is required", nil, "")
		return
	}

	if req.Department == "" {
		req.Department = store.DeptTechnical
	}
	if req.Priority == "" {
		req.Priority = store.PriorityMedium
	}

	userName := claims.Email
	if idx := strings.Index(userName, "@"); idx != -1 {
		userName = userName[:idx]
	}

	ticket := &store.Ticket{
		ID:             uuid.New(),
		OrganizationID: claims.OrganizationID,
		UserID:         claims.UserID,
		UserEmail:      claims.Email,
		UserName:       userName,
		Department:     req.Department,
		Priority:       req.Priority,
		Status:         store.TicketStatusOpen,
		Subject:        strings.TrimSpace(req.Subject),
		RelatedService: strings.TrimSpace(req.RelatedService),
	}

	if err := h.store.CreateTicket(r.Context(), ticket, strings.TrimSpace(req.Message)); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to create ticket", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "ticket.create", "ticket", ticket.ID.String(), "success", fmt.Sprintf("Created support ticket %s: %s", ticket.TicketNumber, ticket.Subject), nil)

	response.JSON(w, http.StatusCreated, ticket, nil)
}

type ReplyTicketRequest struct {
	Message     string   `json:"message"`
	Attachments []string `json:"attachments,omitempty"`
}

func (h *SupportHandler) ReplyTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	idStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ticket ID format", nil, "")
		return
	}

	_, err = h.store.GetTicket(r.Context(), ticketID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Ticket not found", nil, "")
		return
	}

	var req ReplyTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", err.Error(), "")
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Message cannot be empty", nil, "")
		return
	}

	isStaff := claims.Role == "owner" || claims.Role == "superadmin" || claims.Role == "admin"
	userName := claims.Email
	if isStaff {
		userName = "Hostvra Support Staff"
	} else if idx := strings.Index(userName, "@"); idx != -1 {
		userName = userName[:idx]
	}

	reply := &store.TicketReply{
		ID:          uuid.New(),
		TicketID:    ticketID,
		UserID:      claims.UserID,
		UserEmail:   claims.Email,
		UserName:    userName,
		IsStaff:     isStaff,
		Message:     strings.TrimSpace(req.Message),
		Attachments: req.Attachments,
	}

	if err := h.store.AddTicketReply(r.Context(), reply); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to add reply", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "ticket.reply", "ticket", ticketID.String(), "success", "Added reply to ticket", nil)

	response.JSON(w, http.StatusCreated, reply, nil)
}

func (h *SupportHandler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	idStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ticket ID format", nil, "")
		return
	}

	if err := h.store.UpdateTicketStatus(r.Context(), ticketID, store.TicketStatusClosed); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Ticket not found", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ticket.close", "ticket", ticketID.String(), "success", "Closed support ticket", nil)

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Ticket has been marked as closed.",
	}, nil)
}

// ----------------------------------------------------------------------------
// Knowledgebase Articles Handlers
// ----------------------------------------------------------------------------

func (h *SupportHandler) ListArticles(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	query := r.URL.Query().Get("q")

	articles, err := h.store.ListKnowledgeArticles(r.Context(), category, query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to list knowledge articles", err.Error(), "")
		return
	}

	if articles == nil {
		articles = []store.KnowledgeArticle{}
	}

	response.JSON(w, http.StatusOK, articles, nil)
}

func (h *SupportHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "idOrSlug")
	if idOrSlug == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Article identifier is required", nil, "")
		return
	}

	article, err := h.store.GetKnowledgeArticle(r.Context(), idOrSlug)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Article not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, article, nil)
}

func (h *SupportHandler) VoteArticle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid article ID", nil, "")
		return
	}

	var req struct {
		Helpful bool `json:"helpful"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", err.Error(), "")
		return
	}

	if err := h.store.VoteKnowledgeArticle(r.Context(), id, req.Helpful); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Article not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Thank you for your feedback!",
	}, nil)
}

func (h *SupportHandler) SaveArticle(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || (claims.Role != "owner" && claims.Role != "superadmin" && claims.Role != "admin") {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Admin permissions required", nil, "")
		return
	}

	var req store.KnowledgeArticle
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", err.Error(), "")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Title and content are required", nil, "")
		return
	}

	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Title), " ", "-"))
	}
	req.IsPublished = true

	if err := h.store.SaveKnowledgeArticle(r.Context(), &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to save article", err.Error(), "")
		return
	}

	h.audit.Log(r.Context(), r, "article.save", "article", req.Slug, "success", fmt.Sprintf("Saved knowledge article %s", req.Title), nil)

	response.JSON(w, http.StatusOK, req, nil)
}
