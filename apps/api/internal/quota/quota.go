package quota

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

var (
	ErrQuotaExceeded      = fmt.Errorf("resource limit quota exceeded")
	ErrFeatureDisabled    = fmt.Errorf("feature is not permitted on this hosting package")
	ErrUserNotFound       = fmt.Errorf("user not found")
	ErrPlanNotFound       = fmt.Errorf("hosting plan not found")
)

// Service provides thread-safe, centralized resolution of effective limits,
// feature permissions, usage tracking, and admin overrides.
type Service struct {
	store store.Store
	mu    sync.RWMutex
}

func NewService(s store.Store) *Service {
	return &Service{
		store: s,
	}
}

// ResolveEffectivePlan calculates the final effective quotas and permissions for a user
// Hierarchy: User Overrides > Package Defaults > System Defaults
func (s *Service) ResolveEffectivePlan(ctx context.Context, userID uuid.UUID) (*store.EffectiveUserPlan, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 1. SuperAdmin / Root Administrator gets unlimited infrastructure privileges
	if user.IsSuperAdmin || user.Role == "admin" || user.Role == "superadmin" {
		return &store.EffectiveUserPlan{
			UserID:             user.ID,
			UserEmail:          user.Email,
			UserName:           user.FullName,
			Role:               user.Role,
			IsActive:           user.IsActive,
			IsSuperAdmin:       true,
			PlanID:             uuid.Nil,
			PlanName:           "Infrastructure Root Administrator",
			PlanSlug:           "root-admin",
			PlanTier:           "enterprise",
			SubscriptionStatus: "active",
			MaxWebsites:        999999,
			MaxDatabases:       999999,
			MaxMailboxes:       999999,
			MaxFTP:             999999,
			MaxCron:            999999,
			MaxSubdomains:      999999,
			DiskSpaceMB:        10485760, // 10 TB
			BandwidthMB:        104857600,
			Permissions: map[string]bool{
				"terminal":     true,
				"file_manager": true,
				"backups":      true,
				"dns":          true,
				"ssl":          true,
				"cron":         true,
				"apps":         true,
				"php_selector": true,
				"whm_admin":    true,
				"server_admin": true,
			},
			Usage: s.getUsageUnchecked(ctx, user.ID, user.DefaultOrgID),
		}, nil
	}

	// 2. Fetch User's Active Subscription & Plan
	var assignedPlan *store.HostingPlan
	var subStatus string = "active"

	subs, err := s.store.ListSubscriptions(ctx, user.DefaultOrgID)
	if err == nil {
		for _, sub := range subs {
			if sub.UserID == user.ID && (sub.Status == store.SubStatusActive || sub.Status == store.SubStatusTrial) {
				plan, pErr := s.store.GetPlanByID(ctx, sub.PlanID)
				if pErr == nil && plan != nil {
					assignedPlan = plan
					subStatus = string(sub.Status)
					break
				}
			}
		}
	}

	// Fallback to default Starter Cloud plan if no active subscription found
	if assignedPlan == nil {
		plans, err := s.store.ListPlans(ctx)
		if err == nil && len(plans) > 0 {
			// Find starter plan
			for _, p := range plans {
				if strings.Contains(strings.ToLower(p.Slug), "starter") || p.Tier == store.PlanTierStarter {
					assignedPlan = p
					break
				}
			}
			if assignedPlan == nil {
				assignedPlan = plans[0]
			}
		} else {
			// Hard default baseline
			assignedPlan = &store.HostingPlan{
				ID:            uuid.MustParse("10000000-0000-0000-0000-000000000001"),
				Name:          "Starter Cloud",
				Slug:          "starter-cloud",
				Tier:          store.PlanTierStarter,
				DiskSpaceMB:   10240,
				BandwidthMB:   102400,
				MaxWebsites:   1,
				MaxDatabases:  2,
				MaxMailboxes:  5,
				MaxFTP:        2,
				MaxCron:       5,
				MaxSubdomains: 10,
				FreeSSL:       true,
			}
		}
	}

	// 3. Build default effective package structure
	hasTerminal := strings.EqualFold(string(assignedPlan.Tier), "enterprise") || strings.EqualFold(string(assignedPlan.Tier), "business")
	for _, f := range assignedPlan.Features {
		if strings.Contains(strings.ToLower(f), "terminal") || strings.Contains(strings.ToLower(f), "ssh") {
			hasTerminal = true
			break
		}
	}

	effective := &store.EffectiveUserPlan{
		UserID:             user.ID,
		UserEmail:          user.Email,
		UserName:           user.FullName,
		Role:               user.Role,
		IsActive:           user.IsActive,
		IsSuperAdmin:       false,
		PlanID:             assignedPlan.ID,
		PlanName:           assignedPlan.Name,
		PlanSlug:           assignedPlan.Slug,
		PlanTier:           string(assignedPlan.Tier),
		SubscriptionStatus: subStatus,
		MaxWebsites:        assignedPlan.MaxWebsites,
		MaxDatabases:       assignedPlan.MaxDatabases,
		MaxMailboxes:       assignedPlan.MaxMailboxes,
		MaxFTP:             assignedPlan.MaxFTP,
		MaxCron:            assignedPlan.MaxCron,
		MaxSubdomains:      assignedPlan.MaxSubdomains,
		DiskSpaceMB:        assignedPlan.DiskSpaceMB,
		BandwidthMB:        assignedPlan.BandwidthMB,
		Permissions: map[string]bool{
			"terminal":     hasTerminal,
			"file_manager": true,
			"backups":      true,
			"dns":          true,
			"ssl":          assignedPlan.FreeSSL,
			"cron":         assignedPlan.MaxCron > 0,
			"apps":         true,
			"php_selector": true,
			"whm_admin":    false,
			"server_admin": false,
		},
		Usage: s.getUsageUnchecked(ctx, user.ID, user.DefaultOrgID),
	}

	// 4. Layer User-Specific Overrides (if any configured by Admin)
	override, err := s.store.GetUserPlanOverride(ctx, user.ID)
	if err == nil && override != nil {
		effective.Overrides = override

		if override.PlanID != nil {
			if altPlan, err := s.store.GetPlanByID(ctx, *override.PlanID); err == nil && altPlan != nil {
				effective.PlanID = altPlan.ID
				effective.PlanName = altPlan.Name
				effective.PlanSlug = altPlan.Slug
				effective.PlanTier = string(altPlan.Tier)
			}
		}
		if override.MaxWebsites != nil {
			effective.MaxWebsites = *override.MaxWebsites
		}
		if override.MaxDatabases != nil {
			effective.MaxDatabases = *override.MaxDatabases
		}
		if override.MaxMailboxes != nil {
			effective.MaxMailboxes = *override.MaxMailboxes
		}
		if override.MaxFTP != nil {
			effective.MaxFTP = *override.MaxFTP
		}
		if override.MaxCron != nil {
			effective.MaxCron = *override.MaxCron
		}
		if override.MaxSubdomains != nil {
			effective.MaxSubdomains = *override.MaxSubdomains
		}
		if override.DiskSpaceMB != nil {
			effective.DiskSpaceMB = *override.DiskSpaceMB
		}
		if override.BandwidthMB != nil {
			effective.BandwidthMB = *override.BandwidthMB
		}
		if override.PermissionTerminal != nil {
			effective.Permissions["terminal"] = *override.PermissionTerminal
		}
		if override.PermissionBackups != nil {
			effective.Permissions["backups"] = *override.PermissionBackups
		}
		if override.PermissionDNS != nil {
			effective.Permissions["dns"] = *override.PermissionDNS
		}
		if override.PermissionSSL != nil {
			effective.Permissions["ssl"] = *override.PermissionSSL
		}
		if override.PermissionFileManager != nil {
			effective.Permissions["file_manager"] = *override.PermissionFileManager
		}
		if override.PermissionCron != nil {
			effective.Permissions["cron"] = *override.PermissionCron
		}
		if override.PermissionApps != nil {
			effective.Permissions["apps"] = *override.PermissionApps
		}
		if override.PermissionPHPSelector != nil {
			effective.Permissions["php_selector"] = *override.PermissionPHPSelector
		}
	}

	return effective, nil
}

// CheckQuota validates if user has remaining capacity before creating a resource
func (s *Service) CheckQuota(ctx context.Context, userID uuid.UUID, resource string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	effective, err := s.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		return err
	}

	// Superadmins bypass all quotas
	if effective.IsSuperAdmin || effective.Role == "admin" || effective.Role == "superadmin" {
		return nil
	}

	var current, limit int
	switch strings.ToLower(resource) {
	case "website", "websites":
		current = effective.Usage.WebsitesCount
		limit = effective.MaxWebsites
	case "database", "databases":
		current = effective.Usage.DatabasesCount
		limit = effective.MaxDatabases
	case "mailbox", "mailboxes", "email":
		current = effective.Usage.MailboxesCount
		limit = effective.MaxMailboxes
	case "ftp":
		current = effective.Usage.FTPCount
		limit = effective.MaxFTP
	case "cron":
		current = effective.Usage.CronCount
		limit = effective.MaxCron
	case "subdomain", "subdomains":
		current = effective.Usage.SubdomainsCount
		limit = effective.MaxSubdomains
	default:
		return nil
	}

	if limit > 0 && current >= limit {
		return fmt.Errorf("quota exceeded: you have reached the maximum allowed %s (%d/%d) for your plan. Please upgrade your package or contact administration", resource, current, limit)
	}

	return nil
}

// CheckPermission checks if a given feature toggle is enabled for this user's effective plan
func (s *Service) CheckPermission(ctx context.Context, userID uuid.UUID, feature string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	effective, err := s.ResolveEffectivePlan(ctx, userID)
	if err != nil {
		return false
	}

	if effective.IsSuperAdmin || effective.Role == "admin" || effective.Role == "superadmin" {
		return true
	}

	allowed, ok := effective.Permissions[strings.ToLower(feature)]
	if !ok {
		return false
	}
	return allowed
}

// getUsageUnchecked collects live resource counts
func (s *Service) getUsageUnchecked(ctx context.Context, userID, orgID uuid.UUID) store.UserResourceUsage {
	usage := store.UserResourceUsage{}

	// 1. Websites count
	if sites, err := s.store.ListWebsitesByOrg(ctx, orgID); err == nil {
		usage.WebsitesCount = len(sites)
	}

	// 2. Databases count
	if dbs, err := s.store.ListDatabasesByServer(ctx, uuid.Nil); err == nil {
		// Count active non-recycle databases
		for _, db := range dbs {
			if !db.InRecycleBin {
				usage.DatabasesCount++
			}
		}
	}

	// 3. Mailboxes count
	if domains, err := s.store.ListEmailDomainsByOrg(ctx, orgID); err == nil {
		for _, d := range domains {
			if mbs, err := s.store.ListEmailMailboxesByDomain(ctx, d.ID); err == nil {
				usage.MailboxesCount += len(mbs)
			}
		}
	}

	// 4. Hosting Account usage if present
	if accs, err := s.store.ListHostingAccounts(ctx, orgID, nil); err == nil {
		for _, a := range accs {
			usage.DiskUsedMB += a.DiskUsedMB
			usage.BandwidthUsedMB += a.BandwidthUsedMB
			if usage.FTPCount == 0 {
				usage.FTPCount = 1
			}
		}
	}

	// Default minimum calculated storage if databases or sites exist
	if usage.DiskUsedMB == 0 && (usage.WebsitesCount > 0 || usage.DatabasesCount > 0) {
		usage.DiskUsedMB = int64(usage.WebsitesCount*150 + usage.DatabasesCount*50)
	}

	return usage
}
