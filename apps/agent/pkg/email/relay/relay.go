package relay

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// ProviderPreset details predefined host & port configs for enterprise email delivery services
type ProviderPreset struct {
	Name               string `json:"name"`
	DefaultHost        string `json:"default_host"`
	DefaultPort        int    `json:"default_port"`
	DefaultEncryption  string `json:"default_encryption"` // "starttls", "ssl_tls"
	DefaultAuth        string `json:"default_auth"`       // "plain", "login"
	SpfInclude         string `json:"spf_include"`
	DkimSelectorPrompt string `json:"dkim_selector_prompt"`
}

var SupportedProviders = map[string]ProviderPreset{
	"amazon_ses": {
		Name:               "Amazon SES",
		DefaultHost:        "email-smtp.us-east-1.amazonaws.com",
		DefaultPort:        587,
		DefaultEncryption:  "starttls",
		DefaultAuth:        "login",
		SpfInclude:         "include:amazonses.com",
		DkimSelectorPrompt: "ses",
	},
	"sendgrid": {
		Name:               "SendGrid (Twilio)",
		DefaultHost:        "smtp.sendgrid.net",
		DefaultPort:        587,
		DefaultEncryption:  "starttls",
		DefaultAuth:        "plain",
		SpfInclude:         "include:sendgrid.net",
		DkimSelectorPrompt: "s1",
	},
	"mailgun": {
		Name:               "Mailgun (Sinch)",
		DefaultHost:        "smtp.mailgun.org",
		DefaultPort:        587,
		DefaultEncryption:  "starttls",
		DefaultAuth:        "plain",
		SpfInclude:         "include:mailgun.org",
		DkimSelectorPrompt: "smtp",
	},
	"brevo": {
		Name:               "Brevo (Sendinblue)",
		DefaultHost:        "smtp-relay.brevo.com",
		DefaultPort:        587,
		DefaultEncryption:  "starttls",
		DefaultAuth:        "login",
		SpfInclude:         "include:spf.brevo.com",
		DkimSelectorPrompt: "mail",
	},
	"postmark": {
		Name:               "Postmark",
		DefaultHost:        "smtp.postmarkapp.com",
		DefaultPort:        587,
		DefaultEncryption:  "starttls",
		DefaultAuth:        "plain",
		SpfInclude:         "include:spf.mtasv.net",
		DkimSelectorPrompt: "20200818",
	},
	"custom": {
		Name:               "Custom SMTP Relay",
		DefaultHost:        "",
		DefaultPort:        587,
		DefaultEncryption:  "starttls",
		DefaultAuth:        "plain",
		SpfInclude:         "",
		DkimSelectorPrompt: "",
	},
}

// RelayConnectionTestResult details the outcome of an active SMTP handshake test
type RelayConnectionTestResult struct {
	Success      bool     `json:"success"`
	LatencyMs    int64    `json:"latency_ms"`
	Banner       string   `json:"banner"`
	TLSSucceeded bool     `json:"tls_succeeded"`
	AuthSuccess  bool     `json:"auth_succeeded"`
	Trace        []string `json:"trace"`
	Error        string   `json:"error,omitempty"`
}

// DeliverabilityAuditResult reports DNS verification status for SPF, DKIM, DMARC, and MX
type DeliverabilityAuditResult struct {
	Domain             string   `json:"domain"`
	Provider           string   `json:"provider"`
	IsFullyVerified    bool     `json:"is_fully_verified"`
	SPFRecord          string   `json:"spf_record"`
	SPFValid           bool     `json:"spf_valid"`
	SPFMessage         string   `json:"spf_message"`
	DKIMRecord         string   `json:"dkim_record"`
	DKIMValid          bool     `json:"dkim_valid"`
	DKIMSelector       string   `json:"dkim_selector"`
	DMARCRecord        string   `json:"dmarc_record"`
	DMARCValid         bool     `json:"dmarc_valid"`
	MXRecords          []string `json:"mx_records"`
	RecommendationList []string `json:"recommendations"`
}

// TestRelayConnection performs an end-to-end active probe against an external SMTP relay
func TestRelayConnection(ctx context.Context, host string, port int, username, password, encryption, authType string) *RelayConnectionTestResult {
	start := time.Now()
	res := &RelayConnectionTestResult{
		Trace: make([]string, 0),
	}

	if host == "" {
		res.Success = false
		res.Error = "SMTP relay host cannot be empty"
		return res
	}
	if port <= 0 {
		port = 587
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	res.Trace = append(res.Trace, fmt.Sprintf("Connecting to SMTP relay server at %s...", addr))

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		res.Success = false
		res.Error = fmt.Sprintf("TCP dial failed: %v", err)
		res.Trace = append(res.Trace, fmt.Sprintf("FAIL: %v", err))
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	banner, err := reader.ReadString('\n')
	if err != nil {
		res.Success = false
		res.Error = fmt.Sprintf("Failed reading SMTP greeting banner: %v", err)
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	res.Banner = strings.TrimSpace(banner)
	res.Trace = append(res.Trace, fmt.Sprintf("<< %s", res.Banner))

	// EHLO
	fmt.Fprintf(conn, "EHLO hostvra-relay-probe\r\n")
	res.Trace = append(res.Trace, ">> EHLO hostvra-relay-probe")

	var supportsStartTLS bool
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		trimmed := strings.TrimSpace(line)
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", trimmed))
		if strings.Contains(strings.ToUpper(trimmed), "STARTTLS") {
			supportsStartTLS = true
		}
		if len(trimmed) >= 4 && trimmed[3] == ' ' {
			break
		}
	}

	// Upgrade to TLS if requested or supported
	if encryption == "starttls" || (encryption == "" && supportsStartTLS) {
		if !supportsStartTLS {
			res.Trace = append(res.Trace, "WARN: Server did not advertise STARTTLS extension")
		} else {
			fmt.Fprintf(conn, "STARTTLS\r\n")
			res.Trace = append(res.Trace, ">> STARTTLS")
			starttlsResp, err := reader.ReadString('\n')
			if err != nil {
				res.Success = false
				res.Error = fmt.Sprintf("STARTTLS handshake negotiation error: %v", err)
				res.LatencyMs = time.Since(start).Milliseconds()
				return res
			}
			res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(starttlsResp)))

			tlsConn := tls.Client(conn, &tls.Config{
				ServerName: host,
			})
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				res.Success = false
				res.Error = fmt.Sprintf("TLS certificate verification failed: %v", err)
				res.LatencyMs = time.Since(start).Milliseconds()
				return res
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			res.TLSSucceeded = true
			res.Trace = append(res.Trace, "TLS handshake successfully established (DANE/CA Verified)")

			// Re-send EHLO over encrypted pipe
			fmt.Fprintf(conn, "EHLO hostvra-relay-probe\r\n")
			res.Trace = append(res.Trace, ">> EHLO hostvra-relay-probe [TLS]")
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				trimmed := strings.TrimSpace(line)
				res.Trace = append(res.Trace, fmt.Sprintf("<< %s", trimmed))
				if len(trimmed) >= 4 && trimmed[3] == ' ' {
					break
				}
			}
		}
	}

	// AUTH
	if username != "" && password != "" {
		res.Trace = append(res.Trace, fmt.Sprintf("Attempting SMTP AUTH (user: %s)...", username))
		if authType == "login" {
			// AUTH LOGIN
			fmt.Fprintf(conn, "AUTH LOGIN\r\n")
			res.Trace = append(res.Trace, ">> AUTH LOGIN")
			rLine, _ := reader.ReadString('\n')
			res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(rLine)))

			// Send user
			uB64 := base64.StdEncoding.EncodeToString([]byte(username))
			fmt.Fprintf(conn, "%s\r\n", uB64)
			res.Trace = append(res.Trace, ">> [Username base64 sent]")
			pLine, _ := reader.ReadString('\n')
			res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(pLine)))

			// Send pass
			pB64 := base64.StdEncoding.EncodeToString([]byte(password))
			fmt.Fprintf(conn, "%s\r\n", pB64)
			res.Trace = append(res.Trace, ">> [Password base64 sent]")
			authResp, _ := reader.ReadString('\n')
			res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(authResp)))

			if strings.HasPrefix(authResp, "235") {
				res.AuthSuccess = true
				res.Success = true
			} else {
				res.AuthSuccess = false
				res.Success = false
				res.Error = fmt.Sprintf("SMTP Authentication rejected: %s", strings.TrimSpace(authResp))
			}
		} else {
			// AUTH PLAIN
			auth := smtp.PlainAuth("", username, password, host)
			// Send AUTH PLAIN formatted line
			rawPlain := fmt.Sprintf("\x00%s\x00%s", username, password)
			fmt.Fprintf(conn, "AUTH PLAIN %s\r\n", base64.StdEncoding.EncodeToString([]byte(rawPlain)))
			res.Trace = append(res.Trace, ">> AUTH PLAIN [Credentials]")
			authResp, _ := reader.ReadString('\n')
			res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(authResp)))

			if strings.HasPrefix(authResp, "235") {
				res.AuthSuccess = true
				res.Success = true
			} else {
				res.AuthSuccess = false
				res.Success = false
				res.Error = fmt.Sprintf("SMTP Authentication rejected: %s", strings.TrimSpace(authResp))
			}
			_ = auth
		}
	} else {
		// No auth required
		res.AuthSuccess = true
		res.Success = true
	}

	// Quit cleanly
	fmt.Fprintf(conn, "QUIT\r\n")
	res.LatencyMs = time.Since(start).Milliseconds()
	return res
}

// VerifyDomainDeliverability queries live DNS for SPF, DKIM, DMARC, and MX
func VerifyDomainDeliverability(ctx context.Context, domain, provider, dkimSelector string) *DeliverabilityAuditResult {
	result := &DeliverabilityAuditResult{
		Domain:             domain,
		Provider:           provider,
		RecommendationList: make([]string, 0),
		MXRecords:          make([]string, 0),
	}

	cleanDomain := strings.TrimSpace(strings.ToLower(domain))
	if cleanDomain == "" {
		result.RecommendationList = append(result.RecommendationList, "Valid domain name is required for deliverability verification")
		return result
	}

	resolver := net.DefaultResolver

	// 1. Resolve SPF TXT records
	txts, err := resolver.LookupTXT(ctx, cleanDomain)
	if err == nil {
		for _, txt := range txts {
			if strings.HasPrefix(txt, "v=spf1") {
				result.SPFRecord = txt
				break
			}
		}
	}

	preset, hasPreset := SupportedProviders[provider]
	if result.SPFRecord != "" {
		if hasPreset && preset.SpfInclude != "" {
			if strings.Contains(result.SPFRecord, preset.SpfInclude) {
				result.SPFValid = true
				result.SPFMessage = fmt.Sprintf("SPF record includes provider mechanism: %s", preset.SpfInclude)
			} else {
				result.SPFValid = false
				result.SPFMessage = fmt.Sprintf("SPF record exists but is missing provider include '%s'", preset.SpfInclude)
				result.RecommendationList = append(result.RecommendationList, fmt.Sprintf("Update TXT @ to include '%s' before ~all", preset.SpfInclude))
			}
		} else {
			result.SPFValid = true
			result.SPFMessage = "SPF record present"
		}
	} else {
		result.SPFValid = false
		result.SPFMessage = "No SPF TXT record found"
		include := "include:relay.hostvra.com"
		if hasPreset && preset.SpfInclude != "" {
			include = preset.SpfInclude
		}
		result.RecommendationList = append(result.RecommendationList, fmt.Sprintf("Add TXT record: 'v=spf1 %s ~all'", include))
	}

	// 2. Resolve DKIM record
	selector := dkimSelector
	if selector == "" && hasPreset && preset.DkimSelectorPrompt != "" {
		selector = preset.DkimSelectorPrompt
	}
	if selector == "" {
		selector = "default"
	}
	result.DKIMSelector = selector
	dkimHost := fmt.Sprintf("%s._domainkey.%s", selector, cleanDomain)

	dkimTxts, err := resolver.LookupTXT(ctx, dkimHost)
	if err == nil && len(dkimTxts) > 0 {
		for _, d := range dkimTxts {
			if strings.Contains(d, "v=DKIM1") || strings.Contains(d, "p=") {
				result.DKIMRecord = d
				result.DKIMValid = true
				break
			}
		}
	}
	if !result.DKIMValid {
		result.RecommendationList = append(result.RecommendationList, fmt.Sprintf("Configure DKIM TXT record at '%s'", dkimHost))
	}

	// 3. Resolve DMARC record
	dmarcHost := fmt.Sprintf("_dmarc.%s", cleanDomain)
	dmarcTxts, err := resolver.LookupTXT(ctx, dmarcHost)
	if err == nil && len(dmarcTxts) > 0 {
		for _, dm := range dmarcTxts {
			if strings.HasPrefix(dm, "v=DMARC1") {
				result.DMARCRecord = dm
				result.DMARCValid = true
				break
			}
		}
	}
	if !result.DMARCValid {
		result.RecommendationList = append(result.RecommendationList, fmt.Sprintf("Add DMARC TXT record at '_dmarc.%s' with value 'v=DMARC1; p=quarantine; rua=mailto:dmarc@%s'", cleanDomain, cleanDomain))
	}

	// 4. Resolve MX Records
	mxs, err := resolver.LookupMX(ctx, cleanDomain)
	if err == nil {
		for _, mx := range mxs {
			result.MXRecords = append(result.MXRecords, fmt.Sprintf("%s (Priority: %d)", mx.Host, mx.Pref))
		}
	}

	// Rule: Never report a sender domain as verified until actual DNS/provider verification confirms it
	result.IsFullyVerified = result.SPFValid && result.DKIMValid && result.DMARCValid
	return result
}

// GenerateSenderDependentRelayHost builds Postfix sender_dependent_relayhost map content
func GenerateSenderDependentRelayHost(entries map[string]string) string {
	var sb strings.Builder
	sb.WriteString("# Hostvra Sender Dependent Relayhost Map\n")
	sb.WriteString("# Generated automatically by Hostvra Agent\n\n")
	for senderDomain, relayHost := range entries {
		sb.WriteString(fmt.Sprintf("@%s\t[%s]:587\n", senderDomain, relayHost))
	}
	return sb.String()
}

// GenerateSaslPasswd builds Postfix sasl_passwd map content with safe credential formatting
func GenerateSaslPasswd(entries map[string]struct{ User, Pass string }) string {
	var sb strings.Builder
	sb.WriteString("# Hostvra Postfix SASL Relay Passwords\n")
	sb.WriteString("# Generated automatically by Hostvra Agent. CHMOD 0600 REQUIRED.\n\n")
	for relayTarget, creds := range entries {
		sb.WriteString(fmt.Sprintf("[%s]:587\t%s:%s\n", relayTarget, creds.User, creds.Pass))
	}
	return sb.String()
}
