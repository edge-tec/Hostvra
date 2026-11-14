package firewall

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrSSHLockoutBlocked  = errors.New("operation rejected: modifying or denying SSH port (22) without override would lock out administrative access")
	ErrInvalidPort        = errors.New("invalid port specification: must be integer 1-65535 or range 'start:end'")
	ErrInvalidProtocol    = errors.New("invalid protocol: must be 'tcp', 'udp', or 'both'")
	ErrInvalidIP          = errors.New("invalid IP address or CIDR network")
	ErrFail2banNotRunning = errors.New("fail2ban is not installed or service is not running")
	commentSanitizeRegex  = regexp.MustCompile(`[^a-zA-Z0-9 _.-]`)
)

type FirewallRule struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	To       string `json:"to"`       // Port or service
	Action   string `json:"action"`   // ALLOW, DENY, REJECT
	From     string `json:"from"`     // Anywhere, IP, CIDR
	Protocol string `json:"protocol"` // tcp, udp, both
	Comment  string `json:"comment"`
}

type JailInfo struct {
	Name            string   `json:"name"`
	Service         string   `json:"service"`
	IsActive        bool     `json:"is_active"`
	CurrentlyFailed int      `json:"currently_failed"`
	TotalFailed     int      `json:"total_failed"`
	CurrentlyBanned int      `json:"currently_banned"`
	TotalBanned     int      `json:"total_banned"`
	BannedIPs       []string `json:"banned_ips"`
}

type BannedIPItem struct {
	IP       string `json:"ip"`
	Jail     string `json:"jail"`
	BannedAt string `json:"banned_at"`
}

type FirewallStatus struct {
	Backend           string   `json:"backend"` // "ufw", "nftables", "dev"
	IsActive          bool     `json:"is_active"`
	DefaultIncoming   string   `json:"default_incoming"`
	DefaultOutgoing   string   `json:"default_outgoing"`
	RulesCount        int      `json:"rules_count"`
	SSHPortProtected  bool     `json:"ssh_port_protected"`
	Fail2banInstalled bool     `json:"fail2ban_installed"`
	Fail2banActive    bool     `json:"fail2ban_active"`
	JailsCount        int      `json:"jails_count"`
	TotalBannedIPs    int      `json:"total_banned_ips"`
	PingProhibited    bool     `json:"ping_prohibited"`
}

type ListeningPort struct {
	Protocol  string `json:"protocol"`   // tcp, udp, tcp6
	Port      int    `json:"port"`       // e.g. 22, 80, 443
	Process   string `json:"process"`    // e.g. nginx, sshd, mysqld
	PID       int    `json:"pid"`        // process ID
	LocalAddr string `json:"local_addr"` // 0.0.0.0, 127.0.0.1, *
	IsOpen    bool   `json:"is_open"`    // whether permitted through firewall rules
}

type FirewallManager struct {
	backend string // "ufw", "nftables", "dev"
}

func NewFirewallManager() *FirewallManager {
	backend := "dev"
	if _, err := exec.LookPath("ufw"); err == nil {
		backend = "ufw"
	} else if _, err := exec.LookPath("nft"); err == nil {
		backend = "nftables"
	}
	return &FirewallManager{backend: backend}
}

// GetStatus returns the overall firewall and fail2ban health
func (fm *FirewallManager) GetStatus() (*FirewallStatus, error) {
	active, _ := fm.IsActive()
	f2bInstalled := false
	if _, err := exec.LookPath("fail2ban-client"); err == nil {
		f2bInstalled = true
	}

	f2bActive := false
	if f2bInstalled {
		if out, err := exec.Command("fail2ban-client", "ping").Output(); err == nil {
			f2bActive = strings.Contains(string(out), "Server replied: pong")
		}
	}

	rules, _ := fm.ListRules()
	jails, _ := fm.ListJails()

	totalBanned := 0
	for _, j := range jails {
		totalBanned += j.CurrentlyBanned
	}

	pingProhibited, _ := fm.GetPingStatus()

	return &FirewallStatus{
		Backend:           fm.backend,
		IsActive:          active,
		DefaultIncoming:   "deny",
		DefaultOutgoing:   "allow",
		RulesCount:        len(rules),
		SSHPortProtected:  true,
		Fail2banInstalled: f2bInstalled,
		Fail2banActive:    f2bActive,
		JailsCount:        len(jails),
		TotalBannedIPs:    totalBanned,
		PingProhibited:    pingProhibited,
	}, nil
}

// IsActive checks if UFW or firewall daemon is active
func (fm *FirewallManager) IsActive() (bool, error) {
	if fm.backend == "ufw" {
		out, err := exec.Command("ufw", "status").Output()
		if err != nil {
			return false, err
		}
		return strings.Contains(string(out), "Status: active"), nil
	}
	return true, nil
}

// Enable turns the firewall on
func (fm *FirewallManager) Enable() error {
	if fm.backend == "ufw" {
		// Ensure port 22 is explicitly allowed first to prevent SSH lockout
		_ = exec.Command("ufw", "allow", "22/tcp", "comment", "Hostvra SSH Safety Rule").Run()
		out, err := exec.Command("ufw", "--force", "enable").CombinedOutput()
		if err != nil {
			return fmt.Errorf("ufw enable failed: %s (%w)", string(out), err)
		}
	}
	return nil
}

// Disable turns the firewall off
func (fm *FirewallManager) Disable() error {
	if fm.backend == "ufw" {
		out, err := exec.Command("ufw", "disable").CombinedOutput()
		if err != nil {
			return fmt.Errorf("ufw disable failed: %s (%w)", string(out), err)
		}
	}
	return nil
}

// ListRules parses active UFW firewall rules
func (fm *FirewallManager) ListRules() ([]FirewallRule, error) {
	if fm.backend == "ufw" {
		out, err := exec.Command("ufw", "status", "numbered").Output()
		if err != nil {
			return nil, err
		}
		return parseUFWNumberedRules(string(out)), nil
	}

	return []FirewallRule{}, nil
}

// AddRule adds an allow or deny rule for a port or IP
func (fm *FirewallManager) AddRule(port, protocol, fromIP, action, comment string) error {
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "allow" && action != "deny" {
		action = "allow"
	}

	proto := strings.ToLower(strings.TrimSpace(protocol))
	if proto == "" {
		proto = "tcp"
	}

	cleanComment := commentSanitizeRegex.ReplaceAllString(comment, "")
	if len(cleanComment) > 48 {
		cleanComment = cleanComment[:48]
	}

	// SSH Lockout Protection
	if (port == "22" || isSSHPort(port)) && action == "deny" {
		return ErrSSHLockoutBlocked
	}

	if fm.backend == "ufw" {
		var args []string
		args = append(args, action)

		if fromIP != "" && fromIP != "Anywhere" && fromIP != "any" {
			if net.ParseIP(fromIP) == nil && !isValidCIDR(fromIP) {
				return ErrInvalidIP
			}
			args = append(args, "from", fromIP)
		}

		if port != "" {
			if err := validatePortSpec(port); err != nil {
				return err
			}
			if proto != "both" {
				args = append(args, "to", "any", "port", port, "proto", proto)
			} else {
				args = append(args, "to", "any", "port", port)
			}
		}

		if cleanComment != "" {
			args = append(args, "comment", cleanComment)
		}

		cmd := exec.Command("ufw", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ufw rule failed: %s (%w)", string(out), err)
		}
		return nil
	}

	return nil
}

// DeleteRule removes a rule by numbered index or port
func (fm *FirewallManager) DeleteRule(ruleID string) error {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID == "" {
		return fmt.Errorf("rule ID or port required")
	}

	if fm.backend == "ufw" {
		rules, _ := fm.ListRules()

		// 1. Try matching by exact Rule Number or ID in active rules
		for _, r := range rules {
			if r.ID == ruleID || strconv.Itoa(r.Number) == ruleID {
				cmd := exec.Command("ufw", "--force", "delete", strconv.Itoa(r.Number))
				out, err := cmd.CombinedOutput()
				if err == nil {
					return nil
				}
				return fmt.Errorf("ufw delete rule failed: %s (%w)", string(out), err)
			}
		}

		// 2. Try matching by port in active rules (e.g. ruleID is "8080" and r.To is "8080/tcp" or "8080")
		for _, r := range rules {
			cleanTo := strings.TrimSpace(strings.Split(r.To, "/")[0])
			if cleanTo == ruleID || r.To == ruleID {
				cmd := exec.Command("ufw", "--force", "delete", strconv.Itoa(r.Number))
				out, err := cmd.CombinedOutput()
				if err == nil {
					return nil
				}
				return fmt.Errorf("ufw delete rule failed: %s (%w)", string(out), err)
			}
		}

		// 3. If ruleID is a direct small number and within current rule count, try deleting by number
		ruleNum, err := strconv.Atoi(ruleID)
		if err == nil && ruleNum > 0 && ruleNum <= len(rules) {
			cmd := exec.Command("ufw", "--force", "delete", strconv.Itoa(ruleNum))
			if err := cmd.Run(); err == nil {
				return nil
			}
		}

		// 4. Fallback: try deleting by port or syntax directly
		cmdAllowProto := exec.Command("ufw", "--force", "delete", "allow", ruleID+"/tcp")
		if out, err := cmdAllowProto.CombinedOutput(); err == nil && !strings.Contains(string(out), "Could not find rule") {
			return nil
		}
		cmdAllow := exec.Command("ufw", "--force", "delete", "allow", ruleID)
		if out, err := cmdAllow.CombinedOutput(); err == nil && !strings.Contains(string(out), "Could not find rule") {
			return nil
		}
		cmdDeny := exec.Command("ufw", "--force", "delete", "deny", ruleID)
		_ = cmdDeny.Run()

		return nil
	}

	return nil
}

// ----------------------------------------------------------------------------
// FAIL2BAN SUBSYSTEM
// ----------------------------------------------------------------------------

// ListJails queries active Fail2ban jails and parses banned IP statistics
func (fm *FirewallManager) ListJails() ([]JailInfo, error) {
	if _, err := exec.LookPath("fail2ban-client"); err != nil {
		return nil, ErrFail2banNotRunning
	}

	out, err := exec.Command("fail2ban-client", "status").Output()
	if err != nil {
		return nil, err
	}

	jailNames := parseJailList(string(out))
	jails := make([]JailInfo, 0, len(jailNames))

	for _, name := range jailNames {
		statusOut, err := exec.Command("fail2ban-client", "status", name).Output()
		if err != nil {
			continue
		}
		jInfo := parseJailStatus(name, string(statusOut))
		jails = append(jails, jInfo)
	}

	return jails, nil
}

// BanIP bans an IP in a specific Fail2ban jail
func (fm *FirewallManager) BanIP(jail, ip string) error {
	if net.ParseIP(ip) == nil {
		return ErrInvalidIP
	}
	if jail == "" {
		jail = "sshd"
	}

	cmd := exec.Command("fail2ban-client", "set", jail, "banip", ip)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail2ban ban failed: %s (%w)", string(out), err)
	}
	return nil
}

// UnbanIP unbans an IP from a jail or all jails
func (fm *FirewallManager) UnbanIP(jail, ip string) error {
	if net.ParseIP(ip) == nil {
		return ErrInvalidIP
	}

	if jail != "" {
		cmd := exec.Command("fail2ban-client", "set", jail, "unbanip", ip)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("fail2ban unban failed: %s (%w)", string(out), err)
		}
		return nil
	}

	// Unban across all jails
	cmd := exec.Command("fail2ban-client", "unban", ip)
	_ = cmd.Run()
	return nil
}

// ListAllBannedIPs aggregates banned IPs across all jails
func (fm *FirewallManager) ListAllBannedIPs() ([]BannedIPItem, error) {
	jails, err := fm.ListJails()
	if err != nil {
		return nil, err
	}

	var items []BannedIPItem
	for _, j := range jails {
		for _, ip := range j.BannedIPs {
			items = append(items, BannedIPItem{
				IP:       ip,
				Jail:     j.Name,
				BannedAt: "Active",
			})
		}
	}
	return items, nil
}

// ----------------------------------------------------------------------------
// PARSER HELPERS
// ----------------------------------------------------------------------------

func parseUFWNumberedRules(output string) []FirewallRule {
	var rules []FirewallRule
	lines := strings.Split(output, "\n")

	// Regex to match e.g. "[ 1] 22/tcp                     ALLOW IN    Anywhere                   # SSH Rule"
	re := regexp.MustCompile(`^\[\s*(\d+)\]\s+(\S+)\s+(ALLOW(?:\s+IN)?|DENY(?:\s+IN)?|REJECT(?:\s+IN)?)\s+(\S+)(?:\s+#\s*(.*))?$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 5 {
			num, _ := strconv.Atoi(matches[1])
			to := matches[2]
			action := matches[3]
			from := matches[4]
			comment := ""
			if len(matches) >= 6 {
				comment = strings.TrimSpace(matches[5])
			}

			proto := "both"
			if strings.Contains(to, "/tcp") {
				proto = "tcp"
			} else if strings.Contains(to, "/udp") {
				proto = "udp"
			}

			rules = append(rules, FirewallRule{
				ID:       strconv.Itoa(num),
				Number:   num,
				To:       to,
				Action:   action,
				From:     from,
				Protocol: proto,
				Comment:  comment,
			})
		}
	}
	return rules
}

func parseJailList(output string) []string {
	// Look for "Jail list:	sshd, nginx-http-auth"
	idx := strings.Index(output, "Jail list:")
	if idx == -1 {
		return nil
	}
	sub := output[idx+len("Jail list:"):]
	if nl := strings.Index(sub, "\n"); nl != -1 {
		sub = sub[:nl]
	}
	parts := strings.Split(sub, ",")
	var list []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return list
}

func parseJailStatus(name, output string) JailInfo {
	info := JailInfo{
		Name:     name,
		Service:  name,
		IsActive: true,
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if idx := strings.Index(line, "Currently failed:"); idx != -1 {
			val := strings.TrimSpace(line[idx+len("Currently failed:"):])
			info.CurrentlyFailed, _ = strconv.Atoi(strings.Fields(val)[0])
		} else if idx := strings.Index(line, "Total failed:"); idx != -1 {
			val := strings.TrimSpace(line[idx+len("Total failed:"):])
			info.TotalFailed, _ = strconv.Atoi(strings.Fields(val)[0])
		} else if idx := strings.Index(line, "Currently banned:"); idx != -1 {
			val := strings.TrimSpace(line[idx+len("Currently banned:"):])
			info.CurrentlyBanned, _ = strconv.Atoi(strings.Fields(val)[0])
		} else if idx := strings.Index(line, "Total banned:"); idx != -1 {
			val := strings.TrimSpace(line[idx+len("Total banned:"):])
			info.TotalBanned, _ = strconv.Atoi(strings.Fields(val)[0])
		} else if idx := strings.Index(line, "Banned IP list:"); idx != -1 {
			val := strings.TrimSpace(line[idx+len("Banned IP list:"):])
			info.BannedIPs = strings.Fields(val)
		}
	}
	return info
}

func validatePortSpec(port string) error {
	port = strings.TrimSpace(port)
	if strings.Contains(port, ":") {
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return ErrInvalidPort
		}
		p1, err1 := strconv.Atoi(parts[0])
		p2, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || p1 < 1 || p1 > 65535 || p2 < 1 || p2 > 65535 || p1 > p2 {
			return ErrInvalidPort
		}
		return nil
	}

	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return ErrInvalidPort
	}
	return nil
}

func DetectSSHPort() int {
	// Check /etc/ssh/sshd_config and sshd_config.d/*.conf
	paths := []string{"/etc/ssh/sshd_config"}
	if matches, err := filepath.Glob("/etc/ssh/sshd_config.d/*.conf"); err == nil {
		paths = append(paths, matches...)
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), "port ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if port, err := strconv.Atoi(fields[1]); err == nil && port > 0 && port <= 65535 {
						return port
					}
				}
			}
		}
	}
	return 22 // Default SSH port
}

func isSSHPort(portStr string) bool {
	p, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		return false
	}
	activeSSHPort := DetectSSHPort()
	return p == 22 || p == activeSSHPort
}

func isValidCIDR(cidrStr string) bool {
	_, _, err := net.ParseCIDR(cidrStr)
	return err == nil
}

// GetPingStatus returns true if ICMP ping requests are dropped/prohibited (net.ipv4.icmp_echo_ignore_all == 1)
func (fm *FirewallManager) GetPingStatus() (bool, error) {
	if data, err := os.ReadFile("/proc/sys/net/ipv4/icmp_echo_ignore_all"); err == nil {
		val := strings.TrimSpace(string(data))
		return val == "1", nil
	}

	if out, err := exec.Command("sysctl", "-n", "net.ipv4.icmp_echo_ignore_all").Output(); err == nil {
		val := strings.TrimSpace(string(out))
		return val == "1", nil
	}

	// Dev / non-Linux fallback state
	devPath := filepath.Join(os.TempDir(), "hostvra_ping_ignore")
	if data, err := os.ReadFile(devPath); err == nil {
		return strings.TrimSpace(string(data)) == "1", nil
	}

	return false, nil
}

// SetPingStatus configures ICMP echo reply (true = drop ping, false = allow ping)
func (fm *FirewallManager) SetPingStatus(prohibit bool) error {
	val := "0"
	if prohibit {
		val = "1"
	}

	// 1. Live write to proc filesystem
	_ = os.WriteFile("/proc/sys/net/ipv4/icmp_echo_ignore_all", []byte(val+"\n"), 0644)

	// 2. Apply via sysctl
	_ = exec.Command("sysctl", "-w", "net.ipv4.icmp_echo_ignore_all="+val).Run()

	// 3. Persist to sysctl.d so it survives server reboots
	confDir := "/etc/sysctl.d"
	if fi, err := os.Stat(confDir); err == nil && fi.IsDir() {
		confPath := filepath.Join(confDir, "99-hostvra-ping.conf")
		_ = os.WriteFile(confPath, []byte(fmt.Sprintf("# Hostvra aaPanel Parity Ping Control\nnet.ipv4.icmp_echo_ignore_all = %s\n", val)), 0644)
	}

	// Fallback dev state
	devPath := filepath.Join(os.TempDir(), "hostvra_ping_ignore")
	_ = os.WriteFile(devPath, []byte(val), 0644)

	return nil
}

// GetListeningPorts inspects actively listening ports and correlates with firewall rules
func (fm *FirewallManager) GetListeningPorts() ([]ListeningPort, error) {
	rules, _ := fm.ListRules()
	var rawOutput string

	if _, err := exec.LookPath("ss"); err == nil {
		if out, err := exec.Command("ss", "-tulpn", "-H").Output(); err == nil && len(out) > 0 {
			rawOutput = string(out)
		}
	}

	if rawOutput == "" {
		if _, err := exec.LookPath("netstat"); err == nil {
			if out, err := exec.Command("netstat", "-tulpn").Output(); err == nil && len(out) > 0 {
				rawOutput = string(out)
			}
		}
	}

	ports := parseListeningOutput(rawOutput)

	// If no ports detected (e.g. non-root or dev environment), supply common defaults
	if len(ports) == 0 {
		ports = []ListeningPort{
			{Protocol: "tcp", Port: 22, Process: "sshd", PID: 1, LocalAddr: "0.0.0.0"},
			{Protocol: "tcp", Port: 80, Process: "nginx", PID: 2, LocalAddr: "0.0.0.0"},
			{Protocol: "tcp", Port: 443, Process: "nginx", PID: 2, LocalAddr: "0.0.0.0"},
			{Protocol: "tcp", Port: 3306, Process: "mysqld", PID: 3, LocalAddr: "127.0.0.1"},
			{Protocol: "tcp", Port: 8080, Process: "hostvra-api", PID: 4, LocalAddr: "0.0.0.0"},
		}
	}

	// Deduplicate and correlate with firewall rules
	seen := make(map[string]bool)
	var result []ListeningPort
	for _, lp := range ports {
		key := fmt.Sprintf("%s:%d", lp.Protocol, lp.Port)
		if seen[key] {
			continue
		}
		seen[key] = true
		lp.IsOpen = isPortCoveredInRules(lp.Port, lp.Protocol, rules)
		result = append(result, lp)
	}

	return result, nil
}

func parseListeningOutput(output string) []ListeningPort {
	var list []ListeningPort
	lines := strings.Split(output, "\n")
	procRegex := regexp.MustCompile(`(?:users:\(\("([^"]+)",pid=(\d+))|(\d+)\/([a-zA-Z0-9_\-\.]+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Proto") || strings.HasPrefix(line, "Active") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToLower(fields[0])
		if !strings.HasPrefix(proto, "tcp") && !strings.HasPrefix(proto, "udp") {
			continue
		}

		var localAddr string
		for _, f := range fields {
			if strings.Contains(f, ":") && !strings.HasPrefix(f, "users:") {
				localAddr = f
				break
			}
		}
		if localAddr == "" {
			continue
		}

		lastColon := strings.LastIndex(localAddr, ":")
		if lastColon == -1 {
			continue
		}
		portStr := localAddr[lastColon+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		bindIP := localAddr[:lastColon]

		processName := "-"
		pid := 0
		if matches := procRegex.FindStringSubmatch(line); len(matches) > 0 {
			if matches[1] != "" {
				processName = matches[1]
				pid, _ = strconv.Atoi(matches[2])
			} else if matches[4] != "" {
				processName = matches[4]
				pid, _ = strconv.Atoi(matches[3])
			}
		}

		if processName == "-" {
			switch port {
			case 22:
				processName = "sshd"
			case 80, 443:
				processName = "webserver"
			case 3306:
				processName = "mysql"
			case 5432:
				processName = "postgres"
			case 6379:
				processName = "redis"
			case 53:
				processName = "named/dns"
			}
		}

		list = append(list, ListeningPort{
			Protocol:  proto,
			Port:      port,
			Process:   processName,
			PID:       pid,
			LocalAddr: bindIP,
		})
	}
	return list
}

func isPortCoveredInRules(port int, protocol string, rules []FirewallRule) bool {
	for _, r := range rules {
		if !strings.Contains(strings.ToUpper(r.Action), "ALLOW") {
			continue
		}
		target := strings.ToLower(r.To)
		target = strings.TrimSuffix(target, "/tcp")
		target = strings.TrimSuffix(target, "/udp")
		target = strings.TrimSpace(target)

		if strings.Contains(target, ":") {
			parts := strings.Split(target, ":")
			if len(parts) == 2 {
				p1, err1 := strconv.Atoi(parts[0])
				p2, err2 := strconv.Atoi(parts[1])
				if err1 == nil && err2 == nil && port >= p1 && port <= p2 {
					return true
				}
			}
		}

		if p, err := strconv.Atoi(target); err == nil && p == port {
			return true
		}
	}
	return false
}
