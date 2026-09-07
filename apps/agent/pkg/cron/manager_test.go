package cron

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCron_Validation(t *testing.T) {
	cm := NewCronManager()

	validExprs := []string{
		"0 2 * * *",
		"*/15 * * * *",
		"30 4 1,15 * 5",
		"0 0 1-5 * *",
		"@daily",
		"@hourly",
		"@reboot",
		"@weekly",
		"@monthly",
	}

	for _, expr := range validExprs {
		if err := cm.ValidateSchedule(expr); err != nil {
			t.Errorf("expected valid expr %q to pass, got error: %v", expr, err)
		}
	}

	invalidExprs := []string{
		"invalid",
		"* * * *",       // 4 fields
		"* * * * * * *", // 7 fields
		"65 * * * *",     // minute > 59
		"* 25 * * *",     // hour > 23
		"* * 32 * *",     // day > 31
		"* * * 13 *",     // month > 12
		"* * * * 8",      // weekday > 7
	}

	for _, expr := range invalidExprs {
		if err := cm.ValidateSchedule(expr); err == nil {
			t.Errorf("expected invalid expr %q to fail validation, but it passed", expr)
		}
	}
}

func TestCron_DangerousCommandBlocking(t *testing.T) {
	cm := NewCronManager()

	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		"rm -r -f /",
		"mkfs.ext4 /dev/sda1",
		":(){ :|:& };:",
	}

	for _, cmd := range dangerous {
		if err := cm.ValidateCommand(cmd); err != ErrDangerousCommand {
			t.Errorf("expected ErrDangerousCommand for %q, got: %v", cmd, err)
		}
	}

	safe := []string{
		"php /var/www/site/artisan schedule:run",
		"python3 /home/user/script.py --sync",
		"/usr/local/bin/backup-mysql.sh",
		"echo 'hello world'",
	}

	for _, cmd := range safe {
		if err := cm.ValidateCommand(cmd); err != nil {
			t.Errorf("expected safe command %q to pass validation, got: %v", cmd, err)
		}
	}
}

func TestCron_CRUDAndPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-cron-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageFile := filepath.Join(tempDir, "test-crontab")
	cm := NewCronManager()
	cm.SetStoragePath(storageFile)

	// 1. Add Job
	job1, err := cm.AddJob(CronJob{
		ID:          "job-test-1",
		Schedule:    "0 3 * * *",
		Command:     "echo 'task 1'",
		SystemUser:  "root",
		Description: "Daily Backup Task",
	})
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	if job1.ID != "job-test-1" || !job1.IsEnabled {
		t.Fatalf("unexpected job after Add: %+v", job1)
	}

	// 2. Add Second Job
	_, err = cm.AddJob(CronJob{
		ID:          "job-test-2",
		Schedule:    "*/10 * * * *",
		Command:     "php /var/www/app/artisan queue:work",
		SystemUser:  "root",
		Description: "Worker Queue",
	})
	if err != nil {
		t.Fatalf("AddJob 2 failed: %v", err)
	}

	// 3. List Jobs
	jobs, err := cm.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	// 4. Toggle Job (Disable)
	toggled, err := cm.ToggleJob("job-test-1")
	if err != nil {
		t.Fatalf("ToggleJob failed: %v", err)
	}
	if toggled.IsEnabled {
		t.Fatalf("expected job-test-1 to be disabled, got enabled")
	}

	// Verify persistence from file
	reloaded, _ := cm.ListJobs()
	if reloaded[0].IsEnabled {
		t.Fatalf("reloaded job should be disabled in persistent file")
	}

	// 5. Update Job
	err = cm.UpdateJob(CronJob{
		ID:          "job-test-2",
		Schedule:    "*/5 * * * *",
		Command:     "php /var/www/app/artisan queue:work --timeout=60",
		SystemUser:  "root",
		Description: "Updated Worker Queue",
		IsEnabled:   true,
	})
	if err != nil {
		t.Fatalf("UpdateJob failed: %v", err)
	}

	// 6. Delete Job
	err = cm.DeleteJob("job-test-1")
	if err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}

	finalJobs, _ := cm.ListJobs()
	if len(finalJobs) != 1 || finalJobs[0].ID != "job-test-2" {
		t.Fatalf("expected 1 remaining job (job-test-2), got: %+v", finalJobs)
	}
}

func TestCron_ExecuteNow(t *testing.T) {
	cm := NewCronManager()

	res, err := cm.ExecuteNow("echo 'Hostvra Cron Runner Test'", "root")
	if err != nil {
		t.Fatalf("ExecuteNow failed: %v", err)
	}

	if !res.Success || res.ExitCode != 0 {
		t.Fatalf("expected exit code 0 and success, got code %d: %s", res.ExitCode, res.Stdout)
	}

	if res.Stdout == "" {
		t.Fatalf("expected stdout output from echo")
	}
}
