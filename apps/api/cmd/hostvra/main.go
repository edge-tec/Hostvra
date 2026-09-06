package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"hostvra/api/internal/update"
)

const CLIVersion = "1.0.0"

// ANSI formatting helpers
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

type CLIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func newCLIClient(baseURL, token string) *CLIClient {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	// Attempt reading local admin token if not passed
	if token == "" {
		if envToken := os.Getenv("HOSTVRA_TOKEN"); envToken != "" {
			token = envToken
		} else if b, err := os.ReadFile("/etc/hostvra/admin.token"); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}

	return &CLIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *CLIClient) request(method, path string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Hostvra API at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type apiEnvelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	var env apiEnvelope
	if err := json.Unmarshal(respBytes, &env); err != nil {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
		}
		return err
	}

	if !env.Success && env.Error != nil {
		return fmt.Errorf("[%s] %s", env.Error.Code, env.Error.Message)
	}

	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}

	return nil
}

func printBanner() {
	fmt.Printf("%s%sHostvra Command Line Utility v%s%s\n", colorCyan, colorBold, CLIVersion, colorReset)
	fmt.Println("Official Server Fleet & Live Update Management Tool")
	fmt.Println("---------------------------------------------------------")
}

func printHelp() {
	printBanner()
	fmt.Println("Usage:")
	fmt.Println("  hostvra <command> [subcommand] [flags]")
	fmt.Println("")
	fmt.Println("Available Commands:")
	fmt.Printf("  %supdate check%s       Query release server and check for available upgrades\n", colorBold, colorReset)
	fmt.Printf("  %supdate status%s      Display installation matrix and current active update job\n", colorBold, colorReset)
	fmt.Printf("  %supdate install%s     Execute live upgrade pipeline with zero downtime\n", colorBold, colorReset)
	fmt.Printf("  %supdate rollback%s    Immediately revert to the previous verified release\n", colorBold, colorReset)
	fmt.Printf("  %supdate history%s     List historical updates, schema migrations, and rollbacks\n", colorBold, colorReset)
	fmt.Printf("  %supdate channel%s     Switch release channel (stable, beta, nightly)\n", colorBold, colorReset)
	fmt.Printf("  %sversion%s            Show Hostvra CLI version information\n", colorBold, colorReset)
	fmt.Println("")
	fmt.Println("Global Flags:")
	fmt.Println("  --api-url              Hostvra API base URL (default: http://127.0.0.1:8080)")
	fmt.Println("  --token                Admin authentication bearer token")
	fmt.Println("  --yes, -y              Automatic yes to prompts; run non-interactively")
	fmt.Println("")
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	apiURLFlag := flag.String("api-url", "http://127.0.0.1:8080", "Hostvra API endpoint")
	tokenFlag := flag.String("token", "", "Admin authentication token")
	yesFlag := flag.Bool("yes", false, "Assume yes on confirmations")
	targetFlag := flag.String("target", "1.1.0", "Target version for update")
	channelFlag := flag.String("channel", "", "Release channel (stable/beta/nightly)")

	// Parse flags that might appear before or after command
	_ = flag.CommandLine.Parse(os.Args[2:])
	client := newCLIClient(*apiURLFlag, *tokenFlag)

	command := os.Args[1]
	switch command {
	case "version", "-v", "--version":
		fmt.Printf("hostvra CLI version %s (darwin/linux compatible)\n", CLIVersion)
		return

	case "help", "-h", "--help":
		printHelp()
		return

	case "update":
		if len(os.Args) < 3 {
			printUpdateHelp()
			return
		}
		subCmd := os.Args[2]
		switch subCmd {
		case "check":
			handleCheck(client)
		case "status":
			handleStatus(client)
		case "install":
			handleInstall(client, *targetFlag, *channelFlag, *yesFlag)
		case "rollback":
			handleRollback(client, *yesFlag)
		case "history":
			handleHistory(client)
		case "channel":
			if len(os.Args) < 4 {
				fmt.Printf("%sError: channel name required (stable | beta | nightly)%s\n", colorRed, colorReset)
				os.Exit(1)
			}
			handleSetChannel(client, os.Args[3])
		default:
			fmt.Printf("%sUnknown update subcommand: %s%s\n", colorRed, subCmd, colorReset)
			printUpdateHelp()
			os.Exit(1)
		}

	default:
		fmt.Printf("%sUnknown command: %s%s\n", colorRed, command, colorReset)
		printHelp()
		os.Exit(1)
	}
}

func printUpdateHelp() {
	fmt.Println("Update Commands:")
	fmt.Println("  hostvra update check")
	fmt.Println("  hostvra update status")
	fmt.Println("  hostvra update install [--target=1.1.0] [--channel=stable] [-y]")
	fmt.Println("  hostvra update rollback [-y]")
	fmt.Println("  hostvra update history")
	fmt.Println("  hostvra update channel <stable|beta|nightly>")
}

type statusPayload struct {
	System        *update.SystemVersionInfo `json:"system"`
	LatestRelease *update.ReleaseMetadata   `json:"latest_release"`
	ActiveJob     *update.UpdateJob         `json:"active_job"`
}

func handleCheck(client *CLIClient) {
	fmt.Printf("%sChecking Hostvra release network for updates...%s\n", colorCyan, colorReset)
	var status statusPayload
	if err := client.request("GET", "/api/v1/system/updates/status", nil, &status); err != nil {
		fmt.Printf("%sError checking updates: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Printf("Current Installed Version: %s%sv%s%s (Channel: %s)\n", colorBold, colorGreen, status.System.APIVersion, colorReset, status.System.Channel)

	if status.LatestRelease != nil {
		fmt.Printf("Latest Available Version:  %s%sv%s%s (Channel: %s)\n", colorBold, colorPurple, status.LatestRelease.Version, colorReset, status.LatestRelease.Channel)
		fmt.Printf("Release Date:              %s\n", status.LatestRelease.ReleasedAt.Format(time.RFC1123))
		fmt.Printf("Package Size:              %.2f MB\n", float64(status.LatestRelease.PackageSizeBytes)/(1024*1024))
		fmt.Printf("Release Notes:             %s\n", status.LatestRelease.ReleaseNotes)

		if status.System.UpdateAvailable {
			fmt.Println("")
			fmt.Printf("%s%s✓ An update is available!%s Run `%shostvra update install%s` to apply.\n", colorGreen, colorBold, colorReset, colorBold, colorReset)
		} else {
			fmt.Println("")
			fmt.Printf("%s✓ System is already running the latest version.%s\n", colorGreen, colorReset)
		}
	}
}

func handleStatus(client *CLIClient) {
	var status statusPayload
	if err := client.request("GET", "/api/v1/system/updates/status", nil, &status); err != nil {
		fmt.Printf("%sError fetching status: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	printBanner()
	fmt.Println("System Component Versions:")
	fmt.Printf("  • Hostvra API:        v%s\n", status.System.APIVersion)
	fmt.Printf("  • Hostvra Agent:      v%s\n", status.System.AgentVersion)
	fmt.Printf("  • Database Schema:    Revision #%d\n", status.System.DBSchemaVersion)
	fmt.Printf("  • Release Channel:    %s\n", status.System.Channel)
	fmt.Printf("  • OS & Architecture:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  • Runtime Engine:     %s\n", runtime.Version())
	fmt.Println("")

	if status.ActiveJob != nil {
		fmt.Printf("%sActive Update Job In Progress:%s\n", colorYellow, colorReset)
		fmt.Printf("  Job ID:       %s\n", status.ActiveJob.ID)
		fmt.Printf("  Target:       v%s\n", status.ActiveJob.TargetVersion)
		fmt.Printf("  Status:       %s (%d%%)\n", status.ActiveJob.Status, jobProgressPercent(status.ActiveJob.Status))
	} else {
		fmt.Printf("%sNo update job currently active.%s\n", colorGreen, colorReset)
	}
}

func jobProgressPercent(s update.JobStatus) int {
	switch s {
	case update.StatusPending:
		return 5
	case update.StatusPrechecking:
		return 15
	case update.StatusBackingUp:
		return 30
	case update.StatusDownloading:
		return 45
	case update.StatusVerifying:
		return 55
	case update.StatusPreparing:
		return 65
	case update.StatusMigrating:
		return 75
	case update.StatusInstalling:
		return 85
	case update.StatusActivating:
		return 90
	case update.StatusHealthChecking:
		return 95
	case update.StatusCompleted:
		return 100
	case update.StatusFailed, update.StatusRolledBack:
		return 100
	default:
		return 50
	}
}

func handleInstall(client *CLIClient, target, channel string, skipConfirm bool) {
	if !skipConfirm {
		fmt.Printf("%sAre you sure you want to perform a live update to v%s? [y/N]: %s", colorYellow, target, colorReset)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update aborted by user.")
			return
		}
	}

	fmt.Printf("%sInitiating live update pipeline for v%s...%s\n", colorCyan, target, colorReset)
	reqBody := map[string]string{
		"target_version": target,
		"channel":        channel,
	}

	var job update.UpdateJob
	if err := client.request("POST", "/api/v1/system/updates/start", reqBody, &job); err != nil {
		fmt.Printf("%sFailed to start update: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Update job queued (ID: %s)%s\n", colorGreen, job.ID, colorReset)
	fmt.Println("Streaming execution pipeline:")

	// Polling loop with animated progress display
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinIdx := 0

	for {
		var currentJob update.UpdateJob
		err := client.request("GET", fmt.Sprintf("/api/v1/system/updates/jobs/%s", job.ID), nil, &currentJob)
		if err == nil {
			statusStr := strings.ToUpper(string(currentJob.Status))
			percent := jobProgressPercent(currentJob.Status)

			fmt.Printf("\r  %s [%3d%%] %-16s", spinner[spinIdx%len(spinner)], percent, statusStr)
			spinIdx++

			if currentJob.Status == update.StatusCompleted {
				fmt.Printf("\n\n%s%s🎉 Successfully upgraded Hostvra to v%s!%s\n", colorGreen, colorBold, target, colorReset)
				return
			}
			if currentJob.Status == update.StatusFailed || currentJob.Status == update.StatusRolledBack {
				fmt.Printf("\n\n%s⚠️ Update terminated with status: %s%s\n", colorRed, currentJob.Status, colorReset)
				if currentJob.ErrorMessage != "" {
					fmt.Printf("Error detail: %s\n", currentJob.ErrorMessage)
				}
				os.Exit(1)
			}
		}

		time.Sleep(400 * time.Millisecond)
	}
}

func handleRollback(client *CLIClient, skipConfirm bool) {
	if !skipConfirm {
		fmt.Printf("%sWARNING: This will revert Hostvra to the previous release snapshot. Continue? [y/N]: %s", colorRed, colorReset)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Rollback cancelled.")
			return
		}
	}

	fmt.Printf("%sExecuting emergency rollback...%s\n", colorYellow, colorReset)
	var resp map[string]string
	if err := client.request("POST", "/api/v1/system/updates/rollback", nil, &resp); err != nil {
		fmt.Printf("%sRollback failed: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s✓ %s%s\n", colorGreen, resp["message"], colorReset)
}

func handleHistory(client *CLIClient) {
	var jobs []*update.UpdateJob
	if err := client.request("GET", "/api/v1/system/updates/jobs", nil, &jobs); err != nil {
		fmt.Printf("%sError retrieving update history: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	printBanner()
	fmt.Printf("%-36s  %-8s  %-14s  %-8s  %-20s\n", "JOB ID", "TARGET", "STATUS", "CHANNEL", "STARTED AT")
	fmt.Println(strings.Repeat("-", 94))

	if len(jobs) == 0 {
		fmt.Println("No recorded update operations found.")
		return
	}

	for _, j := range jobs {
		statusColor := colorGreen
		if j.Status == update.StatusFailed {
			statusColor = colorRed
		} else if j.Status == update.StatusRolledBack {
			statusColor = colorYellow
		}
		fmt.Printf("%-36s  v%-7s  %s%-14s%s  %-8s  %-20s\n",
			j.ID.String(),
			j.TargetVersion,
			statusColor,
			j.Status,
			colorReset,
			j.Channel,
			j.StartedAt.Format("2006-01-02 15:04:05"),
		)
	}
}

func handleSetChannel(client *CLIClient, channel string) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	reqBody := map[string]string{"channel": channel}
	var resp map[string]string
	if err := client.request("POST", "/api/v1/system/updates/channel", reqBody, &resp); err != nil {
		fmt.Printf("%sFailed to switch channel: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s✓ %s%s\n", colorGreen, resp["message"], colorReset)
}
