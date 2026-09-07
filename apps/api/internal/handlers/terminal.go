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
	"strings"
	"time"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type TerminalHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewTerminalHandler(cfg *config.Config, s store.Store, a *audit.Logger) *TerminalHandler {
	return &TerminalHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
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

// GetInfo returns system environment metadata for the Web Terminal
func (h *TerminalHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
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

// Execute runs a terminal command inside the host environment
func (h *TerminalHandler) Execute(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.GetClaims(r.Context()); !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
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

	// 60-second execution deadline
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
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

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	start := time.Now()
	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			exitCode = 124 // Timeout standard
			stderrBuf.WriteString("\n[Hostvra Terminal] Execution timed out after 60 seconds.")
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
