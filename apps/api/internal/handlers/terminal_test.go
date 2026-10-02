package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/store"
)

func TestTerminal_GetInfoAndExecute(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	quotaSvc := quota.NewService(s)

	h := NewTerminalHandler(cfg, s, auditLogger)
	h.SetQuotaService(quotaSvc)

	adminUserID := uuid.New()
	adminOrgID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         adminUserID,
				OrganizationID: adminOrgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/info", h.GetInfo)
	r.Post("/execute", h.Execute)

	// 1. Test GetInfo
	req := httptest.NewRequest("GET", "/info", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /info, got %d: %s", rec.Code, rec.Body.String())
	}

	var infoResp struct {
		Data TerminalInfoResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &infoResp); err != nil {
		t.Fatalf("failed to decode /info response: %v", err)
	}
	if infoResp.Data.Shell == "" {
		t.Fatalf("expected non-empty shell in info")
	}

	// 2. Test Execute
	execBody, _ := json.Marshal(ExecuteCommandRequest{
		Command: "echo 'hello terminal'",
	})
	reqExec := httptest.NewRequest("POST", "/execute", bytes.NewReader(execBody))
	reqExec.Header.Set("Content-Type", "application/json")
	recExec := httptest.NewRecorder()
	r.ServeHTTP(recExec, reqExec)

	if recExec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /execute, got %d: %s", recExec.Code, recExec.Body.String())
	}

	var execResp struct {
		Data ExecuteCommandResponse `json:"data"`
	}
	if err := json.Unmarshal(recExec.Body.Bytes(), &execResp); err != nil {
		t.Fatalf("failed to decode /execute response: %v", err)
	}
	if !strings.Contains(execResp.Data.Stdout, "hello terminal") {
		t.Fatalf("expected stdout to contain 'hello terminal', got: %s", execResp.Data.Stdout)
	}
	if execResp.Data.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", execResp.Data.ExitCode)
	}
}

func TestTerminal_WebSocket_PTY_Interactive(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	quotaSvc := quota.NewService(s)

	h := NewTerminalHandler(cfg, s, auditLogger)
	h.SetQuotaService(quotaSvc)

	adminUserID := uuid.New()
	adminOrgID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         adminUserID,
				OrganizationID: adminOrgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/ws", h.HandleWebSocket)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Connect to WebSocket PTY
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer ws.Close()

	// Assert immediate "ready" event is sent on PTY allocation
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, firstMsg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read initial message from websocket: %v", err)
	}
	if msgType != websocket.TextMessage || !strings.Contains(string(firstMsg), `"type":"ready"`) {
		t.Fatalf("expected first message to be ready event, got: %s", string(firstMsg))
	}

	// Helper to read output from websocket until expected pattern or timeout
	readUntil := func(target string, timeout time.Duration) (string, bool) {
		deadline := time.Now().Add(timeout)
		var fullOut strings.Builder
		for time.Now().Before(deadline) {
			_ = ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, msg, rErr := ws.ReadMessage()
			if rErr == nil {
				fullOut.Write(msg)
				if strings.Contains(fullOut.String(), target) {
					return fullOut.String(), true
				}
			}
		}
		return fullOut.String(), false
	}

	// 1. Verify standard input is indeed a real TTY: run `[ -t 0 ] && echo "TTY_CONFIRMED"`
	time.Sleep(100 * time.Millisecond) // Let shell initialize
	cmd := "[ -t 0 ] && echo \"TTY_CONFIRMED\"\n"
	if err := ws.WriteMessage(websocket.TextMessage, []byte(cmd)); err != nil {
		t.Fatalf("failed to write command to terminal: %v", err)
	}

	out, ok := readUntil("TTY_CONFIRMED", 3*time.Second)
	if !ok {
		t.Fatalf("Expected stdin to be a real TTY ('TTY_CONFIRMED'), got output: %q", out)
	}

	// 2. Verify state persistence across commands: `cd /tmp && pwd`
	cdCmd := "cd /tmp && pwd\n"
	if err := ws.WriteMessage(websocket.TextMessage, []byte(cdCmd)); err != nil {
		t.Fatalf("failed to write cd command: %v", err)
	}

	out, ok = readUntil("/tmp", 3*time.Second)
	if !ok {
		t.Fatalf("Expected directory state '/tmp' to be persistent in PTY session, got output: %q", out)
	}

	// 3. Test terminal resize control message
	resizeMsg, _ := json.Marshal(WSMessage{
		Type: "resize",
		Cols: 120,
		Rows: 35,
	})
	if err := ws.WriteMessage(websocket.TextMessage, resizeMsg); err != nil {
		t.Fatalf("failed to write resize message: %v", err)
	}

	// 4. Send clean exit command
	exitCmd := "exit\n"
	_ = ws.WriteMessage(websocket.TextMessage, []byte(exitCmd))
}

func TestTerminal_Security_Unauthorized(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	quotaSvc := quota.NewService(s)

	h := NewTerminalHandler(cfg, s, auditLogger)
	h.SetQuotaService(quotaSvc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// No claims in context
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/ws", h.HandleWebSocket)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Dialing without auth should fail
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatalf("expected unauthenticated websocket dial to fail")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing auth, got %d", resp.StatusCode)
	}
}

func TestTerminal_Heartbeat_And_Inactivity(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	quotaSvc := quota.NewService(s)

	h := NewTerminalHandler(cfg, s, auditLogger)
	h.SetQuotaService(quotaSvc)

	adminUserID := uuid.New()
	adminOrgID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         adminUserID,
				OrganizationID: adminOrgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/ws", h.HandleWebSocket)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer ws.Close()

	// Read initial ready event
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, firstMsg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read ready message: %v", err)
	}
	if !strings.Contains(string(firstMsg), `"type":"ready"`) {
		t.Fatalf("expected ready event, got: %s", string(firstMsg))
	}

	// 1. Test Application-Level Heartbeat: Ping -> Pong
	pingMsg := []byte(`{"type":"ping"}`)
	if err := ws.WriteMessage(websocket.TextMessage, pingMsg); err != nil {
		t.Fatalf("failed to send application ping: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, respMsg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to receive pong: %v", err)
	}
	if !strings.Contains(string(respMsg), `"type":"pong"`) {
		t.Fatalf("expected pong response, got: %s", string(respMsg))
	}

	// 2. Simulate User Inactivity: Wait without any keyboard commands
	// The terminal session and PTY must remain alive and responsive
	time.Sleep(1 * time.Second)

	// Send command after idle period to confirm session is completely responsive
	cmd := "echo IDLE_TEST_OK\n"
	if err := ws.WriteMessage(websocket.TextMessage, []byte(cmd)); err != nil {
		t.Fatalf("failed to write command after idle period: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		_ = ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, msg, rErr := ws.ReadMessage()
		if rErr == nil && strings.Contains(string(msg), "IDLE_TEST_OK") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("terminal failed to respond after idle inactivity period")
	}
}

func TestTerminal_ShellExitNotification(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	quotaSvc := quota.NewService(s)

	h := NewTerminalHandler(cfg, s, auditLogger)
	h.SetQuotaService(quotaSvc)

	adminUserID := uuid.New()
	adminOrgID := uuid.New()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         adminUserID,
				OrganizationID: adminOrgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/ws", h.HandleWebSocket)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer ws.Close()

	// Read initial ready event
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, firstMsg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read ready message: %v", err)
	}
	if !strings.Contains(string(firstMsg), `"type":"ready"`) {
		t.Fatalf("expected ready event, got: %s", string(firstMsg))
	}

	// Send "exit 0\n" command
	if err := ws.WriteMessage(websocket.TextMessage, []byte("exit\n")); err != nil {
		t.Fatalf("failed to send exit command: %v", err)
	}

	// Verify server sends an explicit exit control message before closing
	deadline := time.Now().Add(3 * time.Second)
	var gotExitMsg bool
	for time.Now().Before(deadline) {
		_ = ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, msg, rErr := ws.ReadMessage()
		if rErr == nil && strings.Contains(string(msg), `"type":"exit"`) {
			gotExitMsg = true
			break
		}
		if rErr != nil {
			break
		}
	}
	if !gotExitMsg {
		t.Fatalf("expected server to notify client with exit control message")
	}
}
