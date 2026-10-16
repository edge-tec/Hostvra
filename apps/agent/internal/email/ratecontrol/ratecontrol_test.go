package ratecontrol

import (
	"strings"
	"testing"
	"time"
)

func TestClassifySMTPResponse(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		msg            string
		expected421    bool
		expected5xx    bool
		expected4xx    bool
	}{
		{
			name:        "Gmail 421 4.7.28 Rate Limit Exact",
			code:        "421",
			msg:         "4.7.28 Gmail has detected an unusual rate of mail originating from your SPF domain [onlyflrt.net].",
			expected421: true,
			expected5xx: false,
			expected4xx: true,
		},
		{
			name:        "Gmail 421 Multi-line String",
			code:        "421-4.7.28",
			msg:         "Gmail has detected an unusual rate of mail originating from your SPF domain [onlyflrt.net]. To protect our users from spam, mail originating from your domain has been temporarily rate limited.",
			expected421: true,
			expected5xx: false,
			expected4xx: true,
		},
		{
			name:        "Permanent Recipient Rejection 550",
			code:        "550",
			msg:         "5.1.1 <invalid@gmail.com>: Recipient address rejected: User unknown",
			expected421: false,
			expected5xx: true,
			expected4xx: false,
		},
		{
			name:        "Generic Temporary Failure 451",
			code:        "451",
			msg:         "4.3.0 Mail system error, try again later",
			expected421: false,
			expected5xx: false,
			expected4xx: true,
		},
		{
			name:        "Standard Delivery Success 250",
			code:        "250",
			msg:         "2.0.0 OK 1729112345 y4-20020a05620a210400b007a12345abcd.10 - gsmtp",
			expected421: false,
			expected5xx: false,
			expected4xx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is421, is5xx, is4xx, _ := ClassifySMTPResponse(tt.code, tt.msg)
			if is421 != tt.expected421 {
				t.Errorf("expected is421=%v, got %v", tt.expected421, is421)
			}
			if is5xx != tt.expected5xx {
				t.Errorf("expected is5xx=%v, got %v", tt.expected5xx, is5xx)
			}
			if is4xx != tt.expected4xx {
				t.Errorf("expected is4xx=%v, got %v", tt.expected4xx, is4xx)
			}
		})
	}
}

func TestCooldownAndBackoffProgression(t *testing.T) {
	cfg := ControllerConfig{
		MinCooldownDuration: 10 * time.Minute,
		MaxBackoffDuration:  60 * time.Minute,
		InitialConcurrency:  1,
		MaxConcurrency:      5,
		RampUpSuccessCount:  5,
	}
	ctrl := NewDeliveryController(cfg)

	domain := "onlyflrt.net"
	dest := "gmail.com"

	// Initial State: Can deliver
	allowed, _, _ := ctrl.CanDeliver(domain, dest)
	if !allowed {
		t.Fatalf("Domain should initially be allowed to deliver")
	}

	// 1. Encounter 421 Rate Limit
	state := ctrl.RecordDeliveryEvent(domain, dest, "421", "4.7.28 Gmail has detected an unusual rate of mail originating from your SPF domain [onlyflrt.net]")
	if !state.InCooldown {
		t.Errorf("Expected InCooldown to be true after 421 event")
	}
	if state.BackoffTier != 1 {
		t.Errorf("Expected BackoffTier 1, got %d", state.BackoffTier)
	}
	if state.MaxAllowedConcurrency != 1 {
		t.Errorf("Expected concurrency to drop to 1, got %d", state.MaxAllowedConcurrency)
	}
	if state.RemainingSeconds < 590 || state.RemainingSeconds > 670 {
		t.Errorf("Expected tier 1 cooldown ~600s with jitter, got %d", state.RemainingSeconds)
	}

	// CanDeliver must be false during cooldown
	allowed, waitSec, reason := ctrl.CanDeliver(domain, dest)
	if allowed {
		t.Errorf("Delivery must be blocked during cooldown")
	}
	if waitSec <= 0 || !strings.Contains(reason, "cooldown") {
		t.Errorf("Expected valid waitSec and cooldown reason, got %d, %s", waitSec, reason)
	}

	// Other destinations should NOT be blocked!
	allowedYahoo, _, _ := ctrl.CanDeliver(domain, "yahoo.com")
	if !allowedYahoo {
		t.Errorf("Yahoo delivery should NOT be blocked when only Gmail is rate-limited")
	}

	// 2. Encounter second 421 (Tier 2 backoff: ~20m)
	state2 := ctrl.RecordDeliveryEvent(domain, dest, "421", "4.7.28 Gmail unusual rate")
	if state2.BackoffTier != 2 {
		t.Errorf("Expected BackoffTier 2, got %d", state2.BackoffTier)
	}
	if state2.RemainingSeconds < 1180 || state2.RemainingSeconds > 1350 {
		t.Errorf("Expected tier 2 cooldown ~1200s with jitter, got %d", state2.RemainingSeconds)
	}

	// 3. Reset cooldown
	ctrl.ResetCooldown(domain, dest)
	allowedAfterReset, _, _ := ctrl.CanDeliver(domain, dest)
	if !allowedAfterReset {
		t.Errorf("Expected delivery allowed after manual cooldown reset")
	}
}

func TestGradualConcurrencyRampUp(t *testing.T) {
	cfg := ControllerConfig{
		MinCooldownDuration: 10 * time.Minute,
		MaxBackoffDuration:  60 * time.Minute,
		InitialConcurrency:  1,
		MaxConcurrency:      3,
		RampUpSuccessCount:  3,
	}
	ctrl := NewDeliveryController(cfg)

	domain := "onlyflrt.net"
	dest := "gmail.com"

	state := ctrl.GetState(domain, dest)
	if state.MaxAllowedConcurrency != 1 {
		t.Fatalf("Expected initial concurrency 1, got %d", state.MaxAllowedConcurrency)
	}

	// 2 successes - still concurrency 1
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	state = ctrl.GetState(domain, dest)
	if state.MaxAllowedConcurrency != 1 {
		t.Errorf("Expected concurrency 1 before reaching ramp threshold, got %d", state.MaxAllowedConcurrency)
	}

	// 3rd success - reaches threshold, increases to 2
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	state = ctrl.GetState(domain, dest)
	if state.MaxAllowedConcurrency != 2 {
		t.Errorf("Expected concurrency ramp up to 2, got %d", state.MaxAllowedConcurrency)
	}

	// 3 more successes - increases to 3 (MaxConcurrency)
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	state = ctrl.GetState(domain, dest)
	if state.MaxAllowedConcurrency != 3 {
		t.Errorf("Expected concurrency ramp up to 3, got %d", state.MaxAllowedConcurrency)
	}

	// Further successes do not exceed MaxConcurrency (3)
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	ctrl.RecordDeliveryEvent(domain, dest, "250", "2.0.0 OK")
	state = ctrl.GetState(domain, dest)
	if state.MaxAllowedConcurrency != 3 {
		t.Errorf("Concurrency should not exceed MaxConcurrency 3, got %d", state.MaxAllowedConcurrency)
	}

	// Immediate drop back to 1 if 421 occurs!
	ctrl.RecordDeliveryEvent(domain, dest, "421", "4.7.28 rate limit")
	state = ctrl.GetState(domain, dest)
	if state.MaxAllowedConcurrency != 1 {
		t.Errorf("Concurrency must immediately reset to 1 on 421, got %d", state.MaxAllowedConcurrency)
	}
}

func TestSafeFlushQueueBlockedDuringCooldown(t *testing.T) {
	ctrl := NewDeliveryController(DefaultConfig())
	domain := "onlyflrt.net"
	dest := "gmail.com"

	// Put in cooldown
	ctrl.RecordDeliveryEvent(domain, dest, "421", "4.7.28 unusual rate")

	err := ctrl.SafeFlushQueue(domain, dest)
	if err == nil {
		t.Errorf("SafeFlushQueue must return error and block flush during cooldown")
	}
	if !strings.Contains(err.Error(), "blocked by delivery controller") {
		t.Errorf("Expected flush blocked error message, got: %v", err)
	}
}
