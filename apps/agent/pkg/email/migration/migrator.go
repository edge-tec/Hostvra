package migration

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"hostvra/agent/pkg/email/storage"
)

// SourceConnectionOptions defines external IMAP credentials
type SourceConnectionOptions struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseSSL   bool   `json:"use_ssl"`
}

// TestSourceResult reports initial IMAP handshake & auth status
type TestSourceResult struct {
	Success      bool     `json:"success"`
	LatencyMs    int64    `json:"latency_ms"`
	FolderCount  int      `json:"folder_count"`
	Folders      []string `json:"folders"`
	Banner       string   `json:"banner"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// MigrationTask specifies an active migration job
type MigrationTask struct {
	ID                     string                  `json:"id"`
	OrganizationID         string                  `json:"organization_id"`
	DestinationMailboxID   string                  `json:"destination_mailbox_id"`
	DestinationEmail       string                  `json:"destination_email"`
	DestinationMaildirBase string                  `json:"destination_maildir_base"`
	Source                 SourceConnectionOptions `json:"source"`
}

// MigrationProgress tracks live transfer stats
type MigrationProgress struct {
	TaskID            string `json:"task_id"`
	Status            string `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	TotalFolders      int    `json:"total_folders"`
	CurrentFolder     string `json:"current_folder"`
	TotalMessages     int    `json:"total_messages"`
	SyncedMessages    int    `json:"synced_messages"`
	SkippedDuplicates int    `json:"skipped_duplicates"`
	FailedMessages    int    `json:"failed_messages"`
	BytesTransferred  int64  `json:"bytes_transferred"`
	ErrorMessage      string `json:"error_message,omitempty"`
}

type ProgressCallback func(p MigrationProgress)

// IMAPClient implements minimal RFC 3501 client for migration
type IMAPClient struct {
	conn   net.Conn
	reader *bufio.Reader
	tagSeq int
	mu     sync.Mutex
}

func (c *IMAPClient) nextTag() string {
	c.tagSeq++
	return fmt.Sprintf("A%04d", c.tagSeq)
}

func (c *IMAPClient) sendCommand(cmd string) (string, []string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tag := c.nextTag()
	fullCmd := fmt.Sprintf("%s %s\r\n", tag, cmd)
	if _, err := c.conn.Write([]byte(fullCmd)); err != nil {
		return tag, nil, fmt.Errorf("write command failed: %w", err)
	}

	var lines []string
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return tag, lines, fmt.Errorf("read response failed: %w", err)
		}
		trimmed := strings.TrimRight(line, "\r\n")
		lines = append(lines, trimmed)

		if strings.HasPrefix(trimmed, tag+" OK") {
			return tag, lines, nil
		}
		if strings.HasPrefix(trimmed, tag+" NO") || strings.HasPrefix(trimmed, tag+" BAD") {
			return tag, lines, fmt.Errorf("IMAP command '%s' failed: %s", cmd, trimmed)
		}
	}
}

// DialIMAP opens an authenticated IMAP session
func DialIMAP(ctx context.Context, opts SourceConnectionOptions) (*IMAPClient, string, error) {
	if opts.Host == "" {
		return nil, "", fmt.Errorf("source host cannot be empty")
	}
	if opts.Port <= 0 {
		if opts.UseSSL {
			opts.Port = 993
		} else {
			opts.Port = 143
		}
	}

	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	var dialer net.Dialer

	var rawConn net.Conn
	var err error

	if opts.UseSSL {
		tlsConfig := &tls.Config{
			ServerName: opts.Host,
		}
		rawConn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsConfig)
	} else {
		rawConn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, "", fmt.Errorf("connection to %s failed: %w", addr, err)
	}

	client := &IMAPClient{
		conn:   rawConn,
		reader: bufio.NewReader(rawConn),
	}

	// Read initial banner
	banner, err := client.reader.ReadString('\n')
	if err != nil {
		rawConn.Close()
		return nil, "", fmt.Errorf("failed reading IMAP banner: %w", err)
	}
	banner = strings.TrimSpace(banner)
	if !strings.HasPrefix(banner, "* OK") {
		rawConn.Close()
		return nil, banner, fmt.Errorf("invalid IMAP banner from server: %s", banner)
	}

	// Authenticate via LOGIN
	loginCmd := fmt.Sprintf("LOGIN \"%s\" \"%s\"", escapeIMAP(opts.Username), escapeIMAP(opts.Password))
	_, _, err = client.sendCommand(loginCmd)
	if err != nil {
		rawConn.Close()
		return nil, banner, fmt.Errorf("authentication failed for %s: %w", opts.Username, err)
	}

	return client, banner, nil
}

func escapeIMAP(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// TestSourceConnection verifies connectivity, SSL, and credentials
func TestSourceConnection(ctx context.Context, opts SourceConnectionOptions) *TestSourceResult {
	start := time.Now()
	res := &TestSourceResult{
		Folders: make([]string, 0),
	}

	client, banner, err := DialIMAP(ctx, opts)
	if err != nil {
		res.Success = false
		res.ErrorMessage = err.Error()
		res.Banner = banner
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	defer client.conn.Close()

	res.Banner = banner

	// List folders
	_, lines, err := client.sendCommand("LIST \"\" \"*\"")
	if err == nil {
		folderRe := regexp.MustCompile(`^\* LIST \(.*?\) "(.*?)" (.*)$`)
		for _, line := range lines {
			if matches := folderRe.FindStringSubmatch(line); len(matches) == 3 {
				folderName := strings.Trim(matches[2], "\"")
				res.Folders = append(res.Folders, folderName)
			}
		}
	}
	res.FolderCount = len(res.Folders)
	res.Success = true
	res.LatencyMs = time.Since(start).Milliseconds()

	// Clean logout
	_, _, _ = client.sendCommand("LOGOUT")
	return res
}

// RunMigration executes full mailbox transfer while strictly preserving source data
func RunMigration(ctx context.Context, task MigrationTask, progressCb ProgressCallback) error {
	progress := MigrationProgress{
		TaskID: task.ID,
		Status: "running",
	}
	progressCb(progress)

	// Destination Maildir
	parts := strings.Split(task.DestinationEmail, "@")
	if len(parts) != 2 {
		err := fmt.Errorf("invalid destination email: %s", task.DestinationEmail)
		progress.Status = "failed"
		progress.ErrorMessage = err.Error()
		progressCb(progress)
		return err
	}
	domain, localPart := parts[1], parts[0]

	basePath := task.DestinationMaildirBase
	if basePath == "" {
		basePath = "/var/mail/vhosts"
	}

	targetDir, err := storage.EnsureMaildir(basePath, domain, localPart, 5000, 5000)
	if err != nil {
		progress.Status = "failed"
		progress.ErrorMessage = fmt.Sprintf("failed ensuring target maildir: %v", err)
		progressCb(progress)
		return err
	}

	// Dial Source
	client, _, err := DialIMAP(ctx, task.Source)
	if err != nil {
		progress.Status = "failed"
		progress.ErrorMessage = err.Error()
		progressCb(progress)
		return err
	}
	defer client.conn.Close()

	// List folders
	_, lines, err := client.sendCommand("LIST \"\" \"*\"")
	if err != nil {
		progress.Status = "failed"
		progress.ErrorMessage = fmt.Sprintf("failed listing folders: %v", err)
		progressCb(progress)
		return err
	}

	var folderNames []string
	folderRe := regexp.MustCompile(`^\* LIST \(.*?\) "(.*?)" (.*)$`)
	for _, line := range lines {
		if matches := folderRe.FindStringSubmatch(line); len(matches) == 3 {
			folderName := strings.Trim(matches[2], "\"")
			// Skip noselect folders
			if !strings.Contains(strings.ToLower(line), "\\noselect") {
				folderNames = append(folderNames, folderName)
			}
		}
	}

	if len(folderNames) == 0 {
		folderNames = []string{"INBOX"}
	}
	progress.TotalFolders = len(folderNames)
	progressCb(progress)

	// Collect known message-IDs in destination to prevent duplicates
	knownMessageIDs := scanExistingMessageIDs(targetDir)

	for _, folder := range folderNames {
		select {
		case <-ctx.Done():
			progress.Status = "cancelled"
			progress.ErrorMessage = "Migration cancelled by user"
			progressCb(progress)
			return ctx.Err()
		default:
		}

		progress.CurrentFolder = folder
		progressCb(progress)

		// Select folder
		_, selLines, err := client.sendCommand(fmt.Sprintf("SELECT \"%s\"", escapeIMAP(folder)))
		if err != nil {
			// Skip unselectable folder
			continue
		}

		var existsCount int
		for _, sl := range selLines {
			if strings.HasSuffix(sl, "EXISTS") {
				parts := strings.Fields(sl)
				if len(parts) >= 2 {
					fmt.Sscanf(parts[1], "%d", &existsCount)
				}
			}
		}

		if existsCount == 0 {
			continue
		}
		progress.TotalMessages += existsCount
		progressCb(progress)

		// Determine target subfolder in Maildir
		destSubdir := mapFolderToMaildirSubdir(folder)
		folderMaildir := filepath.Join(targetDir, destSubdir, "cur")
		_ = os.MkdirAll(folderMaildir, 0700)

		// Search UIDs
		_, searchLines, err := client.sendCommand("UID SEARCH ALL")
		if err != nil {
			continue
		}

		var uids []string
		for _, sl := range searchLines {
			if strings.HasPrefix(sl, "* SEARCH") {
				uids = append(uids, strings.Fields(strings.TrimPrefix(sl, "* SEARCH"))...)
			}
		}

		// Transfer each message
		for _, uid := range uids {
			select {
			case <-ctx.Done():
				progress.Status = "cancelled"
				progress.ErrorMessage = "Migration cancelled by user"
				progressCb(progress)
				return ctx.Err()
			default:
			}

			msgBytes, err := fetchMessageRFC822(client, uid)
			if err != nil {
				progress.FailedMessages++
				progressCb(progress)
				continue
			}

			// Duplicate check
			msgID := extractMessageID(msgBytes)
			if msgID != "" && knownMessageIDs[msgID] {
				progress.SkippedDuplicates++
				progressCb(progress)
				continue
			}

			// Save to maildir
			filename := fmt.Sprintf("%d.%d.hostvra:2,S", time.Now().Unix(), time.Now().UnixNano()%1000000)
			destPath := filepath.Join(folderMaildir, filename)
			if err := os.WriteFile(destPath, msgBytes, 0600); err == nil {
				progress.SyncedMessages++
				progress.BytesTransferred += int64(len(msgBytes))
				if msgID != "" {
					knownMessageIDs[msgID] = true
				}
			} else {
				progress.FailedMessages++
			}
			progressCb(progress)
		}
	}

	progress.Status = "completed"
	progressCb(progress)

	// Clean logout
	_, _, _ = client.sendCommand("LOGOUT")
	return nil
}

func mapFolderToMaildirSubdir(folder string) string {
	lower := strings.ToLower(folder)
	switch {
	case lower == "inbox":
		return ""
	case strings.Contains(lower, "sent"):
		return ".Sent"
	case strings.Contains(lower, "draft"):
		return ".Drafts"
	case strings.Contains(lower, "trash") || strings.Contains(lower, "bin"):
		return ".Trash"
	case strings.Contains(lower, "junk") || strings.Contains(lower, "spam"):
		return ".Junk"
	default:
		sanitized := strings.ReplaceAll(folder, "/", ".")
		sanitized = strings.ReplaceAll(sanitized, " ", "_")
		return "." + sanitized
	}
}

func scanExistingMessageIDs(maildir string) map[string]bool {
	known := make(map[string]bool)
	_ = filepath.WalkDir(maildir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.Contains(path, "/cur/") {
			f, err := os.Open(path)
			if err == nil {
				defer f.Close()
				buf := make([]byte, 2048)
				n, _ := f.Read(buf)
				if id := extractMessageID(buf[:n]); id != "" {
					known[id] = true
				}
			}
		}
		return nil
	})
	return known
}

func extractMessageID(data []byte) string {
	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		if l == "" || l == "\r" {
			break
		}
		if strings.HasPrefix(strings.ToLower(l), "message-id:") {
			return strings.TrimSpace(strings.TrimPrefix(l[11:], " "))
		}
	}
	return ""
}

func fetchMessageRFC822(c *IMAPClient, uid string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tag := c.nextTag()
	cmd := fmt.Sprintf("%s UID FETCH %s (BODY[])\r\n", tag, uid)
	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return nil, err
	}

	// Read literal response
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// Parse literal byte count: {1234}
	re := regexp.MustCompile(`\{(\d+)\}`)
	match := re.FindStringSubmatch(line)
	if len(match) != 2 {
		return nil, fmt.Errorf("unexpected fetch response: %s", line)
	}

	byteCount, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, err
	}

	buf := make([]byte, byteCount)
	if _, err := io.ReadFull(c.reader, buf); err != nil {
		return nil, fmt.Errorf("failed reading full message body: %w", err)
	}

	// Read trailing lines until tagged OK
	for {
		l, err := c.reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(l, tag+" OK") || strings.HasPrefix(l, tag+" NO") {
			break
		}
	}

	return buf, nil
}
