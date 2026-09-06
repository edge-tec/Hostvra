package osadapter

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func Detect() (OSAdapter, error) {
	arch := runtime.GOARCH
	kernel := detectKernel()

	// 1. Try parsing /etc/os-release (Standard Linux)
	if f, err := os.Open("/etc/os-release"); err == nil {
		defer f.Close()

		data := make(map[string]string)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				val := strings.Trim(parts[1], `"'`)
				data[key] = val
			}
		}

		id := strings.ToLower(data["ID"])
		idLike := strings.ToLower(data["ID_LIKE"])
		name := data["NAME"]
		version := data["VERSION_ID"]

		if name == "" {
			name = id
		}

		// Debian / Ubuntu family
		if id == "ubuntu" || id == "debian" || strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
			return NewDebianAdapter(name, version, arch, kernel), nil
		}

		// RHEL / Alma / Rocky family
		if id == "almalinux" || id == "rocky" || id == "rhel" || id == "centos" || id == "fedora" || strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
			return NewRHELAdapter(name, version, arch, kernel), nil
		}

		// Default to Debian-compatible if unknown Linux
		return NewDebianAdapter(name, version, arch, kernel), nil
	}

	// 2. Development Fallback (e.g. Darwin / macOS development)
	if runtime.GOOS == "darwin" {
		return &DarwinDevAdapter{
			BaseOSInfo: BaseOSInfo{
				OSName:     "macOS",
				OSVersion:  "Darwin Dev",
				OSFamily:   "darwin",
				Arch:       arch,
				Kernel:     kernel,
				PkgManager: "brew",
			},
		}, nil
	}

	return nil, ErrUnsupportedOS
}

func detectKernel() string {
	if b, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		return strings.TrimSpace(string(b))
	}

	out, err := exec.Command("uname", "-r").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}

	return "unknown"
}

// DarwinDevAdapter is a development dummy adapter for macOS testing
type DarwinDevAdapter struct {
	BaseOSInfo
}

func (d *DarwinDevAdapter) ServiceAction(service string, action ServiceAction) error {
	return nil
}

func (d *DarwinDevAdapter) IsServiceRunning(service string) (bool, error) {
	return true, nil
}

func (d *DarwinDevAdapter) GetConfigPath(service string) string {
	return "/usr/local/etc"
}
