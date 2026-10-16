package ratecontrol

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Standard constants for Gmail rate control
const (
	DefaultMinCooldownDuration = 10 * time.Minute
	DefaultMaxBackoffDuration  = 60 * time.Minute
	DefaultInitialConcurrency  = 1
	DefaultMaxConcurrency      = 5
	DefaultRampUpSuccessCount  = 5
	DefaultRateDelay           = "1s"
)

// Regex patterns to identify Gmail 421 4.7.28 rate limiting responses
var (
	gmail421Regex = regexp.MustCompile(`(?i)(421[\s\-]4\.7\.28|unusual rate of mail originating from your SPF domain|UnsolicitedRateLimitError)`)
	perm5xxRegex  = regexp.MustCompile(`(?i)(^5\d\d|5\.\d\.\d|user unknown|recipient address rejected|mailbox unavailable|account has been disabled)`)
	temp4xxRegex  = regexp.MustCompile(`(?i)(^4\d\d|4\.\d\.\d|temporarily deferred|rate limit|try again later|connection timed out)`)
)

// ControllerConfig holds tuning parameters for outbound delivery control
type ControllerConfig struct {
	MinCooldownDuration time.Duration `json:"min_cooldown_duration"`
	MaxBackoffDuration  time.Duration `json:"max_backoff_duration"`
	InitialConcurrency  int           `json:"initial_concurrency"`
	MaxConcurrency      int           `json:"max_concurrency"`
	RampUpSuccessCount  int           `json:"ramp_up_success_count"`
	RateDelay           string        `json:"rate_delay"`
}

// DefaultConfig provides production-safe initial defaults
func DefaultConfig() ControllerConfig {
	return ControllerConfig{
		MinCooldownDuration: DefaultMinCooldownDuration,
		MaxBackoffDuration:  DefaultMaxBackoffDuration,
		InitialConcurrency:  DefaultInitialConcurrency,
		MaxConcurrency:      DefaultMaxConcurrency,
		RampUpSuccessCount:  DefaultRampUpSuccessCount,
		RateDelay:           DefaultRateDelay,
	}
}

// DomainRateState tracks rate limiting, cooldown, and concurrency for a domain
type DomainRateState struct {
	Domain               string     `json:"domain"`
	Destination          string     `json:"destination"` // e.g. "gmail.com"
	InCooldown           bool       `json:"in_cooldown"`
	CooldownUntil        time.Time  `json:"cooldown_until"`
	RemainingSeconds     int64      `json:"remaining_seconds"`
	BackoffTier          int        `json:"backoff_tier"`
	CurrentConcurrency   int        `json:"current_concurrency"`
	MaxAllowedConcurrency int       `json:"max_allowed_concurrency"`
	ConsecutiveSuccesses int        `json:"consecutive_successes"`
	Last421Response      string     `json:"last_421_response,omitempty"`
	Last421At            *time.Time `json:"last_421_at,omitempty"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty"`
	LastDeliveryStatus   string     `json:"last_delivery_status"`
	TotalDeferred        int64      `json:"total_deferred"`
	TotalDelivered       int64      `json:"total_delivered"`
	TotalRateLimitHits   int64      `json:"total_rate_limit_hits"`
}

// DeliveryController coordinates safe rate-limited outbound dispatching
type DeliveryController struct {
	mu     sync.RWMutex
	cfg    ControllerConfig
	states map[string]*DomainRateState // key: "domain:destination"
}

// NewDeliveryController creates a thread-safe delivery controller
func NewDeliveryController(cfg ControllerConfig) *DeliveryController {
	if cfg.MinCooldownDuration <= 0 {
		cfg.MinCooldownDuration = DefaultMinCooldownDuration
	}
	if cfg.MaxBackoffDuration <= 0 {
		cfg.MaxBackoffDuration = DefaultMaxBackoffDuration
	}
	if cfg.InitialConcurrency <= 0 {
		cfg.InitialConcurrency = DefaultInitialConcurrency
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = DefaultMaxConcurrency
	}
	if cfg.RampUpSuccessCount <= 0 {
		cfg.RampUpSuccessCount = DefaultRampUpSuccessCount
	}

	return &DeliveryController{
		cfg:    cfg,
		states: make(map[string]*DomainRateState),
	}
}

// GlobalController provides a singleton instance for in-process rate control
var (
	globalController     *DeliveryController
	globalControllerOnce sync.Once
)

// GetGlobalController returns the shared singleton delivery controller
func GetGlobalController() *DeliveryController {
	globalControllerOnce.Do(func() {
		globalController = NewDeliveryController(DefaultConfig())
	})
	return globalController
}

// ClassifySMTPResponse analyzes an SMTP status code and response body
func ClassifySMTPResponse(statusCode string, responseText string) (is421RateLimit bool, isPermanent5xx bool, isTemporary4xx bool, reason string) {
	combined := fmt.Sprintf("%s %s", statusCode, responseText)

	if gmail421Regex.MatchString(combined) {
		return true, false, true, "Gmail 421 4.7.28: Unusual rate of mail originating from SPF domain"
	}

	if strings.HasPrefix(statusCode, "5") || perm5xxRegex.MatchString(combined) {
		return false, true, false, "Permanent delivery failure (5xx)"
	}

	if strings.HasPrefix(statusCode, "4") || temp4xxRegex.MatchString(combined) {
		return false, false, true, "Temporary delivery failure (4xx)"
	}

	return false, false, false, "Success or unrecognized status"
}

// CalculateBackoff computes exponential backoff with jitter
// Tier 1: 10m (600s) + jitter
// Tier 2: 20m (1200s) + jitter
// Tier 3: 40m (2400s) + jitter
// Tier 4+: capped at MaxBackoffDuration (default 60m)
func (c *DeliveryController) CalculateBackoff(tier int) time.Duration {
	base := c.cfg.MinCooldownDuration
	if tier <= 1 {
		return addJitter(base)
	}

	// Exponential multiplier: 2^(tier-1)
	multiplier := 1 << (tier - 1)
	backoff := time.Duration(multiplier) * base
	if backoff > c.cfg.MaxBackoffDuration {
		backoff = c.cfg.MaxBackoffDuration
	}

	return addJitter(backoff)
}

// addJitter adds up to 10% random jitter to avoid synchronized retry storms
func addJitter(d time.Duration) time.Duration {
	jitterMaxMs := d.Milliseconds() / 10
	if jitterMaxMs <= 0 {
		jitterMaxMs = 1
	}

	n, err := rand.Int(rand.Reader, big.NewInt(jitterMaxMs))
	if err != nil {
		return d
	}
	return d + time.Duration(n.Int64())*time.Millisecond
}

// RecordDeliveryEvent processes an outbound delivery attempt and updates state
func (c *DeliveryController) RecordDeliveryEvent(domain, destination, statusCode, responseText string) *DomainRateState {
	c.mu.Lock()
	defer c.mu.Unlock()

	domain = strings.ToLower(strings.TrimSpace(domain))
	destination = strings.ToLower(strings.TrimSpace(destination))
	key := fmt.Sprintf("%s:%s", domain, destination)

	state, exists := c.states[key]
	if !exists {
		state = &DomainRateState{
			Domain:                domain,
			Destination:           destination,
			InCooldown:            false,
			BackoffTier:           0,
			CurrentConcurrency:    c.cfg.InitialConcurrency,
			MaxAllowedConcurrency: c.cfg.InitialConcurrency,
			LastDeliveryStatus:    statusCode,
		}
		c.states[key] = state
	}

	now := time.Now()
	is421, is5xx, is4xx, _ := ClassifySMTPResponse(statusCode, responseText)

	if is421 {
		// Rate limit event detected
		state.InCooldown = true
		state.BackoffTier++
		backoff := c.CalculateBackoff(state.BackoffTier)
		state.CooldownUntil = now.Add(backoff)
		state.RemainingSeconds = int64(backoff.Seconds())
		state.MaxAllowedConcurrency = 1
		state.CurrentConcurrency = 1
		state.ConsecutiveSuccesses = 0
		state.Last421Response = responseText
		state.Last421At = &now
		state.LastDeliveryStatus = "421_rate_limited"
		state.TotalRateLimitHits++
		state.TotalDeferred++
	} else if strings.HasPrefix(statusCode, "2") {
		// Successful delivery
		state.TotalDelivered++
		state.LastSuccessAt = &now
		state.LastDeliveryStatus = "delivered"
		state.ConsecutiveSuccesses++

		// If cooldown has elapsed, clear cooldown flag
		if state.InCooldown && now.After(state.CooldownUntil) {
			state.InCooldown = false
			state.RemainingSeconds = 0
			state.BackoffTier = 0
		}

		// Gradually increase concurrency after consecutive successful deliveries
		if state.ConsecutiveSuccesses >= c.cfg.RampUpSuccessCount && state.MaxAllowedConcurrency < c.cfg.MaxConcurrency {
			state.MaxAllowedConcurrency++
			state.ConsecutiveSuccesses = 0 // Reset for next tier
		}
	} else if is5xx {
		state.LastDeliveryStatus = "permanent_failure"
	} else if is4xx {
		state.TotalDeferred++
		state.LastDeliveryStatus = "deferred"
	}

	c.updateRemaining(state, now)
	return state
}

// GetState returns the current rate state for a domain and destination
func (c *DeliveryController) GetState(domain, destination string) *DomainRateState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	domain = strings.ToLower(strings.TrimSpace(domain))
	destination = strings.ToLower(strings.TrimSpace(destination))
	key := fmt.Sprintf("%s:%s", domain, destination)

	state, exists := c.states[key]
	if !exists {
		return &DomainRateState{
			Domain:                domain,
			Destination:           destination,
			InCooldown:            false,
			CurrentConcurrency:    c.cfg.InitialConcurrency,
			MaxAllowedConcurrency: c.cfg.InitialConcurrency,
			LastDeliveryStatus:    "unknown",
		}
	}

	now := time.Now()
	// Return copy
	clone := *state
	c.updateRemaining(&clone, now)
	return &clone
}

// GetAllStates returns all tracked domain rate states
func (c *DeliveryController) GetAllStates() []*DomainRateState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	list := make([]*DomainRateState, 0, len(c.states))
	for _, s := range c.states {
		clone := *s
		c.updateRemaining(&clone, now)
		list = append(list, &clone)
	}
	return list
}

// ResetCooldown manually clears the cooldown state for a domain
func (c *DeliveryController) ResetCooldown(domain, destination string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", strings.ToLower(domain), strings.ToLower(destination))
	if state, ok := c.states[key]; ok {
		state.InCooldown = false
		state.CooldownUntil = time.Time{}
		state.RemainingSeconds = 0
		state.BackoffTier = 0
		state.MaxAllowedConcurrency = c.cfg.InitialConcurrency
	}
}

// CanDeliver checks whether delivery to destination is permitted or in cooldown
func (c *DeliveryController) CanDeliver(domain, destination string) (allowed bool, waitSeconds int64, reason string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", strings.ToLower(domain), strings.ToLower(destination))
	state, exists := c.states[key]
	if !exists {
		return true, 0, ""
	}

	now := time.Now()
	if state.InCooldown && now.Before(state.CooldownUntil) {
		rem := int64(state.CooldownUntil.Sub(now).Seconds())
		return false, rem, fmt.Sprintf("Domain %s is in rate-limit cooldown for %s until %s (%ds remaining)",
			domain, destination, state.CooldownUntil.Format(time.RFC3339), rem)
	}

	return true, 0, ""
}

// SafeFlushQueue flushes deferred messages only if target destination is not in cooldown
func (c *DeliveryController) SafeFlushQueue(domain, destination string) error {
	allowed, waitSeconds, reason := c.CanDeliver(domain, destination)
	if !allowed {
		return fmt.Errorf("queue flush blocked by delivery controller: %s (retry in %ds)", reason, waitSeconds)
	}

	if _, err := exec.LookPath("postqueue"); err != nil {
		return nil
	}

	cmd := exec.Command("postqueue", "-f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postqueue -f failed: %s (%w)", string(out), err)
	}

	return nil
}

func (c *DeliveryController) updateRemaining(s *DomainRateState, now time.Time) {
	if s.InCooldown {
		if now.Before(s.CooldownUntil) {
			s.RemainingSeconds = int64(s.CooldownUntil.Sub(now).Seconds())
		} else {
			s.InCooldown = false
			s.RemainingSeconds = 0
		}
	} else {
		s.RemainingSeconds = 0
	}
}

// ----------------------------------------------------------------------------
// MAILBOX-LEVEL OUTBOUND RATE LIMITER
// ----------------------------------------------------------------------------

// MailboxUsage tracks outbound counters for a single mailbox
type MailboxUsage struct {
	Mailbox         string    `json:"mailbox"`
	HourWindowStart time.Time `json:"hour_window_start"`
	DayWindowStart  time.Time `json:"day_window_start"`
	HourSent        int       `json:"hour_sent"`
	DaySent         int       `json:"day_sent"`
	LastSentAt      time.Time `json:"last_sent_at"`
}

// MailboxUsageStats provides calculated limits and remaining capacity
type MailboxUsageStats struct {
	Mailbox       string    `json:"mailbox"`
	HourSent      int       `json:"hour_sent"`
	DaySent       int       `json:"day_sent"`
	HourLimit     int       `json:"hour_limit"`
	DayLimit      int       `json:"day_limit"`
	HourRemaining int       `json:"hour_remaining"`
	DayRemaining  int       `json:"day_remaining"`
	IsRateLimited bool      `json:"is_rate_limited"`
	LimitReason   string    `json:"limit_reason,omitempty"`
	RetryAfterSec int       `json:"retry_after_sec"`
	ResetHourAt   time.Time `json:"reset_hour_at"`
	ResetDayAt    time.Time `json:"reset_day_at"`
}

// MailboxRateLimiter controls outbound sending volumes per mailbox
type MailboxRateLimiter struct {
	mu    sync.RWMutex
	usage map[string]*MailboxUsage
}

var (
	globalMailboxLimiter     *MailboxRateLimiter
	globalMailboxLimiterOnce sync.Once
)

// GetGlobalMailboxRateLimiter returns the process singleton mailbox rate limiter
func GetGlobalMailboxRateLimiter() *MailboxRateLimiter {
	globalMailboxLimiterOnce.Do(func() {
		globalMailboxLimiter = NewMailboxRateLimiter()
	})
	return globalMailboxLimiter
}

// NewMailboxRateLimiter creates a new thread-safe mailbox rate limiter
func NewMailboxRateLimiter() *MailboxRateLimiter {
	return &MailboxRateLimiter{
		usage: make(map[string]*MailboxUsage),
	}
}

// CanMailboxSend checks whether a mailbox is within its hourly and daily quotas
func (m *MailboxRateLimiter) CanMailboxSend(mailbox string, limitHour, limitDay int) (allowed bool, currentHour int, currentDay int, retryAfterSec int, reason string) {
	if limitHour <= 0 {
		limitHour = 100
	}
	if limitDay <= 0 {
		limitDay = 1000
	}

	key := strings.ToLower(strings.TrimSpace(mailbox))
	if key == "" {
		return true, 0, 0, 0, ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	u, exists := m.usage[key]
	if !exists {
		m.usage[key] = &MailboxUsage{
			Mailbox:         key,
			HourWindowStart: now,
			DayWindowStart:  now,
			HourSent:        0,
			DaySent:         0,
			LastSentAt:      now,
		}
		return true, 0, 0, 0, ""
	}

	m.rollWindows(u, now)

	resetHourAt := u.HourWindowStart.Add(time.Hour)
	resetDayAt := u.DayWindowStart.Add(24 * time.Hour)

	if u.HourSent >= limitHour {
		remSec := int(resetHourAt.Sub(now).Seconds())
		if remSec < 1 {
			remSec = 1
		}
		return false, u.HourSent, u.DaySent, remSec, fmt.Sprintf("Hourly outbound rate limit exceeded (%d/%d sent). Reset in %ds", u.HourSent, limitHour, remSec)
	}

	if u.DaySent >= limitDay {
		remSec := int(resetDayAt.Sub(now).Seconds())
		if remSec < 1 {
			remSec = 1
		}
		return false, u.HourSent, u.DaySent, remSec, fmt.Sprintf("Daily outbound rate limit exceeded (%d/%d sent). Reset in %ds", u.DaySent, limitDay, remSec)
	}

	return true, u.HourSent, u.DaySent, 0, ""
}

// RecordMailboxSent records an outbound email sent by a mailbox
func (m *MailboxRateLimiter) RecordMailboxSent(mailbox string) {
	key := strings.ToLower(strings.TrimSpace(mailbox))
	if key == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	u, exists := m.usage[key]
	if !exists {
		u = &MailboxUsage{
			Mailbox:         key,
			HourWindowStart: now,
			DayWindowStart:  now,
			HourSent:        1,
			DaySent:         1,
			LastSentAt:      now,
		}
		m.usage[key] = u
		return
	}

	m.rollWindows(u, now)
	u.HourSent++
	u.DaySent++
	u.LastSentAt = now
}

// GetMailboxUsage returns current usage counters and calculated limits
func (m *MailboxRateLimiter) GetMailboxUsage(mailbox string, limitHour, limitDay int) *MailboxUsageStats {
	if limitHour <= 0 {
		limitHour = 100
	}
	if limitDay <= 0 {
		limitDay = 1000
	}

	key := strings.ToLower(strings.TrimSpace(mailbox))

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	u, exists := m.usage[key]
	if !exists {
		u = &MailboxUsage{
			Mailbox:         key,
			HourWindowStart: now,
			DayWindowStart:  now,
			HourSent:        0,
			DaySent:         0,
			LastSentAt:      now,
		}
		m.usage[key] = u
	} else {
		m.rollWindows(u, now)
	}

	resetHourAt := u.HourWindowStart.Add(time.Hour)
	resetDayAt := u.DayWindowStart.Add(24 * time.Hour)

	hourRem := limitHour - u.HourSent
	if hourRem < 0 {
		hourRem = 0
	}
	dayRem := limitDay - u.DaySent
	if dayRem < 0 {
		dayRem = 0
	}

	isLimited := false
	reason := ""
	retryAfter := 0

	if u.HourSent >= limitHour {
		isLimited = true
		retryAfter = int(resetHourAt.Sub(now).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		reason = fmt.Sprintf("Hourly outbound rate limit exceeded (%d/%d sent)", u.HourSent, limitHour)
	} else if u.DaySent >= limitDay {
		isLimited = true
		retryAfter = int(resetDayAt.Sub(now).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		reason = fmt.Sprintf("Daily outbound rate limit exceeded (%d/%d sent)", u.DaySent, limitDay)
	}

	return &MailboxUsageStats{
		Mailbox:       key,
		HourSent:      u.HourSent,
		DaySent:       u.DaySent,
		HourLimit:     limitHour,
		DayLimit:      limitDay,
		HourRemaining: hourRem,
		DayRemaining:  dayRem,
		IsRateLimited: isLimited,
		LimitReason:   reason,
		RetryAfterSec: retryAfter,
		ResetHourAt:   resetHourAt,
		ResetDayAt:    resetDayAt,
	}
}

// ResetMailboxLimit resets usage counters for a specific mailbox
func (m *MailboxRateLimiter) ResetMailboxLimit(mailbox string) {
	key := strings.ToLower(strings.TrimSpace(mailbox))
	if key == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.usage[key] = &MailboxUsage{
		Mailbox:         key,
		HourWindowStart: now,
		DayWindowStart:  now,
		HourSent:        0,
		DaySent:         0,
		LastSentAt:      now,
	}
}

func (m *MailboxRateLimiter) rollWindows(u *MailboxUsage, now time.Time) {
	if now.Sub(u.HourWindowStart) >= time.Hour {
		u.HourWindowStart = now
		u.HourSent = 0
	}
	if now.Sub(u.DayWindowStart) >= 24*time.Hour {
		u.DayWindowStart = now
		u.DaySent = 0
	}
}

