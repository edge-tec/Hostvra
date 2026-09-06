package queue

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type QueueMessage struct {
	QueueID     string    `json:"queue_id"`
	SizeBytes   int64     `json:"size_bytes"`
	ArrivalTime time.Time `json:"arrival_time"`
	Sender      string    `json:"sender"`
	Recipients  []string  `json:"recipients"`
	Status      string    `json:"status"` // active, deferred, hold
	Reason      string    `json:"reason,omitempty"`
}

var (
	// Matches: "12345ABCDE*     1024 Sun Sep  6 10:00:00  sender@example.com"
	headerRegex = regexp.MustCompile(`^([A-Za-z0-9]+)([\*!]?)\s+(\d+)\s+([A-Za-z]{3}\s+[A-Za-z]{3}\s+\d+\s+\d+:\d+:\d+)\s+(.+)$`)
)

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
				messages = append(messages, *current)
			}

			size, _ := strconv.ParseInt(matches[3], 10, 64)
			status := "deferred"
			if matches[2] == "*" {
				status = "active"
			} else if matches[2] == "!" {
				status = "hold"
			}

			current = &QueueMessage{
				QueueID:    matches[1],
				SizeBytes:  size,
				Sender:     matches[5],
				Status:     status,
				Recipients: make([]string, 0),
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

// DeleteQueueMessage deletes a message from the queue via postsuper -d
func DeleteQueueMessage(queueID string) error {
	// Security check on queueID: alphanumeric only
	validID := regexp.MustCompile(`^[A-Za-z0-9]+$`)
	if !validID.MatchString(queueID) {
		return fmt.Errorf("invalid queue ID format")
	}

	if _, err := exec.LookPath("postsuper"); err != nil {
		return nil
	}

	cmd := exec.Command("postsuper", "-d", queueID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postsuper -d failed: %s (%w)", string(out), err)
	}
	return nil
}
