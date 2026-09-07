package cron

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func generateJobID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("cron-%x", b)
}

var (
	ErrInvalidCronSchedule = errors.New("invalid cron schedule expression (must contain 5 fields: min hour day month weekday or @reboot/@daily/@hourly/@weekly/@monthly)")
	ErrInvalidSystemUser   = errors.New("invalid system username: only alphanumeric, hyphen, and underscores allowed (1-32 chars)")
	ErrDangerousCommand    = errors.New("command rejected: destructive system command pattern detected")
	ErrJobNotFound         = errors.New("cron job not found")

	validUserRegex        = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
	dangerousCmdPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(^|\s)rm\s+.*(-[a-zA-Z]*r[a-zA-Z]*\s+.*-[a-zA-Z]*f[a-zA-Z]*|-[a-zA-Z]*f[a-zA-Z]*\s+.*-[a-zA-Z]*r[a-zA-Z]*|-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*|-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*)\s+(/\s*$|/\*\s*$)`),
		regexp.MustCompile(`(^|\s)mkfs(\.[a-zA-Z0-9]+|\s+)`),
		regexp.MustCompile(`(^|\s)dd\s+if=.*of=/dev/([sh]d[a-z]|nvme)`),
		regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`),
	}
)

type CronJob struct {
	ID          string `json:"id"`
	Schedule    string `json:"schedule"`    // "0 2 * * *" or "@daily"
	Command     string `json:"command"`
	SystemUser  string `json:"system_user"` // "root", "www-data"
	Description string `json:"description"`
	IsEnabled   bool   `json:"is_enabled"`
	LastRunAt   string `json:"last_run_at,omitempty"`
	LastStatus  string `json:"last_status,omitempty"` // "success", "failed", "running"
	LastOutput  string `json:"last_output,omitempty"`
}

type ExecutionResult struct {
	JobID      string `json:"job_id,omitempty"`
	Command    string `json:"command"`
	SystemUser string `json:"system_user"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
	Success    bool   `json:"success"`
}

type CronDaemonStatus struct {
	IsActive bool   `json:"is_active"`
	Daemon   string `json:"daemon"` // "cron", "crond", "systemd-cron", "dev"
	JobsCount int   `json:"jobs_count"`
}

type CronManager struct {
	mu         sync.RWMutex
	filePath   string
	lastRuns   map[string]*ExecutionResult
}

func NewCronManager() *CronManager {
	targetFile := "/etc/cron.d/hostvra"
	// Check if directory exists and is writable, otherwise use local app storage
	dir := filepath.Dir(targetFile)
	if _, err := os.Stat(dir); err != nil {
		homeDir, _ := os.UserHomeDir()
		targetFile = filepath.Join(homeDir, ".hostvra", "crontab")
	}

	_ = os.MkdirAll(filepath.Dir(targetFile), 0755)

	return &CronManager{
		filePath: targetFile,
		lastRuns: make(map[string]*ExecutionResult),
	}
}

// SetStoragePath allows customizing the crontab storage file (for unit testing)
func (cm *CronManager) SetStoragePath(path string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.filePath = path
	_ = os.MkdirAll(filepath.Dir(path), 0755)
}

// GetDaemonStatus checks if the system cron daemon is active
func (cm *CronManager) GetDaemonStatus() (*CronDaemonStatus, error) {
	daemonName := "cron"
	active := false

	if _, err := exec.LookPath("systemctl"); err == nil {
		for _, name := range []string{"cron", "crond", "cronie"} {
			if out, err := exec.Command("systemctl", "is-active", name).Output(); err == nil {
				if strings.TrimSpace(string(out)) == "active" {
					active = true
					daemonName = name
					break
				}
			}
		}
	} else {
		// Non-systemd or dev environment
		active = true
		daemonName = "dev"
	}

	jobs, _ := cm.ListJobs()

	return &CronDaemonStatus{
		IsActive:  active,
		Daemon:    daemonName,
		JobsCount: len(jobs),
	}, nil
}

// ValidateSchedule verifies standard 5-field cron syntax or macros
func (cm *CronManager) ValidateSchedule(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "@reboot" || expr == "@hourly" || expr == "@daily" || expr == "@weekly" || expr == "@monthly" {
		return nil
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return ErrInvalidCronSchedule
	}

	if err := validateCronField(parts[0], 0, 59); err != nil {
		return fmt.Errorf("invalid minute field: %w", err)
	}
	if err := validateCronField(parts[1], 0, 23); err != nil {
		return fmt.Errorf("invalid hour field: %w", err)
	}
	if err := validateCronField(parts[2], 1, 31); err != nil {
		return fmt.Errorf("invalid day-of-month field: %w", err)
	}
	if err := validateCronField(parts[3], 1, 12); err != nil {
		return fmt.Errorf("invalid month field: %w", err)
	}
	if err := validateCronField(parts[4], 0, 7); err != nil {
		return fmt.Errorf("invalid weekday field: %w", err)
	}

	return nil
}

// ValidateCommand inspects for catastrophic/destructive system patterns
func (cm *CronManager) ValidateCommand(cmd string) error {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return errors.New("command cannot be empty")
	}

	for _, pattern := range dangerousCmdPatterns {
		if pattern.MatchString(trimmed) {
			return ErrDangerousCommand
		}
	}
	return nil
}

// ListJobs reads and parses all configured cron jobs
func (cm *CronManager) ListJobs() ([]CronJob, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []CronJob{}, nil
		}
		return nil, err
	}

	return cm.parseCrontab(string(data)), nil
}

// AddJob validates and saves a new scheduled task
func (cm *CronManager) AddJob(job CronJob) (*CronJob, error) {
	if err := cm.ValidateSchedule(job.Schedule); err != nil {
		return nil, err
	}
	if err := cm.ValidateCommand(job.Command); err != nil {
		return nil, err
	}

	if job.SystemUser == "" {
		job.SystemUser = "root"
	}
	job.SystemUser = strings.TrimSpace(job.SystemUser)
	if !validUserRegex.MatchString(job.SystemUser) {
		return nil, fmt.Errorf("%w: '%s'", ErrInvalidSystemUser, job.SystemUser)
	}

	if job.ID == "" {
		job.ID = generateJobID()
	}
	job.IsEnabled = true

	cm.mu.Lock()
	defer cm.mu.Unlock()

	existing, _ := cm.loadJobsUnlocked()
	for _, j := range existing {
		if j.ID == job.ID {
			return nil, fmt.Errorf("cron job with ID '%s' already exists", job.ID)
		}
	}

	existing = append(existing, job)
	if err := cm.saveJobsUnlocked(existing); err != nil {
		return nil, err
	}

	return &job, nil
}

// UpdateJob modifies an existing cron job
func (cm *CronManager) UpdateJob(job CronJob) error {
	if err := cm.ValidateSchedule(job.Schedule); err != nil {
		return err
	}
	if err := cm.ValidateCommand(job.Command); err != nil {
		return err
	}

	if job.SystemUser == "" {
		job.SystemUser = "root"
	}
	job.SystemUser = strings.TrimSpace(job.SystemUser)
	if !validUserRegex.MatchString(job.SystemUser) {
		return fmt.Errorf("%w: '%s'", ErrInvalidSystemUser, job.SystemUser)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	existing, err := cm.loadJobsUnlocked()
	if err != nil {
		return err
	}

	found := false
	for i, j := range existing {
		if j.ID == job.ID {
			existing[i] = job
			found = true
			break
		}
	}

	if !found {
		return ErrJobNotFound
	}

	return cm.saveJobsUnlocked(existing)
}

// DeleteJob removes a cron job by ID
func (cm *CronManager) DeleteJob(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	existing, err := cm.loadJobsUnlocked()
	if err != nil {
		return err
	}

	newJobs := make([]CronJob, 0, len(existing))
	found := false
	for _, j := range existing {
		if j.ID == id {
			found = true
			continue
		}
		newJobs = append(newJobs, j)
	}

	if !found {
		return ErrJobNotFound
	}

	delete(cm.lastRuns, id)
	return cm.saveJobsUnlocked(newJobs)
}

// ToggleJob enables or disables a cron job
func (cm *CronManager) ToggleJob(id string) (*CronJob, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	existing, err := cm.loadJobsUnlocked()
	if err != nil {
		return nil, err
	}

	var updated *CronJob
	for i, j := range existing {
		if j.ID == id {
			existing[i].IsEnabled = !existing[i].IsEnabled
			updated = &existing[i]
			break
		}
	}

	if updated == nil {
		return nil, ErrJobNotFound
	}

	if err := cm.saveJobsUnlocked(existing); err != nil {
		return nil, err
	}

	return updated, nil
}

// ExecuteNow runs a command immediately with a 60-second execution window and captures stdout/stderr
func (cm *CronManager) ExecuteNow(command, user string) (*ExecutionResult, error) {
	if err := cm.ValidateCommand(command); err != nil {
		return nil, err
	}

	if user == "" {
		user = "root"
	}
	user = strings.TrimSpace(user)
	if !validUserRegex.MatchString(user) {
		return nil, fmt.Errorf("%w: '%s'", ErrInvalidSystemUser, user)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	var cmd *exec.Cmd

	if user != "root" && os.Geteuid() == 0 {
		cmd = exec.CommandContext(ctx, "su", "-", user, "-c", command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
	}

	outBytes, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()
	outputStr := string(outBytes)

	exitCode := 0
	isSuccess := true
	if err != nil {
		isSuccess = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	res := &ExecutionResult{
		Command:    command,
		SystemUser: user,
		ExitCode:   exitCode,
		Stdout:     outputStr,
		DurationMs: duration,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Success:    isSuccess,
	}

	return res, nil
}

// ExecuteJobNow executes a registered job and updates its last run metrics
func (cm *CronManager) ExecuteJobNow(id string) (*ExecutionResult, error) {
	jobs, err := cm.ListJobs()
	if err != nil {
		return nil, err
	}

	var target *CronJob
	for _, j := range jobs {
		if j.ID == id {
			target = &j
			break
		}
	}

	if target == nil {
		return nil, ErrJobNotFound
	}

	res, err := cm.ExecuteNow(target.Command, target.SystemUser)
	if err != nil && res == nil {
		return nil, err
	}

	res.JobID = id

	cm.mu.Lock()
	cm.lastRuns[id] = res
	cm.mu.Unlock()

	return res, nil
}

// ----------------------------------------------------------------------------
// PARSER & ATOMIC PERSISTENCE
// ----------------------------------------------------------------------------

func (cm *CronManager) loadJobsUnlocked() ([]CronJob, error) {
	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []CronJob{}, nil
		}
		return nil, err
	}
	return cm.parseCrontab(string(data)), nil
}

func (cm *CronManager) saveJobsUnlocked(jobs []CronJob) error {
	var sb strings.Builder
	sb.WriteString("# Hostvra Automated System Crontab\n")
	sb.WriteString("# Generated automatically - DO NOT EDIT MANUALLY\n\n")
	sb.WriteString("SHELL=/bin/bash\n")
	sb.WriteString("PATH=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/sbin:/usr/local/bin\n\n")

	for _, j := range jobs {
		prefix := ""
		if !j.IsEnabled {
			prefix = "# DISABLED: "
		}
		desc := strings.ReplaceAll(j.Description, "\n", " ")
		sb.WriteString(fmt.Sprintf("%s%s %s %s # HOSTVRA_ID:%s # %s\n",
			prefix, j.Schedule, j.SystemUser, j.Command, j.ID, desc))
	}

	tmpFile := cm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(sb.String()), 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, cm.filePath)
}

func (cm *CronManager) parseCrontab(content string) []CronJob {
	var jobs []CronJob
	lines := strings.Split(content, "\n")

	// Match: [optional # DISABLED: ] <schedule(5 parts or @macro)> <user> <command> # HOSTVRA_ID:<id> # <desc>
	idRegex := regexp.MustCompile(`#\s*HOSTVRA_ID:(\S+)(?:\s+#\s*(.*))?$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# DISABLED:")) {
			continue
		}
		if strings.HasPrefix(line, "SHELL=") || strings.HasPrefix(line, "PATH=") {
			continue
		}

		isEnabled := true
		rawLine := line
		if strings.HasPrefix(line, "# DISABLED:") {
			isEnabled = false
			rawLine = strings.TrimSpace(strings.TrimPrefix(line, "# DISABLED:"))
		}

		matches := idRegex.FindStringSubmatch(rawLine)
		id := ""
		description := ""
		if len(matches) >= 2 {
			id = matches[1]
			if len(matches) >= 3 {
				description = strings.TrimSpace(matches[2])
			}
			rawLine = strings.TrimSpace(rawLine[:strings.Index(rawLine, "# HOSTVRA_ID:")])
		}

		fields := strings.Fields(rawLine)
		if len(fields) < 2 {
			continue
		}

		schedule := ""
		user := "root"
		cmd := ""

		if strings.HasPrefix(fields[0], "@") {
			// Macro syntax e.g. @daily root /cmd
			schedule = fields[0]
			if len(fields) >= 2 {
				user = fields[1]
			}
			if len(fields) >= 3 {
				cmd = strings.Join(fields[2:], " ")
			}
		} else if len(fields) >= 7 {
			// Standard 5-field schedule + user + command
			schedule = strings.Join(fields[0:5], " ")
			user = fields[5]
			cmd = strings.Join(fields[6:], " ")
		} else {
			continue
		}

		if id == "" {
			id = generateJobID()
		}

		job := CronJob{
			ID:          id,
			Schedule:    schedule,
			Command:     cmd,
			SystemUser:  user,
			Description: description,
			IsEnabled:   isEnabled,
		}

		// Attach cached execution status if present
		if run, ok := cm.lastRuns[id]; ok {
			job.LastRunAt = run.Timestamp
			if run.Success {
				job.LastStatus = "success"
			} else {
				job.LastStatus = "failed"
			}
			job.LastOutput = run.Stdout
		}

		jobs = append(jobs, job)
	}

	return jobs
}

// ----------------------------------------------------------------------------
// VALIDATION HELPERS
// ----------------------------------------------------------------------------

func validateCronField(field string, min, max int) error {
	if field == "*" {
		return nil
	}
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, p := range parts {
			if err := validateSingleCronVal(p, min, max); err != nil {
				return err
			}
		}
		return nil
	}
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return errors.New("invalid range expression")
		}
		v1, err1 := strconv.Atoi(parts[0])
		v2, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || v1 < min || v2 > max || v1 > v2 {
			return errors.New("range bounds out of allowed values")
		}
		return nil
	}
	return validateSingleCronVal(field, min, max)
}

func validateSingleCronVal(valStr string, min, max int) error {
	if strings.HasPrefix(valStr, "*/") {
		stepStr := strings.TrimPrefix(valStr, "*/")
		step, err := strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return errors.New("invalid step value")
		}
		return nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return errors.New("must be numeric, wildcard, or range")
	}
	if val < min || val > max {
		return fmt.Errorf("value %d out of bounds [%d, %d]", val, min, max)
	}
	return nil
}
