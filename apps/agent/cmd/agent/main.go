package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hostvra/agent/internal/client"
	"hostvra/agent/internal/collector"
	"hostvra/agent/internal/config"
	"hostvra/agent/internal/osadapter"
)

const AgentVersion = "1.0.0"

func main() {
	enrollToken := flag.String("token", "", "Temporary enrollment token from Hostvra Control Plane")
	endpoint := flag.String("endpoint", "http://localhost:8080", "Hostvra Core API endpoint URL")
	configFile := flag.String("config", "", "Path to agent.json configuration file")
	daemonMode := flag.Bool("daemon", false, "Run in background daemon telemetry mode")
	versionFlag := flag.Bool("version", false, "Display Hostvra Agent version and exit")
	flag.Parse()

	_ = daemonMode // Explicitly acknowledge daemonMode flag

	if *versionFlag {
		fmt.Printf("Hostvra Agent v%s\n", AgentVersion)
		os.Exit(0)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("Initializing Hostvra Node Agent", "version", AgentVersion)

	// 1. Detect Operating System and Distribution
	osInfo, err := osadapter.Detect()
	if err != nil {
		logger.Error("Failed to detect operating system", "error", err)
		os.Exit(1)
	}

	logger.Info("OS detected successfully",
		"name", osInfo.Name(),
		"version", osInfo.Version(),
		"family", osInfo.Family(),
		"arch", osInfo.Architecture(),
		"pkg_manager", osInfo.PackageManager(),
	)

	// 2. Hardware Collector
	metricsCollector := collector.NewCollector()
	apiClient := client.NewAPIClient(*endpoint)

	// 3. Handle Enrollment Mode if token provided
	if *enrollToken != "" {
		logger.Info("Enrollment flag detected, registering node with Hostvra Control Plane...")
		initialMetrics, err := metricsCollector.Collect()
		if err != nil {
			logger.Error("Failed to collect initial metrics for enrollment", "error", err)
			os.Exit(1)
		}

		enrollData, err := apiClient.Enroll(*enrollToken, osInfo, initialMetrics, AgentVersion)
		if err != nil {
			logger.Error("Enrollment rejected by control plane", "error", err)
			os.Exit(1)
		}

		logger.Info("Server enrolled successfully!",
			"server_id", enrollData.ServerID,
			"org_id", enrollData.OrgID,
		)

		cfgPath := *configFile
		if cfgPath == "" {
			cfgPath = config.DefaultConfigPath()
		}

		agentConfig := &config.AgentConfig{
			ServerID:             enrollData.ServerID,
			AgentKey:             enrollData.AgentKey,
			ControlPlaneURL:      *endpoint,
			HeartbeatIntervalSec: 10,
		}

		if err := agentConfig.Save(cfgPath); err != nil {
			logger.Error("Failed to save agent credentials to disk", "error", err, "path", cfgPath)
			os.Exit(1)
		}

		logger.Info("Agent configuration written to disk securely", "path", cfgPath)
		fmt.Println("---------------------------------------------------------------")
		fmt.Println(" Hostvra Agent installed and enrolled successfully!           ")
		fmt.Printf(" Server ID : %s\n", enrollData.ServerID)
		fmt.Println(" Starting telemetry daemon loop...                             ")
		fmt.Println("---------------------------------------------------------------")
	}

	// 4. Load Configuration for Daemon Mode
	agentConfig, err := config.Load(*configFile)
	if err != nil {
		logger.Error("Unable to load agent configuration. Run with --token <token> first to enroll.", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting Hostvra Agent Telemetry Daemon",
		"server_id", agentConfig.ServerID,
		"endpoint", agentConfig.ControlPlaneURL,
		"interval_seconds", agentConfig.HeartbeatIntervalSec,
	)

	// 5. Heartbeat & Metrics Loop
	ticker := time.NewTicker(time.Duration(agentConfig.HeartbeatIntervalSec) * time.Second)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Send initial heartbeat immediately
	sendReport(agentConfig, metricsCollector, apiClient, logger)

	for {
		select {
		case <-ticker.C:
			sendReport(agentConfig, metricsCollector, apiClient, logger)
		case sig := <-quit:
			logger.Info("Hostvra Agent received signal, shutting down cleanly", "signal", sig.String())
			return
		}
	}
}

func sendReport(cfg *config.AgentConfig, c *collector.Collector, cl *client.APIClient, log *slog.Logger) {
	metrics, err := c.Collect()
	if err != nil {
		log.Warn("Failed to sample hardware metrics", "error", err)
		return
	}

	err = cl.SendHeartbeat(cfg.ServerID, cfg.AgentKey, metrics)
	if err != nil {
		log.Warn("Failed to report telemetry heartbeat to control plane", "error", err)
	} else {
		log.Debug("Telemetry heartbeat transmitted successfully",
			"cpu_pct", metrics.CPUPercent,
			"ram_used_mb", metrics.RAMUsedMB,
			"uptime_sec", metrics.UptimeSec,
		)
	}
}
