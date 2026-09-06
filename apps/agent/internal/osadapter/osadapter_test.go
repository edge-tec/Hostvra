package osadapter

import (
	"testing"
)

func TestDebianAdapter(t *testing.T) {
	deb := NewDebianAdapter("Ubuntu", "24.04 LTS", "x86_64", "6.8.0-generic")
	if deb.Family() != "debian" {
		t.Errorf("Expected family debian, got %s", deb.Family())
	}
	if deb.PackageManager() != "apt" {
		t.Errorf("Expected package manager apt, got %s", deb.PackageManager())
	}
	if deb.GetConfigPath("nginx") != "/etc/nginx/sites-available" {
		t.Errorf("Expected /etc/nginx/sites-available, got %s", deb.GetConfigPath("nginx"))
	}
}

func TestRHELAdapter(t *testing.T) {
	rhel := NewRHELAdapter("AlmaLinux", "9.4", "x86_64", "5.14.0-el9")
	if rhel.Family() != "rhel" {
		t.Errorf("Expected family rhel, got %s", rhel.Family())
	}
	if rhel.PackageManager() != "dnf" {
		t.Errorf("Expected package manager dnf, got %s", rhel.PackageManager())
	}
	if rhel.GetConfigPath("nginx") != "/etc/nginx/conf.d" {
		t.Errorf("Expected /etc/nginx/conf.d, got %s", rhel.GetConfigPath("nginx"))
	}
}

func TestDetect(t *testing.T) {
	adapter, err := Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if adapter.Name() == "" {
		t.Error("Expected OS name to be detected")
	}
	if adapter.Architecture() == "" {
		t.Error("Expected architecture to be detected")
	}
}
