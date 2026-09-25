package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"hostvra/api/internal/response"
)

type rateLimiterEntry struct {
	tokens     float64
	lastAccess time.Time
}

// RateLimiter provides a thread-safe token bucket rate limiter per IP address.
type RateLimiter struct {
	mu          sync.Mutex
	visitors    map[string]*rateLimiterEntry
	rate        float64 // tokens per second
	capacity    float64 // max burst capacity
	window      time.Duration
	cleanupStop chan struct{}
	message     string
}

// LoginRateLimiter is an alias for RateLimiter for backwards compatibility.
type LoginRateLimiter = RateLimiter

func NewRateLimiter(maxRequests int, window time.Duration, message string) *RateLimiter {
	if message == "" {
		message = "Too many requests. Please try again later."
	}
	limiter := &RateLimiter{
		visitors:    make(map[string]*rateLimiterEntry),
		rate:        float64(maxRequests) / window.Seconds(),
		capacity:    float64(maxRequests),
		window:      window,
		cleanupStop: make(chan struct{}),
		message:     message,
	}

	// Background cleanup routine to prevent memory leaks
	go limiter.cleanupLoop(10 * time.Minute)

	return limiter
}

func NewLoginRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return NewRateLimiter(maxRequests, window, "Too many login attempts. Please try again later.")
}

func (rl *LoginRateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, entry := range rl.visitors {
				if now.Sub(entry.lastAccess) > rl.window*2 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.cleanupStop:
			ticker.Stop()
			return
		}
	}
}

func (rl *LoginRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &rateLimiterEntry{
			tokens:     rl.capacity - 1,
			lastAccess: now,
		}
		return true
	}

	elapsed := now.Sub(entry.lastAccess).Seconds()
	entry.lastAccess = now

	entry.tokens += elapsed * rl.rate
	if entry.tokens > rl.capacity {
		entry.tokens = rl.capacity
	}

	if entry.tokens < 1.0 {
		return false
	}

	entry.tokens -= 1.0
	return true
}

func (rl *LoginRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !rl.allow(ip) {
			w.Header().Set("Retry-After", "60")
			response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", rl.message, nil, "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
