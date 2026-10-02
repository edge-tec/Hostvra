package quota_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/store"
)

func TestQuotaAndOverrides(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	// 1. Create a customer user & org
	userID := uuid.New()
	orgID := uuid.New()
	timestamp := time.Now().UnixNano()
	org := &store.Organization{
		ID:          orgID,
		Name:        "Test Customer Org",
		Slug:        fmt.Sprintf("test-cust-org-%d", timestamp),
		PlanTier:    "starter",
		MaxWebsites: 1,
	}
	if err := memStore.CreateOrganization(ctx, org); err != nil && err != store.ErrAlreadyExists {
		t.Fatalf("failed to create org: %v", err)
	}

	user := &store.User{
		ID:           userID,
		Email:        fmt.Sprintf("customer_%d@example.com", timestamp),
		FullName:     "Test Customer",
		IsActive:     true,
		IsSuperAdmin: false,
	}
	if err := memStore.CreateUser(ctx, user, orgID, "customer"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	quotaService := quota.NewService(memStore)

	// 2. Check default plan resolution (starter-cloud)
	plan, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve plan: %v", err)
	}
	if plan.PlanName != "Starter Cloud" {
		t.Errorf("expected plan 'Starter Cloud', got '%s'", plan.PlanName)
	}
	if plan.MaxWebsites != 1 {
		t.Errorf("expected MaxWebsites 1, got %d", plan.MaxWebsites)
	}
	if plan.MaxDatabases != 2 {
		t.Errorf("expected MaxDatabases 2, got %d", plan.MaxDatabases)
	}
	if plan.Permissions["terminal"] {
		t.Errorf("expected terminal to be disabled by default for starter plan")
	}

	// 3. Test CheckQuota under limit (0 websites out of 1)
	if err := quotaService.CheckQuota(ctx, userID, "websites"); err != nil {
		t.Fatalf("expected quota check to pass for 0 websites: %v", err)
	}

	// Create 1 website
	site := &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  fmt.Sprintf("custsite-%d.com", timestamp),
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	if err := memStore.CreateWebsite(ctx, site); err != nil {
		t.Fatalf("failed to create website: %v", err)
	}

	// Now check quota again - should FAIL because limit is 1 and usage is 1
	if err := quotaService.CheckQuota(ctx, userID, "websites"); err == nil {
		t.Fatalf("expected quota check to fail when limit reached (1/1)")
	}

	// 4. Test Admin Override on limits: Increase MaxWebsites to 5 & enable terminal
	overrideWebsites := 5
	overrideTerminal := true
	override := &store.UserPlanOverride{
		UserID:             userID,
		MaxWebsites:        &overrideWebsites,
		PermissionTerminal: &overrideTerminal,
	}
	if err := memStore.UpsertUserPlanOverride(ctx, override); err != nil {
		t.Fatalf("failed to upsert override: %v", err)
	}

	// Re-resolve effective plan
	planWithOverride, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve plan with override: %v", err)
	}
	if planWithOverride.MaxWebsites != 5 {
		t.Errorf("expected MaxWebsites override 5, got %d", planWithOverride.MaxWebsites)
	}
	if !planWithOverride.Permissions["terminal"] {
		t.Errorf("expected PermissionTerminal override to be true")
	}
	if planWithOverride.Overrides == nil {
		t.Errorf("expected Overrides to not be nil")
	}

	// Now check quota again - should PASS because usage is 1 and limit is now 5
	if err := quotaService.CheckQuota(ctx, userID, "websites"); err != nil {
		t.Fatalf("expected quota check to pass after admin limit override: %v", err)
	}

	// Check terminal permission - should PASS due to override
	if allowed := quotaService.CheckPermission(ctx, userID, "terminal"); !allowed {
		t.Fatalf("expected terminal permission to pass after admin override")
	}

	// 5. Delete override - should revert back to package defaults
	if err := memStore.DeleteUserPlanOverride(ctx, userID); err != nil {
		t.Fatalf("failed to delete override: %v", err)
	}

	planReverted, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve reverted plan: %v", err)
	}
	if planReverted.MaxWebsites != 1 {
		t.Errorf("expected MaxWebsites to revert to 1, got %d", planReverted.MaxWebsites)
	}
	if planReverted.Permissions["terminal"] {
		t.Errorf("expected PermissionTerminal to revert to false")
	}

	// Quota check should fail again
	if err := quotaService.CheckQuota(ctx, userID, "websites"); err == nil {
		t.Fatalf("expected quota check to fail after override removed (1/1)")
	}
}

func TestStorageQuotaAndTenantIsolation(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()
	quotaService := quota.NewService(memStore)

	// Create User A (Starter plan: 10 GB)
	userA := &store.User{
		ID:           uuid.New(),
		Email:        "usera@example.com",
		FullName:     "User Alpha",
		IsActive:     true,
		IsSuperAdmin: false,
	}
	orgA := &store.Organization{
		ID:          uuid.New(),
		Name:        "Alpha Org",
		Slug:        "alpha-org",
		PlanTier:    "starter",
		MaxWebsites: 5,
	}
	_ = memStore.CreateOrganization(ctx, orgA)
	_ = memStore.CreateUser(ctx, userA, orgA.ID, "customer")

	// Create User B (Starter plan: 10 GB)
	userB := &store.User{
		ID:           uuid.New(),
		Email:        "userb@example.com",
		FullName:     "User Beta",
		IsActive:     true,
		IsSuperAdmin: false,
	}
	orgB := &store.Organization{
		ID:          uuid.New(),
		Name:        "Beta Org",
		Slug:        "beta-org",
		PlanTier:    "starter",
		MaxWebsites: 5,
	}
	_ = memStore.CreateOrganization(ctx, orgB)
	_ = memStore.CreateUser(ctx, userB, orgB.ID, "customer")

	// 1. Initial State: Neither user should have 64 GB hardcoded or 100% full!
	planA, err := quotaService.ResolveEffectivePlan(ctx, userA.ID)
	if err != nil {
		t.Fatalf("failed to resolve plan A: %v", err)
	}
	expectedQuotaBytes := int64(10240) * 1024 * 1024 // 10 GB
	if planA.Storage.QuotaBytes != expectedQuotaBytes {
		t.Errorf("expected 10 GB quota (%d), got %d", expectedQuotaBytes, planA.Storage.QuotaBytes)
	}
	if planA.Storage.UsedBytes != 0 {
		t.Errorf("expected 0 used bytes for fresh user, got %d", planA.Storage.UsedBytes)
	}
	if planA.Storage.FreeBytes != expectedQuotaBytes {
		t.Errorf("expected %d free bytes, got %d", expectedQuotaBytes, planA.Storage.FreeBytes)
	}
	if planA.Storage.UsagePercent != 0.0 {
		t.Errorf("expected 0%% used, got %.1f%%", planA.Storage.UsagePercent)
	}

	// 2. Add real files to User A's website
	tempDir := t.TempDir()
	siteADocRoot := filepath.Join(tempDir, "var_www_alpha")
	if err := os.MkdirAll(siteADocRoot, 0755); err != nil {
		t.Fatalf("failed to create site A dir: %v", err)
	}
	// Write a 2 MB file for User A
	dummyData := make([]byte, 2*1024*1024)
	if err := os.WriteFile(filepath.Join(siteADocRoot, "app.bin"), dummyData, 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	siteA := &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgA.ID,
		PrimaryDomain:  "alpha.com",
		DocumentRoot:   siteADocRoot,
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(ctx, siteA)

	// Force refresh cache for User A
	quotaService.InvalidateUserStorageCache(userA.ID)
	planAAfter, err := quotaService.ResolveEffectivePlan(ctx, userA.ID)
	if err != nil {
		t.Fatalf("failed to resolve plan A after file write: %v", err)
	}
	if planAAfter.Storage.UsedBytes != 2*1024*1024 {
		t.Errorf("expected 2MB (%d bytes) used for User A, got %d", 2*1024*1024, planAAfter.Storage.UsedBytes)
	}
	if planAAfter.Storage.FreeBytes != expectedQuotaBytes-2*1024*1024 {
		t.Errorf("expected free bytes %d, got %d", expectedQuotaBytes-2*1024*1024, planAAfter.Storage.FreeBytes)
	}

	// 3. Tenant Isolation Check: User B MUST still have 0 bytes used!
	planB, err := quotaService.ResolveEffectivePlan(ctx, userB.ID)
	if err != nil {
		t.Fatalf("failed to resolve plan B: %v", err)
	}
	if planB.Storage.UsedBytes != 0 {
		t.Errorf("tenant leak! User B has %d bytes used, should be 0", planB.Storage.UsedBytes)
	}
	if planB.Storage.FreeBytes != expectedQuotaBytes {
		t.Errorf("expected User B free bytes %d, got %d", expectedQuotaBytes, planB.Storage.FreeBytes)
	}

	// 4. Admin Storage Override Test: Admin sets 25 GB for User A
	overrideDiskMB := int64(25600) // 25 GB
	override := &store.UserPlanOverride{
		UserID:      userA.ID,
		DiskSpaceMB: &overrideDiskMB,
	}
	_ = memStore.UpsertUserPlanOverride(ctx, override)
	planAOverride, _ := quotaService.ResolveEffectivePlan(ctx, userA.ID)
	expectedOverrideBytes := int64(25600) * 1024 * 1024
	if planAOverride.Storage.QuotaBytes != expectedOverrideBytes {
		t.Errorf("expected override quota %d, got %d", expectedOverrideBytes, planAOverride.Storage.QuotaBytes)
	}
	if planAOverride.Storage.Source != "admin_override" {
		t.Errorf("expected source 'admin_override', got '%s'", planAOverride.Storage.Source)
	}

	// 5. Quota Check on Storage
	if err := quotaService.CheckQuota(ctx, userA.ID, "storage"); err != nil {
		t.Errorf("expected storage quota check to pass: %v", err)
	}
}

