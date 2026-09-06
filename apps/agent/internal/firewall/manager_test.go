package firewall

import (
	"testing"
)

func TestSSHLockoutProtection(t *testing.T) {
	fm := NewFirewallManager()

	// Denying port 22 without override must fail
	err := fm.DenyPort("22", "tcp", false)
	if err == nil {
		t.Fatal("Expected DenyPort('22') without override to be rejected, but it succeeded")
	}

	// Deleting port 22 must fail
	err = fm.DeleteRule("22", "tcp")
	if err == nil {
		t.Fatal("Expected DeleteRule('22') to be rejected")
	}

	// Denying standard port 8080 should not be blocked by SSH lockout
	err = fm.DenyPort("8080", "tcp", false)
	if err != nil && err == ErrSSHLockoutBlocked {
		t.Fatal("Did not expect ErrSSHLockoutBlocked for non-SSH port")
	}
}
