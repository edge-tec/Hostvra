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

// TestScenariosAthroughF tests exact scenarios A through F from requirements:
// Scenario A: Physical disk = 64 GB, Package = 10 GB, Customer usage = 0 GB -> 0 / 10 GB, 0%, 10 GB free
// Scenario B: Physical disk = 64 GB, Package = 10 GB, Customer usage = 1 GB -> 1 / 10 GB, 10%, 9 GB free
// Scenario C: Physical disk = 64 GB, Package = 10 GB, Customer usage = 5 GB -> 5 / 10 GB, 50%, 5 GB free
// Scenario D: Physical disk = 64 GB, Package = 10 GB, Customer usage = 10 GB -> 10 / 10 GB, 100%, 0 B free
// Scenario E: Physical disk = 64 GB, Package = 25 GB, Customer usage = 5 GB -> 5 / 25 GB, 20%, 20 GB free
// Scenario F: Physical disk = 64 GB, Package = 50 GB, Customer usage = 5 GB -> 5 / 50 GB, 10%, 45 GB free
func TestScenariosAthroughF(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()
	quotaService := quota.NewService(memStore)

	testCases := []struct {
		name            string
		packageMB       int64
		usageBytes      int64
		expectedQuota   int64
		expectedUsed    int64
		expectedFree    int64
		expectedPercent float64
	}{
		{
			name:            "Scenario A: 0 GB used of 10 GB",
			packageMB:       10240, // 10 GB
			usageBytes:      0,
			expectedQuota:   10240 * 1024 * 1024,
			expectedUsed:    0,
			expectedFree:    10240 * 1024 * 1024,
			expectedPercent: 0.0,
		},
		{
			name:            "Scenario B: 1 GB used of 10 GB",
			packageMB:       10240, // 10 GB
			usageBytes:      1024 * 1024 * 1024,
			expectedQuota:   10240 * 1024 * 1024,
			expectedUsed:    1024 * 1024 * 1024,
			expectedFree:    9216 * 1024 * 1024,
			expectedPercent: 10.0,
		},
		{
			name:            "Scenario C: 5 GB used of 10 GB",
			packageMB:       10240, // 10 GB
			usageBytes:      5120 * 1024 * 1024,
			expectedQuota:   10240 * 1024 * 1024,
			expectedUsed:    5120 * 1024 * 1024,
			expectedFree:    5120 * 1024 * 1024,
			expectedPercent: 50.0,
		},
		{
			name:            "Scenario D: 10 GB used of 10 GB",
			packageMB:       10240, // 10 GB
			usageBytes:      10240 * 1024 * 1024,
			expectedQuota:   10240 * 1024 * 1024,
			expectedUsed:    10240 * 1024 * 1024,
			expectedFree:    0,
			expectedPercent: 100.0,
		},
		{
			name:            "Scenario E: 5 GB used of 25 GB",
			packageMB:       25600, // 25 GB
			usageBytes:      5120 * 1024 * 1024,
			expectedQuota:   25600 * 1024 * 1024,
			expectedUsed:    5120 * 1024 * 1024,
			expectedFree:    20480 * 1024 * 1024,
			expectedPercent: 20.0,
		},
		{
			name:            "Scenario F: 5 GB used of 50 GB",
			packageMB:       51200, // 50 GB
			usageBytes:      5120 * 1024 * 1024,
			expectedQuota:   51200 * 1024 * 1024,
			expectedUsed:    5120 * 1024 * 1024,
			expectedFree:    46080 * 1024 * 1024,
			expectedPercent: 10.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userID := uuid.New()
			orgID := uuid.New()

			user := &store.User{
				ID:           userID,
				Email:        fmt.Sprintf("user_%s@example.com", userID.String()[:8]),
				FullName:     "Scenario User",
				IsActive:     true,
				IsSuperAdmin: false,
			}
			org := &store.Organization{
				ID:          orgID,
				Name:        "Scenario Org",
				Slug:        "scenario-org-" + userID.String()[:8],
				PlanTier:    "starter",
				MaxWebsites: 5,
			}
			_ = memStore.CreateOrganization(ctx, org)
			_ = memStore.CreateUser(ctx, user, orgID, "customer")

			// Set package disk space via admin override to simulate exact package tiers
			overrideDisk := tc.packageMB
			_ = memStore.UpsertUserPlanOverride(ctx, &store.UserPlanOverride{
				UserID:      userID,
				DiskSpaceMB: &overrideDisk,
			})

			// Add database resource to simulate the exact customer usage
			if tc.usageBytes > 0 {
				dbName := fmt.Sprintf("scenario_db_%s", userID.String()[:8])
				_ = memStore.CreateDatabase(ctx, &store.Database{
					ID:             uuid.New(),
					OrganizationID: orgID,
					Name:           dbName,
					SizeBytes:      tc.usageBytes,
				})
			}

			plan, err := quotaService.ResolveEffectivePlan(ctx, userID)
			if err != nil {
				t.Fatalf("failed to resolve plan: %v", err)
			}

			if plan.Storage.QuotaBytes != tc.expectedQuota {
				t.Errorf("expected quota %d bytes, got %d", tc.expectedQuota, plan.Storage.QuotaBytes)
			}
			if plan.Storage.UsedBytes != tc.expectedUsed {
				t.Errorf("expected used %d bytes, got %d", tc.expectedUsed, plan.Storage.UsedBytes)
			}
			if plan.Storage.FreeBytes != tc.expectedFree {
				t.Errorf("expected free %d bytes, got %d", tc.expectedFree, plan.Storage.FreeBytes)
			}
			if plan.Storage.UsagePercent != tc.expectedPercent {
				t.Errorf("expected usage %.1f%%, got %.1f%%", tc.expectedPercent, plan.Storage.UsagePercent)
			}

			// Verify CheckStorageQuota
			if tc.expectedFree > 0 {
				// Uploading 1 MB should succeed
				if err := quotaService.CheckStorageQuota(ctx, userID, 1024*1024); err != nil {
					t.Errorf("expected CheckStorageQuota to pass: %v", err)
				}
			} else {
				// Full quota: uploading 1 MB must fail
				if err := quotaService.CheckStorageQuota(ctx, userID, 1024*1024); err == nil {
					t.Errorf("expected CheckStorageQuota to fail when quota is full")
				}
			}
		})
	}
}

func TestSystemPathAndUserExclusion(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()
	quotaService := quota.NewService(memStore)

	userID := uuid.New()
	orgID := uuid.New()
	user := &store.User{
		ID:           userID,
		Email:        "system_check@example.com",
		FullName:     "System Check User",
		IsActive:     true,
		IsSuperAdmin: false,
	}
	org := &store.Organization{
		ID:          orgID,
		Name:        "System Org",
		Slug:        "system-org",
		PlanTier:    "starter",
		MaxWebsites: 5,
	}
	_ = memStore.CreateOrganization(ctx, org)
	_ = memStore.CreateUser(ctx, user, orgID, "customer")

	// Create hosting accounts with system usernames (e.g. ubuntu, root, hostvra)
	// These must be strictly ignored and NOT measured towards customer usage!
	_ = memStore.CreateHostingAccount(ctx, &store.HostingAccount{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Username:       "ubuntu",
	})
	_ = memStore.CreateHostingAccount(ctx, &store.HostingAccount{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Username:       "root",
	})
	_ = memStore.CreateHostingAccount(ctx, &store.HostingAccount{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Username:       "", // Empty username
	})

	// Add a website with forbidden system paths
	_ = memStore.CreateWebsite(ctx, &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "systemsite.com",
		DocumentRoot:   "/var/www", // Bare prefix - forbidden!
		Status:         "active",
	})
	_ = memStore.CreateWebsite(ctx, &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "systemsite2.com",
		DocumentRoot:   "/etc", // Prohibited system root!
		Status:         "active",
	})

	plan, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve plan: %v", err)
	}

	// Must be exactly 0 bytes used!
	if plan.Storage.UsedBytes != 0 {
		t.Fatalf("system isolation failed! Used bytes is %d, expected 0", plan.Storage.UsedBytes)
	}
	if plan.Storage.UsagePercent != 0.0 {
		t.Fatalf("usage percent is %.1f%%, expected 0.0%%", plan.Storage.UsagePercent)
	}
}

func TestPackageSwitchingAndStorageEnforcement(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()
	quotaService := quota.NewService(memStore)

	userID := uuid.New()
	orgID := uuid.New()
	user := &store.User{
		ID:           userID,
		Email:        "switch_test@example.com",
		FullName:     "Package Switcher",
		IsActive:     true,
		IsSuperAdmin: false,
	}
	org := &store.Organization{
		ID:          orgID,
		Name:        "Switch Org",
		Slug:        "switch-org",
		PlanTier:    "starter", // 10 GB
		MaxWebsites: 5,
	}
	_ = memStore.CreateOrganization(ctx, org)
	_ = memStore.CreateUser(ctx, user, orgID, "customer")

	// Customer uses 4 GB via database
	_ = memStore.CreateDatabase(ctx, &store.Database{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "switch_db",
		SizeBytes:      4 * 1024 * 1024 * 1024, // 4 GB
	})

	// 1. Initial 10 GB Plan
	plan1, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve plan: %v", err)
	}
	expected10GB := int64(10240) * 1024 * 1024
	if plan1.Storage.QuotaBytes != expected10GB {
		t.Fatalf("expected 10 GB quota, got %d", plan1.Storage.QuotaBytes)
	}
	if plan1.Storage.FreeBytes != expected10GB-4*1024*1024*1024 {
		t.Fatalf("expected 6 GB free, got %d", plan1.Storage.FreeBytes)
	}

	// 2. Package upgrade to Pro (25 GB)
	override25GB := int64(25600)
	_ = memStore.UpsertUserPlanOverride(ctx, &store.UserPlanOverride{
		UserID:      userID,
		DiskSpaceMB: &override25GB,
	})
	quotaService.InvalidateUserStorageCache(userID)

	plan2, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve plan: %v", err)
	}
	expected25GB := int64(25600) * 1024 * 1024
	if plan2.Storage.QuotaBytes != expected25GB {
		t.Fatalf("expected 25 GB quota after upgrade, got %d", plan2.Storage.QuotaBytes)
	}
	if plan2.Storage.FreeBytes != expected25GB-4*1024*1024*1024 {
		t.Fatalf("expected 21 GB free, got %d", plan2.Storage.FreeBytes)
	}

	// 3. Package upgrade to Enterprise (50 GB)
	override50GB := int64(51200)
	_ = memStore.UpsertUserPlanOverride(ctx, &store.UserPlanOverride{
		UserID:      userID,
		DiskSpaceMB: &override50GB,
	})
	quotaService.InvalidateUserStorageCache(userID)

	plan3, err := quotaService.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		t.Fatalf("failed to resolve plan: %v", err)
	}
	expected50GB := int64(51200) * 1024 * 1024
	if plan3.Storage.QuotaBytes != expected50GB {
		t.Fatalf("expected 50 GB quota after upgrade, got %d", plan3.Storage.QuotaBytes)
	}
	if plan3.Storage.FreeBytes != expected50GB-4*1024*1024*1024 {
		t.Fatalf("expected 46 GB free, got %d", plan3.Storage.FreeBytes)
	}

	// 4. Test CheckQuota storage rejection when quota is exceeded
	// Override limit down to 2 GB while using 4 GB
	override2GB := int64(2048)
	_ = memStore.UpsertUserPlanOverride(ctx, &store.UserPlanOverride{
		UserID:      userID,
		DiskSpaceMB: &override2GB,
	})
	quotaService.InvalidateUserStorageCache(userID)

	planExceeded, _ := quotaService.ResolveEffectivePlan(ctx, userID)
	if planExceeded.Storage.FreeBytes != 0 {
		t.Fatalf("expected 0 free bytes when over-quota, got %d", planExceeded.Storage.FreeBytes)
	}
	if planExceeded.Storage.UsagePercent != 100.0 {
		t.Fatalf("expected 100%% usage percent when over-quota, got %.1f", planExceeded.Storage.UsagePercent)
	}

	// CheckQuota should fail
	if err := quotaService.CheckQuota(ctx, userID, "storage"); err == nil {
		t.Fatalf("expected CheckQuota to return error when over storage quota")
	}
}


