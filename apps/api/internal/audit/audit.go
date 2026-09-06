package audit

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"hostvra/api/internal/auth"
	"hostvra/api/internal/store"
)

type Logger struct {
	store  store.Store
	logger *slog.Logger
}

func NewLogger(s store.Store, logger *slog.Logger) *Logger {
	return &Logger{
		store:  s,
		logger: logger,
	}
}

func (l *Logger) Log(ctx context.Context, r *http.Request, action, resourceType, resourceID, status, errorMessage string, metadata map[string]interface{}) {
	var orgID *uuid.UUID
	var userID *uuid.UUID

	if claims, ok := auth.GetClaims(ctx); ok {
		u := claims.UserID
		userID = &u
		o := claims.OrganizationID
		orgID = &o
	}

	ip := ""
	userAgent := ""
	if r != nil {
		ip = r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		userAgent = r.UserAgent()
	}

	entry := &store.AuditLog{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      ip,
		UserAgent:      userAgent,
		Status:         status,
		ErrorMessage:   errorMessage,
		Metadata:       metadata,
	}

	if err := l.store.CreateAuditLog(ctx, entry); err != nil {
		l.logger.Error("Failed to record audit log entry", "error", err, "action", action)
	} else {
		l.logger.Info("Audit log recorded", "action", action, "resource_type", resourceType, "status", status)
	}
}
