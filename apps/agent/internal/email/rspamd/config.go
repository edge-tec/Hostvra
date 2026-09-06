package rspamd

import (
	"fmt"
	"os"
	"path/filepath"
)

type Options struct {
	DKIMBaseDir      string  // default "/var/lib/hostvra/dkim"
	SpamAddHeader    float64 // default 6.0
	SpamGreylist     float64 // default 4.0
	SpamReject       float64 // default 15.0
	EnableClamAV     bool    // toggle based on host RAM
	ClamAVSocket     string  // default "tcp:127.0.0.1:3310"
	TotalHostRAMMB   int64
}

// GenerateActionsConf generates local.d/actions.conf for spam scoring
func GenerateActionsConf(opts Options) string {
	if opts.SpamAddHeader == 0 {
		opts.SpamAddHeader = 6.0
	}
	if opts.SpamGreylist == 0 {
		opts.SpamGreylist = 4.0
	}
	if opts.SpamReject == 0 {
		opts.SpamReject = 15.0
	}

	return fmt.Sprintf(`# Hostvra Rspamd Actions Configuration
# local.d/actions.conf
reject = %.1f;
add_header = %.1f;
greylist = %.1f;
`, opts.SpamReject, opts.SpamAddHeader, opts.SpamGreylist)
}

// GenerateDKIMSigningConf configures Rspamd dkim_signing module
func GenerateDKIMSigningConf(opts Options) string {
	if opts.DKIMBaseDir == "" {
		opts.DKIMBaseDir = "/var/lib/hostvra/dkim"
	}

	return fmt.Sprintf(`# Hostvra Rspamd DKIM Signing
# local.d/dkim_signing.conf
allow_username_mismatch = true;
path = "%s/$domain/$selector.private";
selector_map = "%s/selectors.map";
sign_local = true;
`, opts.DKIMBaseDir, opts.DKIMBaseDir)
}

// GenerateAntivirusConf generates local.d/antivirus.conf with RAM safeguard
func GenerateAntivirusConf(opts Options) string {
	// Guard: If host RAM is below 2GB and ClamAV is requested, disable to prevent OOM panic
	if opts.TotalHostRAMMB > 0 && opts.TotalHostRAMMB < 2048 {
		opts.EnableClamAV = false
	}

	if !opts.EnableClamAV {
		return `# Hostvra Rspamd Antivirus (Disabled due to low RAM or toggle)
# local.d/antivirus.conf
clamav {
  enabled = false;
}
`
	}

	sock := opts.ClamAVSocket
	if sock == "" {
		sock = "tcp:127.0.0.1:3310"
	}

	return fmt.Sprintf(`# Hostvra Rspamd Antivirus (ClamAV)
# local.d/antivirus.conf
clamav {
  attachments_only = false;
  servers = "%s";
  symbol = "CLAM_VIRUS";
  type = "clamav";
  patterns {
    JUST_EICAR = "^Eicar-Test-Signature$";
  }
}
`, sock)
}

// GenerateMilterHeadersConf ensures X-Spam headers are added
func GenerateMilterHeadersConf() string {
	return `# Hostvra Rspamd Milter Headers
# local.d/milter_headers.conf
use = ["x-spamd-bar", "x-spam-level", "x-spam-status", "authentication-results"];
routines {
  authentication-results {
    header = "Authentication-Results";
    remove = 1;
  }
}
`
}

// GenerateClassifierBayesConf sets up bayes learning
func GenerateClassifierBayesConf() string {
	return `# Hostvra Rspamd Bayes Classifier
# local.d/classifier-bayes.conf
backend = "sqlite3";
autolearn = [-0.5, 6.0];
`
}

// ApplyRspamdConfig writes all Rspamd local configs
func ApplyRspamdConfig(configDir string, opts Options) error {
	if configDir == "" {
		configDir = "/etc/rspamd/local.d"
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create rspamd config directory: %w", err)
	}

	configs := map[string]string{
		"actions.conf":          GenerateActionsConf(opts),
		"dkim_signing.conf":     GenerateDKIMSigningConf(opts),
		"antivirus.conf":        GenerateAntivirusConf(opts),
		"milter_headers.conf":   GenerateMilterHeadersConf(),
		"classifier-bayes.conf": GenerateClassifierBayesConf(),
	}

	for fname, content := range configs {
		path := filepath.Join(configDir, fname)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fname, err)
		}
	}

	return nil
}
