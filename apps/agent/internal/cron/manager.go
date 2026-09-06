package cron

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidCronSchedule = errors.New("invalid cron schedule expression (must contain 5 fields: min hour day month weekday)")
)

type CronJob struct {
	ID          string    `json:"id"`
	Schedule    string    `json:"schedule"` // e.g. "0 2 * * *"
	Command     string    `json:"command"`
	SystemUser  string    `json:"system_user"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"is_enabled"`
	LastRunAt   time.Time `json:"last_run_at,omitempty"`
}

type CronManager struct{}

func NewCronManager() *CronManager {
	return &CronManager{}
}

func (cm *CronManager) ValidateSchedule(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "@reboot" || expr == "@hourly" || expr == "@daily" || expr == "@weekly" || expr == "@monthly" {
		return nil
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return ErrInvalidCronSchedule
	}

	// Validate field 0 (minute 0-59 or */n or *)
	if err := validateCronField(parts[0], 0, 59); err != nil {
		return fmt.Errorf("invalid minute field: %w", err)
	}
	// Validate field 1 (hour 0-23)
	if err := validateCronField(parts[1], 0, 23); err != nil {
		return fmt.Errorf("invalid hour field: %w", err)
	}
	// Validate field 2 (day of month 1-31)
	if err := validateCronField(parts[2], 1, 31); err != nil {
		return fmt.Errorf("invalid day-of-month field: %w", err)
	}
	// Validate field 3 (month 1-12)
	if err := validateCronField(parts[3], 1, 12); err != nil {
		return fmt.Errorf("invalid month field: %w", err)
	}
	// Validate field 4 (weekday 0-7)
	if err := validateCronField(parts[4], 0, 7); err != nil {
		return fmt.Errorf("invalid weekday field: %w", err)
	}

	return nil
}

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
		return errors.New("must be numeric or wildcard")
	}
	if val < min || val > max {
		return fmt.Errorf("value %d out of bounds [%d, %d]", val, min, max)
	}
	return nil
}

func (cm *CronManager) ExecuteNow(command, user string) (string, error) {
	if user == "" {
		user = "root"
	}

	var cmd *exec.Cmd
	if user != "root" {
		cmd = exec.Command("su", "-", user, "-c", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("execution error: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	return string(out), nil
}
