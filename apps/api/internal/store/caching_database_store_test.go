package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryStore_CachingAndDatabaseSubsystem(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStoreWithPath("")

	websiteID := uuid.New()
	orgID := uuid.New()
	dbID := uuid.New()

	// 1. Website Cache Config
	_, err := store.GetWebsiteCacheConfig(ctx, websiteID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for non-existent cache config, got %v", err)
	}

	cfg := &WebsiteCacheConfigRecord{
		WebsiteID:            websiteID,
		OrganizationID:       orgID,
		CacheType:            "redis",
		ConnectionType:       "socket",
		SocketPath:           "/var/run/redis/redis-" + websiteID.String() + ".sock",
		Host:                 "127.0.0.1",
		Port:                 6379,
		Password:             "secret_redis_pass_123",
		MaxMemoryMB:          256,
		EvictionPolicy:       "allkeys-lru",
		WPObjectCacheEnabled: true,
		Status:               "active",
	}

	if err := store.SaveWebsiteCacheConfig(ctx, cfg); err != nil {
		t.Fatalf("failed to save cache config: %v", err)
	}

	savedCfg, err := store.GetWebsiteCacheConfig(ctx, websiteID)
	if err != nil {
		t.Fatalf("failed to get cache config: %v", err)
	}
	if savedCfg.CacheType != "redis" || savedCfg.MaxMemoryMB != 256 || !savedCfg.WPObjectCacheEnabled {
		t.Fatalf("unexpected cache config: %+v", savedCfg)
	}

	// Update cache config
	savedCfg.MaxMemoryMB = 512
	if err := store.SaveWebsiteCacheConfig(ctx, savedCfg); err != nil {
		t.Fatalf("failed to update cache config: %v", err)
	}
	updatedCfg, _ := store.GetWebsiteCacheConfig(ctx, websiteID)
	if updatedCfg.MaxMemoryMB != 512 {
		t.Fatalf("expected 512 MB, got %d", updatedCfg.MaxMemoryMB)
	}

	// 2. Remote Database Access
	remotes, err := store.ListDatabaseRemoteAccess(ctx, dbID)
	if err != nil {
		t.Fatalf("failed to list remote access: %v", err)
	}
	if len(remotes) != 0 {
		t.Fatalf("expected 0 remotes, got %d", len(remotes))
	}

	exp := time.Now().Add(24 * time.Hour)
	grant := &DatabaseRemoteAccessRecord{
		DatabaseID:          dbID,
		OrganizationID:      orgID,
		DBUser:              "app_user",
		AuthorizedIP:        "198.51.100.25",
		GrantedPrivileges:   "SELECT, INSERT, UPDATE, DELETE",
		FirewallRuleApplied: true,
		Status:              "active",
		Notes:               "Staging microservice access",
		ExpiresAt:           &exp,
	}

	if err := store.CreateDatabaseRemoteAccess(ctx, grant); err != nil {
		t.Fatalf("failed to create remote access grant: %v", err)
	}

	remotes, err = store.ListDatabaseRemoteAccess(ctx, dbID)
	if err != nil || len(remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d (err: %v)", len(remotes), err)
	}
	if remotes[0].AuthorizedIP != "198.51.100.25" || !remotes[0].FirewallRuleApplied {
		t.Fatalf("unexpected remote access data: %+v", remotes[0])
	}

	grantID := remotes[0].ID
	gotGrant, err := store.GetDatabaseRemoteAccessByID(ctx, grantID)
	if err != nil {
		t.Fatalf("failed to get grant by ID: %v", err)
	}
	gotGrant.Status = "revoked"
	if err := store.UpdateDatabaseRemoteAccess(ctx, gotGrant); err != nil {
		t.Fatalf("failed to update grant: %v", err)
	}

	updatedGrant, _ := store.GetDatabaseRemoteAccessByID(ctx, grantID)
	if updatedGrant.Status != "revoked" {
		t.Fatalf("expected revoked status, got %s", updatedGrant.Status)
	}

	// 3. Database Maintenance Logs
	mLog := &DatabaseMaintenanceLogRecord{
		DatabaseID:      dbID,
		OrganizationID:  orgID,
		Operation:       "optimize",
		Status:          "success",
		TablesProcessed: 14,
		BytesReclaimed:  10485760, // 10MB
		DurationMs:      450,
		Details:         "Optimized 14 InnoDB tables, freed 10MB fragmented space",
		InitiatedBy:     "admin@hostvra.local",
	}

	if err := store.RecordDatabaseMaintenanceLog(ctx, mLog); err != nil {
		t.Fatalf("failed to record maintenance log: %v", err)
	}

	logs, err := store.ListDatabaseMaintenanceLogs(ctx, dbID, 10)
	if err != nil {
		t.Fatalf("failed to list maintenance logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].TablesProcessed != 14 || logs[0].BytesReclaimed != 10485760 {
		t.Fatalf("unexpected maintenance log: %+v", logs[0])
	}

	// Delete tests
	if err := store.DeleteDatabaseRemoteAccess(ctx, grantID); err != nil {
		t.Fatalf("failed to delete remote access grant: %v", err)
	}
	if err := store.DeleteWebsiteCacheConfig(ctx, websiteID); err != nil {
		t.Fatalf("failed to delete cache config: %v", err)
	}
	_, err = store.GetWebsiteCacheConfig(ctx, websiteID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}
