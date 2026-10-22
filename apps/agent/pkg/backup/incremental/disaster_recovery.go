package incremental

import (
	"fmt"
	"time"
)

// RunbookRequest holds the parameters for generating a disaster recovery runbook
type RunbookRequest struct {
	OrganizationName   string   `json:"organization_name"`
	TargetHost         string   `json:"target_host"`
	RepositoryProvider string   `json:"repository_provider"`
	RepositoryURI      string   `json:"repository_uri"`
	EncryptionMode     string   `json:"encryption_mode"`
	LatestSnapshotID   string   `json:"latest_snapshot_id"`
	Websites           []string `json:"websites"`
	Databases          []string `json:"databases"`
	MailDomains        []string `json:"mail_domains"`
}

// RunbookStep represents a single actionable procedure
type RunbookStep struct {
	StepNumber  int      `json:"step_number"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	SafetyNote  string   `json:"safety_note,omitempty"`
}

// RunbookDocument represents the full disaster recovery documentation
type RunbookDocument struct {
	Title       string        `json:"title"`
	GeneratedAt time.Time     `json:"generated_at"`
	Summary     string        `json:"summary"`
	Steps       []RunbookStep `json:"steps"`
	Markdown    string        `json:"markdown"`
}

// GenerateRunbook produces a comprehensive, actionable Disaster Recovery Runbook
func GenerateRunbook(req RunbookRequest) *RunbookDocument {
	now := time.Now().UTC()

	steps := []RunbookStep{
		{
			StepNumber:  1,
			Title:       "Target Host Provisioning & Base Installation",
			Description: "Provision a clean Ubuntu 22.04/24.04 or Debian 12 server. Ensure base web server (Nginx/Apache), PHP-FPM, Database (MySQL/MariaDB/PostgreSQL), and Mail (Dovecot/Postfix) packages are installed.",
			Commands: []string{
				"curl -fsSL https://get.hostvra.com/install.sh | bash",
				"systemctl status hostvra-agent",
			},
			SafetyNote: "Always test restoration in an isolated sandbox environment first to avoid DNS flapping.",
		},
		{
			StepNumber:  2,
			Title:       "Authenticate & Verify Off-Site Backup Repository",
			Description: fmt.Sprintf("Connect to the remote %s repository and verify cryptographic seal integrity before unpacking.", req.RepositoryProvider),
			Commands: []string{
				fmt.Sprintf("export RESTIC_REPOSITORY='%s'", req.RepositoryURI),
				"export RESTIC_PASSWORD='<SECURE_REPOSITORY_PASSPHRASE>'",
				"restic check --read-data-subset=10%",
			},
			SafetyNote: "Do not proceed if repository integrity verification fails. Verify access keys with off-site provider.",
		},
		{
			StepNumber:  3,
			Title:       "Restore Application-Consistent Databases",
			Description: "Restore MySQL or PostgreSQL transactional dumps before launching application workers.",
			Commands: []string{
				fmt.Sprintf("restic restore %s --target /tmp/recovery --include '*_consistent_backup.sql'", req.LatestSnapshotID),
				"mysql -u root -p < /tmp/recovery/*_consistent_backup.sql",
				"rm -rf /tmp/recovery",
			},
			SafetyNote: "Foreign key checks will be automatically disabled during schema initialization.",
		},
		{
			StepNumber:  4,
			Title:       "Restore Website Document Roots & Permissions",
			Description: "Restore website files into standard Hostvra vhost paths and enforce correct webmaster ownership.",
			Commands: []string{
				fmt.Sprintf("restic restore %s --target / --include '/var/www/*'", req.LatestSnapshotID),
				"chown -R www-data:www-data /var/www/",
				"chmod -R 755 /var/www/",
			},
		},
		{
			StepNumber:  5,
			Title:       "Restore Mailboxes & Maildir Spool",
			Description: "Restore incoming and sent email Maildirs with Dovecot permissions preserved.",
			Commands: []string{
				fmt.Sprintf("restic restore %s --target / --include '/var/mail/vhosts/*'", req.LatestSnapshotID),
				"chown -R vmail:vmail /var/mail/vhosts/",
				"doveadm quota recalc -A",
			},
		},
		{
			StepNumber:  6,
			Title:       "Service Health Verification & DNS Cutover",
			Description: "Run local curl tests against 127.0.0.1 with Host headers before updating authoritative DNS records.",
			Commands: []string{
				"curl -kv -H 'Host: example.com' http://127.0.0.1/",
				"systemctl restart nginx php8.2-fpm mysql dovecot postfix",
			},
			SafetyNote: "Set TTL to 300 seconds on your DNS provider (Cloudflare, Route53, Hostvra) prior to cutover.",
		},
	}

	// Build markdown document
	md := fmt.Sprintf("# Hostvra Disaster Recovery Plan & Standard Operating Procedure\n\n")
	md += fmt.Sprintf("**Target Host:** `%s`  \n", req.TargetHost)
	md += fmt.Sprintf("**Organization:** %s  \n", req.OrganizationName)
	md += fmt.Sprintf("**Generated At:** %s UTC  \n", now.Format(time.RFC1123))
	md += fmt.Sprintf("**Repository URI:** `%s` (%s)  \n\n", req.RepositoryURI, req.RepositoryProvider)
	md += "---\n\n"

	for _, s := range steps {
		md += fmt.Sprintf("### Step %d: %s\n\n%s\n\n", s.StepNumber, s.Title, s.Description)
		if len(s.Commands) > 0 {
			md += "```bash\n"
			for _, cmd := range s.Commands {
				md += cmd + "\n"
			}
			md += "```\n\n"
		}
		if s.SafetyNote != "" {
			md += fmt.Sprintf("> **Safety Warning:** %s\n\n", s.SafetyNote)
		}
	}

	return &RunbookDocument{
		Title:       fmt.Sprintf("Disaster Recovery Plan: %s", req.TargetHost),
		GeneratedAt: now,
		Summary:     fmt.Sprintf("Automated 6-step recovery procedure for %d websites and %d databases.", len(req.Websites), len(req.Databases)),
		Steps:       steps,
		Markdown:    md,
	}
}
