package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMiddleware_BearerAuth(t *testing.T) {
	jwtSecret := "super-secret-jwt-key-for-testing-purposes-12345"
	userID := uuid.New()
	orgID := uuid.New()

	tokens, _, err := GenerateTokenPair(userID, orgID, "test@hostvra.com", "admin", true, jwtSecret, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	mw := Middleware(jwtSecret)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if !ok || claims == nil {
			t.Fatalf("Expected valid claims in request context")
		}
		if claims.Email != "test@hostvra.com" {
			t.Fatalf("Expected email test@hostvra.com, got %s", claims.Email)
		}
		w.WriteHeader(http.StatusOK)
	})

	// 1. Missing auth
	reqNoAuth := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	rrNoAuth := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrNoAuth, reqNoAuth)
	if rrNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for missing token, got %d", rrNoAuth.Code)
	}

	// 2. Valid Bearer token
	reqValid := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	reqValid.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rrValid := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrValid, reqValid)
	if rrValid.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for valid Bearer token, got %d", rrValid.Code)
	}
}

func TestMiddleware_CookieAuth_CSRFProtection(t *testing.T) {
	jwtSecret := "super-secret-jwt-key-for-testing-purposes-12345"
	userID := uuid.New()
	orgID := uuid.New()

	tokens, _, err := GenerateTokenPair(userID, orgID, "user@tenant.com", "customer", false, jwtSecret, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	mw := Middleware(jwtSecret)
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// 1. Safe GET request with Cookie auth should succeed
	called = false
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/websites", nil)
	reqGet.AddCookie(&http.Cookie{Name: "access_token", Value: tokens.AccessToken})
	rrGet := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrGet, reqGet)
	if rrGet.Code != http.StatusOK || !called {
		t.Fatalf("Expected 200 OK for GET with Cookie auth, got %d", rrGet.Code)
	}

	// 2. State-changing POST with cross-site Sec-Fetch-Site must be blocked
	called = false
	reqCrossSite := httptest.NewRequest(http.MethodPost, "/api/v1/websites", nil)
	reqCrossSite.Host = "panel.hostvra.com"
	reqCrossSite.AddCookie(&http.Cookie{Name: "access_token", Value: tokens.AccessToken})
	reqCrossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	rrCrossSite := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrCrossSite, reqCrossSite)
	if rrCrossSite.Code != http.StatusForbidden || called {
		t.Fatalf("Expected 403 Forbidden for cross-site Sec-Fetch-Site, got %d", rrCrossSite.Code)
	}

	// 3. State-changing POST with evil Origin must be blocked
	called = false
	reqEvilOrigin := httptest.NewRequest(http.MethodPost, "/api/v1/websites", nil)
	reqEvilOrigin.Host = "panel.hostvra.com"
	reqEvilOrigin.AddCookie(&http.Cookie{Name: "access_token", Value: tokens.AccessToken})
	reqEvilOrigin.Header.Set("Origin", "https://evil-attacker.com")
	rrEvilOrigin := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrEvilOrigin, reqEvilOrigin)
	if rrEvilOrigin.Code != http.StatusForbidden || called {
		t.Fatalf("Expected 403 Forbidden for evil Origin, got %d", rrEvilOrigin.Code)
	}

	// 4. State-changing POST with evil Referer must be blocked
	called = false
	reqEvilReferer := httptest.NewRequest(http.MethodPost, "/api/v1/websites", nil)
	reqEvilReferer.Host = "panel.hostvra.com"
	reqEvilReferer.AddCookie(&http.Cookie{Name: "access_token", Value: tokens.AccessToken})
	reqEvilReferer.Header.Set("Referer", "https://attacker.site/phishing")
	rrEvilReferer := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrEvilReferer, reqEvilReferer)
	if rrEvilReferer.Code != http.StatusForbidden || called {
		t.Fatalf("Expected 403 Forbidden for evil Referer, got %d", rrEvilReferer.Code)
	}

	// 5. State-changing POST with matching Origin should succeed
	called = false
	reqValidOrigin := httptest.NewRequest(http.MethodPost, "/api/v1/websites", nil)
	reqValidOrigin.Host = "panel.hostvra.com"
	reqValidOrigin.AddCookie(&http.Cookie{Name: "access_token", Value: tokens.AccessToken})
	reqValidOrigin.Header.Set("Origin", "https://panel.hostvra.com")
	rrValidOrigin := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrValidOrigin, reqValidOrigin)
	if rrValidOrigin.Code != http.StatusOK || !called {
		t.Fatalf("Expected 200 OK for valid Origin, got %d", rrValidOrigin.Code)
	}

	// 6. State-changing POST with X-CSRF-Token should succeed even if cross-origin headers are present
	called = false
	reqCSRFToken := httptest.NewRequest(http.MethodPost, "/api/v1/websites", nil)
	reqCSRFToken.Host = "panel.hostvra.com"
	reqCSRFToken.AddCookie(&http.Cookie{Name: "access_token", Value: tokens.AccessToken})
	reqCSRFToken.Header.Set("X-CSRF-Token", "valid-anti-csrf-token")
	rrCSRFToken := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrCSRFToken, reqCSRFToken)
	if rrCSRFToken.Code != http.StatusOK || !called {
		t.Fatalf("Expected 200 OK with X-CSRF-Token header, got %d", rrCSRFToken.Code)
	}

	// 7. Bearer token auth is immune to CSRF checks (APIs called via Authorization header)
	called = false
	reqBearer := httptest.NewRequest(http.MethodPost, "/api/v1/websites", nil)
	reqBearer.Host = "panel.hostvra.com"
	reqBearer.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	reqBearer.Header.Set("Origin", "https://thirdparty-client.com")
	rrBearer := httptest.NewRecorder()
	mw(nextHandler).ServeHTTP(rrBearer, reqBearer)
	if rrBearer.Code != http.StatusOK || !called {
		t.Fatalf("Expected 200 OK for Bearer token despite external Origin, got %d", rrBearer.Code)
	}
}
