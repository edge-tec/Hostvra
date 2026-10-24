package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
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
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
}

type InviteMemberRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"` // admin, manager, developer, viewer
}

func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	users, err := h.store.ListUsers(r.Context())
	if err == nil && len(users) > 0 {
		members := make([]TeamMember, 0, len(users))
		for _, u := range users {
			if u.DefaultOrgID == claims.OrganizationID || claims.IsSuperAdmin {
				displayName := u.FullName
				if displayName == "" {
					displayName = u.Email
					if at := strings.Index(u.Email, "@"); at > 0 {
						displayName = strings.Title(strings.ReplaceAll(u.Email[:at], ".", " "))
					}
				}
				roleStr := u.Role
				if roleStr == "" {
					roleStr = "viewer"
				}
				statusStr := "active"
				if !u.IsActive {
					statusStr = "inactive"
				}
				members = append(members, TeamMember{
					ID:       u.ID,
					Email:    u.Email,
					Name:     displayName,
					Role:     roleStr,
					Status:   statusStr,
					JoinedAt: u.CreatedAt,
				})
			}
		}
		if len(members) > 0 {
			response.JSON(w, http.StatusOK, members, &response.Meta{Total: len(members)})
			return
		}
	}

	// Fallback current authenticated user
	members := []TeamMember{
		{
			ID:       claims.UserID,
			Email:    claims.Email,
			Name:     "Root Administrator",
			Role:     claims.Role,
			Status:   "active",
			JoinedAt: time.Now().UTC(),
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

	cleanEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if cleanEmail == "" || req.Role == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Email and role are required", nil, "")
		return
	}

	// Check if user already exists
	existing, _ := h.store.GetUserByEmail(r.Context(), cleanEmail)
	if existing != nil {
		response.Error(w, http.StatusConflict, "USER_EXISTS", "A user with this email address already exists", nil, "")
		return
	}

	// Generate temporary password
	randBytes := make([]byte, 12)
	_, _ = rand.Read(randBytes)
	tempPass := "Hv_" + hex.EncodeToString(randBytes)
	hash, err := auth.HashPassword(tempPass, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HASH_ERROR", "Password hashing failed", nil, "")
		return
	}

	displayName := strings.TrimSpace(req.Name)
	if displayName == "" {
		displayName = cleanEmail
	}

	user := &store.User{
		ID:           uuid.New(),
		Email:        cleanEmail,
		FullName:     displayName,
		PasswordHash: hash,
		DefaultOrgID: claims.OrganizationID,
		Role:         req.Role,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := h.store.CreateUser(r.Context(), user, claims.OrganizationID, req.Role); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create user account", nil, "")
		return
	}

	member := TeamMember{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.FullName,
		Role:     user.Role,
		Status:   "active",
		JoinedAt: user.CreatedAt,
	}

	h.audit.Log(r.Context(), r, "team.member.invite", "team_member", member.ID.String(), "success", "", map[string]interface{}{
		"email": req.Email,
		"role":  req.Role,
		"org":   claims.OrganizationID.String(),
	})

	response.JSON(w, http.StatusCreated, member, nil)
}

func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	memberIDStr := chi.URLParam(r, "memberID")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid member UUID", nil, "")
		return
	}

	if memberID == claims.UserID {
		response.Error(w, http.StatusBadRequest, "CANNOT_REMOVE_SELF", "You cannot remove your own account from the team", nil, "")
		return
	}

	_ = h.store.DeleteUser(r.Context(), memberID)

	h.audit.Log(r.Context(), r, "team.member.remove", "team_member", memberIDStr, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"removed": true, "member_id": memberIDStr}, nil)
}

