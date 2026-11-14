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

func TestFirewall_PingStatus(t *testing.T) {
	fm := NewFirewallManager()

	if err := fm.SetPingStatus(true); err != nil {
		t.Fatalf("failed to set ping prohibit: %v", err)
	}
	prohibited, err := fm.GetPingStatus()
	if err != nil {
		t.Fatalf("failed to get ping status: %v", err)
	}
	if !prohibited {
		t.Errorf("expected ping to be prohibited, got false")
	}

	if err := fm.SetPingStatus(false); err != nil {
		t.Fatalf("failed to set ping allow: %v", err)
	}
	prohibited2, err := fm.GetPingStatus()
	if err != nil {
		t.Fatalf("failed to get ping status: %v", err)
	}
	if prohibited2 {
		t.Errorf("expected ping to be allowed (false), got true")
	}
}

func TestFirewall_ParseSSOutput(t *testing.T) {
	sample := `tcp   LISTEN 0      128          0.0.0.0:22        0.0.0.0:*    users:(("sshd",pid=762,fd=3))
tcp   LISTEN 0      511          0.0.0.0:80        0.0.0.0:*    users:(("nginx",pid=1200,fd=6))
tcp   LISTEN 0      70         127.0.0.1:3306      0.0.0.0:*    users:(("mysqld",pid=950,fd=21))
udp   UNCONN 0      0            0.0.0.0:53        0.0.0.0:*    users:(("named",pid=600,fd=512))
`
	ports := parseListeningOutput(sample)
	if len(ports) != 4 {
		t.Fatalf("expected 4 parsed ports, got %d", len(ports))
	}

	if ports[0].Port != 22 || ports[0].Process != "sshd" || ports[0].PID != 762 {
		t.Errorf("unexpected port 0: %+v", ports[0])
	}
	if ports[1].Port != 80 || ports[1].Process != "nginx" {
		t.Errorf("unexpected port 1: %+v", ports[1])
	}
	if ports[2].Port != 3306 || ports[2].Process != "mysql" && ports[2].Process != "mysqld" {
		t.Errorf("unexpected port 2: %+v", ports[2])
	}
	if ports[3].Port != 53 || ports[3].Protocol != "udp" {
		t.Errorf("unexpected port 3: %+v", ports[3])
	}
}

func TestFirewall_IsPortCoveredInRules(t *testing.T) {
	rules := []FirewallRule{
		{To: "22/tcp", Action: "ALLOW IN"},
		{To: "80", Action: "ALLOW IN"},
		{To: "8000:9000/tcp", Action: "ALLOW IN"},
		{To: "3306", Action: "DENY IN"},
	}

	if !isPortCoveredInRules(22, "tcp", rules) {
		t.Errorf("expected port 22 to be covered in rules")
	}
	if !isPortCoveredInRules(80, "tcp", rules) {
		t.Errorf("expected port 80 to be covered in rules")
	}
	if !isPortCoveredInRules(8500, "tcp", rules) {
		t.Errorf("expected port 8500 to be covered in range 8000:9000")
	}
	if isPortCoveredInRules(3306, "tcp", rules) {
		t.Errorf("port 3306 has DENY rule, should not be covered as open")
	}
	if isPortCoveredInRules(9999, "tcp", rules) {
		t.Errorf("port 9999 is not in rules, should not be covered")
	}
}
