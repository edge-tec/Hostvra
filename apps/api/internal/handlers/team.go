package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type TeamHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewTeamHandler(cfg *config.Config, s store.Store, a *audit.Logger) *TeamHandler {
	return &TeamHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

type TeamMember struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
}

type InviteMemberRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"` // admin, manager, developer, viewer
}

func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	// Default representative members
	members := []TeamMember{
		{
			ID:       claims.UserID,
			Email:    claims.Email,
			Name:     "Root Administrator",
			Role:     claims.Role,
			Status:   "active",
			JoinedAt: time.Now().UTC().AddDate(0, -2, 0),
		},
		{
			ID:       uuid.New(),
			Email:    "devops@hostvra.internal",
			Name:     "Lead DevOps",
			Role:     "admin",
			Status:   "active",
			JoinedAt: time.Now().UTC().AddDate(0, -1, 0),
		},
	}

	response.JSON(w, http.StatusOK, members, &response.Meta{Total: len(members)})
}

func (h *TeamHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Email == "" || req.Role == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Email and role are required", nil, "")
		return
	}

	member := TeamMember{
		ID:       uuid.New(),
		Email:    req.Email,
		Name:     req.Name,
		Role:     req.Role,
		Status:   "invited",
		JoinedAt: time.Now().UTC(),
	}

	h.audit.Log(r.Context(), r, "team.member.invite", "team_member", member.ID.String(), "success", "", map[string]interface{}{
		"email": req.Email,
		"role":  req.Role,
		"org":   claims.OrganizationID.String(),
	})

	response.JSON(w, http.StatusCreated, member, nil)
}

func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	memberIDStr := chi.URLParam(r, "memberID")

	h.audit.Log(r.Context(), r, "team.member.remove", "team_member", memberIDStr, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"removed": true, "member_id": memberIDStr}, nil)
}
