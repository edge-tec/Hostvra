package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

// getMaildirBasePaths returns the directories where virtual mailboxes are located.
func getMaildirBasePaths() []string {
	paths := []string{}
	if env := os.Getenv("STORAGE_LOCATION"); env != "" {
		paths = append(paths, env)
	}
	if env := os.Getenv("MAILDIR_BASE"); env != "" {
		paths = append(paths, env)
	}
	paths = append(paths, "/var/mail/vhosts", "/var/vmail", "/var/mail")
	return paths
}

// findMailboxDir locates the on-disk directory for a mailbox (e.g. /var/mail/vhosts/hostvra.com/info).
func findMailboxDir(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	domain := strings.TrimSpace(parts[1])
	localPart := strings.TrimSpace(parts[0])

	for _, base := range getMaildirBasePaths() {
		candidate := filepath.Join(base, domain, localPart)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
		candidate2 := filepath.Join(base, email)
		if fi, err := os.Stat(candidate2); err == nil && fi.IsDir() {
			return candidate2
		}
	}
	return filepath.Join("/var/mail/vhosts", domain, localPart)
}

// getSubdirCandidates returns candidate Maildir subdirectories to scan for a folder.
func getSubdirCandidates(mbDir, folder string) []string {
	switch folder {
	case "inbox":
		return []string{
			filepath.Join(mbDir, "new"),
			filepath.Join(mbDir, "cur"),
		}
	case "sent":
		return []string{
			filepath.Join(mbDir, ".Sent", "new"),
			filepath.Join(mbDir, ".Sent", "cur"),
			filepath.Join(mbDir, "Sent", "new"),
			filepath.Join(mbDir, "Sent", "cur"),
		}
	case "drafts":
		return []string{
			filepath.Join(mbDir, ".Drafts", "new"),
			filepath.Join(mbDir, ".Drafts", "cur"),
			filepath.Join(mbDir, "Drafts", "new"),
			filepath.Join(mbDir, "Drafts", "cur"),
		}
	case "trash":
		return []string{
			filepath.Join(mbDir, ".Trash", "new"),
			filepath.Join(mbDir, ".Trash", "cur"),
			filepath.Join(mbDir, "Trash", "new"),
			filepath.Join(mbDir, "Trash", "cur"),
		}
	case "spam", "junk":
		return []string{
			filepath.Join(mbDir, ".Junk", "new"),
			filepath.Join(mbDir, ".Junk", "cur"),
			filepath.Join(mbDir, ".Spam", "new"),
			filepath.Join(mbDir, ".Spam", "cur"),
			filepath.Join(mbDir, "Junk", "new"),
			filepath.Join(mbDir, "Junk", "cur"),
		}
	case "archive":
		return []string{
			filepath.Join(mbDir, ".Archive", "new"),
			filepath.Join(mbDir, ".Archive", "cur"),
			filepath.Join(mbDir, "Archive", "new"),
			filepath.Join(mbDir, "Archive", "cur"),
		}
	case "all", "starred":
		return []string{
			filepath.Join(mbDir, "new"),
			filepath.Join(mbDir, "cur"),
			filepath.Join(mbDir, ".Sent", "new"),
			filepath.Join(mbDir, ".Sent", "cur"),
			filepath.Join(mbDir, ".Drafts", "new"),
			filepath.Join(mbDir, ".Drafts", "cur"),
			filepath.Join(mbDir, ".Trash", "new"),
			filepath.Join(mbDir, ".Trash", "cur"),
			filepath.Join(mbDir, ".Junk", "new"),
			filepath.Join(mbDir, ".Junk", "cur"),
			filepath.Join(mbDir, ".Archive", "new"),
			filepath.Join(mbDir, ".Archive", "cur"),
		}
	default:
		return []string{
			filepath.Join(mbDir, "new"),
			filepath.Join(mbDir, "cur"),
		}
	}
}

// folderFromDir detects standard folder type from directory path.
func folderFromDir(dirPath string) string {
	lower := strings.ToLower(dirPath)
	if strings.Contains(lower, "sent") {
		return "sent"
	}
	if strings.Contains(lower, "draft") {
		return "drafts"
	}
	if strings.Contains(lower, "trash") {
		return "trash"
	}
	if strings.Contains(lower, "junk") || strings.Contains(lower, "spam") {
		return "spam"
	}
	if strings.Contains(lower, "archive") {
		return "archive"
	}
	return "inbox"
}

// parseMailBodyAndAttachments extracts plain text, HTML and attachment descriptors.
func parseMailBodyAndAttachments(contentTypeHeader string, body io.Reader) (string, string, []store.WebmailAttachment) {
	if contentTypeHeader == "" {
		contentTypeHeader = "text/plain; charset=UTF-8"
	}

	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		b, _ := io.ReadAll(body)
		return string(b), "", nil
	}

	var bodyText, bodyHTML string
	var attachments []store.WebmailAttachment

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary, ok := params["boundary"]
		if !ok || boundary == "" {
			b, _ := io.ReadAll(body)
			return string(b), "", nil
		}
		mr := multipart.NewReader(body, boundary)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}

			partType := p.Header.Get("Content-Type")
			partMediaType, partParams, _ := mime.ParseMediaType(partType)
			partDisp := p.Header.Get("Content-Disposition")
			disposition, dispParams, _ := mime.ParseMediaType(partDisp)
			encoding := strings.ToLower(p.Header.Get("Content-Transfer-Encoding"))

			var partReader io.Reader = p
			if encoding == "quoted-printable" {
				partReader = quotedprintable.NewReader(p)
			} else if encoding == "base64" {
				partReader = base64.NewDecoder(base64.StdEncoding, p)
			}

			data, _ := io.ReadAll(partReader)

			filename := dispParams["filename"]
			if filename == "" {
				filename = partParams["name"]
			}

			if disposition == "attachment" || filename != "" {
				attachments = append(attachments, store.WebmailAttachment{
					ID:          uuid.New(),
					Filename:    filename,
					ContentType: partMediaType,
					SizeBytes:   int64(len(data)),
				})
			} else if strings.HasPrefix(partMediaType, "text/html") && bodyHTML == "" {
				bodyHTML = string(data)
			} else if strings.HasPrefix(partMediaType, "text/plain") && bodyText == "" {
				bodyText = string(data)
			} else if strings.HasPrefix(partMediaType, "multipart/") {
				subText, subHTML, subAtt := parseMailBodyAndAttachments(partType, bytes.NewReader(data))
				if bodyText == "" {
					bodyText = subText
				}
				if bodyHTML == "" {
					bodyHTML = subHTML
				}
				attachments = append(attachments, subAtt...)
			}
		}
	} else if strings.HasPrefix(mediaType, "text/html") {
		data, _ := io.ReadAll(body)
		bodyHTML = string(data)
		bodyText = stripHTMLTags(bodyHTML)
	} else {
		data, _ := io.ReadAll(body)
		bodyText = string(data)
	}

	return bodyText, bodyHTML, attachments
}

// parseRawMail parses raw RFC 5322 byte slice into WebmailMessage.
func parseRawMail(data []byte, defaultFolder, fileName string, isNewDir bool, mb *store.EmailMailbox) (*store.WebmailMessage, error) {
	reader := bytes.NewReader(data)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	header := msg.Header
	msgID := strings.TrimSpace(header.Get("Message-ID"))
	if msgID == "" {
		hash := sha256.Sum256(data)
		msgID = fmt.Sprintf("<%s@hostvra.local>", hex.EncodeToString(hash[:8]))
	}

	dec := new(mime.WordDecoder)
	subject, _ := dec.DecodeHeader(header.Get("Subject"))
	if subject == "" {
		subject = header.Get("Subject")
	}
	if subject == "" {
		subject = "(No Subject)"
	}

	fromRaw := header.Get("From")
	fromName := ""
	fromEmail := fromRaw
	if addr, err := mail.ParseAddress(fromRaw); err == nil && addr != nil {
		fromName = addr.Name
		fromEmail = addr.Address
	}

	toRaw := header.Get("To")
	toName := toRaw
	toEmail := toRaw
	if addrs, err := mail.ParseAddressList(toRaw); err == nil && len(addrs) > 0 {
		toName = addrs[0].Name
		toEmail = addrs[0].Address
	}

	ccRaw := header.Get("Cc")
	bccRaw := header.Get("Bcc")

	dateParsed := time.Now().UTC()
	if dateStr := header.Get("Date"); dateStr != "" {
		if d, err := mail.ParseDate(dateStr); err == nil {
			dateParsed = d.UTC()
		}
	}

	bodyText, bodyHTML, attachments := parseMailBodyAndAttachments(header.Get("Content-Type"), msg.Body)

	isUnread := isNewDir
	isStarred := false
	if strings.Contains(fileName, ":2,") || strings.Contains(fileName, ",") {
		infoFlags := fileName[strings.LastIndex(fileName, ",")+1:]
		if strings.Contains(infoFlags, "S") {
			isUnread = false
		}
		if strings.Contains(infoFlags, "F") {
			isStarred = true
		}
	}

	wmMsg := &store.WebmailMessage{
		ID:            uuid.New(),
		MailboxID:     mb.ID,
		AccountEmail:  mb.Email,
		Folder:        defaultFolder,
		MessageID:     msgID,
		FromName:      fromName,
		FromEmail:     fromEmail,
		ToName:        toName,
		ToEmail:       toEmail,
		Cc:            ccRaw,
		Bcc:           bccRaw,
		Subject:       subject,
		BodyText:      bodyText,
		BodyHTML:      bodyHTML,
		IsUnread:      isUnread,
		IsStarred:     isStarred,
		IsImportant:   header.Get("X-Priority") == "1" || strings.Contains(strings.ToLower(header.Get("Priority")), "high"),
		HasAttachment: len(attachments) > 0,
		Priority:      "normal",
		SizeBytes:     int64(len(data)),
		CreatedAt:     dateParsed,
		UpdatedAt:     dateParsed,
		Attachments:   attachments,
	}

	if wmMsg.IsImportant {
		wmMsg.Priority = "high"
	}

	if wmMsg.Snippet == "" && wmMsg.BodyText != "" {
		if len(wmMsg.BodyText) > 120 {
			wmMsg.Snippet = wmMsg.BodyText[:120] + "..."
		} else {
			wmMsg.Snippet = wmMsg.BodyText
		}
	}

	return wmMsg, nil
}

// syncMaildirFolder scans Maildir files for the mailbox and imports any new or changed emails into store.
func (h *WebmailHandler) syncMaildirFolder(ctx context.Context, mb *store.EmailMailbox, folder string) {
	mbDir := findMailboxDir(mb.Email)
	if mbDir == "" {
		return
	}

	subdirs := getSubdirCandidates(mbDir, folder)
	for _, dir := range subdirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		isNewDir := strings.HasSuffix(dir, "new")
		dirFolder := folderFromDir(dir)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			filePath := filepath.Join(dir, entry.Name())
			fi, err := entry.Info()
			if err != nil {
				continue
			}

			data, err := os.ReadFile(filePath)
			if err != nil || len(data) == 0 {
				continue
			}

			parsed, err := parseRawMail(data, dirFolder, entry.Name(), isNewDir, mb)
			if err != nil || parsed == nil {
				continue
			}

			existing, err := h.store.GetWebmailMessageByMessageID(ctx, mb.ID, parsed.MessageID)
			if err == nil && existing != nil {
				// If status changed in Maildir
				if !isNewDir && strings.Contains(entry.Name(), ",S") && existing.IsUnread {
					isFalse := false
					_ = h.store.UpdateWebmailMessageFlags(ctx, existing.ID, &isFalse, nil, nil)
				}
				continue
			}

			if fi.ModTime().Before(parsed.CreatedAt) || parsed.CreatedAt.IsZero() {
				parsed.CreatedAt = fi.ModTime().UTC()
			}

			_ = h.store.CreateWebmailMessage(ctx, parsed)
		}
	}
}

// writeEmailToMaildir writes raw email bytes to appropriate Maildir folder.
func writeEmailToMaildir(email, folder string, rawBytes []byte, isRead bool) {
	mbDir := findMailboxDir(email)
	if mbDir == "" {
		return
	}

	sub := "new"
	if isRead {
		sub = "cur"
	}

	targetDir := ""
	switch folder {
	case "inbox":
		targetDir = filepath.Join(mbDir, sub)
	case "sent":
		targetDir = filepath.Join(mbDir, ".Sent", sub)
	case "drafts":
		targetDir = filepath.Join(mbDir, ".Drafts", sub)
	case "trash":
		targetDir = filepath.Join(mbDir, ".Trash", sub)
	case "spam", "junk":
		targetDir = filepath.Join(mbDir, ".Junk", sub)
	case "archive":
		targetDir = filepath.Join(mbDir, ".Archive", sub)
	default:
		targetDir = filepath.Join(mbDir, sub)
	}

	_ = os.MkdirAll(targetDir, 0700)
	flags := ""
	if isRead {
		flags = ":2,S"
	}
	fName := fmt.Sprintf("%d.M%dP%d.hostvra,S=%d%s", time.Now().Unix(), time.Now().UnixNano()%100000, os.Getpid(), len(rawBytes), flags)
	_ = os.WriteFile(filepath.Join(targetDir, fName), rawBytes, 0600)
}

// moveMaildirFile moves email file on disk when moved in Webmail.
func moveMaildirFile(email, messageID, sourceFolder, targetFolder string) {
	mbDir := findMailboxDir(email)
	if mbDir == "" {
		return
	}

	srcDirs := getSubdirCandidates(mbDir, sourceFolder)
	var foundPath string
	var fileName string

	for _, d := range srcDirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fPath := filepath.Join(d, e.Name())
			if data, err := os.ReadFile(fPath); err == nil {
				if strings.Contains(string(data), messageID) {
					foundPath = fPath
					fileName = e.Name()
					break
				}
			}
		}
		if foundPath != "" {
			break
		}
	}

	if foundPath == "" {
		return
	}

	if targetFolder == "" || targetFolder == "delete" {
		_ = os.Remove(foundPath)
		return
	}

	destDirs := getSubdirCandidates(mbDir, targetFolder)
	if len(destDirs) == 0 {
		return
	}
	destDir := destDirs[0]
	_ = os.MkdirAll(destDir, 0700)
	_ = os.Rename(foundPath, filepath.Join(destDir, fileName))
}
