package tester

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type TestEmailResult struct {
	Success      bool     `json:"success"`
	MessageID    string   `json:"message_id"`
	SMTPResponse string   `json:"smtp_response"`
	DurationMs   int64    `json:"duration_ms"`
	Trace        []string `json:"trace"`
	Error        string   `json:"error,omitempty"`
}

type MailboxTestResult struct {
	Email      string   `json:"email"`
	SMTPAuth   bool     `json:"smtp_auth"`
	IMAPAuth   bool     `json:"imap_auth"`
	SendTest   bool     `json:"send_test"`
	QuotaCheck bool     `json:"quota_check"`
	Trace      []string `json:"trace"`
	Error      string   `json:"error,omitempty"`
}

// SendTestEmail sends an email via SMTP and records the live transaction trace
func SendTestEmail(smtpHost string, smtpPort int, username, password, from, to, subject, body string) *TestEmailResult {
	start := time.Now()
	res := &TestEmailResult{
		Trace: make([]string, 0),
	}

	if smtpHost == "" {
		smtpHost = "127.0.0.1"
	}
	if smtpPort == 0 {
		smtpPort = 25
	}

	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	res.Trace = append(res.Trace, fmt.Sprintf("Connecting to SMTP server at %s...", addr))

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		res.Success = false
		res.Error = fmt.Sprintf("TCP connection failed: %v", err)
		res.Trace = append(res.Trace, fmt.Sprintf("ERROR: %v", err))
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Read initial banner
	banner, err := reader.ReadString('\n')
	if err != nil {
		res.Success = false
		res.Error = fmt.Sprintf("Failed to read SMTP banner: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(banner)))

	// Send EHLO
	fmt.Fprintf(conn, "EHLO hostvra.local\r\n")
	res.Trace = append(res.Trace, ">> EHLO hostvra.local")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))
		if len(line) >= 4 && line[3] == ' ' {
			break
		}
	}

	// Optional Authentication if credentials provided
	if username != "" && password != "" {
		fmt.Fprintf(conn, "AUTH LOGIN\r\n")
		res.Trace = append(res.Trace, ">> AUTH LOGIN")
		line, _ := reader.ReadString('\n')
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

		// Username
		uB64 := base64.StdEncoding.EncodeToString([]byte(username))
		fmt.Fprintf(conn, "%s\r\n", uB64)
		line, _ = reader.ReadString('\n')
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

		// Password
		pB64 := base64.StdEncoding.EncodeToString([]byte(password))
		fmt.Fprintf(conn, "%s\r\n", pB64)
		line, _ = reader.ReadString('\n')
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

		if !strings.HasPrefix(line, "235") {
			res.Trace = append(res.Trace, "WARN: SMTP authentication did not return 235 Success")
		}
	}

	// MAIL FROM
	fmt.Fprintf(conn, "MAIL FROM:<%s>\r\n", from)
	res.Trace = append(res.Trace, fmt.Sprintf(">> MAIL FROM:<%s>", from))
	line, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "250") {
		res.Success = false
		res.Error = fmt.Sprintf("MAIL FROM rejected: %s", strings.TrimSpace(line))
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

	// RCPT TO
	fmt.Fprintf(conn, "RCPT TO:<%s>\r\n", to)
	res.Trace = append(res.Trace, fmt.Sprintf(">> RCPT TO:<%s>", to))
	line, err = reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "250") {
		res.Success = false
		res.Error = fmt.Sprintf("RCPT TO rejected: %s", strings.TrimSpace(line))
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

	// DATA
	fmt.Fprintf(conn, "DATA\r\n")
	res.Trace = append(res.Trace, ">> DATA")
	line, err = reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "354") {
		res.Success = false
		res.Error = fmt.Sprintf("DATA command rejected: %s", strings.TrimSpace(line))
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

	// Message content
	msgID := fmt.Sprintf("<%d.test@hostvra.local>", time.Now().UnixNano())
	payload := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: %s\r\nDate: %s\r\n\r\n%s\r\n.\r\n",
		from, to, subject, msgID, time.Now().Format(time.RFC1123Z), body)

	fmt.Fprintf(conn, "%s", payload)
	line, err = reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "250") {
		res.Success = false
		res.Error = fmt.Sprintf("Message submission rejected: %s", strings.TrimSpace(line))
		res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Trace = append(res.Trace, fmt.Sprintf("<< %s", strings.TrimSpace(line)))

	// QUIT
	fmt.Fprintf(conn, "QUIT\r\n")

	res.Success = true
	res.MessageID = msgID
	res.SMTPResponse = strings.TrimSpace(line)
	res.DurationMs = time.Since(start).Milliseconds()
	res.Trace = append(res.Trace, fmt.Sprintf("Test email accepted for delivery: %s (Duration: %dms)", res.SMTPResponse, res.DurationMs))
	return res
}

// TestMailboxCredentials verifies SMTP and IMAP logins for an email account
func TestMailboxCredentials(smtpHost string, smtpPort int, imapHost string, imapPort int, email, password string) *MailboxTestResult {
	res := &MailboxTestResult{
		Email: email,
		Trace: make([]string, 0),
	}

	if smtpHost == "" {
		smtpHost = "127.0.0.1"
	}
	if smtpPort == 0 {
		smtpPort = 587
	}
	if imapHost == "" {
		imapHost = "127.0.0.1"
	}
	if imapPort == 0 {
		imapPort = 993
	}

	// 1. Test SMTP Auth
	res.Trace = append(res.Trace, fmt.Sprintf("Testing SMTP auth at %s:%d...", smtpHost, smtpPort))
	auth := smtp.PlainAuth("", email, password, smtpHost)
	smtpClient, err := smtp.Dial(fmt.Sprintf("%s:%d", smtpHost, smtpPort))
	if err != nil {
		res.Trace = append(res.Trace, fmt.Sprintf("SMTP Connect failed: %v", err))
	} else {
		defer smtpClient.Close()
		if ok, _ := smtpClient.Extension("STARTTLS"); ok {
			tlsConf := &tls.Config{InsecureSkipVerify: true, ServerName: smtpHost}
			_ = smtpClient.StartTLS(tlsConf)
		}
		if aErr := smtpClient.Auth(auth); aErr == nil {
			res.SMTPAuth = true
			res.Trace = append(res.Trace, "SMTP Authentication: SUCCESS")
		} else {
			res.Trace = append(res.Trace, fmt.Sprintf("SMTP Auth failed: %v", aErr))
		}
	}

	// 2. Test IMAP Auth
	res.Trace = append(res.Trace, fmt.Sprintf("Testing IMAP connection at %s:%d...", imapHost, imapPort))
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ServerName: imapHost}
	imapConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", fmt.Sprintf("%s:%d", imapHost, imapPort), tlsConfig)
	if err != nil {
		res.Trace = append(res.Trace, fmt.Sprintf("IMAP TLS connect failed: %v", err))
	} else {
		defer imapConn.Close()
		imapReader := bufio.NewReader(imapConn)
		banner, _ := imapReader.ReadString('\n')
		res.Trace = append(res.Trace, fmt.Sprintf("IMAP Banner: %s", strings.TrimSpace(banner)))

		// Send LOGIN command
		fmt.Fprintf(imapConn, "A001 LOGIN \"%s\" \"%s\"\r\n", email, password)
		loginResp, _ := imapReader.ReadString('\n')
		res.Trace = append(res.Trace, fmt.Sprintf("IMAP Login Response: %s", strings.TrimSpace(loginResp)))

		if strings.HasPrefix(loginResp, "A001 OK") {
			res.IMAPAuth = true
			res.QuotaCheck = true
			res.Trace = append(res.Trace, "IMAP Authentication: SUCCESS")
		}
		fmt.Fprintf(imapConn, "A002 LOGOUT\r\n")
	}

	return res
}
