package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func BenchmarkAPI_Health(b *testing.B) {
	h := NewHealthHandler("1.0.0")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.Health(rec, req)
	}
}

func BenchmarkAPI_AuthJWTVerification(b *testing.B) {
	secret := "test-jwt-secret-at-least-32-chars-long!"
	userID := uuid.New()
	orgID := uuid.New()
	tokenPair, _, _ := auth.GenerateTokenPair(userID, orgID, "admin@hostvra.com", "owner", true, secret, 24*time.Hour, 7*24*time.Hour)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := auth.ValidateAccessToken(tokenPair.AccessToken, secret)
		if err != nil {
			b.Fatalf("token validation failed: %v", err)
		}
	}
}

func BenchmarkAPI_DashboardGatherTelemetry(b *testing.B) {
	memStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	cfg := &config.Config{Port: "8080", Host: "127.0.0.1"}
	h := NewDashboardHandler(cfg, memStore, auditLogger)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = h.gatherTelemetry()
	}
}

func BenchmarkStore_DatabaseOperations(b *testing.B) {
	ctx := context.Background()
	s := store.NewMemoryStore()
	dbID := uuid.New()
	db := &store.Database{
		ID:           dbID,
		Name:         "bench_db",
		CharacterSet: "utf8mb4",
		Collation:    "utf8mb4_unicode_ci",
		CreatedAt:    time.Now(),
	}
	_ = s.CreateDatabase(ctx, db)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = s.GetDatabaseByID(ctx, dbID)
	}
}
