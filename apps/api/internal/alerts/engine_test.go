package alerts

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAlertEngineLifecycle(t *testing.T) {
	webhookHit := make(chan string, 5)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookHit <- r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	engine := NewEngine()
	orgID := uuid.New()
	serverID := uuid.New()

	// 1. Add CPU Alert Rule (Threshold: 90%)
	rule := &AlertRule{
		OrganizationID: orgID,
		Name:           "High CPU Usage",
		Type:           RuleCPU,
		Threshold:      90.0,
		Severity:       SeverityCritical,
		Enabled:        true,
	}
	engine.AddRule(rule)

	// 2. Add Webhook Channel
	ch := &NotificationChannel{
		OrganizationID: orgID,
		Name:           "Operations Webhook",
		Type:           ChannelWebhook,
		Target:         ts.URL,
		Enabled:        true,
	}
	engine.AddChannel(ch)

	// 3. Normal metric: 50% CPU -> No incidents
	incidents := engine.EvaluateMetric(serverID, "prod-vps-01", 50.0, 40.0, 30.0)
	if len(incidents) != 0 {
		t.Errorf("expected 0 incidents at 50%% CPU, got %d", len(incidents))
	}

	// 4. Spike metric: 95% CPU -> Should fire 1 incident
	incidents = engine.EvaluateMetric(serverID, "prod-vps-01", 95.0, 40.0, 30.0)
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident at 95%% CPU, got %d", len(incidents))
	}
	if incidents[0].Status != StatusFiring {
		t.Errorf("expected incident to be 'firing', got %s", incidents[0].Status)
	}

	// Verify webhook was called
	select {
	case method := <-webhookHit:
		if method != http.MethodPost {
			t.Errorf("expected POST webhook, got %s", method)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for webhook dispatch")
	}

	// 5. Normal metric returns: 70% CPU -> Should resolve the incident
	engine.EvaluateMetric(serverID, "prod-vps-01", 70.0, 40.0, 30.0)
	allIncidents := engine.ListIncidents(10)
	if len(allIncidents) != 1 {
		t.Fatalf("expected 1 total incident record, got %d", len(allIncidents))
	}
	if allIncidents[0].Status != StatusResolved {
		t.Errorf("expected incident to be 'resolved', got %s", allIncidents[0].Status)
	}
}
