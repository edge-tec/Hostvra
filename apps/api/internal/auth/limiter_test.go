package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginRateLimiter(t *testing.T) {
	// Allow 2 requests per second
	limiter := NewLoginRateLimiter(2, time.Second)

	handler := limiter.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Request 1: should pass
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req1.RemoteAddr = "192.168.1.100:1234"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected request 1 status 200, got %d", w1.Code)
	}

	// Request 2: should pass
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req2.RemoteAddr = "192.168.1.100:1234"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected request 2 status 200, got %d", w2.Code)
	}

	// Request 3: should be rate limited (429)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req3.RemoteAddr = "192.168.1.100:1234"
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("expected request 3 status 429 Too Many Requests, got %d", w3.Code)
	}

	// Request from a different IP: should pass
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req4.RemoteAddr = "10.0.0.1:5678"
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("expected request 4 (different IP) status 200, got %d", w4.Code)
	}
}
