package store

import (
	"time"

	"github.com/google/uuid"
)

type AlertRule struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"` // cpu, memory, disk, load, service
	Threshold      float64   `json:"threshold"`
	DurationSec    int       `json:"duration_sec"`
	Severity       string    `json:"severity"` // info, warning, critical
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
}

type AlertIncident struct {
	ID         uuid.UUID  `json:"id"`
	ServerID   uuid.UUID  `json:"server_id"`
	ServerName string     `json:"server_name"`
	RuleID     uuid.UUID  `json:"rule_id"`
	RuleName   string     `json:"rule_name"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"` // firing, resolved
	Value      float64    `json:"value"`
	Threshold  float64    `json:"threshold"`
	Message    string     `json:"message"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type NotificationChannel struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"` // webhook, email, telegram
	Target         string    `json:"target"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
}

type StoredAPIKey struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	Name           string     `json:"name"`
	KeyPrefix      string     `json:"key_prefix"`
	KeyHash        string     `json:"-"`
	Role           string     `json:"role"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}
