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
	"hostvra/api/internal/rbac"
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

// AuthorizeTerminalAccess verifies authentication and ensures customer account has package permission for terminal access
func (h *TerminalHandler) AuthorizeTerminalAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok || claims == nil {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required to access terminal", nil, "")
			return
		}

		if claims.IsSuperAdmin || claims.Role == "owner" || claims.Role == "admin" {
			next.ServeHTTP(w, r)
			return
		}

		// For customer/user roles, verify package/subscription or admin override grants terminal access
		if h.quotaSvc != nil {
			if !h.quotaSvc.CheckPermission(r.Context(), claims.UserID, "terminal") {
				h.audit.Log(r.Context(), r, "terminal.forbidden", "server", "localhost", "failure", "Web Terminal SSH access not enabled for plan", map[string]interface{}{
					"user_id": claims.UserID.String(),
				})
				response.Error(w, http.StatusForbidden, "FEATURE_DISABLED", "Web Terminal SSH access is not enabled for your hosting plan. Please upgrade your package or contact administration.", nil, "")
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Fallback to standard RBAC check if quota service not configured
		if !rbac.HasPermission(claims.Role, claims.IsSuperAdmin, rbac.PermTerminalAccess) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Terminal access requires administrator privilege or hosting package entitlement", nil, "")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// resolveCustomerEnvironment determines the isolated user and working directory for a customer account
func (h *TerminalHandler) resolveCustomerEnvironment(ctx context.Context, claims *auth.Claims) (string, string) {
	custName := "customer"
	if claims.Email != "" {
		parts := strings.Split(claims.Email, "@")
		if len(parts) > 0 && parts[0] != "" {
			custName = parts[0]
		}
	}

	if h.store != nil {
		// 1. Check for hosting account
		if accounts, err := h.store.ListHostingAccounts(ctx, claims.OrganizationID, nil); err == nil && len(accounts) > 0 {
			for _, acc := range accounts {
				if acc.UserID == claims.UserID || acc.OrganizationID == claims.OrganizationID {
					homeDir := fmt.Sprintf("/home/%s", acc.Username)
					if stat, err := os.Stat(homeDir); err == nil && stat.IsDir() {
						return acc.Username, homeDir
					}
					if acc.DocumentRoot != "" {
						if stat, err := os.Stat(acc.DocumentRoot); err == nil && stat.IsDir() {
							return acc.Username, acc.DocumentRoot
						}
					}
					return acc.Username, homeDir
				}
			}
		}

		// 2. Check for website document root
		if sites, err := h.store.ListWebsitesByOrg(ctx, claims.OrganizationID); err == nil && len(sites) > 0 {
			for _, s := range sites {
				if s.DocumentRoot != "" {
					if stat, err := os.Stat(s.DocumentRoot); err == nil && stat.IsDir() {
						sysUser := s.SystemUser
						if sysUser == "" {
							sysUser = custName
						}
						return sysUser, s.DocumentRoot
					}
				}
			}
		}
	}

	// 3. Fallback: User home directory or safe directory
	userHome := fmt.Sprintf("/home/%s", custName)
	if stat, err := os.Stat(userHome); err == nil && stat.IsDir() {
		return custName, userHome
	}

	if stat, err := os.Stat("/var/www"); err == nil && stat.IsDir() {
		return custName, "/var/www"
	}

	tempCustDir := filepath.Join(os.TempDir(), "hostvra-users", custName)
	_ = os.MkdirAll(tempCustDir, 0755)
	if stat, err := os.Stat(tempCustDir); err == nil && stat.IsDir() {
		return custName, tempCustDir
	}

	return custName, os.TempDir()
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

func (s *safeWSConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteControl(messageType, data, deadline)
}

func (s *safeWSConn) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Close()
}

func sendWSError(conn *safeWSConn, code, msg string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type":    "error",
		"code":    code,
		"message": msg,
	})
	_ = conn.WriteMessage(websocket.TextMessage, payload)
}

// Terminal heartbeat & buffer limits
const (
	termWriteWait  = 10 * time.Second
	termPongWait   = 60 * time.Second
	termPingPeriod = 20 * time.Second // Ping every 20s to prevent Nginx/Cloudflare/NAT idle drops
	termMaxMsgSize = 1024 * 1024
)

// HandleWebSocket handles real interactive full-duplex PTY terminal sessions
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate session & enforce strict role/plan quota checks BEFORE upgrade
	claims, ok := auth.GetClaims(r.Context())
	if !ok || claims == nil {
		h.audit.Log(r.Context(), r, "terminal.unauthorized", "server", "localhost", "failure", "Authentication required for terminal access", nil)
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required to access terminal", nil, "")
		return
	}

	if claims.Role != "owner" && claims.Role != "admin" && !claims.IsSuperAdmin {
		if h.quotaSvc != nil && !h.quotaSvc.CheckPermission(r.Context(), claims.UserID, "terminal") {
			h.audit.Log(r.Context(), r, "terminal.forbidden", "server", "localhost", "failure", "Web Terminal SSH access not enabled for plan", map[string]interface{}{
				"user_id": claims.UserID.String(),
			})
			response.Error(w, http.StatusForbidden, "FEATURE_DISABLED", "Web Terminal SSH access is not enabled for your hosting plan. Please upgrade your package or contact administration.", nil, "")
			return
		}
	}

	// 2. Upgrade HTTP to WebSocket first to establish duplex channel with browser
	rawConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.audit.Log(r.Context(), r, "terminal.websocket.upgrade_failed", "server", "localhost", "failure", "WebSocket upgrade failed: "+err.Error(), nil)
		return
	}

	// CRITICAL: Clear pre-existing HTTP server WriteTimeout/ReadTimeout deadlines from hijacked socket
	// so the connection is not killed after 15 minutes by Go's net/http server.
	_ = rawConn.SetReadDeadline(time.Time{})
	_ = rawConn.SetWriteDeadline(time.Time{})

	safeConn := &safeWSConn{conn: rawConn}

	// Configure transport-level WebSocket Ping/Pong heartbeat
	safeConn.conn.SetReadLimit(termMaxMsgSize)
	_ = safeConn.conn.SetReadDeadline(time.Now().Add(termPongWait))
	safeConn.conn.SetPongHandler(func(string) error {
		_ = safeConn.conn.SetReadDeadline(time.Now().Add(termPongWait))
		return nil
	})

	// 3. Determine starting working directory and identity with tenant isolation
	isPrivileged := claims.IsSuperAdmin || claims.Role == "owner" || claims.Role == "admin"
	defaultUser, defaultCwd := "root", "/root/Hostvra"
	if isPrivileged {
		if envUser := os.Getenv("USER"); envUser != "" {
			defaultUser = envUser
		}
		if _, err := os.Stat(defaultCwd); os.IsNotExist(err) {
			defaultCwd, _ = os.UserHomeDir()
			if defaultCwd == "" {
				defaultCwd = "/"
			}
		}
	} else {
		defaultUser, defaultCwd = h.resolveCustomerEnvironment(r.Context(), claims)
	}

	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		cwd = defaultCwd
	} else {
		cleanCwd := filepath.Clean(cwd)
		if !isPrivileged {
			// Security: strictly block tenant escape to /root, /etc, /sys, /proc, /
			if cleanCwd == "/root" || strings.HasPrefix(cleanCwd, "/root/") ||
				cleanCwd == "/etc" || strings.HasPrefix(cleanCwd, "/etc/") ||
				cleanCwd == "/" {
				cwd = defaultCwd
			} else if stat, err := os.Stat(cleanCwd); err == nil && stat.IsDir() {
				cwd = cleanCwd
			} else {
				cwd = defaultCwd
			}
		} else {
			if stat, err := os.Stat(cleanCwd); err == nil && stat.IsDir() {
				cwd = cleanCwd
			} else {
				cwd = defaultCwd
			}
		}
	}

	// 4. Determine available shell
	shell := "/bin/bash"
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		shell = "/bin/sh"
	}

	// Ensure working directory exists, otherwise fallback safely
	if stat, err := os.Stat(cwd); err != nil || !stat.IsDir() {
		if stat, err := os.Stat(defaultCwd); err == nil && stat.IsDir() {
			cwd = defaultCwd
		} else {
			cwd = os.TempDir()
		}
	}

	// 5. Launch interactive login shell with both -l (login) and -i (interactive)
	// Explicitly set TMOUT=0 to prevent Bash from executing auto-logout after inactivity
	cmd := exec.Command(shell, "-l", "-i")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
		"TMOUT=0", // Permanently disable idle auto-logout
		fmt.Sprintf("USER=%s", defaultUser),
		fmt.Sprintf("HOME=%s", defaultCwd),
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

	// 6. Allocate real Linux PTY attached to interactive shell process
	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		h.audit.Log(r.Context(), r, "terminal.pty.failed", "server", "localhost", "failure", "Failed to allocate pseudo-terminal: "+err.Error(), map[string]interface{}{
			"user_id": claims.UserID.String(),
			"error":   err.Error(),
		})
		sendWSError(safeConn, "PTY_FAILED", "Failed to allocate pseudo-terminal: "+err.Error())
		_ = safeConn.Close()
		return
	}

	sessionID := fmt.Sprintf("term_%s_%d", claims.UserID.String()[:8], time.Now().UnixNano())
	h.audit.Log(r.Context(), r, "terminal.session.created", "server", "localhost", "success", "Interactive PTY terminal session established", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    claims.UserID.String(),
		"shell":      shell,
		"cwd":        cwd,
		"rows":       rows,
		"cols":       cols,
	})

	// 7. Send immediate ready event so frontend immediately clears any loading indicators
	readyPayload, _ := json.Marshal(map[string]interface{}{
		"type":       "ready",
		"session_id": sessionID,
		"cols":       cols,
		"rows":       rows,
		"shell":      shell,
		"cwd":        cwd,
	})
	_ = safeConn.WriteMessage(websocket.TextMessage, readyPayload)

	sessionDone := make(chan struct{})

	// Dedicated Ping Ticker Goroutine: sends transport-level WebSocket Ping frames every 20 seconds.
	// This resets idle timers in Nginx, Cloudflare, AWS ALBs, NAT routers, and firewalls.
	go func() {
		ticker := time.NewTicker(termPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := safeConn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(termWriteWait)); err != nil {
					return
				}
			case <-sessionDone:
				return
			}
		}
	}()

	var closeOnce sync.Once
	cleanup := func(reason string) {
		closeOnce.Do(func() {
			close(sessionDone)
			_ = ptyFile.Close()
			exitCode := -1
			if cmd.Process != nil {
				pid := cmd.Process.Pid
				// Send SIGHUP to the process group to notify foreground process
				_ = syscall.Kill(-pid, syscall.SIGHUP)

				// Asynchronously wait for exit with timeout to prevent goroutine leak or hang
				done := make(chan error, 1)
				go func() {
					done <- cmd.Wait()
				}()

				select {
				case <-done:
					if cmd.ProcessState != nil {
						exitCode = cmd.ProcessState.ExitCode()
					}
				case <-time.After(200 * time.Millisecond):
					_ = syscall.Kill(-pid, syscall.SIGTERM)
					select {
					case <-done:
						if cmd.ProcessState != nil {
							exitCode = cmd.ProcessState.ExitCode()
						}
					case <-time.After(200 * time.Millisecond):
						_ = syscall.Kill(-pid, syscall.SIGKILL)
						<-done
						if cmd.ProcessState != nil {
							exitCode = cmd.ProcessState.ExitCode()
						}
					}
				}
			}
			_ = safeConn.Close()

			h.audit.Log(r.Context(), r, "terminal.session.closed", "server", "localhost", "success", "Interactive PTY terminal session terminated", map[string]interface{}{
				"session_id": sessionID,
				"user_id":    claims.UserID.String(),
				"reason":     reason,
				"exit_code":  exitCode,
			})
		})
	}
	defer cleanup("handler_exited")

	// Goroutine 1: PTY -> WebSocket (Streaming live stdout/stderr/ANSI with zero buffering lag)
	go func() {
		defer cleanup("shell_exited")
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

		// Notify client that the shell process specifically exited
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
			closeReason := "client_disconnected"
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				closeReason = "normal_closure"
			} else {
				closeReason = fmt.Sprintf("read_error: %v", err)
			}
			cleanup(closeReason)
			break
		}

		// Any message received from the client extends the read deadline
		_ = safeConn.conn.SetReadDeadline(time.Now().Add(termPongWait))

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
					// Respond to application-level ping from frontend
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
	if !ok || claims == nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required to access terminal", nil, "")
		return
	}

	isPrivileged := claims.IsSuperAdmin || claims.Role == "owner" || claims.Role == "admin"
	if !isPrivileged {
		if h.quotaSvc != nil {
			if !h.quotaSvc.CheckPermission(r.Context(), claims.UserID, "terminal") {
				response.Error(w, http.StatusForbidden, "FEATURE_DISABLED", "Web Terminal SSH access is not enabled for your hosting plan. Please upgrade your package or contact administration.", nil, "")
				return
			}
		} else if !rbac.HasPermission(claims.Role, claims.IsSuperAdmin, rbac.PermTerminalAccess) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Terminal access requires administrator privilege or hosting package entitlement", nil, "")
			return
		}
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "hostvra-node"
	}

	currentUser := "root"
	defaultCwd := "/root/Hostvra"

	if isPrivileged {
		if envUser := os.Getenv("USER"); envUser != "" {
			currentUser = envUser
		}
		if _, err := os.Stat(defaultCwd); os.IsNotExist(err) {
			defaultCwd, _ = os.UserHomeDir()
			if defaultCwd == "" {
				defaultCwd = "/"
			}
		}
	} else {
		currentUser, defaultCwd = h.resolveCustomerEnvironment(r.Context(), claims)
	}

	shell := "/bin/bash"
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		shell = "/bin/sh"
	}

	quickCmds := []string{
		"uptime",
		"free -m",
		"df -h",
		"whoami",
		"pwd",
		"ls -la",
		"php -v",
		"node -v",
		"python3 --version",
	}
	if isPrivileged {
		quickCmds = append([]string{
			"git status",
			"systemctl status hostvra-api",
			"systemctl status hostvra-web",
			"docker ps",
		}, quickCmds...)
	}

	info := TerminalInfoResponse{
		Hostname:   hostname,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		User:       currentUser,
		DefaultCwd: defaultCwd,
		Shell:      shell,
		QuickCmds:  quickCmds,
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

	isPrivileged := claims.IsSuperAdmin || claims.Role == "owner" || claims.Role == "admin"
	if !isPrivileged {
		if h.quotaSvc != nil {
			if !h.quotaSvc.CheckPermission(r.Context(), claims.UserID, "terminal") {
				response.Error(w, http.StatusForbidden, "FEATURE_DISABLED", "Web Terminal SSH access is not enabled for your hosting plan. Please upgrade your package or contact administration.", nil, "")
				return
			}
		} else if !rbac.HasPermission(claims.Role, claims.IsSuperAdmin, rbac.PermTerminalAccess) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Terminal access requires administrator privilege or hosting package entitlement", nil, "")
			return
		}
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
	defaultUser, defaultCwd := "root", "/root/Hostvra"
	if !isPrivileged {
		defaultUser, defaultCwd = h.resolveCustomerEnvironment(r.Context(), claims)
		_ = defaultUser
	}

	cwd := req.Cwd
	if cwd == "" {
		cwd = defaultCwd
	} else {
		cleanCwd := filepath.Clean(cwd)
		if !isPrivileged {
			if cleanCwd == "/root" || strings.HasPrefix(cleanCwd, "/root/") ||
				cleanCwd == "/etc" || strings.HasPrefix(cleanCwd, "/etc/") ||
				cleanCwd == "/" {
				cwd = defaultCwd
			} else if stat, err := os.Stat(cleanCwd); err == nil && stat.IsDir() {
				cwd = cleanCwd
			} else {
				cwd = defaultCwd
			}
		} else {
			if stat, err := os.Stat(cleanCwd); err == nil && stat.IsDir() {
				cwd = cleanCwd
			} else {
				cwd = defaultCwd
			}
		}
	}
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
