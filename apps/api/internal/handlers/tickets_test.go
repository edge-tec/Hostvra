package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestSupportHandler_Flow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewSupportHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         userID,
				OrganizationID: orgID,
				Email:          "client@example.com",
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Get("/support/tickets", h.ListTickets)
	r.Post("/support/tickets", h.CreateTicket)
	r.Get("/support/tickets/{id}", h.GetTicket)
	r.Post("/support/tickets/{id}/reply", h.ReplyTicket)
	r.Post("/support/tickets/{id}/close", h.CloseTicket)
	r.Get("/support/articles", h.ListArticles)
	r.Get("/support/articles/{idOrSlug}", h.GetArticle)
	r.Post("/support/articles/{id}/vote", h.VoteArticle)

	// 1. List seeded articles
	t.Run("ListArticles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/support/articles", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp struct {
			Success bool                     `json:"success"`
			Data    []store.KnowledgeArticle `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data) == 0 {
			t.Fatalf("expected seeded articles, got 0")
		}
	})

	// 2. Create Support Ticket
	var createdTicketID uuid.UUID
	t.Run("CreateTicket", func(t *testing.T) {
		payload := map[string]interface{}{
			"subject":         "Database Connection Timeout during backup",
			"department":      "technical",
			"priority":        "high",
			"related_service": "apexagency.com",
			"message":         "I am getting SQL server timeout error when running mysqldump at 2am.",
		}
		buf, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/support/tickets", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool         `json:"success"`
			Data    store.Ticket `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Data.Subject != "Database Connection Timeout during backup" {
			t.Fatalf("unexpected subject: %s", resp.Data.Subject)
		}
		createdTicketID = resp.Data.ID
	})

	// 3. Get Ticket & Replies
	t.Run("GetTicket", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/support/tickets/"+createdTicketID.String(), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Ticket  store.Ticket        `json:"ticket"`
				Replies []store.TicketReply `json:"replies"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data.Replies) != 1 {
			t.Fatalf("expected 1 initial reply, got %d", len(resp.Data.Replies))
		}
	})

	// 4. Reply to Ticket
	t.Run("ReplyTicket", func(t *testing.T) {
		payload := map[string]interface{}{
			"message": "We have adjusted the max_execution_time and net_read_timeout in my.cnf.",
		}
		buf, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/support/tickets/"+createdTicketID.String()+"/reply", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 5. Close Ticket
	t.Run("CloseTicket", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/support/tickets/"+createdTicketID.String()+"/close", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
