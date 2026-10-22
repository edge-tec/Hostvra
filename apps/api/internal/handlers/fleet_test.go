package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func setupFleetTestEnv() (*config.Config, store.Store, *audit.Logger, *ServerHandler, *CrossNodeMigrationHandler, uuid.UUID, *auth.Claims) {
	st := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(st, logger)
	cfg := &config.Config{JWTSecret: "test-secret-fleet-1234567890"}

	serverH := NewServerHandler(cfg, st, auditLogger)
	migrationH := NewCrossNodeMigrationHandler(cfg, st, auditLogger)

	orgID := uuid.New()
	_ = st.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Fleet Test Org",
		Slug: "fleet-org",
	})

	adminClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "admin@hostvra.local",
		OrganizationID: orgID,
		Role:           "admin",
		IsSuperAdmin:   true,
	}

	return cfg, st, auditLogger, serverH, migrationH, orgID, adminClaims
}

func TestFleet_MaintenanceAndDrain(t *testing.T) {
	_, st, _, serverH, _, orgID, claims := setupFleetTestEnv()
	ctx := context.Background()

	serverID := uuid.New()
	srv := &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "node-primary",
		Hostname:       "node1.hostvra.local",
		IPAddress:      "192.168.1.10",
		Status:         "online",
		CPUCores:       4,
		RAMTotalMB:     8192,
		DiskTotalGB:    200,
		MaxWebsites:    50,
		MaxDatabases:   50,
		Capabilities: store.ServerCapabilities{
			WebServers:  []string{"nginx"},
			PHPVersions: []string{"8.1", "8.2", "8.3"},
			Databases:   []string{"mysql"},
			Services:    []string{"fail2ban"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := st.CreateServer(ctx, srv); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// 1. Toggle Maintenance (Turn ON)
	maintReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID.String()+"/maintenance", bytes.NewReader([]byte(`{"maintenance": true}`)))
	maintReq = maintReq.WithContext(auth.WithClaims(maintReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID.String())
	maintReq = maintReq.WithContext(context.WithValue(maintReq.Context(), chi.RouteCtxKey, rctx))
	maintRec := httptest.NewRecorder()

	serverH.ToggleMaintenance(maintRec, maintReq)
	if maintRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", maintRec.Code, maintRec.Body.String())
	}

	updated, _ := st.GetServerByID(ctx, serverID)
	if !updated.IsMaintenance || updated.Status != "maintenance" {
		t.Fatalf("expected server in maintenance mode, got is_maintenance=%v, status=%s", updated.IsMaintenance, updated.Status)
	}

	// 2. Toggle Maintenance (Turn OFF)
	maintOffReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID.String()+"/maintenance", bytes.NewReader([]byte(`{"maintenance": false}`)))
	maintOffReq = maintOffReq.WithContext(auth.WithClaims(maintOffReq.Context(), claims))
	maintOffReq = maintOffReq.WithContext(context.WithValue(maintOffReq.Context(), chi.RouteCtxKey, rctx))
	maintOffRec := httptest.NewRecorder()

	serverH.ToggleMaintenance(maintOffRec, maintOffReq)
	if maintOffRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", maintOffRec.Code, maintOffRec.Body.String())
	}

	updated, _ = st.GetServerByID(ctx, serverID)
	if updated.IsMaintenance || updated.Status != "online" {
		t.Fatalf("expected server online, got is_maintenance=%v, status=%s", updated.IsMaintenance, updated.Status)
	}

	// 3. Drain Node
	drainReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID.String()+"/drain", bytes.NewReader([]byte(`{"draining": true}`)))
	drainReq = drainReq.WithContext(auth.WithClaims(drainReq.Context(), claims))
	drainReq = drainReq.WithContext(context.WithValue(drainReq.Context(), chi.RouteCtxKey, rctx))
	drainRec := httptest.NewRecorder()

	serverH.DrainNode(drainRec, drainReq)
	if drainRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", drainRec.Code, drainRec.Body.String())
	}

	updated, _ = st.GetServerByID(ctx, serverID)
	if !updated.Draining {
		t.Fatalf("expected server draining=true, got %v", updated.Draining)
	}
}

func TestFleet_WorkloadSummaryAndDeleteProtection(t *testing.T) {
	_, st, _, serverH, _, orgID, claims := setupFleetTestEnv()
	ctx := context.Background()

	serverID := uuid.New()
	srv := &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "node-workload",
		Hostname:       "workload.hostvra.local",
		IPAddress:      "192.168.1.20",
		Status:         "online",
		MaxWebsites:    10,
		MaxDatabases:   10,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	_ = st.CreateServer(ctx, srv)

	// Create 2 websites on this node
	_ = st.CreateWebsite(ctx, &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ServerID:       serverID,
		PrimaryDomain:  "site1.hostvra.local",
		DocumentRoot:   "/var/www/site1",
		SystemUser:     "user1",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})
	_ = st.CreateWebsite(ctx, &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ServerID:       serverID,
		PrimaryDomain:  "site2.hostvra.local",
		DocumentRoot:   "/var/www/site2",
		SystemUser:     "user2",
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	})

	// Create 1 database on this node
	_ = st.CreateDatabase(ctx, &store.Database{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ServerID:       serverID,
		DBType:         "mysql",
		Name:           "db_workload",
		CreatedAt:      time.Now().UTC(),
	})

	// 1. Get Workload Summary
	workReq := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID.String()+"/workloads", nil)
	workReq = workReq.WithContext(auth.WithClaims(workReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID.String())
	workReq = workReq.WithContext(context.WithValue(workReq.Context(), chi.RouteCtxKey, rctx))
	workRec := httptest.NewRecorder()

	serverH.GetWorkloadSummary(workRec, workReq)
	if workRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", workRec.Code, workRec.Body.String())
	}

	var workResp struct {
		Data store.ServerDrainSummary `json:"data"`
	}
	if err := json.NewDecoder(workRec.Body).Decode(&workResp); err != nil {
		t.Fatalf("failed to decode workload response: %v", err)
	}
	if workResp.Data.WebsitesCount != 2 || workResp.Data.DatabasesCount != 1 {
		t.Fatalf("expected 2 websites and 1 db, got %d sites, %d dbs", workResp.Data.WebsitesCount, workResp.Data.DatabasesCount)
	}

	// 2. Safe Delete Protection: Attempting delete without force must fail with 409 CONFLICT
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/servers/"+serverID.String(), nil)
	delReq = delReq.WithContext(auth.WithClaims(delReq.Context(), claims))
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))
	delRec := httptest.NewRecorder()

	serverH.DeleteServer(delRec, delReq)
	if delRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when deleting server with active workloads, got %d: %s", delRec.Code, delRec.Body.String())
	}

	// 3. Force Delete: Passing ?force=true must successfully delete
	forceDelReq := httptest.NewRequest(http.MethodDelete, "/api/v1/servers/"+serverID.String()+"?force=true", nil)
	forceDelReq = forceDelReq.WithContext(auth.WithClaims(forceDelReq.Context(), claims))
	forceDelReq = forceDelReq.WithContext(context.WithValue(forceDelReq.Context(), chi.RouteCtxKey, rctx))
	forceDelRec := httptest.NewRecorder()

	serverH.DeleteServer(forceDelRec, forceDelReq)
	if forceDelRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on force delete, got %d: %s", forceDelRec.Code, forceDelRec.Body.String())
	}

	deleted, err := st.GetServerByID(ctx, serverID)
	if err == nil && deleted != nil {
		t.Fatalf("expected server to be deleted from store, but found: %+v", deleted)
	}
}

func TestFleet_CrossNodeMigrationLifecycle(t *testing.T) {
	_, st, _, _, migrationH, orgID, claims := setupFleetTestEnv()
	ctx := context.Background()

	srcID := uuid.New()
	srcServer := &store.Server{
		ID:             srcID,
		OrganizationID: orgID,
		Name:           "source-node",
		Hostname:       "src.hostvra.local",
		IPAddress:      "10.0.0.1",
		Status:         "online",
		MaxWebsites:    100,
		MaxDatabases:   100,
		Capabilities: store.ServerCapabilities{
			WebServers:  []string{"nginx"},
			PHPVersions: []string{"8.1", "8.2", "8.3"},
			Databases:   []string{"mysql"},
			Services:    []string{"fail2ban"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = st.CreateServer(ctx, srcServer)

	destID := uuid.New()
	destServer := &store.Server{
		ID:             destID,
		OrganizationID: orgID,
		Name:           "dest-node",
		Hostname:       "dest.hostvra.local",
		IPAddress:      "10.0.0.2",
		Status:         "online",
		MaxWebsites:    100,
		MaxDatabases:   100,
		Capabilities: store.ServerCapabilities{
			WebServers:  []string{"nginx", "apache"},
			PHPVersions: []string{"8.1", "8.2", "8.3"},
			Databases:   []string{"mysql", "postgresql"},
			Services:    []string{"fail2ban"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = st.CreateServer(ctx, destServer)

	phpVer := "8.2"
	siteID := uuid.New()
	site := &store.Website{
		ID:             siteID,
		OrganizationID: orgID,
		ServerID:       srcID,
		PrimaryDomain:  "migration-test.com",
		DocumentRoot:   "/var/www/migration-test",
		SystemUser:     "miguser",
		Status:         "active",
		PHPVersion:     &phpVer,
		WebServerType:  "nginx",
		CreatedAt:      time.Now().UTC(),
	}
	_ = st.CreateWebsite(ctx, site)

	// 1. Preflight Check
	preflightBody, _ := json.Marshal(PreflightCheckRequest{
		SourceServerID:      srcID,
		DestinationServerID: destID,
		WebsiteID:           &siteID,
	})
	preReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/migrations/preflight", bytes.NewReader(preflightBody))
	preReq = preReq.WithContext(auth.WithClaims(preReq.Context(), claims))
	preRec := httptest.NewRecorder()

	migrationH.PreflightCheck(preRec, preReq)
	if preRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on preflight check, got %d: %s", preRec.Code, preRec.Body.String())
	}

	var preResp struct {
		Data PreflightCheckResponse `json:"data"`
	}
	_ = json.NewDecoder(preRec.Body).Decode(&preResp)
	if !preResp.Data.Compatible {
		t.Fatalf("expected compatible=true, got false: %+v", preResp.Data.Checks)
	}

	// 2. Create Migration Job
	migReqBody, _ := json.Marshal(CreateCrossNodeMigrationRequest{
		SourceServerID:      srcID,
		DestinationServerID: destID,
		WebsiteID:           &siteID,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/migrations", bytes.NewReader(migReqBody))
	createReq = createReq.WithContext(auth.WithClaims(createReq.Context(), claims))
	createRec := httptest.NewRecorder()

	migrationH.CreateMigration(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on migration start, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var createResp struct {
		Data store.CrossNodeMigration `json:"data"`
	}
	_ = json.NewDecoder(createRec.Body).Decode(&createResp)
	jobID := createResp.Data.ID

	job, err := st.GetCrossNodeMigration(ctx, jobID)
	if err != nil {
		t.Fatalf("failed to retrieve migration job: %v", err)
	}
	if job.ID == uuid.Nil {
		t.Fatalf("expected valid job ID")
	}

	// Prepare job for cutover test
	testChecksum := "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945"
	job.Status = "cutover_ready"
	job.SourceChecksum = testChecksum
	job.DestinationChecksum = testChecksum
	_ = st.UpdateCrossNodeMigration(ctx, job)

	// 3. Cutover Migration
	cutReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/migrations/"+jobID.String()+"/cutover", nil)
	cutReq = cutReq.WithContext(auth.WithClaims(cutReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	cutReq = cutReq.WithContext(context.WithValue(cutReq.Context(), chi.RouteCtxKey, rctx))
	cutRec := httptest.NewRecorder()

	migrationH.CutoverMigration(cutRec, cutReq)
	if cutRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on cutover, got %d: %s", cutRec.Code, cutRec.Body.String())
	}

	// Verify website ServerID was atomically updated to destID
	updatedSite, err := st.GetWebsiteByID(ctx, siteID)
	if err != nil {
		t.Fatalf("failed to get website: %v", err)
	}
	if updatedSite.ServerID != destID {
		t.Fatalf("expected site ServerID to be %s, got %s", destID, updatedSite.ServerID)
	}

	// 4. Test Rollback Prevention on Completed Migration
	rollReq := httptest.NewRequest(http.MethodPost, "/api/v1/servers/migrations/"+jobID.String()+"/rollback", nil)
	rollReq = rollReq.WithContext(auth.WithClaims(rollReq.Context(), claims))
	rollReq = rollReq.WithContext(context.WithValue(rollReq.Context(), chi.RouteCtxKey, rctx))
	rollRec := httptest.NewRecorder()

	migrationH.RollbackMigration(rollRec, rollReq)
	if rollRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when rolling back completed migration, got %d: %s", rollRec.Code, rollRec.Body.String())
	}
}
