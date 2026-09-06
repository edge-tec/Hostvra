package collector

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type MetricsSnapshot struct {
	CPUPercent  float64 `json:"cpu_percent"`
	RAMUsedMB   int64   `json:"ram_used_mb"`
	RAMTotalMB  int64   `json:"ram_total_mb"`
	DiskUsedGB  int64   `json:"disk_used_gb"`
	DiskTotalGB int64   `json:"disk_total_gb"`
	Load1m      float64 `json:"load_1m"`
	Load5m      float64 `json:"load_5m"`
	Load15m     float64 `json:"load_15m"`
	NetRxBytes  int64   `json:"net_rx_bytes"`
	NetTxBytes  int64   `json:"net_tx_bytes"`
	UptimeSec   int64   `json:"uptime_seconds"`
	CPUCores    int     `json:"cpu_cores"`
	CPUModel    string  `json:"cpu_model"`
	Hostname    string  `json:"hostname"`
	IPAddress   string  `json:"ip_address"`
}

type Collector struct {
	mu           sync.Mutex
	startTime    time.Time
	prevCPUTotal uint64
	prevCPUIdle  uint64
	hasPrevCPU   bool
}

func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now().UTC(),
	}
}

func (c *Collector) Collect() (*MetricsSnapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hostname, _ := os.Hostname()
	snapshot := &MetricsSnapshot{
		CPUCores:  runtime.NumCPU(),
		CPUModel:  "Virtual Processor",
		Hostname:  hostname,
		IPAddress: "127.0.0.1",
		UptimeSec: int64(time.Since(c.startTime).Seconds()),
	}

	// 1. Try Native Linux /proc collectors
	if _, err := os.Stat("/proc/meminfo"); err == nil {
		c.collectLinuxMem(snapshot)
		c.collectLinuxCPU(snapshot)
		c.collectLinuxLoad(snapshot)
		c.collectLinuxDisk(snapshot)
		c.collectLinuxNetwork(snapshot)
		c.collectLinuxCPUInfo(snapshot)
		return snapshot, nil
	}

	// 2. Development Fallback (e.g. Darwin / macOS dev environment)
	c.collectDevFallback(snapshot)
	return snapshot, nil
}

func (c *Collector) collectLinuxMem(s *MetricsSnapshot) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()

	var memTotalKB, memAvailableKB, memFreeKB, buffersKB, cachedKB int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(parts[1], 10, 64)
		switch parts[0] {
		case "MemTotal:":
			memTotalKB = val
		case "MemAvailable:":
			memAvailableKB = val
		case "MemFree:":
			memFreeKB = val
		case "Buffers:":
			buffersKB = val
		case "Cached:":
			cachedKB = val
		}
	}

	s.RAMTotalMB = memTotalKB / 1024
	if memAvailableKB > 0 {
		s.RAMUsedMB = (memTotalKB - memAvailableKB) / 1024
	} else {
		s.RAMUsedMB = (memTotalKB - memFreeKB - buffersKB - cachedKB) / 1024
	}
	if s.RAMUsedMB < 0 {
		s.RAMUsedMB = 0
	}
}

func (c *Collector) collectLinuxCPU(s *MetricsSnapshot) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return
	}

	var user, nice, system, idle, iowait, irq, softirq, steal uint64
	user, _ = strconv.ParseUint(fields[1], 10, 64)
	nice, _ = strconv.ParseUint(fields[2], 10, 64)
	system, _ = strconv.ParseUint(fields[3], 10, 64)
	idle, _ = strconv.ParseUint(fields[4], 10, 64)
	if len(fields) > 5 {
		iowait, _ = strconv.ParseUint(fields[5], 10, 64)
	}
	if len(fields) > 6 {
		irq, _ = strconv.ParseUint(fields[6], 10, 64)
	}
	if len(fields) > 7 {
		softirq, _ = strconv.ParseUint(fields[7], 10, 64)
	}
	if len(fields) > 8 {
		steal, _ = strconv.ParseUint(fields[8], 10, 64)
	}

	totalIdle := idle + iowait
	totalNonIdle := user + nice + system + irq + softirq + steal
	total := totalIdle + totalNonIdle

	if c.hasPrevCPU && total > c.prevCPUTotal {
		totalDelta := total - c.prevCPUTotal
		idleDelta := totalIdle - c.prevCPUIdle
		if totalDelta > 0 && idleDelta <= totalDelta {
			cpuUsed := float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
			s.CPUPercent = float64(int(cpuUsed*100)) / 100.0
		}
	} else {
		s.CPUPercent = 2.5 // Initial warm-up baseline
	}

	c.prevCPUTotal = total
	c.prevCPUIdle = totalIdle
	c.hasPrevCPU = true
}

func (c *Collector) collectLinuxLoad(s *MetricsSnapshot) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	parts := strings.Fields(string(b))
	if len(parts) >= 3 {
		s.Load1m, _ = strconv.ParseFloat(parts[0], 64)
		s.Load5m, _ = strconv.ParseFloat(parts[1], 64)
		s.Load15m, _ = strconv.ParseFloat(parts[2], 64)
	}
}

func (c *Collector) collectLinuxDisk(s *MetricsSnapshot) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return
	}

	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	s.DiskTotalGB = int64(totalBytes / (1024 * 1024 * 1024))
	s.DiskUsedGB = int64(usedBytes / (1024 * 1024 * 1024))
}

func (c *Collector) collectLinuxNetwork(s *MetricsSnapshot) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer f.Close()

	var totalRx, totalTx int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, ":") || strings.HasPrefix(line, "lo:") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 10 {
			continue
		}
		// Interface name is parts[0] e.g. "eth0:"
		rx, _ := strconv.ParseInt(parts[1], 10, 64)
		tx, _ := strconv.ParseInt(parts[9], 10, 64)
		totalRx += rx
		totalTx += tx
	}

	s.NetRxBytes = totalRx
	s.NetTxBytes = totalTx
}

func (c *Collector) collectLinuxCPUInfo(s *MetricsSnapshot) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				s.CPUModel = strings.TrimSpace(parts[1])
				break
			}
		}
	}
}

// Development Fallback for macOS development workstation
func (c *Collector) collectDevFallback(s *MetricsSnapshot) {
	s.CPUPercent = 4.2
	s.RAMTotalMB = 16384
	s.RAMUsedMB = 4820
	s.DiskTotalGB = 250
	s.DiskUsedGB = 78
	s.Load1m = 0.85
	s.Load5m = 0.92
	s.Load15m = 0.76
	s.CPUModel = fmt.Sprintf("Apple Silicon (%s)", runtime.GOARCH)
	s.NetRxBytes = 104857600
	s.NetTxBytes = 52428800
}
