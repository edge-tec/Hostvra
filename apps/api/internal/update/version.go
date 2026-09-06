package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Channel defines the release stability stream
type Channel string

const (
	ChannelStable  Channel = "stable"
	ChannelBeta    Channel = "beta"
	ChannelNightly Channel = "nightly"
)

// Version represents a parsed Semantic Version (SemVer 2.0.0)
type Version struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	PreRelease string `json:"pre_release,omitempty"`
	Build      string `json:"build,omitempty"`
	Raw        string `json:"raw"`
}

var semverRegex = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

// ParseVersion parses a semver string into a Version struct
func ParseVersion(vStr string) (*Version, error) {
	vStr = strings.TrimSpace(vStr)
	matches := semverRegex.FindStringSubmatch(vStr)
	if len(matches) < 4 {
		return nil, fmt.Errorf("invalid semantic version string: %q", vStr)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}

	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %w", err)
	}

	var preRelease, build string
	if len(matches) > 4 {
		preRelease = matches[4]
	}
	if len(matches) > 5 {
		build = matches[5]
	}

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		PreRelease: preRelease,
		Build:      build,
		Raw:        vStr,
	}, nil
}

// Compare returns:
//
//	-1 if v < other
//	 0 if v == other
//	 1 if v > other
func (v *Version) Compare(other *Version) int {
	if other == nil {
		return 1
	}

	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}

	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}

	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}

	// Pre-release versions have lower precedence than normal versions
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && other.PreRelease == "" {
		return -1
	}
	if v.PreRelease != "" && other.PreRelease != "" {
		if v.PreRelease > other.PreRelease {
			return 1
		} else if v.PreRelease < other.PreRelease {
			return -1
		}
	}

	return 0
}

func (v *Version) IsGreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

func (v *Version) IsLessThan(other *Version) bool {
	return v.Compare(other) < 0
}

func (v *Version) Equals(other *Version) bool {
	return v.Compare(other) == 0
}

// String returns formatted semver string
func (v *Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		base += "-" + v.PreRelease
	}
	if v.Build != "" {
		base += "+" + v.Build
	}
	return base
}

// IsUpgradeFrom checks if candidate target is a valid newer upgrade than current
func IsUpgradeFrom(currentStr, targetStr string) (bool, error) {
	curr, err := ParseVersion(currentStr)
	if err != nil {
		return false, fmt.Errorf("invalid current version: %w", err)
	}
	target, err := ParseVersion(targetStr)
	if err != nil {
		return false, fmt.Errorf("invalid target version: %w", err)
	}
	return target.IsGreaterThan(curr), nil
}

// CheckCompatibility verifies if currentVersion satisfies minimum supported upgrade version
func CheckCompatibility(currentVersion, minSupportedVersion string) (bool, error) {
	curr, err := ParseVersion(currentVersion)
	if err != nil {
		return false, fmt.Errorf("invalid current version: %w", err)
	}
	minVer, err := ParseVersion(minSupportedVersion)
	if err != nil {
		return false, fmt.Errorf("invalid minimum supported version: %w", err)
	}

	// Current version must be >= minSupportedVersion
	return curr.Compare(minVer) >= 0, nil
}
