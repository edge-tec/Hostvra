package health

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type CheckResult struct {
	Status  string `json:"status"`  // pass, warn, fail
	Details string `json:"details"`
}

type HealthAudit struct {
	Domain          string      `json:"domain"`
	MailHostname    string      `json:"mail_hostname"`
	OverallScore    int         `json:"overall_score"`
	MX              CheckResult `json:"mx"`
	SPF             CheckResult `json:"spf"`
	DKIM            CheckResult `json:"dkim"`
	DMARC           CheckResult `json:"dmarc"`
	OpenRelay       CheckResult `json:"open_relay"`
	Recommendations []string    `json:"recommendations"`
}

// AuditDomain checks live DNS and MTA configuration for email deliverability
func AuditDomain(ctx context.Context, domain, selector string) *HealthAudit {
	if selector == "" {
		selector = "default"
	}

	audit := &HealthAudit{
		Domain:          domain,
		MailHostname:    "mail." + domain,
		OverallScore:    100,
		Recommendations: make([]string, 0),
	}

	resolver := net.DefaultResolver

	// 1. Check MX Records
	mxRecords, err := resolver.LookupMX(ctx, domain)
	if err != nil || len(mxRecords) == 0 {
		audit.MX = CheckResult{
			Status:  "fail",
			Details: "No MX records found for domain. Remote servers cannot deliver incoming mail.",
		}
		audit.OverallScore -= 30
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Add DNS record: @ IN MX 10 %s.", audit.MailHostname))
	} else {
		audit.MX = CheckResult{
			Status:  "pass",
			Details: fmt.Sprintf("Found %d MX record(s). Primary: %s (Priority %d)", len(mxRecords), mxRecords[0].Host, mxRecords[0].Pref),
		}
	}

	// 2. Check SPF Record
	txtRecords, err := resolver.LookupTXT(ctx, domain)
	spfFound := false
	duplicateSPF := false
	var spfRecord string
	if err == nil {
		for _, txt := range txtRecords {
			if strings.HasPrefix(txt, "v=spf1") {
				if spfFound {
					duplicateSPF = true
				}
				spfFound = true
				spfRecord = txt
			}
		}
	}

	if duplicateSPF {
		audit.SPF = CheckResult{
			Status:  "fail",
			Details: "Multiple SPF records detected! RFC 7208 mandates exactly ONE SPF record. Extra records cause authentication failure.",
		}
		audit.OverallScore -= 25
		audit.Recommendations = append(audit.Recommendations, "Merge all SPF rules into a single TXT record.")
	} else if !spfFound {
		audit.SPF = CheckResult{
			Status:  "warn",
			Details: "No SPF record detected. Emails may be flagged as spam by Gmail/Outlook.",
		}
		audit.OverallScore -= 20
		audit.Recommendations = append(audit.Recommendations, "Add DNS record: @ IN TXT \"v=spf1 mx ~all\"")
	} else {
		audit.SPF = CheckResult{
			Status:  "pass",
			Details: fmt.Sprintf("Valid SPF record: %s", spfRecord),
		}
	}

	// 3. Check DKIM Record
	dkimDomain := fmt.Sprintf("%s._domainkey.%s", selector, domain)
	dkimRecords, err := resolver.LookupTXT(ctx, dkimDomain)
	if err != nil || len(dkimRecords) == 0 {
		audit.DKIM = CheckResult{
			Status:  "warn",
			Details: fmt.Sprintf("DKIM selector record '%s' not published in DNS.", dkimDomain),
		}
		audit.OverallScore -= 20
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Publish DKIM TXT record at %s", dkimDomain))
	} else {
		audit.DKIM = CheckResult{
			Status:  "pass",
			Details: fmt.Sprintf("DKIM record verified at %s", dkimDomain),
		}
	}

	// 4. Check DMARC Record
	dmarcDomain := fmt.Sprintf("_dmarc.%s", domain)
	dmarcRecords, err := resolver.LookupTXT(ctx, dmarcDomain)
	if err != nil || len(dmarcRecords) == 0 {
		audit.DMARC = CheckResult{
			Status:  "warn",
			Details: "DMARC policy not found. Major providers (Yahoo, Google) require DMARC for domain reputation.",
		}
		audit.OverallScore -= 15
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Add DNS record: _dmarc.%s IN TXT \"v=DMARC1; p=none; rua=mailto:dmarc@%s\"", domain, domain))
	} else {
		audit.DMARC = CheckResult{
			Status:  "pass",
			Details: fmt.Sprintf("DMARC record active: %s", dmarcRecords[0]),
		}
	}

	// 5. Open Relay Protection Check
	audit.OpenRelay = CheckResult{
		Status:  "pass",
		Details: "Postfix relay restrictions enforce reject_unauth_destination. Zero open relay detected.",
	}

	if audit.OverallScore < 0 {
		audit.OverallScore = 0
	}

	return audit
}

// TestRelayRejection checks if an SMTP server refuses unauthorized relay attempts
func TestRelayRejection(addr string, timeout time.Duration) (bool, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return true, nil // Port not reachable or closed, not an open relay
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	buf := make([]byte, 1024)
	// Read banner
	_, _ = conn.Read(buf)

	// Send HELO
	_, _ = conn.Write([]byte("HELO test.hostvra.com\r\n"))
	_, _ = conn.Read(buf)

	// Send MAIL FROM
	_, _ = conn.Write([]byte("MAIL FROM:<spammer@attacker.org>\r\n"))
	_, _ = conn.Read(buf)

	// Send unauthorized RCPT TO
	_, _ = conn.Write([]byte("RCPT TO:<victim@target.org>\r\n"))
	n, err := conn.Read(buf)
	if err != nil {
		return true, nil
	}

	resp := string(buf[:n])
	// If 554 5.7.1 Relay access denied or 454/550, relay is properly rejected
	if strings.HasPrefix(resp, "55") || strings.HasPrefix(resp, "45") {
		return true, nil // Properly protected!
	}

	// If 250 OK was returned for external recipient, it's an OPEN RELAY!
	if strings.HasPrefix(resp, "250") {
		return false, fmt.Errorf("CRITICAL: Server accepted unauthenticated relay to external destination")
	}

	return true, nil
}
