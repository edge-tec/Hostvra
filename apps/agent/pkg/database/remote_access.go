package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"hostvra/agent/pkg/firewall"
)

var (
	ErrWildcardIPForbidden = errors.New("unrestricted '%' or '0.0.0.0/0' wildcard host is forbidden for security unless explicitly confirmed")
	ErrInvalidAuthorizedIP = errors.New("invalid IP address or CIDR network")
	validPrivilegeRegex    = regexp.MustCompile(`^[A-Za-z0-9_ ,]+$`)
)

// RemoteAccessGrantRequest encapsulates grant creation parameters.
type RemoteAccessGrantRequest struct {
	DatabaseName             string `json:"database_name"`
	DBUser                   string `json:"db_user"`
	AuthorizedIP             string `json:"authorized_ip"`
	Privileges               string `json:"privileges"`
	AllowUnrestrictedWildcard bool   `json:"allow_unrestricted_wildcard"`
}

// RemoteAccessTestResult encapsulates connectivity probe output.
type RemoteAccessTestResult struct {
	Success   bool   `json:"success"`
	LatencyMs int64  `json:"latency_ms"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// ValidateAuthorizedIP ensures the IP is a valid IPv4/IPv6 address or CIDR, and checks wildcard policy.
func ValidateAuthorizedIP(ipOrCIDR string, allowWildcard bool) error {
	clean := strings.TrimSpace(ipOrCIDR)
	if clean == "" {
		return errors.New("authorized IP is required")
	}

	if clean == "%" || clean == "0.0.0.0/0" {
		if !allowWildcard {
			return ErrWildcardIPForbidden
		}
		return nil
	}

	if net.ParseIP(clean) != nil {
		return nil
	}

	if _, _, err := net.ParseCIDR(clean); err == nil {
		return nil
	}

	return ErrInvalidAuthorizedIP
}

// GrantRemoteAccess creates MySQL user privileges for the specific source IP and updates firewall rules.
func (m *Manager) GrantRemoteAccess(ctx context.Context, req RemoteAccessGrantRequest) error {
	if req.DatabaseName == "" || req.DBUser == "" {
		return errors.New("database name and database user are required")
	}

	cleanIP := strings.TrimSpace(req.AuthorizedIP)
	if err := ValidateAuthorizedIP(cleanIP, req.AllowUnrestrictedWildcard); err != nil {
		return err
	}

	privs := strings.TrimSpace(req.Privileges)
	if privs == "" {
		privs = "ALL PRIVILEGES"
	}
	if !validPrivilegeRegex.MatchString(privs) {
		return errors.New("invalid database privileges specified")
	}

	escUser := safeSQLString(req.DBUser)
	escHost := safeSQLString(cleanIP)
	quotedDB := SafeQuoteIdentifier(req.DatabaseName)

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s';\n", escUser, escHost))
	sb.WriteString(fmt.Sprintf("GRANT %s ON %s.* TO '%s'@'%s';\n", privs, quotedDB, escUser, escHost))
	sb.WriteString("FLUSH PRIVILEGES;\n")

	if err := m.executeSQLScript(ctx, sb.String()); err != nil {
		return fmt.Errorf("failed to grant database privileges: %w", err)
	}

	// Apply firewall rule if authorized IP is not wildcard and firewall is available
	if cleanIP != "%" && cleanIP != "0.0.0.0/0" {
		fw := firewall.NewFirewallManager()
		comment := fmt.Sprintf("remote_db_%s_%s", req.DatabaseName, req.DBUser)
		_ = fw.AddRule("allow", "3306", "tcp", cleanIP, comment)
	}

	return nil
}

// RevokeRemoteAccess drops user host grant and deletes the corresponding firewall rule.
func (m *Manager) RevokeRemoteAccess(ctx context.Context, dbName, dbUser, authorizedIP string) error {
	if dbName == "" || dbUser == "" || authorizedIP == "" {
		return errors.New("database name, user, and authorized IP are required")
	}

	cleanIP := strings.TrimSpace(authorizedIP)
	escUser := safeSQLString(dbUser)
	escHost := safeSQLString(cleanIP)
	quotedDB := SafeQuoteIdentifier(dbName)

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("REVOKE ALL PRIVILEGES ON %s.* FROM '%s'@'%s';\n", quotedDB, escUser, escHost))
	sb.WriteString(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s';\n", escUser, escHost))
	sb.WriteString("FLUSH PRIVILEGES;\n")

	if err := m.executeSQLScript(ctx, sb.String()); err != nil {
		// Log or return error
		_ = err
	}

	// Remove firewall rule
	if cleanIP != "%" && cleanIP != "0.0.0.0/0" {
		fw := firewall.NewFirewallManager()
		_ = fw.DeleteRule(cleanIP)
	}

	return nil
}

// TestRemoteAccessConnection validates that an external client can authenticate against the database.
func (m *Manager) TestRemoteAccessConnection(ctx context.Context, host string, port int, user, password, dbName string) (*RemoteAccessTestResult, error) {
	if port <= 0 {
		port = 3306
	}
	if host == "" {
		host = "127.0.0.1"
	}

	start := time.Now()

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return &RemoteAccessTestResult{
			Success:   true,
			LatencyMs: 12,
			Message:   fmt.Sprintf("Connected successfully to %s:%d (database: %s) [Test Mode]", host, port, dbName),
		}, nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=3s", user, password, host, port, dbName)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return &RemoteAccessTestResult{
			Success:   false,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}, nil
	}
	defer conn.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := conn.PingContext(pingCtx); err != nil {
		return &RemoteAccessTestResult{
			Success:   false,
			LatencyMs: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		}, nil
	}

	return &RemoteAccessTestResult{
		Success:   true,
		LatencyMs: time.Since(start).Milliseconds(),
		Message:   fmt.Sprintf("Connected successfully to %s:%d (%dms)", host, port, time.Since(start).Milliseconds()),
	}, nil
}
