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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/agenttask"
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

func TestAgentRemoteTaskExecutionLoop(t *testing.T) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(st, logger)
	cfg := &config.Config{JWTSecret: "test-secret-1234567890"}

	taskMgr := agenttask.NewManager()

	agentHandler := NewAgentHandler(cfg, st, auditLogger)
	agentHandler.SetTaskManager(taskMgr)

	serverHandler := NewServerHandler(cfg, st, auditLogger)
	serverHandler.SetTaskManager(taskMgr)

	orgID := uuid.New()
	serverID := uuid.New()
	rawAgentKey := "hv_agt_valid_test_key_12345"
	agentKeyHash := auth.HashOpaqueToken(rawAgentKey)

	_ = st.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "node-test",
		AgentTokenHash: agentKeyHash,
		Status:         "online",
	})

	adminClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Email:          "admin@hostvra.internal",
		Role:           "superadmin",
		IsSuperAdmin:   true,
	}

	// 1. Control plane dispatches task to server
	taskReqBody, _ := json.Marshal(CreateServerTaskRequest{
		Action: "service.restart",
		Payload: map[string]interface{}{
			"service": "nginx",
		},
	})
	dispatchReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID.String()+"/tasks", bytes.NewReader(taskReqBody))
	dispatchReq = dispatchReq.WithContext(auth.WithClaims(dispatchReq.Context(), adminClaims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID.String())
	dispatchReq = dispatchReq.WithContext(context.WithValue(dispatchReq.Context(), chi.RouteCtxKey, rctx))
	dispatchRec := httptest.NewRecorder()

	serverHandler.DispatchTask(dispatchRec, dispatchReq)
	if dispatchRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 StatusAccepted on task dispatch, got %d: %s", dispatchRec.Code, dispatchRec.Body.String())
	}

	var dispatchResp struct {
		Data *agenttask.AgentTask `json:"data"`
	}
	_ = json.Unmarshal(dispatchRec.Body.Bytes(), &dispatchResp)
	dispatchedTaskID := dispatchResp.Data.ID

	// 2. Node Agent sends heartbeat -> receives pending task in response
	hbBody, _ := json.Marshal(AgentHeartbeatRequest{
		UptimeSeconds: 120,
	})
	hbReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/heartbeat", bytes.NewReader(hbBody))
	hbReq.Header.Set("Authorization", "Bearer "+rawAgentKey)
	hbReq.Header.Set("X-Server-ID", serverID.String())
	hbRec := httptest.NewRecorder()

	agentHandler.Heartbeat(hbRec, hbReq)
	if hbRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on heartbeat, got %d", hbRec.Code)
	}

	var hbResp struct {
		Data struct {
			Acknowledged bool                   `json:"acknowledged"`
			Tasks        []*agenttask.AgentTask `json:"tasks"`
		} `json:"data"`
	}
	_ = json.Unmarshal(hbRec.Body.Bytes(), &hbResp)

	if len(hbResp.Data.Tasks) != 1 {
		t.Fatalf("expected 1 task in heartbeat response, got %d", len(hbResp.Data.Tasks))
	}
	if hbResp.Data.Tasks[0].ID != dispatchedTaskID {
		t.Fatalf("task ID mismatch: expected %s, got %s", dispatchedTaskID, hbResp.Data.Tasks[0].ID)
	}

	// 3. Node Agent executes task and reports result back
	resultBody, _ := json.Marshal(ReportTaskResultRequest{
		Status:   "completed",
		ExitCode: 0,
		Output:   "Nginx service reloaded cleanly without downtime",
	})
	resultReq := httptest.NewRequest(http.MethodPost, "/api/v1/agent/tasks/"+dispatchedTaskID.String()+"/result", bytes.NewReader(resultBody))
	resultReq.Header.Set("Authorization", "Bearer "+rawAgentKey)
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("id", dispatchedTaskID.String())
	resultReq = resultReq.WithContext(context.WithValue(resultReq.Context(), chi.RouteCtxKey, rCtx))
	resultRec := httptest.NewRecorder()

	agentHandler.ReportTaskResult(resultRec, resultReq)
	if resultRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on report task result, got %d: %s", resultRec.Code, resultRec.Body.String())
	}

	// 4. Control Plane / Admin queries task status -> verified completed
	getTaskReq := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/tasks/"+dispatchedTaskID.String(), nil)
	getTaskReq = getTaskReq.WithContext(auth.WithClaims(getTaskReq.Context(), adminClaims))
	taskCtx := chi.NewRouteContext()
	taskCtx.URLParams.Add("id", serverID.String())
	taskCtx.URLParams.Add("taskId", dispatchedTaskID.String())
	getTaskReq = getTaskReq.WithContext(context.WithValue(getTaskReq.Context(), chi.RouteCtxKey, taskCtx))
	getTaskRec := httptest.NewRecorder()

	serverHandler.GetTask(getTaskRec, getTaskReq)
	if getTaskRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get task, got %d: %s", getTaskRec.Code, getTaskRec.Body.String())
	}

	var getTaskResp struct {
		Data *agenttask.AgentTask `json:"data"`
	}
	_ = json.Unmarshal(getTaskRec.Body.Bytes(), &getTaskResp)
	if getTaskResp.Data.Status != agenttask.StatusCompleted || getTaskResp.Data.ExitCode != 0 {
		t.Fatalf("unexpected task final status: %+v", getTaskResp.Data)
	}
	if getTaskResp.Data.Output != "Nginx service reloaded cleanly without downtime" {
		t.Fatalf("unexpected task output: %s", getTaskResp.Data.Output)
	}
}
