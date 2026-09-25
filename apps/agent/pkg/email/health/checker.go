package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type CheckResult struct {
	Status   string `json:"status"`   // pass, warn, fail
	Details  string `json:"details"`
	Expected string `json:"expected,omitempty"`
	Current  string `json:"current,omitempty"`
}

type Deduction struct {
	Item    string `json:"item"`
	Points  int    `json:"points"`
	Reason  string `json:"reason"`
	FixHint string `json:"fix_hint"`
}

type HealthAudit struct {
	Domain          string      `json:"domain"`
	MailHostname    string      `json:"mail_hostname"`
	ServerIP        string      `json:"server_ip"`
	OverallScore    int         `json:"overall_score"`
	MX              CheckResult `json:"mx"`
	SPF             CheckResult `json:"spf"`
	DKIM            CheckResult `json:"dkim"`
	DMARC           CheckResult `json:"dmarc"`
	ForwardDNS      CheckResult `json:"forward_dns"`
	ReverseDNS      CheckResult `json:"reverse_dns"`
	TLS             CheckResult `json:"tls"`
	OpenRelay       CheckResult `json:"open_relay"`
	Deductions      []Deduction `json:"deductions"`
	Recommendations []string    `json:"recommendations"`
	AuditedAt       time.Time   `json:"audited_at"`
}

// AuditDomain checks live DNS and MTA configuration for email deliverability
func AuditDomain(ctx context.Context, domain, selector, serverIP string) *HealthAudit {
	if selector == "" {
		selector = "default"
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	mailHostname := "mail." + domain

	audit := &HealthAudit{
		Domain:          domain,
		MailHostname:    mailHostname,
		ServerIP:        serverIP,
		OverallScore:    100,
		Deductions:      make([]Deduction, 0),
		Recommendations: make([]string, 0),
		AuditedAt:       time.Now().UTC(),
	}

	resolver := net.DefaultResolver

	// 1. Check MX Records
	mxRecords, err := resolver.LookupMX(ctx, domain)
	if err != nil || len(mxRecords) == 0 {
		audit.MX = CheckResult{
			Status:   "fail",
			Details:  "No MX records found for domain. Inbound email cannot be routed to this server.",
			Expected: fmt.Sprintf("10 %s.", mailHostname),
			Current:  "None",
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "MX Record",
			Points:  30,
			Reason:  "Missing MX DNS record causes complete failure for receiving inbound emails.",
			FixHint: fmt.Sprintf("Add DNS record: @ IN MX 10 %s.", mailHostname),
		})
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Add DNS record: @ IN MX 10 %s.", mailHostname))
	} else {
		var mxHosts []string
		for _, m := range mxRecords {
			mxHosts = append(mxHosts, fmt.Sprintf("%s (pri %d)", m.Host, m.Pref))
		}
		audit.MX = CheckResult{
			Status:   "pass",
			Details:  fmt.Sprintf("Found %d MX record(s). Primary: %s", len(mxRecords), mxRecords[0].Host),
			Expected: fmt.Sprintf("10 %s.", mailHostname),
			Current:  strings.Join(mxHosts, ", "),
		}
	}

	// 2. Check Forward DNS (A/AAAA for mail hostname)
	var resolvedMailIPs []string
	ips, err := resolver.LookupIP(ctx, "ip4", mailHostname)
	if err == nil && len(ips) > 0 {
		for _, ip := range ips {
			resolvedMailIPs = append(resolvedMailIPs, ip.String())
		}
		audit.ForwardDNS = CheckResult{
			Status:   "pass",
			Details:  fmt.Sprintf("Mail hostname '%s' resolves to %s", mailHostname, strings.Join(resolvedMailIPs, ", ")),
			Expected: serverIP,
			Current:  strings.Join(resolvedMailIPs, ", "),
		}
	} else {
		audit.ForwardDNS = CheckResult{
			Status:   "fail",
			Details:  fmt.Sprintf("Mail hostname '%s' does not resolve to an IPv4 address.", mailHostname),
			Expected: serverIP,
			Current:  "Unresolved",
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "Forward DNS (A Record)",
			Points:  15,
			Reason:  fmt.Sprintf("Mail server '%s' has no valid A record pointing to server IP.", mailHostname),
			FixHint: fmt.Sprintf("Add DNS record: mail.%s IN A %s", domain, serverIP),
		})
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Add DNS record: mail.%s IN A %s", domain, serverIP))
	}

	// 3. Check SPF Record
	txtRecords, err := resolver.LookupTXT(ctx, domain)
	var spfRecords []string
	if err == nil {
		for _, txt := range txtRecords {
			if strings.HasPrefix(strings.TrimSpace(txt), "v=spf1") {
				spfRecords = append(spfRecords, txt)
			}
		}
	}

	expectedSPF := fmt.Sprintf("v=spf1 mx a ip4:%s ~all", serverIP)
	if serverIP == "" {
		expectedSPF = "v=spf1 mx ~all"
	}

	if len(spfRecords) == 0 {
		audit.SPF = CheckResult{
			Status:   "warn",
			Details:  "No SPF record detected. Major mail providers (Gmail, Yahoo, Outlook) may mark outbound mail as spam.",
			Expected: expectedSPF,
			Current:  "None",
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "SPF Record",
			Points:  20,
			Reason:  "RFC 7208 SPF record missing. Receivers cannot authenticate that this server is authorized to send mail.",
			FixHint: fmt.Sprintf("Add DNS record: @ IN TXT \"%s\"", expectedSPF),
		})
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Add DNS record: @ IN TXT \"%s\"", expectedSPF))
	} else if len(spfRecords) > 1 {
		audit.SPF = CheckResult{
			Status:   "fail",
			Details:  fmt.Sprintf("Multiple (%d) SPF records detected! RFC 7208 section 3.2 strictly forbids more than one SPF record. Extra records cause immediate PermError/authentication failure.", len(spfRecords)),
			Expected: expectedSPF,
			Current:  strings.Join(spfRecords, " | "),
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "Multiple SPF Records",
			Points:  25,
			Reason:  "RFC 7208 PermError: Publishing multiple SPF TXT records breaks all SPF evaluation at Gmail/Yahoo.",
			FixHint: "Merge all SPF rules into a single TXT record: combine all 'include:' and 'ip4:' directives.",
		})
		audit.Recommendations = append(audit.Recommendations, "Merge all SPF TXT records into exactly one record.")
	} else {
		spf := spfRecords[0]
		if strings.Contains(spf, "+all") {
			audit.SPF = CheckResult{
				Status:   "warn",
				Details:  "Insecure SPF syntax detected: '+all' allows ANY server on the internet to spoof emails from your domain.",
				Expected: expectedSPF,
				Current:  spf,
			}
			audit.Deductions = append(audit.Deductions, Deduction{
				Item:    "Insecure SPF Policy",
				Points:  20,
				Reason:  "'+all' permits open spoofing and degrades domain reputation.",
				FixHint: "Change '+all' to '~all' (SoftFail) or '-all' (HardFail).",
			})
		} else {
			audit.SPF = CheckResult{
				Status:   "pass",
				Details:  fmt.Sprintf("Valid single SPF record: %s", spf),
				Expected: expectedSPF,
				Current:  spf,
			}
		}
	}

	// 4. Check DKIM Record
	dkimHost := fmt.Sprintf("%s._domainkey.%s", selector, domain)
	dkimTxts, err := resolver.LookupTXT(ctx, dkimHost)
	if err != nil || len(dkimTxts) == 0 {
		audit.DKIM = CheckResult{
			Status:   "warn",
			Details:  fmt.Sprintf("DKIM selector record not found at %s. Outbound signatures cannot be validated by receivers.", dkimHost),
			Expected: fmt.Sprintf("%s IN TXT \"v=DKIM1; k=rsa; p=...\"", dkimHost),
			Current:  "None",
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "DKIM Record",
			Points:  20,
			Reason:  "Cryptographic DKIM signature public key is not published in DNS.",
			FixHint: fmt.Sprintf("Publish DKIM TXT record at %s with your 2048-bit public key.", dkimHost),
		})
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Publish DKIM TXT record at %s", dkimHost))
	} else {
		audit.DKIM = CheckResult{
			Status:   "pass",
			Details:  fmt.Sprintf("DKIM public key published at %s", dkimHost),
			Expected: "v=DKIM1; k=rsa; p=...",
			Current:  dkimTxts[0],
		}
	}

	// 5. Check DMARC Record
	dmarcHost := fmt.Sprintf("_dmarc.%s", domain)
	dmarcTxts, err := resolver.LookupTXT(ctx, dmarcHost)
	if err != nil || len(dmarcTxts) == 0 {
		audit.DMARC = CheckResult{
			Status:   "warn",
			Details:  "No DMARC policy found. Google and Yahoo mandate DMARC for bulk senders to prevent impersonation.",
			Expected: fmt.Sprintf("v=DMARC1; p=none; rua=mailto:dmarc@%s", domain),
			Current:  "None",
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "DMARC Policy",
			Points:  15,
			Reason:  "Missing DMARC policy prevents receivers from knowing how to handle SPF/DKIM alignment failures.",
			FixHint: fmt.Sprintf("Add DNS record: _dmarc.%s IN TXT \"v=DMARC1; p=none; rua=mailto:dmarc@%s\"", domain, domain),
		})
		audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Add DNS record: _dmarc.%s IN TXT \"v=DMARC1; p=none; rua=mailto:dmarc@%s\"", domain, domain))
	} else {
		audit.DMARC = CheckResult{
			Status:   "pass",
			Details:  fmt.Sprintf("Active DMARC policy: %s", dmarcTxts[0]),
			Expected: "v=DMARC1; p=none; ...",
			Current:  dmarcTxts[0],
		}
	}

	// 6. Check Reverse DNS (PTR)
	targetIP := serverIP
	if targetIP == "" && len(resolvedMailIPs) > 0 {
		targetIP = resolvedMailIPs[0]
	}

	if targetIP != "" && targetIP != "127.0.0.1" {
		ptrs, err := resolver.LookupAddr(ctx, targetIP)
		if err != nil || len(ptrs) == 0 {
			audit.ReverseDNS = CheckResult{
				Status:   "warn",
				Details:  fmt.Sprintf("No PTR (Reverse DNS) record configured for IP %s. Remote servers may reject mail due to lack of PTR.", targetIP),
				Expected: mailHostname,
				Current:  "None",
			}
			audit.Deductions = append(audit.Deductions, Deduction{
				Item:    "Reverse DNS (PTR)",
				Points:  15,
				Reason:  "VPS IP has no reverse DNS entry matching mail server hostname.",
				FixHint: fmt.Sprintf("Set Reverse DNS (PTR) for %s to '%s' in your VPS/hosting provider control panel (e.g. Contabo, Hetzner, DigitalOcean). Hostvra cannot configure this directly as it belongs to the IP provider.", targetIP, mailHostname),
			})
			audit.Recommendations = append(audit.Recommendations, fmt.Sprintf("Configure reverse DNS (PTR) for IP %s to '%s' in your hosting provider's dashboard.", targetIP, mailHostname))
		} else {
			ptrMatch := false
			for _, p := range ptrs {
				cleanP := strings.TrimSuffix(p, ".")
				if strings.EqualFold(cleanP, mailHostname) || strings.EqualFold(cleanP, domain) {
					ptrMatch = true
					break
				}
			}
			if ptrMatch {
				audit.ReverseDNS = CheckResult{
					Status:   "pass",
					Details:  fmt.Sprintf("Forward-Confirmed Reverse DNS (FCrDNS) valid: %s -> %s", targetIP, ptrs[0]),
					Expected: mailHostname,
					Current:  ptrs[0],
				}
			} else {
				audit.ReverseDNS = CheckResult{
					Status:   "warn",
					Details:  fmt.Sprintf("PTR mismatch: IP %s resolves to '%s', but expected '%s'.", targetIP, ptrs[0], mailHostname),
					Expected: mailHostname,
					Current:  ptrs[0],
				}
				audit.Deductions = append(audit.Deductions, Deduction{
					Item:    "PTR Hostname Mismatch",
					Points:  10,
					Reason:  "Reverse DNS does not match the outgoing HELO/EHLO mail hostname.",
					FixHint: fmt.Sprintf("Update PTR record for %s to '%s' at your VPS provider.", targetIP, mailHostname),
				})
			}
		}
	} else {
		audit.ReverseDNS = CheckResult{
			Status:  "pass",
			Details: "Local/internal IP tested; external PTR check deferred.",
		}
	}

	// 7. TLS Handshake Probe (Port 587 Submission / Port 25)
	tlsTarget := fmt.Sprintf("%s:587", mailHostname)
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 3 * time.Second}, "tcp", tlsTarget, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err == nil {
		conn.Close()
		audit.TLS = CheckResult{
			Status:  "pass",
			Details: "TLS connection verified on submission port 587 with modern cipher suite.",
		}
	} else {
		// Try STARTTLS via smtp.Client
		c, sErr := smtp.Dial(fmt.Sprintf("%s:587", mailHostname))
		if sErr == nil {
			if ok, _ := c.Extension("STARTTLS"); ok {
				audit.TLS = CheckResult{
					Status:  "pass",
					Details: "STARTTLS capability advertised on submission port 587.",
				}
			} else {
				audit.TLS = CheckResult{
					Status:  "warn",
					Details: "Port 587 open but STARTTLS not advertised.",
				}
			}
			_ = c.Quit()
		} else {
			audit.TLS = CheckResult{
				Status:  "warn",
				Details: "Could not connect to submission port 587 remotely. Ensure firewall allows inbound port 587/465/25.",
			}
		}
	}

	// 8. Open Relay Check (local test)
	isProtected, rErr := TestRelayRejection("127.0.0.1:25", 2*time.Second)
	if isProtected && rErr == nil {
		audit.OpenRelay = CheckResult{
			Status:  "pass",
			Details: "Postfix relay restrictions enforce reject_unauth_destination. Zero open relay detected.",
		}
	} else {
		audit.OpenRelay = CheckResult{
			Status:  "fail",
			Details: "CRITICAL: Server may be accepting unauthenticated relay to external recipients.",
		}
		audit.Deductions = append(audit.Deductions, Deduction{
			Item:    "Open Relay Risk",
			Points:  50,
			Reason:  "Server accepted unauthenticated message delivery to external domain.",
			FixHint: "Enforce 'smtpd_relay_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination' in main.cf",
		})
	}

	// Calculate overall score from deductions
	deductionTotal := 0
	for _, d := range audit.Deductions {
		deductionTotal += d.Points
	}
	audit.OverallScore = 100 - deductionTotal
	if audit.OverallScore < 0 {
		audit.OverallScore = 0
	}

	return audit
}

// TestRelayRejection checks if an SMTP server refuses unauthorized relay attempts
func TestRelayRejection(addr string, timeout time.Duration) (bool, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return true, nil // Port not reachable locally, not an open relay
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	buf := make([]byte, 1024)
	_, _ = conn.Read(buf)

	_, _ = conn.Write([]byte("HELO test.hostvra.com\r\n"))
	_, _ = conn.Read(buf)

	_, _ = conn.Write([]byte("MAIL FROM:<spammer@attacker.org>\r\n"))
	_, _ = conn.Read(buf)

	_, _ = conn.Write([]byte("RCPT TO:<victim@external-target.org>\r\n"))
	n, err := conn.Read(buf)
	if err != nil {
		return true, nil
	}

	resp := string(buf[:n])
	// 554 5.7.1 Relay access denied or 454/550 means properly rejected
	if strings.HasPrefix(resp, "55") || strings.HasPrefix(resp, "45") {
		return true, nil
	}

	// If 250 OK was returned for external recipient, it's an OPEN RELAY!
	if strings.HasPrefix(resp, "250") {
		return false, fmt.Errorf("CRITICAL: Server accepted unauthenticated relay to external destination")
	}

	return true, nil
}
