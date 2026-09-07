package firewall

import (
	"testing"
)

func TestFirewall_SSHLockoutProtection(t *testing.T) {
	fm := NewFirewallManager()

	// Denying port 22 must fail immediately with ErrSSHLockoutBlocked
	err := fm.AddRule("22", "tcp", "", "deny", "test block ssh")
	if err != ErrSSHLockoutBlocked {
		t.Fatalf("expected ErrSSHLockoutBlocked, got: %v", err)
	}

	// Allowing port 22 must NOT trigger ErrSSHLockoutBlocked
	errAllow := fm.AddRule("22", "tcp", "", "allow", "allow ssh")
	if errAllow == ErrSSHLockoutBlocked {
		t.Fatalf("allowing port 22 should not trigger lockout error")
	}
}

func TestFirewall_PortValidation(t *testing.T) {
	validPorts := []string{"80", "443", "8080", "1", "65535", "3000:4000", "80:90"}
	for _, p := range validPorts {
		if err := validatePortSpec(p); err != nil {
			t.Errorf("expected port %q to be valid, got: %v", p, err)
		}
	}

	invalidPorts := []string{"0", "-1", "65536", "70000", "abc", "5000:4000", "1:2:3", "80:", ":80"}
	for _, p := range invalidPorts {
		if err := validatePortSpec(p); err == nil {
			t.Errorf("expected port %q to be invalid, but validation passed", p)
		}
	}
}

func TestFirewall_ParseUFWNumberedRules(t *testing.T) {
	sampleOutput := `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere                   # SSH Management
[ 2] 80/tcp                     ALLOW IN    Anywhere                   # HTTP
[ 3] 443/tcp                    ALLOW IN    Anywhere                   # HTTPS
[ 4] 3306                       DENY IN     192.168.1.100              # MySQL Block
[ 5] 22/tcp (v6)                ALLOW IN    Anywhere (v6)
`

	rules := parseUFWNumberedRules(sampleOutput)
	if len(rules) < 4 {
		t.Fatalf("expected at least 4 parsed rules, got %d", len(rules))
	}

	r1 := rules[0]
	if r1.Number != 1 || r1.To != "22/tcp" || r1.Action != "ALLOW IN" || r1.Comment != "SSH Management" {
		t.Errorf("unexpected rule 1: %+v", r1)
	}

	r4 := rules[3]
	if r4.Number != 4 || r4.To != "3306" || r4.Action != "DENY IN" || r4.From != "192.168.1.100" || r4.Comment != "MySQL Block" {
		t.Errorf("unexpected rule 4: %+v", r4)
	}
}

func TestFirewall_ParseJailStatus(t *testing.T) {
	statusOutput := "Status for the jail: sshd\n" +
		"|- Filter\n" +
		"|  |- Currently failed: 3\n" +
		"|  |- Total failed:     25\n" +
		"|  `- File list:        /var/log/auth.log\n" +
		"`- Actions\n" +
		"   |- Currently banned: 2\n" +
		"   |- Total banned:     10\n" +
		"   `- Banned IP list:   198.51.100.1 203.0.113.5\n"

	info := parseJailStatus("sshd", statusOutput)
	if info.Name != "sshd" {
		t.Errorf("expected jail name sshd, got %s", info.Name)
	}
	if info.CurrentlyFailed != 3 {
		t.Errorf("expected currently failed 3, got %d", info.CurrentlyFailed)
	}
	if info.TotalFailed != 25 {
		t.Errorf("expected total failed 25, got %d", info.TotalFailed)
	}
	if info.CurrentlyBanned != 2 {
		t.Errorf("expected currently banned 2, got %d", info.CurrentlyBanned)
	}
	if info.TotalBanned != 10 {
		t.Errorf("expected total banned 10, got %d", info.TotalBanned)
	}
	if len(info.BannedIPs) != 2 || info.BannedIPs[0] != "198.51.100.1" || info.BannedIPs[1] != "203.0.113.5" {
		t.Errorf("unexpected banned IPs: %v", info.BannedIPs)
	}
}
