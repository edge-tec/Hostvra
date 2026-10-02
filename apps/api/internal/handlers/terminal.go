package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type TerminalHandler struct {
	cfg      *config.Config
	store    store.Store
	audit    *audit.Logger
	quotaSvc *quota.Service
}

func NewTerminalHandler(cfg *config.Config, s store.Store, a *audit.Logger) *TerminalHandler {
	return &TerminalHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

func (h *TerminalHandler) SetQuotaService(q *quota.Service) {
	h.quotaSvc = q
}

type TerminalInfoResponse struct {
	Hostname   string   `json:"hostname"`
	OS         string   `json:"os"`
	Arch       string   `json:"arch"`
	User       string   `json:"user"`
	DefaultCwd string   `json:"default_cwd"`
	Shell      string   `json:"shell"`
	QuickCmds  []string `json:"quick_cmds"`
}

type ExecuteCommandRequest struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd,omitempty"`
}

type ExecuteCommandResponse struct {
	Command    string `json:"command"`
	Cwd        string `json:"cwd"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // Origin checked via auth claims/JWT
	},
}

type WSMessage struct {
	Type string `json:"type"` // "input", "resize", "ping"
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

type safeWSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeWSConn) WriteMessage(messageType int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(messageType, data)
}

func (s *safeWSConn) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Close()
}

// HandleWebSocket handles real interactive full-duplex PTY terminal sessions
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	if claims.Role != "owner" && claims.Role != "admin" && !claims.IsSuperAdmin {
		if h.quotaSvc != nil && !h.quotaSvc.CheckPermission(r.Context(), claims.UserID, "terminal") {
			response.Error(w, http.StatusForbidden, "FEATURE_DISABLED", "Web Terminal SSH access is not enabled for your hosting plan. Please upgrade your package or contact administration.", nil, "")
			return
		}
	}

	// Determine starting working directory
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		cwd = "/root/Hostvra"
		if _, err := os.Stat(cwd); os.IsNotExist(err) {
			cwd, _ = os.UserHomeDir()
			if cwd == "" {
				cwd = "/"
			}
		}
	} else {
		cleanCwd := filepath.Clean(cwd)
		if stat, err := os.Stat(cleanCwd); err == nil && stat.IsDir() {
			cwd = cleanCwd
		} else {
			cwd = "/"
		}
	}

	// Determine available shell
	shell := "/bin/bash"
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		shell = "/bin/sh"
	}

	// Launch interactive login shell
	cmd := exec.Command(shell, "-l")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	)

	// Window dimensions
	rows := uint16(24)
	cols := uint16(80)
	if rStr := r.URL.Query().Get("rows"); rStr != "" {
		if parsedRows, err := strconv.ParseUint(rStr, 10, 16); err == nil && parsedRows > 0 {
			rows = uint16(parsedRows)
		}
	}
	if cStr := r.URL.Query().Get("cols"); cStr != "" {
		if parsedCols, err := strconv.ParseUint(cStr, 10, 16); err == nil && parsedCols > 0 {
			cols = uint16(parsedCols)
		}
	}

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PTY_FAILED", "Failed to allocate pseudo-terminal: "+err.Error(), nil, "")
		return
	}

	rawConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = ptyFile.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return
	}

	safeConn := &safeWSConn{conn: rawConn}

	sessionID := fmt.Sprintf("term_%s_%d", claims.UserID.String()[:8], time.Now().Unix())
	h.audit.Log(r.Context(), r, "terminal.session.created", "server", "localhost", "success", "Interactive PTY terminal session established", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    claims.UserID.String(),
		"shell":      shell,
		"cwd":        cwd,
		"rows":       rows,
		"cols":       cols,
	})

	var closeOnce sync.Once
	cleanup := func() {
		closeOnce.Do(func() {
			_ = ptyFile.Close()
			if cmd.Process != nil {
				// Kill the shell process group cleanly
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGHUP)
				time.Sleep(50 * time.Millisecond)
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
				_ = cmd.Wait()
			}
			_ = safeConn.Close()

			h.audit.Log(r.Context(), r, "terminal.session.closed", "server", "localhost", "success", "Interactive PTY terminal session terminated", map[string]interface{}{
				"session_id": sessionID,
				"user_id":    claims.UserID.String(),
			})
		})
	}
	defer cleanup()

	// Goroutine 1: PTY -> WebSocket (Streaming live stdout/stderr/ANSI with zero buffering lag)
	go func() {
		defer cleanup()
		buf := make([]byte, 4096)
		for {
			n, readErr := ptyFile.Read(buf)
			if n > 0 {
				if writeErr := safeConn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if readErr != nil {
				// EOF or EIO means child shell process exited
				break
			}
		}

		// Notify client of exit if connection still alive
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		exitPayload, _ := json.Marshal(map[string]interface{}{
			"type":      "exit",
			"exit_code": exitCode,
		})
		_ = safeConn.WriteMessage(websocket.TextMessage, exitPayload)
	}()

	// Goroutine 2: WebSocket -> PTY (Handling keyboard input, control keys, and terminal resize)
	for {
		msgType, message, err := safeConn.conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			if _, wErr := ptyFile.Write(message); wErr != nil {
				break
			}
		} else if msgType == websocket.TextMessage {
			// Check if it's a JSON control command
			var wsMsg WSMessage
			if len(message) > 0 && message[0] == '{' && json.Unmarshal(message, &wsMsg) == nil && wsMsg.Type != "" {
				switch wsMsg.Type {
				case "resize":
					if wsMsg.Cols > 0 && wsMsg.Rows > 0 {
						_ = pty.Setsize(ptyFile, &pty.Winsize{
							Rows: wsMsg.Rows,
							Cols: wsMsg.Cols,
						})
					}
				case "input":
					if _, wErr := ptyFile.Write([]byte(wsMsg.Data)); wErr != nil {
						return
					}
				case "ping":
					_ = safeConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
				}
			} else {
				// Raw keystrokes or text
				if _, wErr := ptyFile.Write(message); wErr != nil {
					break
				}
			}
		}
	}
}

// GetInfo returns system environment metadata for the Web Terminal
func (h *TerminalHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || claims == nil || (claims.Role != "owner" && claims.Role != "admin" && !claims.IsSuperAdmin) {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only panel owner or administrator can access terminal metadata", nil, "")
		return
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "hostvra-node"
	}

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "root"
	}

	defaultCwd := "/root/Hostvra"
	if _, err := os.Stat(defaultCwd); os.IsNotExist(err) {
		defaultCwd, _ = os.UserHomeDir()
		if defaultCwd == "" {
			defaultCwd = "/"
		}
	}

	shell := "/bin/bash"
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		shell = "/bin/sh"
	}

	info := TerminalInfoResponse{
		Hostname:   hostname,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		User:       currentUser,
		DefaultCwd: defaultCwd,
		Shell:      shell,
		QuickCmds: []string{
			"git status",
			"systemctl status hostvra-api",
			"systemctl status hostvra-web",
			"uptime",
			"free -m",
			"df -h",
			"docker ps",
		},
	}

	response.JSON(w, http.StatusOK, info, nil)
}

// Execute runs a terminal command inside the host environment (kept for backward compatibility with scripts & APIs)
func (h *TerminalHandler) Execute(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	if claims.Role != "owner" && claims.Role != "admin" && !claims.IsSuperAdmin {
		if h.quotaSvc != nil && !h.quotaSvc.CheckPermission(r.Context(), claims.UserID, "terminal") {
			response.Error(w, http.StatusForbidden, "FEATURE_DISABLED", "Web Terminal SSH access is not enabled for your hosting plan. Please upgrade your package or contact administration.", nil, "")
			return
		}
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only panel owner or administrator can execute system terminal commands", nil, "")
		return
	}

	var req ExecuteCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	req.Command = strings.TrimSpace(req.Command)
	if req.Command == "" {
		response.Error(w, http.StatusBadRequest, "EMPTY_COMMAND", "Command cannot be empty", nil, "")
		return
	}

	// Guard against accidental catastrophic system destruction
	lowerCmd := strings.ToLower(req.Command)
	blockedDestructivePatterns := []string{
		"rm -rf /", "rm -rf /*", "rm -rf --no-preserve-root /",
		"> /dev/sda", "> /dev/nvme", "mkfs.", "dd if=/dev/zero of=/dev/sda",
	}
	for _, blocked := range blockedDestructivePatterns {
		if strings.Contains(lowerCmd, blocked) {
			response.Error(w, http.StatusBadRequest, "COMMAND_BLOCKED", "Execution of catastrophic command is strictly blocked for system safety", nil, "")
			return
		}
	}

	// Determine starting working directory
	cwd := req.Cwd
	if cwd == "" {
		cwd = "/root/Hostvra"
		if _, err := os.Stat(cwd); os.IsNotExist(err) {
			cwd, _ = os.UserHomeDir()
			if cwd == "" {
				cwd = "/"
			}
		}
	} else {
		cleanCwd := filepath.Clean(cwd)
		if stat, err := os.Stat(cleanCwd); err == nil && stat.IsDir() {
			cwd = cleanCwd
		} else {
			cwd = "/"
		}
	}

	// 900-second (15 minutes) execution deadline for builds, docker operations, and maintenance tasks
	ctx, cancel := context.WithTimeout(r.Context(), 900*time.Second)
	defer cancel()

	const pwdSep = "___HV_PWD_DELIM___"

	// Construct bash script that preserves output, isolates exit code, and extracts resulting directory
	script := fmt.Sprintf(`cd "%s" 2>/dev/null || cd /
%s
__HV_EXIT__=$?
printf "\n%s\n"
pwd
exit $__HV_EXIT__
`, strings.ReplaceAll(cwd, `"`, `\"`), req.Command, pwdSep)

	shell := "/bin/bash"
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		shell = "/bin/sh"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", script)

	const maxOutputBytes = 1024 * 1024 // 1MB buffer ceiling per command output
	stdoutBuf := &limitedBuffer{limit: maxOutputBytes}
	stderrBuf := &limitedBuffer{limit: maxOutputBytes}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	start := time.Now()
	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			exitCode = 124 // Timeout standard
			stderrBuf.WriteString("\n[Hostvra Terminal] Execution timed out.")
		} else {
			exitCode = 1
			stderrBuf.WriteString(fmt.Sprintf("\n[Hostvra Terminal] Process error: %s", err.Error()))
		}
	}

	rawOut := stdoutBuf.String()
	newCwd := cwd

	// Parse out resulting directory
	if idx := strings.LastIndex(rawOut, pwdSep); idx != -1 {
		cleanOut := rawOut[:idx]
		postSep := strings.TrimSpace(rawOut[idx+len(pwdSep):])
		if postSep != "" {
			// Extract last non-empty line
			lines := strings.Split(postSep, "\n")
			for i := len(lines) - 1; i >= 0; i-- {
				trimmed := strings.TrimSpace(lines[i])
				if trimmed != "" {
					newCwd = trimmed
					break
				}
			}
		}
		rawOut = strings.TrimRight(cleanOut, "\r\n")
	}

	// Audit log terminal execution
	auditStatus := "success"
	auditErrMsg := ""
	if exitCode != 0 {
		auditStatus = "failure"
		auditErrMsg = strings.TrimSpace(stderrBuf.String())
	}
	h.audit.Log(r.Context(), r, "terminal.execute", "server", "localhost", auditStatus, auditErrMsg, map[string]interface{}{
		"command":     req.Command,
		"exit_code":   exitCode,
		"duration_ms": durationMs,
		"cwd":         newCwd,
	})

	respData := ExecuteCommandResponse{
		Command:    req.Command,
		Cwd:        newCwd,
		Stdout:     rawOut,
		Stderr:     strings.TrimRight(stderrBuf.String(), "\r\n"),
		ExitCode:   exitCode,
		DurationMs: durationMs,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	response.JSON(w, http.StatusOK, respData, nil)
}

// limitedBuffer bounds output capturing to limit bytes to prevent unbounded memory allocation
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.buf.Len() >= l.limit {
		l.truncated = true
		return len(p), nil
	}
	remaining := l.limit - l.buf.Len()
	if len(p) > remaining {
		l.buf.Write(p[:remaining])
		l.truncated = true
		return len(p), nil
	}
	return l.buf.Write(p)
}

func (l *limitedBuffer) WriteString(s string) (int, error) {
	return l.Write([]byte(s))
}

func (l *limitedBuffer) String() string {
	res := l.buf.String()
	if l.truncated {
		res += "\n\n[Hostvra Terminal: Output truncated at 1MB limit to protect server memory]"
	}
	return res
}
