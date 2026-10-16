package queue

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"hostvra/agent/internal/email/ratecontrol"
)

type QueueMessage struct {
	ID                   string    `json:"id"`
	QueueID              string    `json:"queue_id"`
	SizeBytes            int64     `json:"size_bytes"`
	ArrivalTime          time.Time `json:"arrival_time"`
	ArrivalStr           string    `json:"arrival_str,omitempty"`
	Sender               string    `json:"sender"`
	Recipients           []string  `json:"recipients"`
	Status               string    `json:"status"` // active, deferred, hold
	Reason               string    `json:"reason,omitempty"`
	IsRateLimited        bool      `json:"is_rate_limited"`
	RateLimitCode        string    `json:"rate_limit_code,omitempty"`
	RateLimitDomain      string    `json:"rate_limit_domain,omitempty"`
	RateLimitDestination string    `json:"rate_limit_destination,omitempty"`
}

var (
	// Matches: "12345ABCDE*     1024 Sun Sep  6 10:00:00  sender@example.com"
	headerRegex    = regexp.MustCompile(`^([A-Za-z0-9]+)([\*!]?)\s+(\d+)\s+([A-Za-z]{3}\s+[A-Za-z]{3}\s+\d+\s+\d+:\d+:\d+)\s+(.+)$`)
	spfDomainRegex = regexp.MustCompile(`(?i)SPF domain\s+\[([a-zA-Z0-9\.\-]+)\]`)
)

func enrichRateLimitInfo(msg *QueueMessage) {
	if msg == nil || msg.Reason == "" {
		return
	}
	is421, _, _, _ := ratecontrol.ClassifySMTPResponse("421", msg.Reason)
	if is421 {
		msg.IsRateLimited = true
		msg.RateLimitCode = "421 4.7.28"
		msg.RateLimitDestination = "gmail.com"

		// Extract SPF domain from reason if formatted as: SPF domain [domain]
		spfMatch := spfDomainRegex.FindStringSubmatch(msg.Reason)
		if len(spfMatch) > 1 {
			msg.RateLimitDomain = strings.ToLower(spfMatch[1])
		} else if strings.Contains(msg.Sender, "@") {
			parts := strings.Split(msg.Sender, "@")
			msg.RateLimitDomain = strings.ToLower(parts[len(parts)-1])
		}

		if msg.RateLimitDomain != "" {
			ratecontrol.GetGlobalController().RecordDeliveryEvent(msg.RateLimitDomain, "gmail.com", "421", msg.Reason)
		}
	}
}

// ParsePostqueueOutput parses output from `postqueue -p`
func ParsePostqueueOutput(output string) ([]QueueMessage, error) {
	var messages []QueueMessage
	scanner := bufio.NewScanner(strings.NewReader(output))

	var current *QueueMessage

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "-Queue ID-") || strings.HasPrefix(trimmed, "-- ") {
			continue
		}

		if matches := headerRegex.FindStringSubmatch(trimmed); matches != nil {
			if current != nil {
				enrichRateLimitInfo(current)
				messages = append(messages, *current)
			}

			size, _ := strconv.ParseInt(matches[3], 10, 64)
			status := "deferred"
			if matches[2] == "*" {
				status = "active"
			} else if matches[2] == "!" {
				status = "hold"
			}

			arrivalStr := matches[4]
			var arrivalTime time.Time
			for _, layout := range []string{"Mon Jan _2 15:04:05", "Mon Jan 2 15:04:05", "Jan _2 15:04:05", "Jan 2 15:04:05"} {
				if t, err := time.ParseInLocation(layout, arrivalStr, time.Local); err == nil {
					arrivalTime = t.AddDate(time.Now().Year(), 0, 0)
					break
				}
			}
			if arrivalTime.IsZero() {
				arrivalTime = time.Now()
			}

			current = &QueueMessage{
				ID:          matches[1],
				QueueID:     matches[1],
				SizeBytes:   size,
				ArrivalTime: arrivalTime,
				ArrivalStr:  arrivalStr,
				Sender:      matches[5],
				Status:      status,
				Recipients:  make([]string, 0),
			}
			continue
		}

		if current != nil {
			if strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
				current.Reason = strings.Trim(trimmed, "()")
			} else if strings.Contains(trimmed, "@") {
				current.Recipients = append(current.Recipients, trimmed)
			}
		}
	}

	if current != nil {
		enrichRateLimitInfo(current)
		messages = append(messages, *current)
	}

	return messages, nil
}

// ListQueue executes postqueue -p and returns parsed messages
func ListQueue() ([]QueueMessage, error) {
	if _, err := exec.LookPath("postqueue"); err != nil {
		return []QueueMessage{}, nil // Return empty in non-postfix environment
	}

	cmd := exec.Command("postqueue", "-p")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Mail queue is empty") {
			return []QueueMessage{}, nil
		}
		return nil, fmt.Errorf("postqueue failed: %s (%w)", string(out), err)
	}

	return ParsePostqueueOutput(string(out))
}

// FlushQueue executes postqueue -f to retry deferred messages immediately
func FlushQueue() error {
	if _, err := exec.LookPath("postqueue"); err != nil {
		return nil
	}
	cmd := exec.Command("postqueue", "-f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postqueue -f failed: %s (%w)", string(out), err)
	}
	return nil
}

// SafeFlushQueue checks destination cooldown before triggering a queue flush
func SafeFlushQueue(domain, destination string) error {
	if domain != "" && destination != "" {
		allowed, waitSec, reason := ratecontrol.GetGlobalController().CanDeliver(domain, destination)
		if !allowed {
			return fmt.Errorf("queue flush blocked: %s (cooldown active: %ds remaining)", reason, waitSec)
		}
	}
	return FlushQueue()
}

// executePostsuper runs postsuper with privilege fallback
func executePostsuper(args ...string) error {
	postsuperPath, err := exec.LookPath("postsuper")
	if err != nil {
		return nil
	}

	var cmd *exec.Cmd
	if os.Geteuid() == 0 {
		cmd = exec.Command(postsuperPath, args...)
	} else if _, sudoErr := exec.LookPath("sudo"); sudoErr == nil {
		cmd = exec.Command("sudo", append([]string{"-n", postsuperPath}, args...)...)
	} else {
		cmd = exec.Command(postsuperPath, args...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "reserved for the superuser") || strings.Contains(outStr, "password is required") {
			return nil
		}
		return fmt.Errorf("postsuper %s failed: %s (%w)", strings.Join(args, " "), outStr, err)
	}
	return nil
}

// DeleteQueueMessage deletes a message from the queue via postsuper -d
func DeleteQueueMessage(queueID string) error {
	// Security check on queueID: alphanumeric only
	validID := regexp.MustCompile(`^[A-Za-z0-9]+$`)
	if !validID.MatchString(queueID) {
		return fmt.Errorf("invalid queue ID format")
	}
	return executePostsuper("-d", queueID)
}

// ClearQueue deletes all messages from all Postfix queues via postsuper -d ALL
func ClearQueue() error {
	return executePostsuper("-d", "ALL")
}
