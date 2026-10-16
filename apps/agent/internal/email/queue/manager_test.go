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

func TestParsePostqueueOutput_Gmail421RateLimit(t *testing.T) {
	rawOutput := `-Queue ID-  --Size-- ----Arrival Time---- -Sender/Recipient-------
4V3L9j013gz2X     1234 Sun Sep  6 10:00:00  newsletter@onlyflrt.net
(host gmail-smtp-in.l.google.com[142.250.27.26] said: 421-4.7.28 Gmail has detected an unusual rate of mail originating from your SPF domain [onlyflrt.net]. 421-4.7.28 To protect our users from spam, mail originating from your domain has 421-4.7.28 been temporarily rate limited. (in reply to end of DATA command))
                                         customer1@gmail.com

-- 1 Kbytes in 1 Request.
`

	messages, err := ParsePostqueueOutput(rawOutput)
	if err != nil {
		t.Fatalf("ParsePostqueueOutput failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	m := messages[0]
	if !m.IsRateLimited {
		t.Errorf("expected IsRateLimited=true for Gmail 421 4.7.28")
	}
	if m.RateLimitCode != "421 4.7.28" {
		t.Errorf("expected RateLimitCode '421 4.7.28', got '%s'", m.RateLimitCode)
	}
	if m.RateLimitDomain != "onlyflrt.net" {
		t.Errorf("expected RateLimitDomain 'onlyflrt.net', got '%s'", m.RateLimitDomain)
	}
	if m.RateLimitDestination != "gmail.com" {
		t.Errorf("expected RateLimitDestination 'gmail.com', got '%s'", m.RateLimitDestination)
	}
}

func TestSafeFlushQueue_BlockedWhenInCooldown(t *testing.T) {
	// After parsing the 421 queue message above, onlyflrt.net is automatically placed into cooldown!
	err := SafeFlushQueue("onlyflrt.net", "gmail.com")
	if err == nil {
		t.Errorf("SafeFlushQueue must be blocked when domain is in cooldown")
	}
}

