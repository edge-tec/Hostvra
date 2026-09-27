package iputil

import (
	"context"
	"strings"
	"testing"
)

func TestIsPublicIPv4_RejectsLoopback(t *testing.T) {
	loopbacks := []string{
		"127.0.0.1",
		"127.0.1.1",
		"127.255.255.254",
		"localhost",
		"LOCALHOST",
	}

	for _, ip := range loopbacks {
		if IsPublicIPv4(ip) {
			t.Errorf("expected %q to be rejected as loopback, but was accepted", ip)
		}
		if err := ValidatePublicIPv4(ip); err == nil {
			t.Errorf("expected ValidatePublicIPv4(%q) to return error, got nil", ip)
		}
	}
}

func TestIsPublicIPv4_RejectsUnspecified(t *testing.T) {
	unspecified := []string{
		"0.0.0.0",
		"0.0.0.1",
		"",
		"   ",
	}

	for _, ip := range unspecified {
		if IsPublicIPv4(ip) {
			t.Errorf("expected %q to be rejected as unspecified, but was accepted", ip)
		}
		if err := ValidatePublicIPv4(ip); err == nil {
			t.Errorf("expected ValidatePublicIPv4(%q) to return error, got nil", ip)
		}
	}
}

func TestIsPublicIPv4_RejectsPrivateRFC1918(t *testing.T) {
	privates := []string{
		// 10.0.0.0/8
		"10.0.0.1",
		"10.255.255.254",
		"10.50.100.200",
		// 172.16.0.0/12
		"172.16.0.1",
		"172.24.10.20",
		"172.31.255.254",
		// 192.168.0.0/16
		"192.168.0.1",
		"192.168.1.100",
		"192.168.254.254",
	}

	for _, ip := range privates {
		if IsPublicIPv4(ip) {
			t.Errorf("expected %q to be rejected as RFC1918 private, but was accepted", ip)
		}
		if err := ValidatePublicIPv4(ip); err == nil {
			t.Errorf("expected ValidatePublicIPv4(%q) to return error, got nil", ip)
		}
	}
}

func TestIsPublicIPv4_RejectsLinkLocalAndCGNAT(t *testing.T) {
	reserved := []string{
		"169.254.1.1",     // Link-local
		"169.254.254.254", // Link-local
		"100.64.0.1",      // CGNAT
		"100.127.255.254", // CGNAT
		"224.0.0.1",       // Multicast
		"255.255.255.255", // Broadcast
		"240.0.0.1",       // Reserved
	}

	for _, ip := range reserved {
		if IsPublicIPv4(ip) {
			t.Errorf("expected %q to be rejected as reserved/CGNAT/link-local, but was accepted", ip)
		}
		if err := ValidatePublicIPv4(ip); err == nil {
			t.Errorf("expected ValidatePublicIPv4(%q) to return error, got nil", ip)
		}
	}
}

func TestIsPublicIPv4_AcceptsValidPublicIPs(t *testing.T) {
	publics := []string{
		"185.193.17.42",
		"109.123.45.67",
		"142.250.190.46",
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34",
		"151.101.1.140",
		"199.232.69.194",
		"52.95.110.1",
	}

	for _, ip := range publics {
		if !IsPublicIPv4(ip) {
			t.Errorf("expected %q to be accepted as valid public IPv4, but was rejected", ip)
		}
		if err := ValidatePublicIPv4(ip); err != nil {
			t.Errorf("expected ValidatePublicIPv4(%q) to succeed, got error: %v", ip, err)
		}
	}
}

func TestValidatePublicIPv4_DescriptiveErrors(t *testing.T) {
	tests := []struct {
		ip       string
		errSubstr string
	}{
		{"", "cannot be empty"},
		{"localhost", "loopback hostname"},
		{"127.0.0.1", "loopback address"},
		{"0.0.0.0", "unspecified address"},
		{"192.168.1.1", "private RFC1918"},
		{"10.0.0.5", "private RFC1918"},
		{"172.20.0.1", "private RFC1918"},
		{"169.254.10.20", "link-local address"},
		{"not-an-ip", "not a valid IPv4 address"},
		{"2001:db8::1", "IPv6"},
	}

	for _, tc := range tests {
		err := ValidatePublicIPv4(tc.ip)
		if err == nil {
			t.Errorf("expected error for %q, got nil", tc.ip)
			continue
		}
		if !strings.Contains(err.Error(), tc.errSubstr) {
			t.Errorf("for IP %q, expected error containing %q, got %q", tc.ip, tc.errSubstr, err.Error())
		}
	}
}

func TestDetectPublicIPv4_LocalOrDiscovery(t *testing.T) {
	ctx := context.Background()
	ip, err := DetectPublicIPv4(ctx)
	if err == nil {
		if !IsPublicIPv4(ip) {
			t.Errorf("DetectPublicIPv4 returned non-public IP: %q", ip)
		}
	} else {
		t.Logf("DetectPublicIPv4 returned expected error in offline/NAT testing environment: %v", err)
	}
}
