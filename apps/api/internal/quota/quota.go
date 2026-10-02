package quota

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

var (
	ErrQuotaExceeded   = fmt.Errorf("resource limit quota exceeded")
	ErrFeatureDisabled = fmt.Errorf("feature is not permitted on this hosting package")
	ErrUserNotFound    = fmt.Errorf("user not found")
	ErrPlanNotFound    = fmt.Errorf("hosting plan not found")
)

type storageCacheEntry struct {
	usedBytes int64
	timestamp time.Time
}

// Service provides thread-safe, centralized resolution of effective limits,
// feature permissions, usage tracking, and admin overrides.
type Service struct {
	store        store.Store
	mu           sync.RWMutex
	userLocks    sync.Map // map of uuid.UUID -> *sync.Mutex
	storageCache sync.Map // map of uuid.UUID -> storageCacheEntry
}

func NewService(s store.Store) *Service {
	return &Service{
		store: s,
	}
}

func (s *Service) InvalidateUserStorageCache(userID uuid.UUID) {
	s.storageCache.Delete(userID)
}

func (s *Service) getUserMutex(userID uuid.UUID) *sync.Mutex {
	val, _ := s.userLocks.LoadOrStore(userID, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// LockUser locks resource creation for a specific user to serialize concurrent creation and prevent race condition quota bypasses.
func (s *Service) LockUser(userID uuid.UUID) func() {
	m := s.getUserMutex(userID)
	m.Lock()
	return func() {
		m.Unlock()
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
		actualUsage := s.getUsageUnchecked(ctx, user.ID, user.DefaultOrgID)
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
			Usage: actualUsage,
			Storage: store.UserStorageQuotaDetails{
				QuotaBytes:   10 * 1024 * 1024 * 1024 * 1024,
				UsedBytes:    actualUsage.DiskUsedMB * 1024 * 1024,
				FreeBytes:    0,
				UsagePercent: 0.0,
				IsUnlimited:  true,
				Source:       "root_admin",
			},
		}, nil
	}

	// 2. Fetch User's Active Subscription & Plan
	var assignedPlan *store.HostingPlan
	var subStatus string = "active"
	var latestSub *store.Subscription
	now := time.Now().UTC()

	subs, err := s.store.ListSubscriptions(ctx, user.DefaultOrgID)
	if err == nil {
		for _, sub := range subs {
			if sub.UserID == user.ID {
				if latestSub == nil || sub.CreatedAt.After(latestSub.CreatedAt) {
					latestSub = sub
				}

				// Check trial expiration
				if sub.Status == store.SubStatusTrial && sub.TrialEndsAt != nil && sub.TrialEndsAt.Before(now) {
					sub.Status = store.SubStatusExpired
					_ = s.store.UpdateSubscription(ctx, sub)
				}

				// Check overdue active subscription expiration (grace period 7 days)
				if sub.Status == store.SubStatusActive && !sub.NextBillingDate.IsZero() && sub.NextBillingDate.AddDate(0, 0, 7).Before(now) && !sub.AutoRenew {
					sub.Status = store.SubStatusExpired
					_ = s.store.UpdateSubscription(ctx, sub)
				}

				if sub.Status == store.SubStatusActive || sub.Status == store.SubStatusTrial {
					plan, pErr := s.store.GetPlanByID(ctx, sub.PlanID)
					if pErr == nil && plan != nil {
						assignedPlan = plan
						subStatus = string(sub.Status)
						break
					}
				}
			}
		}
	}

	if assignedPlan == nil && latestSub != nil {
		// User had a subscription that is now expired/suspended/cancelled/pending
		subStatus = string(latestSub.Status)
		if plan, pErr := s.store.GetPlanByID(ctx, latestSub.PlanID); pErr == nil && plan != nil {
			assignedPlan = plan
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

	userRole := user.Role
	if userRole == "owner" || userRole == "" {
		userRole = "customer"
	}

	effective := &store.EffectiveUserPlan{
		UserID:             user.ID,
		UserEmail:          user.Email,
		UserName:           user.FullName,
		Role:               userRole,
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

	// If subscription is expired or suspended, restrict creation limits and features
	if subStatus == "expired" || subStatus == "suspended" || subStatus == "cancelled" {
		effective.MaxWebsites = 0
		effective.MaxDatabases = 0
		effective.MaxMailboxes = 0
		effective.MaxFTP = 0
		effective.MaxCron = 0
		effective.MaxSubdomains = 0
		effective.DiskSpaceMB = 0
		effective.BandwidthMB = 0
		for k := range effective.Permissions {
			effective.Permissions[k] = false
		}
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

	// 5. Authoritative Storage Quota & Filesystem Usage Calculation
	quotaMB := effective.DiskSpaceMB
	var quotaBytes int64
	isUnlimited := false

	// Quota <= 0 or >= 10TB is treated as unlimited
	if quotaMB <= 0 || quotaMB >= 10485760 {
		isUnlimited = true
		quotaBytes = 0
	} else {
		quotaBytes = quotaMB * 1024 * 1024
	}

	usedBytes := s.CalculateActualUserStorage(ctx, user.ID, user.DefaultOrgID, false)

	var freeBytes int64
	var usagePercent float64

	if isUnlimited {
		freeBytes = 0
		usagePercent = 0.0
	} else if quotaBytes > 0 {
		if usedBytes >= quotaBytes {
			freeBytes = 0
			usagePercent = 100.0
		} else {
			freeBytes = quotaBytes - usedBytes
			usagePercent = float64(usedBytes) / float64(quotaBytes) * 100.0
		}
	}

	source := "package"
	if effective.Overrides != nil && effective.Overrides.DiskSpaceMB != nil {
		source = "admin_override"
	}

	effective.Storage = store.UserStorageQuotaDetails{
		QuotaBytes:   quotaBytes,
		UsedBytes:    usedBytes,
		FreeBytes:    freeBytes,
		UsagePercent: math.Round(usagePercent*10) / 10,
		IsUnlimited:  isUnlimited,
		Source:       source,
	}

	// Synchronize Usage.DiskUsedMB from accurate on-disk bytes
	if usedBytes > 0 {
		effective.Usage.DiskUsedMB = (usedBytes + (1024*1024 - 1)) / (1024 * 1024)
	} else {
		effective.Usage.DiskUsedMB = 0
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

	// Expired or suspended subscriptions cannot create new resources
	if effective.SubscriptionStatus == "expired" || effective.SubscriptionStatus == "suspended" || effective.SubscriptionStatus == "cancelled" {
		return fmt.Errorf("subscription is %s: please renew your package to create or manage resources", effective.SubscriptionStatus)
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
	case "storage", "disk":
		if effective.Storage.IsUnlimited {
			return nil
		}
		if effective.Storage.QuotaBytes > 0 && effective.Storage.UsedBytes >= effective.Storage.QuotaBytes {
			return fmt.Errorf("quota exceeded: your storage quota is full (%d MB / %d MB). Please upgrade your package or delete unnecessary files", effective.Storage.UsedBytes/(1024*1024), effective.Storage.QuotaBytes/(1024*1024))
		}
		return nil
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

	if effective.SubscriptionStatus == "expired" || effective.SubscriptionStatus == "suspended" || effective.SubscriptionStatus == "cancelled" {
		return false
	}

	allowed, ok := effective.Permissions[strings.ToLower(feature)]
	if !ok {
		return false
	}
	return allowed
}

// isAllowedUserStoragePath checks if a path is a safe user directory to measure and not a critical system path.
func isAllowedUserStoragePath(path string) bool {
	if path == "" {
		return false
	}
	clean := filepath.Clean(path)
	if clean == "/" || clean == "." {
		return false
	}

	// Strictly prohibited system roots
	prohibited := []string{
		"/etc", "/usr", "/boot", "/root", "/bin", "/sbin", "/lib", "/lib64",
		"/sys", "/proc", "/dev", "/opt", "/run", "/var/lib/docker",
	}
	for _, p := range prohibited {
		if clean == p || strings.HasPrefix(clean, p+"/") {
			return false
		}
	}

	// Allowed resource prefixes
	allowedPrefixes := []string{
		"/var/www",
		"/home",
		"/var/mail/vhosts",
		"/var/backups/hostvra",
		"/var/lib/hostvra",
	}
	for _, ap := range allowedPrefixes {
		if clean == ap || strings.HasPrefix(clean, ap+"/") {
			return true
		}
	}

	// If in a testing environment (e.g. /tmp, /var/folders), allow
	if strings.HasPrefix(clean, "/tmp") || strings.HasPrefix(clean, "/var/folders") {
		return true
	}

	return false
}

// calculateDirSizeBytes walks directory safely without following symlinks to prevent symlink escape attacks.
func calculateDirSizeBytes(dirPath string) int64 {
	cleanPath := filepath.Clean(dirPath)
	info, err := os.Lstat(cleanPath)
	if err != nil {
		return 0
	}
	// If symlink itself, do not measure
	if info.Mode()&os.ModeSymlink != 0 {
		return 0
	}
	if !info.IsDir() {
		return info.Size()
	}

	var totalBytes int64
	_ = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable entries gracefully
		}
		// Security: NEVER follow symlinks to avoid symlink escape attacks
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		fi, fErr := d.Info()
		if fErr == nil {
			totalBytes += fi.Size()
		}
		return nil
	})

	return totalBytes
}

// CalculateActualUserStorage calculates actual user-owned resources on filesystem with tenant isolation
func (s *Service) CalculateActualUserStorage(ctx context.Context, userID, orgID uuid.UUID, forceRefresh bool) int64 {
	if !forceRefresh {
		if val, ok := s.storageCache.Load(userID); ok {
			entry := val.(storageCacheEntry)
			if time.Since(entry.timestamp) < 45*time.Second {
				return entry.usedBytes
			}
		}
	}

	var totalBytes int64
	measuredPaths := make(map[string]bool)

	// 1. Websites document roots
	if sites, err := s.store.ListWebsitesByOrg(ctx, orgID); err == nil {
		for _, site := range sites {
			docRoot := filepath.Clean(site.DocumentRoot)
			if docRoot != "" && isAllowedUserStoragePath(docRoot) && !measuredPaths[docRoot] {
				measuredPaths[docRoot] = true
				totalBytes += calculateDirSizeBytes(docRoot)
			}
		}
	}

	// 2. Email Mailboxes (/var/mail/vhosts/<domain>)
	if domains, err := s.store.ListEmailDomainsByOrg(ctx, orgID); err == nil {
		for _, d := range domains {
			mailDir := filepath.Clean(filepath.Join("/var/mail/vhosts", d.Domain))
			if isAllowedUserStoragePath(mailDir) && !measuredPaths[mailDir] {
				measuredPaths[mailDir] = true
				totalBytes += calculateDirSizeBytes(mailDir)
			}
		}
	}

	// 3. Databases (actual recorded size in DB or database folder)
	if dbs, err := s.store.ListDatabasesByOrg(ctx, orgID); err == nil {
		for _, db := range dbs {
			if !db.InRecycleBin && db.SizeBytes > 0 {
				totalBytes += db.SizeBytes
			}
		}
	}

	// 4. Hosting Accounts (/home/<username>) if distinct from measured doc roots
	if accs, err := s.store.ListHostingAccounts(ctx, orgID, nil); err == nil {
		for _, acc := range accs {
			// Ensure tenant isolation: verify user or organization ownership
			if acc.UserID != userID && (acc.OrganizationID == uuid.Nil || acc.OrganizationID != orgID) {
				continue
			}
			homeDir := filepath.Clean(fmt.Sprintf("/home/%s", acc.Username))
			if isAllowedUserStoragePath(homeDir) && !measuredPaths[homeDir] {
				measuredPaths[homeDir] = true
				totalBytes += calculateDirSizeBytes(homeDir)
			}
		}
	}

	// 5. Backups (/var/backups/hostvra/<orgID>)
	if orgID != uuid.Nil {
		backupDir := filepath.Clean(fmt.Sprintf("/var/backups/hostvra/%s", orgID.String()))
		if isAllowedUserStoragePath(backupDir) && !measuredPaths[backupDir] {
			measuredPaths[backupDir] = true
			totalBytes += calculateDirSizeBytes(backupDir)
		}
	}

	// Cache the result
	s.storageCache.Store(userID, storageCacheEntry{
		usedBytes: totalBytes,
		timestamp: time.Now(),
	})

	return totalBytes
}

// getUsageUnchecked collects live resource counts
func (s *Service) getUsageUnchecked(ctx context.Context, userID, orgID uuid.UUID) store.UserResourceUsage {
	usage := store.UserResourceUsage{}

	// 1. Websites count
	if sites, err := s.store.ListWebsitesByOrg(ctx, orgID); err == nil {
		usage.WebsitesCount = len(sites)
	}

	// 2. Databases count
	if dbs, err := s.store.ListDatabasesByOrg(ctx, orgID); err == nil {
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

	// 4. Hosting Account usage if present (strictly tenant isolated)
	if accs, err := s.store.ListHostingAccounts(ctx, orgID, nil); err == nil {
		for _, a := range accs {
			if a.UserID != userID && (a.OrganizationID == uuid.Nil || a.OrganizationID != orgID) {
				continue
			}
			usage.BandwidthUsedMB += a.BandwidthUsedMB
			if usage.FTPCount == 0 {
				usage.FTPCount = 1
			}
		}
	}

	return usage
}

