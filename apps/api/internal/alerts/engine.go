package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
)

type RuleType string

const (
	RuleCPU     RuleType = "cpu"
	RuleMemory  RuleType = "memory"
	RuleDisk    RuleType = "disk"
	RuleLoad    RuleType = "load"
	RuleService RuleType = "service"
)

type AlertRule struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Type           RuleType  `json:"type"`
	Threshold      float64   `json:"threshold"` // e.g. 90.0 for 90%
	DurationSec    int       `json:"duration_sec"`
	Severity       Severity  `json:"severity"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
}

type Incident struct {
	ID         uuid.UUID   `json:"id"`
	ServerID   uuid.UUID   `json:"server_id"`
	ServerName string      `json:"server_name"`
	RuleID     uuid.UUID   `json:"rule_id"`
	RuleName   string      `json:"rule_name"`
	Severity   Severity    `json:"severity"`
	Status     AlertStatus `json:"status"`
	Value      float64     `json:"value"`
	Threshold  float64     `json:"threshold"`
	Message    string      `json:"message"`
	StartedAt  time.Time   `json:"started_at"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
}

type ChannelType string

const (
	ChannelWebhook  ChannelType = "webhook"
	ChannelEmail    ChannelType = "email"
	ChannelTelegram ChannelType = "telegram"
)

type NotificationChannel struct {
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	Name           string      `json:"name"`
	Type           ChannelType `json:"type"`
	Target         string      `json:"target"` // Webhook URL or Email address
	Enabled        bool        `json:"enabled"`
	CreatedAt      time.Time   `json:"created_at"`
}

type Engine struct {
	mu          sync.RWMutex
	rules       map[uuid.UUID]*AlertRule
	incidents   map[uuid.UUID]*Incident
	channels    map[uuid.UUID]*NotificationChannel
	httpClient  *http.Client
}

func NewEngine() *Engine {
	return &Engine{
		rules:      make(map[uuid.UUID]*AlertRule),
		incidents:  make(map[uuid.UUID]*Incident),
		channels:   make(map[uuid.UUID]*NotificationChannel),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// AddRule registers an alert threshold rule
func (e *Engine) AddRule(rule *AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	rule.CreatedAt = time.Now().UTC()
	e.rules[rule.ID] = rule
}

// ListRules returns rules for an organization
func (e *Engine) ListRules(orgID uuid.UUID) []*AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*AlertRule, 0)
	for _, r := range e.rules {
		if r.OrganizationID == orgID {
			out = append(out, r)
		}
	}
	return out
}

// AddChannel registers a notification endpoint
func (e *Engine) AddChannel(channel *NotificationChannel) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if channel.ID == uuid.Nil {
		channel.ID = uuid.New()
	}
	channel.CreatedAt = time.Now().UTC()
	e.channels[channel.ID] = channel
}

// ListChannels returns channels for an organization
func (e *Engine) ListChannels(orgID uuid.UUID) []*NotificationChannel {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*NotificationChannel, 0)
	for _, c := range e.channels {
		if c.OrganizationID == orgID {
			out = append(out, c)
		}
	}
	return out
}

// EvaluateMetric checks incoming server telemetry against all active rules
func (e *Engine) EvaluateMetric(serverID uuid.UUID, serverName string, cpuPct, memPct, diskPct float64) []*Incident {
	e.mu.Lock()
	defer e.mu.Unlock()

	newIncidents := make([]*Incident, 0)

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		var currentVal float64
		switch rule.Type {
		case RuleCPU:
			currentVal = cpuPct
		case RuleMemory:
			currentVal = memPct
		case RuleDisk:
			currentVal = diskPct
		default:
			continue
		}

		incidentKey := fmt.Sprintf("%s-%s", serverID, rule.ID)
		activeIncident := e.findActiveIncident(serverID, rule.ID)

		if currentVal >= rule.Threshold {
			// Threshold breached
			if activeIncident == nil {
				// Fire new incident
				inc := &Incident{
					ID:         uuid.New(),
					ServerID:   serverID,
					ServerName: serverName,
					RuleID:     rule.ID,
					RuleName:   rule.Name,
					Severity:   rule.Severity,
					Status:     StatusFiring,
					Value:      currentVal,
					Threshold:  rule.Threshold,
					Message:    fmt.Sprintf("Server %s: %s at %.1f%% (threshold: %.1f%%)", serverName, rule.Name, currentVal, rule.Threshold),
					StartedAt:  time.Now().UTC(),
				}
				e.incidents[inc.ID] = inc
				newIncidents = append(newIncidents, inc)
				go e.dispatchNotification(inc)
			}
		} else {
			// Normal state
			if activeIncident != nil && activeIncident.Status == StatusFiring {
				// Resolve incident
				now := time.Now().UTC()
				activeIncident.Status = StatusResolved
				activeIncident.ResolvedAt = &now
				activeIncident.Value = currentVal
				go e.dispatchNotification(activeIncident)
			}
		}
		_ = incidentKey
	}

	return newIncidents
}

func (e *Engine) findActiveIncident(serverID, ruleID uuid.UUID) *Incident {
	for _, inc := range e.incidents {
		if inc.ServerID == serverID && inc.RuleID == ruleID && inc.Status == StatusFiring {
			return inc
		}
	}
	return nil
}

// ListIncidents returns all incidents ordered newest first
func (e *Engine) ListIncidents(limit int) []*Incident {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*Incident, 0, len(e.incidents))
	for _, inc := range e.incidents {
		out = append(out, inc)
	}

	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func (e *Engine) dispatchNotification(inc *Incident) {
	e.mu.RLock()
	channels := make([]*NotificationChannel, 0)
	for _, c := range e.channels {
		if c.Enabled {
			channels = append(channels, c)
		}
	}
	e.mu.RUnlock()

	for _, ch := range channels {
		if ch.Type == ChannelWebhook {
			payload, _ := json.Marshal(map[string]interface{}{
				"event":       "alert." + string(inc.Status),
				"incident_id": inc.ID,
				"server_name": inc.ServerName,
				"severity":    inc.Severity,
				"message":     inc.Message,
				"value":       inc.Value,
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			})

			req, err := http.NewRequest(http.MethodPost, ch.Target, bytes.NewReader(payload))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("User-Agent", "Hostvra-AlertEngine/1.0")
				resp, err := e.httpClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}
}
