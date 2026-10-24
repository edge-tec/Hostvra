package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"hostvra/api/internal/response"
)

type contextKey string

const (
	UserContextKey contextKey = "hostvra_user_claims"
)

func validateCSRF(r *http.Request) bool {
	// 1. Modern browser Sec-Fetch-Site protection
	if secSite := r.Header.Get("Sec-Fetch-Site"); secSite == "cross-site" {
		return false
	}

	// 2. Explicit anti-CSRF token header
	if csrfToken := r.Header.Get("X-CSRF-Token"); csrfToken != "" {
		return true
	}

	// 3. Origin header validation
	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err == nil {
			if strings.EqualFold(u.Host, r.Host) {
				return true
			}
			h := u.Hostname()
			if h == "localhost" || h == "127.0.0.1" || h == "::1" {
				return true
			}
		}
		return false
	}

	// 4. Referer header validation
	referer := r.Header.Get("Referer")
	if referer != "" {
		u, err := url.Parse(referer)
		if err == nil {
			if strings.EqualFold(u.Host, r.Host) {
				return true
			}
			h := u.Hostname()
			if h == "localhost" || h == "127.0.0.1" || h == "::1" {
				return true
			}
		}
		return false
	}

	// If non-browser client without browser origin headers, allow
	return true
}

func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			isCookieAuth := false
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenStr = parts[1]
				} else {
					response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format", nil, "")
					return
				}
			} else {
				// Security: Accept token in URL query parameter strictly for WebSocket handshakes
				// (due to browser JS WebSocket API header limitations). REST APIs must use Bearer header or Cookie.
				isWebSocket := strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || strings.Contains(r.URL.Path, "/ws")
				if isWebSocket {
					tokenStr = r.URL.Query().Get("token")
				}
				if tokenStr == "" {
					if c, err := r.Cookie("hostvra_token"); err == nil && c != nil {
						tokenStr = c.Value
						isCookieAuth = true
					} else if c, err := r.Cookie("access_token"); err == nil && c != nil {
						tokenStr = c.Value
						isCookieAuth = true
					}
				}
			}

			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header", nil, "")
				return
			}

			// CSRF defense for cookie-authenticated state-changing requests
			if isCookieAuth && (r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" || r.Method == "DELETE") {
				if !validateCSRF(r) {
					response.Error(w, http.StatusForbidden, "CSRF_BLOCKED", "Cross-site request forgery protection triggered", nil, "")
					return
				}
			}

			claims, err := ValidateAccessToken(tokenStr, jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired access token", nil, "")
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware extracts claims if an Authorization header is present,
// but does NOT block or return 401 if missing or invalid.
func OptionalMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := ValidateAccessToken(parts[1], jwtSecret)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	return claims, ok
}

// WithClaims injects authenticated claims into the context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, UserContextKey, claims)
}

