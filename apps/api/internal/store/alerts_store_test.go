package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAlertsAndAPIKeysStore_MemoryLifecycle(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryStore()
	orgID := uuid.New()

	// 1. Alert Rule Lifecycle
	rule := &AlertRule{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "High CPU Alert",
		Type:           "cpu",
		Threshold:      90.0,
		DurationSec:    60,
		Severity:       "critical",
		Enabled:        true,
	}
	if err := mem.CreateAlertRule(ctx, rule); err != nil {
		t.Fatalf("failed to create alert rule: %v", err)
	}

	rules, err := mem.ListAlertRules(ctx, orgID)
	if err != nil {
		t.Fatalf("failed to list alert rules: %v", err)
	}
	if len(rules) != 1 || rules[0].Name != "High CPU Alert" {
		t.Fatalf("unexpected rules list: %+v", rules)
	}

	// 2. Alert Incident Lifecycle
	inc := &AlertIncident{
		ID:         uuid.New(),
		ServerID:   uuid.New(),
		ServerName: "production-node-1",
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Severity:   rule.Severity,
		Status:     "firing",
		Value:      94.5,
		Threshold:  90.0,
		Message:    "CPU at 94.5%",
	}
	if err := mem.CreateAlertIncident(ctx, inc); err != nil {
		t.Fatalf("failed to create incident: %v", err)
	}

	incidents, err := mem.ListAlertIncidents(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list incidents: %v", err)
	}
	if len(incidents) != 1 || incidents[0].Status != "firing" {
		t.Fatalf("unexpected incidents: %+v", incidents)
	}

	// Resolve incident
	now := time.Now().UTC()
	inc.Status = "resolved"
	inc.ResolvedAt = &now
	if err := mem.UpdateAlertIncident(ctx, inc); err != nil {
		t.Fatalf("failed to update incident: %v", err)
	}

	// 3. Notification Channel Lifecycle
	ch := &NotificationChannel{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Slack Webhook",
		Type:           "webhook",
		Target:         "https://hooks.slack.com/services/test",
		Enabled:        true,
	}
	if err := mem.CreateNotificationChannel(ctx, ch); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	channels, err := mem.ListNotificationChannels(ctx, orgID)
	if err != nil {
		t.Fatalf("failed to list channels: %v", err)
	}
	if len(channels) != 1 || channels[0].Target != ch.Target {
		t.Fatalf("unexpected channels: %+v", channels)
	}

	// 4. API Key Lifecycle
	key := &StoredAPIKey{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "CI/CD Token",
		KeyPrefix:      "hv_live_1234",
		KeyHash:        "hashed_secret",
		Role:           "manager",
	}
	if err := mem.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	keys, err := mem.ListAPIKeys(ctx, orgID)
	if err != nil {
		t.Fatalf("failed to list api keys: %v", err)
	}
	if len(keys) != 1 || keys[0].KeyPrefix != "hv_live_1234" {
		t.Fatalf("unexpected api keys: %+v", keys)
	}

	// Revoke Key
	if err := mem.RevokeAPIKey(ctx, key.ID); err != nil {
		t.Fatalf("failed to revoke api key: %v", err)
	}

	keysAfterRevoke, err := mem.ListAPIKeys(ctx, orgID)
	if err != nil {
		t.Fatalf("failed to list api keys: %v", err)
	}
	if len(keysAfterRevoke) != 0 {
		t.Fatalf("expected 0 active keys after revocation, got %d", len(keysAfterRevoke))
	}

	// Delete rule
	if err := mem.DeleteAlertRule(ctx, rule.ID); err != nil {
		t.Fatalf("failed to delete rule: %v", err)
	}
	rulesAfterDelete, _ := mem.ListAlertRules(ctx, orgID)
	if len(rulesAfterDelete) != 0 {
		t.Fatalf("expected 0 rules after deletion, got %d", len(rulesAfterDelete))
	}
}
