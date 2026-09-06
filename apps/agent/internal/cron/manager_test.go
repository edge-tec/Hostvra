package cron

import (
	"testing"
)

func TestCronValidation(t *testing.T) {
	cm := NewCronManager()

	// Valid cron expressions
	validExprs := []string{
		"0 2 * * *",
		"*/15 * * * *",
		"30 4 1,15 * 5",
		"@daily",
		"@hourly",
		"@reboot",
	}

	for _, expr := range validExprs {
		if err := cm.ValidateSchedule(expr); err != nil {
			t.Errorf("Expected valid expression '%s' to pass, got error: %v", expr, err)
		}
	}

	// Invalid cron expressions
	invalidExprs := []string{
		"invalid",
		"* * * *",       // 4 fields
		"* * * * * * *", // 7 fields
		"65 * * * *",     // minute 65 > 59
		"* 25 * * *",     // hour 25 > 23
	}

	for _, expr := range invalidExprs {
		if err := cm.ValidateSchedule(expr); err == nil {
			t.Errorf("Expected invalid expression '%s' to fail validation", expr)
		}
	}
}
