package migration

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
)

func handleMockIMAPConn(conn net.Conn) {
	defer conn.Close()

	conn.Write([]byte("* OK [CAPABILITY IMAP4rev1] Mock IMAP Server Ready\r\n"))
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		tag := parts[0]
		cmd := strings.ToUpper(parts[1])

		switch cmd {
		case "LOGIN":
			conn.Write([]byte(fmt.Sprintf("%s OK [CAPABILITY IMAP4rev1] Logged in\r\n", tag)))
		case "LIST":
			conn.Write([]byte("* LIST (\\HasNoChildren) \"/\" \"INBOX\"\r\n"))
			conn.Write([]byte("* LIST (\\HasNoChildren) \"/\" \"Sent\"\r\n"))
			conn.Write([]byte(fmt.Sprintf("%s OK LIST completed\r\n", tag)))
		case "SELECT":
			conn.Write([]byte("* 2 EXISTS\r\n"))
			conn.Write([]byte("* 0 RECENT\r\n"))
			conn.Write([]byte(fmt.Sprintf("%s OK [READ-WRITE] SELECT completed\r\n", tag)))
		case "UID":
			if len(parts) >= 3 && strings.ToUpper(parts[2]) == "SEARCH" {
				conn.Write([]byte("* SEARCH 101 102\r\n"))
				conn.Write([]byte(fmt.Sprintf("%s OK UID SEARCH completed\r\n", tag)))
			} else if len(parts) >= 3 && strings.ToUpper(parts[2]) == "FETCH" {
				body := "From: alice@test.com\r\nTo: bob@test.com\r\nMessage-ID: <msg-101@test.com>\r\nSubject: Hello\r\n\r\nTest message body"
				conn.Write([]byte(fmt.Sprintf("* 1 FETCH (UID %s BODY[] {%d}\r\n%s)\r\n", parts[3], len(body), body)))
				conn.Write([]byte(fmt.Sprintf("%s OK UID FETCH completed\r\n", tag)))
			}
		case "LOGOUT":
			conn.Write([]byte("* BYE Logging out\r\n"))
			conn.Write([]byte(fmt.Sprintf("%s OK LOGOUT completed\r\n", tag)))
			return
		}
	}
}

func TestIMAP_MigrationEngine(t *testing.T) {
	// 1. Mock IMAP Server with accept loop
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleMockIMAPConn(conn)
		}
	}()

	// 2. Test TestSourceConnection
	opts := SourceConnectionOptions{
		Host:     "127.0.0.1",
		Port:     port,
		Username: "bob@test.com",
		Password: "secretpassword",
		UseSSL:   false,
	}

	testRes := TestSourceConnection(context.Background(), opts)
	if !testRes.Success {
		t.Fatalf("expected test connection success, got error: %s", testRes.ErrorMessage)
	}
	if testRes.FolderCount != 2 {
		t.Errorf("expected 2 folders, got %d", testRes.FolderCount)
	}

	// 3. Test RunMigration
	tmpDir, err := os.MkdirTemp("", "maildir_test_*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	task := MigrationTask{
		ID:                     "mig-12345",
		OrganizationID:         "org-1",
		DestinationEmail:       "dest@example.com",
		DestinationMaildirBase: tmpDir,
		Source:                 opts,
	}

	var lastProgress MigrationProgress
	err = RunMigration(context.Background(), task, func(p MigrationProgress) {
		lastProgress = p
	})

	if err != nil {
		t.Fatalf("RunMigration failed: %v", err)
	}
	if lastProgress.Status != "completed" {
		t.Errorf("expected completed status, got %s", lastProgress.Status)
	}
	if lastProgress.SyncedMessages == 0 {
		t.Errorf("expected synced messages > 0")
	}
}
