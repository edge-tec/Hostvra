package iputil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoPublicIPDetected = errors.New("no public IPv4 address detected on server interfaces or via external discovery")
	ErrInvalidPublicIP    = errors.New("invalid public IPv4 address")
)

// Private CIDR ranges per RFC 1918, RFC 3927 (link-local), RFC 6598 (CGNAT), etc.
var privateCIDRs []*net.IPNet

func init() {
	rawCIDRs := []string{
		"10.0.0.0/8",      // RFC 1918 Private
		"172.16.0.0/12",   // RFC 1918 Private
		"192.168.0.0/16",  // RFC 1918 Private
		"127.0.0.0/8",     // RFC 1122 Loopback
		"0.0.0.0/8",       // RFC 1122 Current network ("this" host)
		"169.254.0.0/16",  // RFC 3927 Link-Local
		"100.64.0.0/10",   // RFC 6598 Shared Address Space (Carrier-Grade NAT)
		"192.0.0.0/24",    // RFC 6890 IETF Protocol Assignments
		"192.0.2.0/24",    // RFC 5737 Documentation (TEST-NET-1)
		"198.18.0.0/15",   // RFC 2544 Network Interconnect Device Benchmark
		"198.51.100.0/24", // RFC 5737 Documentation (TEST-NET-2)
		"203.0.113.0/24",  // RFC 5737 Documentation (TEST-NET-3)
		"224.0.0.0/4",     // RFC 5771 Multicast
		"240.0.0.0/4",     // RFC 1112 Reserved
		"255.255.255.255/32", // Broadcast
	}

	for _, cidr := range rawCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			privateCIDRs = append(privateCIDRs, ipNet)
		}
	}
}

// IsPublicIPv4 checks if the input is a valid, publicly routable IPv4 address.
// Rejects 127.0.0.1, localhost, 0.0.0.0, private RFC1918 addresses, link-local, multicast, etc.
func IsPublicIPv4(ipStr string) bool {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" || strings.EqualFold(ipStr, "localhost") {
		return false
	}

	parsed := net.ParseIP(ipStr)
	if parsed == nil {
		return false
	}

	// Must be IPv4
	ipv4 := parsed.To4()
	if ipv4 == nil {
		return false
	}

	// Check loopback, unspecified, multicast
	if parsed.IsLoopback() || parsed.IsUnspecified() || parsed.IsMulticast() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return false
	}

	// Check against reserved / private CIDRs
	for _, cidr := range privateCIDRs {
		if cidr.Contains(parsed) {
			return false
		}
	}

	return true
}

// ValidatePublicIPv4 returns a detailed human-friendly error explaining why an IP is invalid.
func ValidatePublicIPv4(ipStr string) error {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	if strings.EqualFold(ipStr, "localhost") {
		return fmt.Errorf("'localhost' is a local loopback hostname, not a valid public IPv4 address")
	}

	parsed := net.ParseIP(ipStr)
	if parsed == nil {
		return fmt.Errorf("'%s' is not a valid IPv4 address", ipStr)
	}

	ipv4 := parsed.To4()
	if ipv4 == nil {
		return fmt.Errorf("'%s' is an IPv6 address; mail DNS A record requires an IPv4 address", ipStr)
	}

	if parsed.IsLoopback() || strings.HasPrefix(ipStr, "127.") {
		return fmt.Errorf("'%s' is a loopback address and cannot be used as a public mail server IP", ipStr)
	}

	if parsed.IsUnspecified() || ipStr == "0.0.0.0" {
		return fmt.Errorf("0.0.0.0 is an unspecified address and cannot be used as a public mail server IP")
	}

	if parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() || strings.HasPrefix(ipStr, "169.254.") {
		return fmt.Errorf("'%s' is a link-local address and cannot be routed across the public internet", ipStr)
	}

	// Check private RFC 1918
	if strings.HasPrefix(ipStr, "10.") ||
		strings.HasPrefix(ipStr, "192.168.") ||
		(strings.HasPrefix(ipStr, "172.") && isRFC1918ClassB(parsed)) {
		return fmt.Errorf("'%s' is a private RFC1918 network address and cannot be routed across the public internet", ipStr)
	}

	// Check other reserved
	for _, cidr := range privateCIDRs {
		if cidr.Contains(parsed) {
			return fmt.Errorf("'%s' is a reserved/non-routable IP address (%s)", ipStr, cidr.String())
		}
	}

	return nil
}

func isRFC1918ClassB(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31
}

// Thread-safe detection cache
type ipCache struct {
	mu        sync.RWMutex
	ip        string
	timestamp time.Time
	ttl       time.Duration
}

var globalCache = &ipCache{
	ttl: 5 * time.Minute,
}

// External discovery endpoints (fallback for NAT / Cloud VMs)
var defaultDiscoveryEndpoints = []string{
	"https://api.ipify.org",
	"https://icanhazip.com",
	"https://ifconfig.me/ip",
	"https://checkip.amazonaws.com",
}

// DetectPublicIPv4 detects the server's real public IPv4 address.
// 1. Checks local non-loopback network interfaces first (0ms overhead if public IP is bound directly).
// 2. If behind NAT (private interface), queries external secure discovery endpoints.
// 3. Caches result for 5 minutes.
func DetectPublicIPv4(ctx context.Context) (string, error) {
	return detectPublicIPv4Internal(ctx, false)
}

// ForceDetectPublicIPv4 bypasses cache and re-queries interfaces & discovery endpoints.
func ForceDetectPublicIPv4(ctx context.Context) (string, error) {
	return detectPublicIPv4Internal(ctx, true)
}

func detectPublicIPv4Internal(ctx context.Context, force bool) (string, error) {
	globalCache.mu.RLock()
	cachedIP := globalCache.ip
	validUntil := globalCache.timestamp.Add(globalCache.ttl)
	globalCache.mu.RUnlock()

	if !force && cachedIP != "" && time.Now().Before(validUntil) && IsPublicIPv4(cachedIP) {
		return cachedIP, nil
	}

	// 1. Try local network interfaces
	if ip, err := detectFromLocalInterfaces(); err == nil && IsPublicIPv4(ip) {
		setCache(ip)
		return ip, nil
	}

	// 2. Try external discovery endpoints (NAT / Cloud VMs)
	if ip, err := detectFromDiscoveryEndpoints(ctx); err == nil && IsPublicIPv4(ip) {
		setCache(ip)
		return ip, nil
	}

	// If cache still has a valid previous IP, use it as fallback
	if cachedIP != "" && IsPublicIPv4(cachedIP) {
		return cachedIP, nil
	}

	return "", ErrNoPublicIPDetected
}

func setCache(ip string) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.ip = ip
	globalCache.timestamp = time.Now()
}

// detectFromLocalInterfaces inspects physical / virtual network adapters for a public IP
func detectFromLocalInterfaces() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, ifi := range interfaces {
		// Skip loopback and down interfaces
		if ifi.Flags&net.FlagLoopback != 0 || ifi.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil {
				continue
			}

			ipv4 := ip.To4()
			if ipv4 == nil {
				continue
			}

			ipStr := ipv4.String()
			if IsPublicIPv4(ipStr) {
				return ipStr, nil
			}
		}
	}

	return "", errors.New("no public IPv4 address found on local interfaces")
}

// detectFromDiscoveryEndpoints queries secure external echo endpoints with a short timeout
func detectFromDiscoveryEndpoints(ctx context.Context) (string, error) {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	for _, url := range defaultDiscoveryEndpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Hostvra-Mail-IP-Detector/1.0")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
		_ = resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		discovered := strings.TrimSpace(string(body))
		if IsPublicIPv4(discovered) {
			return discovered, nil
		}
	}

	return "", errors.New("all public IP discovery endpoints timed out or failed")
}
