package queue

import (
	"testing"
)

func TestParsePostqueueOutput(t *testing.T) {
	rawOutput := `-Queue ID-  --Size-- ----Arrival Time---- -Sender/Recipient-------
4V3L9j013gz2X     1234 Sun Sep  6 10:00:00  sender@example.com
(connect to mail.remote.com[198.51.100.1]:25: Connection refused)
                                         recipient1@remote.com

4V3L9k987gz9Y*    4567 Sun Sep  6 10:05:00  admin@hostvra.com
                                         client@gmail.com

-- 5 Kbytes in 2 Requests.
`

	messages, err := ParsePostqueueOutput(rawOutput)
	if err != nil {
		t.Fatalf("ParsePostqueueOutput failed: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	m1 := messages[0]
	if m1.QueueID != "4V3L9j013gz2X" || m1.SizeBytes != 1234 || m1.Sender != "sender@example.com" {
		t.Errorf("m1 mismatch: %+v", m1)
	}
	if len(m1.Recipients) != 1 || m1.Recipients[0] != "recipient1@remote.com" {
		t.Errorf("m1 recipients mismatch: %v", m1.Recipients)
	}
	if m1.Status != "deferred" {
		t.Errorf("expected deferred status, got %s", m1.Status)
	}
	if m1.Reason != "connect to mail.remote.com[198.51.100.1]:25: Connection refused" {
		t.Errorf("m1 reason mismatch: %s", m1.Reason)
	}

	m2 := messages[1]
	if m2.QueueID != "4V3L9k987gz9Y" || m2.Status != "active" {
		t.Errorf("m2 mismatch: %+v", m2)
	}
}

func TestDeleteQueueMessage_Sanitization(t *testing.T) {
	if err := DeleteQueueMessage("malicious;rm -rf /"); err == nil {
		t.Errorf("expected validation failure for malicious queue ID")
	}
}
