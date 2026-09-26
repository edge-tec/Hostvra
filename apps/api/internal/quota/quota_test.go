package quota_test

import (
	"context"
	"fmt"
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
