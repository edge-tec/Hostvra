package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
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
	mu         sync.RWMutex
	rules      map[uuid.UUID]*AlertRule
	incidents  map[uuid.UUID]*Incident
	channels   map[uuid.UUID]*NotificationChannel
	httpClient *http.Client
	store      store.Store
}

func NewEngine() *Engine {
	return &Engine{
		rules:      make(map[uuid.UUID]*AlertRule),
		incidents:  make(map[uuid.UUID]*Incident),
		channels:   make(map[uuid.UUID]*NotificationChannel),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (e *Engine) SetStore(s store.Store) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store = s
}

// AddRule registers an alert threshold rule and persists to store
func (e *Engine) AddRule(rule *AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now().UTC()
	}
	e.rules[rule.ID] = rule

	if e.store != nil {
		_ = e.store.CreateAlertRule(context.Background(), &store.AlertRule{
			ID:             rule.ID,
			OrganizationID: rule.OrganizationID,
			Name:           rule.Name,
			Type:           string(rule.Type),
			Threshold:      rule.Threshold,
			DurationSec:    rule.DurationSec,
			Severity:       string(rule.Severity),
			Enabled:        rule.Enabled,
			CreatedAt:      rule.CreatedAt,
		})
	}
}

// ListRules returns rules for an organization, syncing from store if available
func (e *Engine) ListRules(orgID uuid.UUID) []*AlertRule {
	if e.store != nil {
		storedRules, err := e.store.ListAlertRules(context.Background(), orgID)
		if err == nil && len(storedRules) > 0 {
			e.mu.Lock()
			out := make([]*AlertRule, 0, len(storedRules))
			for _, sr := range storedRules {
				r := &AlertRule{
					ID:             sr.ID,
					OrganizationID: sr.OrganizationID,
					Name:           sr.Name,
					Type:           RuleType(sr.Type),
					Threshold:      sr.Threshold,
					DurationSec:    sr.DurationSec,
					Severity:       Severity(sr.Severity),
					Enabled:        sr.Enabled,
					CreatedAt:      sr.CreatedAt,
				}
				e.rules[r.ID] = r
				out = append(out, r)
			}
			e.mu.Unlock()
			return out
		}
	}

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

// AddChannel registers a notification endpoint and persists to store
func (e *Engine) AddChannel(channel *NotificationChannel) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if channel.ID == uuid.Nil {
		channel.ID = uuid.New()
	}
	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = time.Now().UTC()
	}
	e.channels[channel.ID] = channel

	if e.store != nil {
		_ = e.store.CreateNotificationChannel(context.Background(), &store.NotificationChannel{
			ID:             channel.ID,
			OrganizationID: channel.OrganizationID,
			Name:           channel.Name,
			Type:           string(channel.Type),
			Target:         channel.Target,
			Enabled:        channel.Enabled,
			CreatedAt:      channel.CreatedAt,
		})
	}
}

// ListChannels returns channels for an organization
func (e *Engine) ListChannels(orgID uuid.UUID) []*NotificationChannel {
	if e.store != nil {
		storedChannels, err := e.store.ListNotificationChannels(context.Background(), orgID)
		if err == nil && len(storedChannels) > 0 {
			e.mu.Lock()
			out := make([]*NotificationChannel, 0, len(storedChannels))
			for _, sc := range storedChannels {
				c := &NotificationChannel{
					ID:             sc.ID,
					OrganizationID: sc.OrganizationID,
					Name:           sc.Name,
					Type:           ChannelType(sc.Type),
					Target:         sc.Target,
					Enabled:        sc.Enabled,
					CreatedAt:      sc.CreatedAt,
				}
				e.channels[c.ID] = c
				out = append(out, c)
			}
			e.mu.Unlock()
			return out
		}
	}

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

				if e.store != nil {
					_ = e.store.CreateAlertIncident(context.Background(), &store.AlertIncident{
						ID:         inc.ID,
						ServerID:   inc.ServerID,
						ServerName: inc.ServerName,
						RuleID:     inc.RuleID,
						RuleName:   inc.RuleName,
						Severity:   string(inc.Severity),
						Status:     string(inc.Status),
						Value:      inc.Value,
						Threshold:  inc.Threshold,
						Message:    inc.Message,
						StartedAt:  inc.StartedAt,
					})
				}

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

				if e.store != nil {
					_ = e.store.UpdateAlertIncident(context.Background(), &store.AlertIncident{
						ID:         activeIncident.ID,
						Status:     string(activeIncident.Status),
						Value:      activeIncident.Value,
						ResolvedAt: activeIncident.ResolvedAt,
					})
				}

				go e.dispatchNotification(activeIncident)
			}
		}
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
	if e.store != nil {
		storedIncidents, err := e.store.ListAlertIncidents(context.Background(), limit)
		if err == nil && len(storedIncidents) > 0 {
			out := make([]*Incident, 0, len(storedIncidents))
			for _, si := range storedIncidents {
				out = append(out, &Incident{
					ID:         si.ID,
					ServerID:   si.ServerID,
					ServerName: si.ServerName,
					RuleID:     si.RuleID,
					RuleName:   si.RuleName,
					Severity:   Severity(si.Severity),
					Status:     AlertStatus(si.Status),
					Value:      si.Value,
					Threshold:  si.Threshold,
					Message:    si.Message,
					StartedAt:  si.StartedAt,
					ResolvedAt: si.ResolvedAt,
				})
			}
			return out
		}
	}

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
