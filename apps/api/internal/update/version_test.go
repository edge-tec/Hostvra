package update

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSemanticVersionParsing(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
		major       int
		minor       int
		patch       int
		preRelease  string
	}{
		{"1.0.0", false, 1, 0, 0, ""},
		{"v1.2.3", false, 1, 2, 3, ""},
		{"2.10.5-beta.1", false, 2, 10, 5, "beta.1"},
		{"0.9.1+build.123", false, 0, 9, 1, ""},
		{"invalid", true, 0, 0, 0, ""},
		{"1.0", true, 0, 0, 0, ""},
		{"", true, 0, 0, 0, ""},
	}

	for _, tt := range tests {
		v, err := ParseVersion(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("unexpected error for input %q: %v", tt.input, err)
			continue
		}

		if v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch {
			t.Errorf("for input %q, expected %d.%d.%d, got %d.%d.%d",
				tt.input, tt.major, tt.minor, tt.patch, v.Major, v.Minor, v.Patch)
		}

		if v.PreRelease != tt.preRelease {
			t.Errorf("for input %q, expected preRelease %q, got %q", tt.input, tt.preRelease, v.PreRelease)
		}
	}
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int // -1 (v1 < v2), 0 (v1 == v2), 1 (v1 > v2)
	}{
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0", "1.0.0", 0},
		{"1.2.0-beta.1", "1.2.0", -1}, // normal version has precedence over pre-release
		{"1.2.0", "1.2.0-beta.1", 1},
	}

	for _, tt := range tests {
		ver1, err1 := ParseVersion(tt.v1)
		ver2, err2 := ParseVersion(tt.v2)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected parse error: %v, %v", err1, err2)
		}

		cmp := ver1.Compare(ver2)
		if cmp != tt.expected {
			t.Errorf("expected %s.Compare(%s) = %d, got %d", tt.v1, tt.v2, tt.expected, cmp)
		}
	}
}

func TestUpgradeAndDowngradeDetection(t *testing.T) {
	// Upgrade: 1.0.0 -> 1.1.0
	isUpgrade, err := IsUpgradeFrom("1.0.0", "1.1.0")
	if err != nil || !isUpgrade {
		t.Errorf("expected 1.0.0 -> 1.1.0 to be recognized as upgrade, got %v (err: %v)", isUpgrade, err)
	}

	// Downgrade: 1.2.0 -> 1.1.0
	isUpgrade, err = IsUpgradeFrom("1.2.0", "1.1.0")
	if err != nil || isUpgrade {
		t.Errorf("expected 1.2.0 -> 1.1.0 to NOT be recognized as upgrade, got %v", isUpgrade)
	}
}

func TestCompatibilityEvaluation(t *testing.T) {
	rel := &ReleaseMetadata{
		ID:                  uuid.New(),
		Version:             "1.2.0",
		Channel:             ChannelStable,
		MinSupportedVersion: "1.0.0",
		ArchCompatibility:   []string{"amd64", "arm64"},
		OSCompatibility:     []string{"ubuntu", "debian"},
		ReleasedAt:          time.Now().UTC(),
	}

	// 1. Compatible upgrade from 1.0.0 on Ubuntu amd64
	report, err := EvaluateCompatibility("1.0.0", "ubuntu-22.04", "amd64", rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.IsCompatible || !report.IsUpgrade || report.IsDowngrade {
		t.Errorf("expected compatible upgrade, got %+v", report)
	}
	if len(report.Blockers) > 0 {
		t.Errorf("expected no blockers, got %v", report.Blockers)
	}

	// 2. Incompatible Downgrade attempt: 1.3.0 -> 1.2.0
	reportDowngrade, err := EvaluateCompatibility("1.3.0", "ubuntu-22.04", "amd64", rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reportDowngrade.IsCompatible {
		t.Errorf("downgrade should NOT be compatible")
	}
	if !reportDowngrade.IsDowngrade {
		t.Errorf("expected downgrade to be flagged")
	}

	// 3. Incompatible Architecture: riscv64
	reportArch, err := EvaluateCompatibility("1.0.0", "ubuntu-22.04", "riscv64", rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reportArch.IsCompatible {
		t.Errorf("unsupported arch should NOT be compatible")
	}

	// 4. Incompatible Min Supported Version
	relHighMin := &ReleaseMetadata{
		Version:             "2.0.0",
		MinSupportedVersion: "1.5.0",
		ArchCompatibility:   []string{"amd64"},
		ReleasedAt:          time.Now().UTC(),
	}
	reportMinVer, err := EvaluateCompatibility("1.0.0", "ubuntu-22.04", "amd64", relHighMin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reportMinVer.IsCompatible {
		t.Errorf("version below min_supported_version should be blocked")
	}
}

func TestSystemVersionInfo(t *testing.T) {
	svc := NewReleaseService("1.0.0", "1.0.0", 4)
	ctx := context.Background()

	// When no update is available
	info := svc.GetSystemVersionInfo(ctx, nil)
	if info.UpdateAvailable {
		t.Errorf("expected no update available when release is nil")
	}
	if info.APIVersion != "1.0.0" || info.DBSchemaVersion != 4 {
		t.Errorf("unexpected version info: %+v", info)
	}

	// When newer release exists
	newerRel := &ReleaseMetadata{
		Version:    "1.1.0",
		Channel:    ChannelStable,
		PackageURL: "https://updates.hostvra.com/releases/1.1.0.tar.gz",
	}
	infoUpdate := svc.GetSystemVersionInfo(ctx, newerRel)
	if !infoUpdate.UpdateAvailable {
		t.Errorf("expected update available for 1.1.0")
	}
	if infoUpdate.LatestVersion != "1.1.0" {
		t.Errorf("expected latest version 1.1.0, got %s", infoUpdate.LatestVersion)
	}
}
