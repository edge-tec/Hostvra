package update

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ReleaseMetadata holds the complete information of an available Hostvra release
type ReleaseMetadata struct {
	ID                  uuid.UUID `json:"id"`
	Version             string    `json:"version"`
	Channel             Channel   `json:"channel"`
	Component           string    `json:"component"` // "bundle", "api", "agent", "web"
	ReleaseNotes        string    `json:"release_notes"`
	MinSupportedVersion string    `json:"min_supported_version"`
	PackageURL          string    `json:"package_url"`
	SHA256Checksum      string    `json:"sha256_checksum"`
	Ed25519Signature    string    `json:"ed25519_signature"`
	PackageSizeBytes    int64     `json:"package_size_bytes"`
	OSCompatibility     []string  `json:"os_compatibility"`
	ArchCompatibility   []string  `json:"arch_compatibility"`
	RequiresDBMigration bool      `json:"requires_db_migration"`
	RequiresReboot      bool      `json:"requires_reboot"`
	IsRevoked           bool      `json:"is_revoked"`
	ReleasedAt          time.Time `json:"released_at"`
}

// CompatibilityReport details whether a release can be safely installed
type CompatibilityReport struct {
	IsCompatible        bool     `json:"is_compatible"`
	IsUpgrade           bool     `json:"is_upgrade"`
	IsDowngrade         bool     `json:"is_downgrade"`
	CurrentVersion      string   `json:"current_version"`
	TargetVersion       string   `json:"target_version"`
	MinSupportedVersion string   `json:"min_supported_version"`
	HostOS              string   `json:"host_os"`
	HostArch            string   `json:"host_arch"`
	Blockers            []string `json:"blockers,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

// EvaluateCompatibility tests target release against current runtime environment
func EvaluateCompatibility(currentVersion string, hostOS string, hostArch string, release *ReleaseMetadata) (*CompatibilityReport, error) {
	report := &CompatibilityReport{
		CurrentVersion:      currentVersion,
		TargetVersion:       release.Version,
		MinSupportedVersion: release.MinSupportedVersion,
		HostOS:              hostOS,
		HostArch:            hostArch,
		Blockers:            make([]string, 0),
		Warnings:            make([]string, 0),
	}

	if hostArch == "" {
		hostArch = runtime.GOARCH
	}

	currVer, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current version: %w", err)
	}

	targetVer, err := ParseVersion(release.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target release version: %w", err)
	}

	// 1. Revocation check
	if release.IsRevoked {
		report.Blockers = append(report.Blockers, fmt.Sprintf("Release %s has been revoked by security policy", release.Version))
	}

	// 2. Upgrade vs Downgrade check
	cmp := targetVer.Compare(currVer)
	if cmp > 0 {
		report.IsUpgrade = true
	} else if cmp < 0 {
		report.IsDowngrade = true
		report.Blockers = append(report.Blockers, fmt.Sprintf("Downgrade from %s to %s is strictly prohibited to prevent database and configuration corruption", currentVersion, release.Version))
	} else {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Target version %s is identical to current active version", release.Version))
	}

	// 3. Minimum Supported Version Check
	if release.MinSupportedVersion != "" {
		minVer, err := ParseVersion(release.MinSupportedVersion)
		if err == nil {
			if currVer.Compare(minVer) < 0 {
				report.Blockers = append(report.Blockers, fmt.Sprintf("Current version %s is below minimum required version %s for this upgrade. Stepwise upgrade required.", currentVersion, release.MinSupportedVersion))
			}
		}
	}

	// 4. Architecture Compatibility
	if len(release.ArchCompatibility) > 0 {
		archMatch := false
		for _, a := range release.ArchCompatibility {
			if strings.EqualFold(a, hostArch) {
				archMatch = true
				break
			}
		}
		if !archMatch {
			report.Blockers = append(report.Blockers, fmt.Sprintf("Host architecture %q is not supported by release %s (supported: %s)", hostArch, release.Version, strings.Join(release.ArchCompatibility, ", ")))
		}
	}

	// 5. Operating System Compatibility
	if hostOS != "" && len(release.OSCompatibility) > 0 {
		osMatch := false
		for _, osName := range release.OSCompatibility {
			if strings.Contains(strings.ToLower(hostOS), strings.ToLower(osName)) {
				osMatch = true
				break
			}
		}
		if !osMatch {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Host OS %q may not be officially certified for release %s", hostOS, release.Version))
		}
	}

	report.IsCompatible = len(report.Blockers) == 0
	return report, nil
}

// SystemVersionInfo represents active component versions on a server
type SystemVersionInfo struct {
	APIVersion       string  `json:"api_version"`
	AgentVersion     string  `json:"agent_version"`
	WebVersion       string  `json:"web_version"`
	DBSchemaVersion  int     `json:"db_schema_version"`
	Channel          Channel `json:"channel"`
	UpdateAvailable  bool    `json:"update_available"`
	LatestVersion    string  `json:"latest_version,omitempty"`
	LatestReleaseURL string  `json:"latest_release_url,omitempty"`
}

// ReleaseService provides version query and compatibility operations
type ReleaseService struct {
	currentAPIVersion   string
	currentAgentVersion string
	currentDBSchemaVer  int
}

// NewReleaseService creates a new ReleaseService
func NewReleaseService(apiVersion, agentVersion string, dbSchemaVer int) *ReleaseService {
	return &ReleaseService{
		currentAPIVersion:   apiVersion,
		currentAgentVersion: agentVersion,
		currentDBSchemaVer:  dbSchemaVer,
	}
}

// GetSystemVersionInfo returns full component version matrix
func (s *ReleaseService) GetSystemVersionInfo(ctx context.Context, latestRelease *ReleaseMetadata) *SystemVersionInfo {
	info := &SystemVersionInfo{
		APIVersion:      s.currentAPIVersion,
		AgentVersion:    s.currentAgentVersion,
		WebVersion:      s.currentAPIVersion,
		DBSchemaVersion: s.currentDBSchemaVer,
		Channel:         ChannelStable,
	}

	if latestRelease != nil {
		info.Channel = latestRelease.Channel
		currVer, _ := ParseVersion(s.currentAPIVersion)
		targetVer, _ := ParseVersion(latestRelease.Version)
		if currVer != nil && targetVer != nil && targetVer.IsGreaterThan(currVer) {
			info.UpdateAvailable = true
			info.LatestVersion = latestRelease.Version
			info.LatestReleaseURL = latestRelease.PackageURL
		}
	}

	return info
}
