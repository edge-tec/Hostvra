package caching

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrPublicCacheExposure   = errors.New("caching service must not be bound to public network; use isolated Unix socket or 127.0.0.1 loopback")
	ErrUnsupportedCacheType  = errors.New("unsupported cache type; must be 'redis', 'memcached', or 'none'")
	ErrWPConfigNotFound      = errors.New("wp-config.php not found in website document root")
	ErrServiceNotReachable   = errors.New("cache service is not reachable on configured socket or host:port")
)

// CacheServiceStatus represents detected cache daemon binaries and runtime status.
type CacheServiceStatus struct {
	ServiceType string `json:"service_type"` // "redis", "memcached"
	Installed   bool   `json:"installed"`
	Running     bool   `json:"running"`
	Version     string `json:"version"`
	BinaryPath  string `json:"binary_path"`
	Port        int    `json:"port"`
	SocketPath  string `json:"socket_path"`
	Error       string `json:"error,omitempty"`
}

// CacheConfig represents per-website caching configuration.
type CacheConfig struct {
	WebsiteID            string `json:"website_id"`
	Domain               string `json:"domain"`
	CacheType            string `json:"cache_type"`      // "redis", "memcached", "none"
	ConnectionType       string `json:"connection_type"` // "socket", "tcp"
	SocketPath           string `json:"socket_path"`
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	Password             string `json:"password"`
	MaxMemoryMB          int    `json:"max_memory_mb"`
	EvictionPolicy       string `json:"eviction_policy"` // "allkeys-lru", "volatile-lru", etc.
	WPObjectCacheEnabled bool   `json:"wp_object_cache_enabled"`
	WebRoot              string `json:"web_root"` // Root directory of site
}

// CacheMetrics encapsulates real-time operational telemetry.
type CacheMetrics struct {
	CacheType         string  `json:"cache_type"`
	Running           bool    `json:"running"`
	UptimeSeconds     int64   `json:"uptime_seconds"`
	ConnectedClients  int     `json:"connected_clients"`
	UsedMemoryBytes   int64   `json:"used_memory_bytes"`
	UsedMemoryHuman   string  `json:"used_memory_human"`
	MaxMemoryBytes    int64   `json:"max_memory_bytes"`
	TotalKeys         int64   `json:"total_keys"`
	Hits              int64   `json:"hits"`
	Misses            int64   `json:"misses"`
	HitRatePercent    float64 `json:"hit_rate_percent"`
}

// BenchmarkResult records latency and throughput benchmark results.
type BenchmarkResult struct {
	CacheType       string  `json:"cache_type"`
	Operations      int     `json:"operations"`
	TotalDurationMs int64   `json:"total_duration_ms"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	MinLatencyMs    float64 `json:"min_latency_ms"`
	MaxLatencyMs    float64 `json:"max_latency_ms"`
	OpsPerSec       float64 `json:"ops_per_sec"`
}

// Manager handles caching daemons, per-site isolation, metrics, and CMS integration.
type Manager struct {
	mu sync.RWMutex
}

// NewManager creates a caching engine manager.
func NewManager() *Manager {
	return &Manager{}
}

// DetectServices inspects the host system for Redis and Memcached availability.
func (m *Manager) DetectServices() []CacheServiceStatus {
	results := make([]CacheServiceStatus, 0, 2)

	// 1. Detect Redis
	redisStatus := CacheServiceStatus{
		ServiceType: "redis",
		Port:        6379,
		SocketPath:  "/var/run/redis/redis-server.sock",
	}
	if path, err := exec.LookPath("redis-server"); err == nil {
		redisStatus.Installed = true
		redisStatus.BinaryPath = path
		if out, err := exec.Command(path, "--version").Output(); err == nil {
			redisStatus.Version = parseRedisVersion(string(out))
		}
	} else if _, err := os.Stat("/usr/bin/redis-server"); err == nil {
		redisStatus.Installed = true
		redisStatus.BinaryPath = "/usr/bin/redis-server"
	}
	redisStatus.Running = isServiceActive("redis") || isServiceActive("redis-server") || isPortOpen("127.0.0.1", 6379)
	results = append(results, redisStatus)

	// 2. Detect Memcached
	memStatus := CacheServiceStatus{
		ServiceType: "memcached",
		Port:        11211,
		SocketPath:  "/var/run/memcached/memcached.sock",
	}
	if path, err := exec.LookPath("memcached"); err == nil {
		memStatus.Installed = true
		memStatus.BinaryPath = path
		if out, err := exec.Command(path, "-h").CombinedOutput(); err == nil {
			memStatus.Version = parseMemcachedVersion(string(out))
		}
	} else if _, err := os.Stat("/usr/bin/memcached"); err == nil {
		memStatus.Installed = true
		memStatus.BinaryPath = "/usr/bin/memcached"
	}
	memStatus.Running = isServiceActive("memcached") || isPortOpen("127.0.0.1", 11211)
	results = append(results, memStatus)

	return results
}

// ConfigureWebsiteCache validates isolation parameters and applies WordPress object cache if requested.
func (m *Manager) ConfigureWebsiteCache(ctx context.Context, cfg CacheConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.CacheType == "none" || cfg.CacheType == "" {
		return m.disableWebsiteCacheLocked(cfg)
	}

	if cfg.CacheType != "redis" && cfg.CacheType != "memcached" {
		return ErrUnsupportedCacheType
	}

	// Security rule: Never allow binding to public interface
	if cfg.ConnectionType == "tcp" {
		if cfg.Host != "127.0.0.1" && cfg.Host != "localhost" && cfg.Host != "" {
			return ErrPublicCacheExposure
		}
		if cfg.Port <= 0 {
			if cfg.CacheType == "redis" {
				cfg.Port = 6379
			} else {
				cfg.Port = 11211
			}
		}
	} else {
		cfg.ConnectionType = "socket"
		if cfg.SocketPath == "" {
			if cfg.CacheType == "redis" {
				cfg.SocketPath = fmt.Sprintf("/var/run/redis/redis-%s.sock", cfg.WebsiteID)
			} else {
				cfg.SocketPath = fmt.Sprintf("/var/run/memcached/memcached-%s.sock", cfg.WebsiteID)
			}
		}
		// Ensure socket directory exists with secure permissions
		sockDir := filepath.Dir(cfg.SocketPath)
		_ = os.MkdirAll(sockDir, 0770)
	}

	// WordPress Object Cache injection
	if cfg.WPObjectCacheEnabled && cfg.WebRoot != "" {
		if err := m.configureWordPressObjectCacheLocked(cfg); err != nil {
			return err
		}
	} else if cfg.WebRoot != "" {
		_ = m.disableWordPressObjectCacheLocked(cfg)
	}

	return nil
}

// DisableWebsiteCache removes caching configuration and clears WordPress constants.
func (m *Manager) DisableWebsiteCache(ctx context.Context, cfg CacheConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.disableWebsiteCacheLocked(cfg)
}

func (m *Manager) disableWebsiteCacheLocked(cfg CacheConfig) error {
	if cfg.WebRoot != "" {
		_ = m.disableWordPressObjectCacheLocked(cfg)
	}
	return nil
}

// GetCacheMetrics fetches real-time telemetry from Redis or Memcached.
func (m *Manager) GetCacheMetrics(ctx context.Context, cfg CacheConfig) (*CacheMetrics, error) {
	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return &CacheMetrics{
			CacheType:        cfg.CacheType,
			Running:          true,
			UptimeSeconds:    86400,
			ConnectedClients: 4,
			UsedMemoryBytes:  18 * 1024 * 1024,
			UsedMemoryHuman:  "18.00M",
			MaxMemoryBytes:   int64(cfg.MaxMemoryMB) * 1024 * 1024,
			TotalKeys:        1450,
			Hits:             9820,
			Misses:           180,
			HitRatePercent:   98.2,
		}, nil
	}

	conn, err := m.dialCache(cfg, 3*time.Second)
	if err != nil {
		return &CacheMetrics{
			CacheType: cfg.CacheType,
			Running:   false,
		}, nil
	}
	defer conn.Close()

	if cfg.CacheType == "redis" {
		return m.queryRedisMetrics(conn, cfg.Password)
	} else if cfg.CacheType == "memcached" {
		return m.queryMemcachedMetrics(conn)
	}

	return nil, ErrUnsupportedCacheType
}

// FlushCache clears the database or cache namespace.
func (m *Manager) FlushCache(ctx context.Context, cfg CacheConfig) error {
	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return nil
	}

	conn, err := m.dialCache(cfg, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	if cfg.CacheType == "redis" {
		if cfg.Password != "" {
			if _, err := fmt.Fprintf(conn, "AUTH %s\r\n", cfg.Password); err != nil {
				return err
			}
			reader := bufio.NewReader(conn)
			_, _ = reader.ReadString('\n')
		}
		if _, err := fmt.Fprintf(conn, "FLUSHDB\r\n"); err != nil {
			return err
		}
		reader := bufio.NewReader(conn)
		resp, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if !strings.HasPrefix(resp, "+OK") {
			return fmt.Errorf("redis flush error: %s", strings.TrimSpace(resp))
		}
		return nil
	} else if cfg.CacheType == "memcached" {
		if _, err := fmt.Fprintf(conn, "flush_all\r\n"); err != nil {
			return err
		}
		reader := bufio.NewReader(conn)
		resp, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if !strings.HasPrefix(resp, "OK") {
			return fmt.Errorf("memcached flush error: %s", strings.TrimSpace(resp))
		}
		return nil
	}

	return ErrUnsupportedCacheType
}

// BenchmarkCache measures round-trip latency and throughput.
func (m *Manager) BenchmarkCache(ctx context.Context, cfg CacheConfig, iterations int) (*BenchmarkResult, error) {
	if iterations <= 0 {
		iterations = 30
	}

	if os.Getenv("HOSTVRA_TEST_MODE") == "1" {
		return &BenchmarkResult{
			CacheType:       cfg.CacheType,
			Operations:      iterations,
			TotalDurationMs: 15,
			AvgLatencyMs:    0.50,
			MinLatencyMs:    0.28,
			MaxLatencyMs:    1.12,
			OpsPerSec:       2000.0,
		}, nil
	}

	conn, err := m.dialCache(cfg, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if cfg.CacheType == "redis" && cfg.Password != "" {
		fmt.Fprintf(conn, "AUTH %s\r\n", cfg.Password)
		reader := bufio.NewReader(conn)
		_, _ = reader.ReadString('\n')
	}

	var latencies []float64
	startAll := time.Now()
	reader := bufio.NewReader(conn)

	for i := 0; i < iterations; i++ {
		t0 := time.Now()
		if cfg.CacheType == "redis" {
			fmt.Fprintf(conn, "PING\r\n")
			_, err = reader.ReadString('\n')
		} else {
			fmt.Fprintf(conn, "version\r\n")
			_, err = reader.ReadString('\n')
		}
		if err != nil {
			return nil, fmt.Errorf("benchmark probe failed: %w", err)
		}
		lat := float64(time.Since(t0).Microseconds()) / 1000.0
		latencies = append(latencies, lat)
	}

	totalDur := time.Since(startAll).Milliseconds()
	if totalDur == 0 {
		totalDur = 1
	}

	var sum, min, max float64
	min = 999999
	for _, l := range latencies {
		sum += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	avg := sum / float64(len(latencies))
	opsPerSec := float64(iterations) / (float64(totalDur) / 1000.0)

	return &BenchmarkResult{
		CacheType:       cfg.CacheType,
		Operations:      iterations,
		TotalDurationMs: totalDur,
		AvgLatencyMs:    roundFloat(avg, 2),
		MinLatencyMs:    roundFloat(min, 2),
		MaxLatencyMs:    roundFloat(max, 2),
		OpsPerSec:       roundFloat(opsPerSec, 1),
	}, nil
}

// ----------------------------------------------------------------------------
// WORDPRESS OBJECT CACHE INJECTION
// ----------------------------------------------------------------------------

func (m *Manager) configureWordPressObjectCacheLocked(cfg CacheConfig) error {
	wpConfigPath := filepath.Join(cfg.WebRoot, "wp-config.php")
	contentBytes, err := os.ReadFile(wpConfigPath)
	if err != nil {
		return ErrWPConfigNotFound
	}
	content := string(contentBytes)

	// Clean existing block
	content = stripHostvraCacheBlock(content)

	var sb strings.Builder
	sb.WriteString("/* Hostvra Object Cache Start */\n")
	sb.WriteString("define( 'WP_CACHE', true );\n")

	if cfg.CacheType == "redis" {
		if cfg.ConnectionType == "socket" {
			sb.WriteString("define( 'WP_REDIS_SCHEME', 'unix' );\n")
			sb.WriteString(fmt.Sprintf("define( 'WP_REDIS_PATH', '%s' );\n", cfg.SocketPath))
		} else {
			sb.WriteString("define( 'WP_REDIS_SCHEME', 'tcp' );\n")
			sb.WriteString("define( 'WP_REDIS_HOST', '127.0.0.1' );\n")
			sb.WriteString(fmt.Sprintf("define( 'WP_REDIS_PORT', %d );\n", cfg.Port))
		}
		if cfg.Password != "" {
			sb.WriteString(fmt.Sprintf("define( 'WP_REDIS_PASSWORD', '%s' );\n", cfg.Password))
		}
	} else if cfg.CacheType == "memcached" {
		if cfg.ConnectionType == "socket" {
			sb.WriteString(fmt.Sprintf("global $memcached_servers;\n$memcached_servers = array( array( '%s', 0 ) );\n", cfg.SocketPath))
		} else {
			sb.WriteString(fmt.Sprintf("global $memcached_servers;\n$memcached_servers = array( array( '127.0.0.1', %d ) );\n", cfg.Port))
		}
	}
	sb.WriteString("/* Hostvra Object Cache End */\n")

	// Inject block right before "/* That's all, stop editing!" or at top of php
	marker := "/* That's all, stop editing!"
	if strings.Contains(content, marker) {
		content = strings.Replace(content, marker, sb.String()+"\n"+marker, 1)
	} else if strings.Contains(content, "<?php") {
		content = strings.Replace(content, "<?php", "<?php\n\n"+sb.String(), 1)
	} else {
		content = sb.String() + "\n" + content
	}

	return os.WriteFile(wpConfigPath, []byte(content), 0644)
}

func (m *Manager) disableWordPressObjectCacheLocked(cfg CacheConfig) error {
	wpConfigPath := filepath.Join(cfg.WebRoot, "wp-config.php")
	contentBytes, err := os.ReadFile(wpConfigPath)
	if err != nil {
		return nil
	}
	cleaned := stripHostvraCacheBlock(string(contentBytes))
	return os.WriteFile(wpConfigPath, []byte(cleaned), 0644)
}

func stripHostvraCacheBlock(content string) string {
	re := regexp.MustCompile(`/\* Hostvra Object Cache Start \*/[\s\S]*?/\* Hostvra Object Cache End \*/\n?`)
	return re.ReplaceAllString(content, "")
}

// ----------------------------------------------------------------------------
// INTERNAL PROTOCOL HELPERS
// ----------------------------------------------------------------------------

func (m *Manager) dialCache(cfg CacheConfig, timeout time.Duration) (net.Conn, error) {
	if cfg.ConnectionType == "socket" && cfg.SocketPath != "" {
		return net.DialTimeout("unix", cfg.SocketPath, timeout)
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		if cfg.CacheType == "redis" {
			port = 6379
		} else {
			port = 11211
		}
	}
	return net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
}

func (m *Manager) queryRedisMetrics(conn net.Conn, password string) (*CacheMetrics, error) {
	reader := bufio.NewReader(conn)
	if password != "" {
		fmt.Fprintf(conn, "AUTH %s\r\n", password)
		_, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}

	fmt.Fprintf(conn, "INFO\r\n")
	metrics := &CacheMetrics{
		CacheType: "redis",
		Running:   true,
	}

	// Read RESP bulk string or lines
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "$") {
		return nil, fmt.Errorf("unexpected redis info response: %s", line)
	}

	for {
		l, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(l) == "" {
			break
		}
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") || !strings.Contains(l, ":") {
			continue
		}
		parts := strings.SplitN(l, ":", 2)
		k, v := parts[0], parts[1]
		switch k {
		case "uptime_in_seconds":
			metrics.UptimeSeconds, _ = strconv.ParseInt(v, 10, 64)
		case "connected_clients":
			metrics.ConnectedClients, _ = strconv.Atoi(v)
		case "used_memory":
			metrics.UsedMemoryBytes, _ = strconv.ParseInt(v, 10, 64)
		case "used_memory_human":
			metrics.UsedMemoryHuman = v
		case "maxmemory":
			metrics.MaxMemoryBytes, _ = strconv.ParseInt(v, 10, 64)
		case "keyspace_hits":
			metrics.Hits, _ = strconv.ParseInt(v, 10, 64)
		case "keyspace_misses":
			metrics.Misses, _ = strconv.ParseInt(v, 10, 64)
		}
	}

	totalOps := metrics.Hits + metrics.Misses
	if totalOps > 0 {
		metrics.HitRatePercent = roundFloat((float64(metrics.Hits)/float64(totalOps))*100, 2)
	}
	return metrics, nil
}

func (m *Manager) queryMemcachedMetrics(conn net.Conn) (*CacheMetrics, error) {
	fmt.Fprintf(conn, "stats\r\n")
	reader := bufio.NewReader(conn)
	metrics := &CacheMetrics{
		CacheType: "memcached",
		Running:   true,
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.HasPrefix(line, "END") {
			break
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[0] == "STAT" {
			k, v := parts[1], parts[2]
			switch k {
			case "uptime":
				metrics.UptimeSeconds, _ = strconv.ParseInt(v, 10, 64)
			case "curr_connections":
				metrics.ConnectedClients, _ = strconv.Atoi(v)
			case "bytes":
				metrics.UsedMemoryBytes, _ = strconv.ParseInt(v, 10, 64)
				metrics.UsedMemoryHuman = fmt.Sprintf("%.2fM", float64(metrics.UsedMemoryBytes)/(1024*1024))
			case "limit_maxbytes":
				metrics.MaxMemoryBytes, _ = strconv.ParseInt(v, 10, 64)
			case "curr_items":
				metrics.TotalKeys, _ = strconv.ParseInt(v, 10, 64)
			case "get_hits":
				metrics.Hits, _ = strconv.ParseInt(v, 10, 64)
			case "get_misses":
				metrics.Misses, _ = strconv.ParseInt(v, 10, 64)
			}
		}
	}

	totalOps := metrics.Hits + metrics.Misses
	if totalOps > 0 {
		metrics.HitRatePercent = roundFloat((float64(metrics.Hits)/float64(totalOps))*100, 2)
	}
	return metrics, nil
}

func parseRedisVersion(out string) string {
	re := regexp.MustCompile(`v=([0-9.]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) > 1 {
		return matches[1]
	}
	return "7.x"
}

func parseMemcachedVersion(out string) string {
	re := regexp.MustCompile(`memcached\s+([0-9.]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) > 1 {
		return matches[1]
	}
	return "1.6.x"
}

func isServiceActive(service string) bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", service)
	return cmd.Run() == nil
}

func isPortOpen(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

func roundFloat(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10.0
	}
	return float64(int(val*ratio+0.5)) / ratio
}
