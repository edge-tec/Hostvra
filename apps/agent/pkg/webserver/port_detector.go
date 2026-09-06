package webserver

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PortConflict describes an active process holding a required web port
type PortConflict struct {
	Port        int    `json:"port"`
	ProcessName string `json:"process_name"`
	PID         int    `json:"pid"`
	CommandLine string `json:"command_line"`
	ServerType  string `json:"server_type,omitempty"`
}

// PortDetector inspects active network listeners on the host
type PortDetector struct{}

// NewPortDetector creates a new PortDetector
func NewPortDetector() *PortDetector {
	return &PortDetector{}
}

// IsPortInUse checks if a specific port is actively listening
func (pd *PortDetector) IsPortInUse(port int) bool {
	timeout := 200 * time.Millisecond
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err == nil {
		_ = conn.Close()
		return true
	}
	// Try ipv6 localhost or listen probe
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	_ = l.Close()
	return false
}

// DetectPortListeners inspects ports 80 and 443 on the system
func (pd *PortDetector) DetectPortListeners(ctx context.Context) (map[int]*PortConflict, error) {
	results := make(map[int]*PortConflict)

	if runtime.GOOS != "linux" {
		// Non-linux development fallback using socket connectivity
		for _, port := range []int{80, 443} {
			if pd.IsPortInUse(port) {
				results[port] = &PortConflict{
					Port:        port,
					ProcessName: "unknown_daemon",
					PID:         0,
					CommandLine: "listening on port " + strconv.Itoa(port),
				}
			}
		}
		return results, nil
	}

	// 1. Try ss -tlpn
	if ssOut, err := exec.CommandContext(ctx, "ss", "-tlpn").CombinedOutput(); err == nil {
		parsed := parseSSOutput(string(ssOut))
		for k, v := range parsed {
			results[k] = v
		}
	}

	// 2. If ss didn't find or if lsof is available, verify with lsof -i :80,443 -sTCP:LISTEN -P -n
	if len(results) < 2 {
		if lsofOut, err := exec.CommandContext(ctx, "lsof", "-i", ":80,443", "-sTCP:LISTEN", "-P", "-n").CombinedOutput(); err == nil {
			parsed := parseLsofOutput(string(lsofOut))
			for k, v := range parsed {
				if _, exists := results[k]; !exists {
					results[k] = v
				}
			}
		}
	}

	// Fallback to basic port dial check if nothing was parsed but port is actually open
	for _, port := range []int{80, 443} {
		if _, exists := results[port]; !exists && pd.IsPortInUse(port) {
			results[port] = &PortConflict{
				Port:        port,
				ProcessName: "active_process",
				PID:         0,
				CommandLine: fmt.Sprintf("Active listener on 0.0.0.0:%d", port),
			}
		}
	}

	return results, nil
}

// CheckConflicts inspects whether any process other than expectedServer is holding ports 80 or 443
func (pd *PortDetector) CheckConflicts(ctx context.Context, expectedServer WebServerType) ([]PortConflict, error) {
	listeners, err := pd.DetectPortListeners(ctx)
	if err != nil {
		return nil, err
	}

	var conflicts []PortConflict
	for _, port := range []int{80, 443} {
		if conflict, exists := listeners[port]; exists {
			conflict.ServerType = classifyServerProcess(conflict.ProcessName)
			if conflict.ServerType != string(expectedServer) {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	return conflicts, nil
}

func classifyServerProcess(procName string) string {
	lower := strings.ToLower(procName)
	switch {
	case strings.Contains(lower, "nginx"):
		return string(TypeNginx)
	case strings.Contains(lower, "apache") || strings.Contains(lower, "httpd"):
		return string(TypeApache)
	case strings.Contains(lower, "openlitespeed"):
		return string(TypeOpenLiteSpeed)
	case strings.Contains(lower, "litespeed") || strings.Contains(lower, "lsws"):
		return string(TypeLiteSpeedEnterprise)
	default:
		return "other"
	}
}

// parseSSOutput parses output of `ss -tlpn`
// Sample:
// LISTEN 0 511 0.0.0.0:80 0.0.0.0:* users:(("nginx",pid=12345,fd=6),("nginx",pid=12346,fd=6))
func parseSSOutput(output string) map[int]*PortConflict {
	result := make(map[int]*PortConflict)
	scanner := bufio.NewScanner(strings.NewReader(output))

	rePort := regexp.MustCompile(`:(\d+)\s+`)
	reProc := regexp.MustCompile(`users:\(\("([^"]+)",pid=(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "LISTEN") {
			continue
		}

		portMatch := rePort.FindStringSubmatch(line)
		if len(portMatch) < 2 {
			continue
		}

		port, err := strconv.Atoi(portMatch[1])
		if err != nil || (port != 80 && port != 443) {
			continue
		}

		procMatch := reProc.FindStringSubmatch(line)
		procName := "unknown"
		pid := 0
		if len(procMatch) >= 3 {
			procName = procMatch[1]
			pid, _ = strconv.Atoi(procMatch[2])
		}

		result[port] = &PortConflict{
			Port:        port,
			ProcessName: procName,
			PID:         pid,
			CommandLine: line,
			ServerType:  classifyServerProcess(procName),
		}
	}

	return result
}

// parseLsofOutput parses output of `lsof -i :80,443 -sTCP:LISTEN -P -n`
// Sample:
// COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
// nginx   12345 root    6u  IPv4  34567      0t0  TCP *:80 (LISTEN)
func parseLsofOutput(output string) map[int]*PortConflict {
	result := make(map[int]*PortConflict)
	scanner := bufio.NewScanner(strings.NewReader(output))

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		if lineCount == 1 || strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		procName := fields[0]
		pid, _ := strconv.Atoi(fields[1])
		nameField := fields[8]

		port := 0
		if strings.Contains(nameField, ":80") {
			port = 80
		} else if strings.Contains(nameField, ":443") {
			port = 443
		}

		if port > 0 {
			result[port] = &PortConflict{
				Port:        port,
				ProcessName: procName,
				PID:         pid,
				CommandLine: line,
				ServerType:  classifyServerProcess(procName),
			}
		}
	}

	return result
}
