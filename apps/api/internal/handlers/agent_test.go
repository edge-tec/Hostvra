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
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestAgentHandler_EnrollAndHeartbeatSecurity(t *testing.T) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(st, logger)
	cfg := &config.Config{JWTSecret: "test-secret-1234567890"}
	handler := NewAgentHandler(cfg, st, auditLogger)

	orgID := uuid.New()
	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Fleet Org",
		Slug: "fleet-org",
	})

	// 1. Create Enrollment Token in store
	rawToken := "hv_enr_test1234567890abcdef"
	tokenHash := auth.HashOpaqueToken(rawToken)
	tokenRecord := &store.ServerEnrollmentToken{
		ID:             uuid.New(),
		OrganizationID: orgID,
		TokenHash:      tokenHash,
		Label:          "Test Node",
		ExpiresAt:      time.Now().UTC().Add(1 * time.Hour),
	}
	_ = st.CreateEnrollmentToken(context.Background(), tokenRecord)

	// 2. Enroll Node
	enrollBody, _ := json.Marshal(AgentEnrollRequest{
		EnrollmentToken: rawToken,
		Name:            "prod-vps-01",
		Hostname:        "vps01.example.com",
		IPAddress:       "198.51.100.10",
		OSName:          "Ubuntu 24.04 LTS",
		CPUCores:        4,
		RAMTotalMB:      8192,
		DiskTotalGB:     160,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", bytes.NewReader(enrollBody))
	rec := httptest.NewRecorder()
	handler.Enroll(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Enroll failed: status %d, body: %s", rec.Code, rec.Body.String())
	}

	var enrollResp struct {
		Data struct {
			ServerID uuid.UUID `json:"server_id"`
			AgentKey string    `json:"agent_key"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &enrollResp)
	serverID := enrollResp.Data.ServerID
	agentKey := enrollResp.Data.AgentKey

	if serverID == uuid.Nil || agentKey == "" {
		t.Fatalf("invalid enroll response: %+v", enrollResp)
	}

	// 3. Send Heartbeat with Valid Agent Key -> Must succeed (200 OK)
	hbBody, _ := json.Marshal(AgentHeartbeatRequest{
		UptimeSeconds: 3600,
		Metric: &store.ServerMetric{
			CPUPercent: 12.5,
			RAMUsedMB:  4096,
			RAMTotalMB: 8192,
			DiskUsedGB: 50,
			DiskTotalGB: 160,
		},
	})

	hbReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", bytes.NewReader(hbBody))
	hbReq.Header.Set("Authorization", "Bearer "+agentKey)
	hbReq.Header.Set("X-Server-ID", serverID.String())
	hbRec := httptest.NewRecorder()
	handler.Heartbeat(hbRec, hbReq)

	if hbRec.Code != http.StatusOK {
		t.Fatalf("Valid heartbeat failed: status %d, body: %s", hbRec.Code, hbRec.Body.String())
	}

	// 4. Send Heartbeat with Forged / Invalid Agent Key -> Must be rejected (401 Unauthorized)
	forgedReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", bytes.NewReader(hbBody))
	forgedReq.Header.Set("Authorization", "Bearer hv_agt_forged_key_12345")
	forgedReq.Header.Set("X-Server-ID", serverID.String())
	forgedRec := httptest.NewRecorder()
	handler.Heartbeat(forgedRec, forgedReq)

	if forgedRec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected forged agent key to be rejected with 401, got status %d, body: %s", forgedRec.Code, forgedRec.Body.String())
	}
}
