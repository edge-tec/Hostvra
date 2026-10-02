package auth

import (
	"context"
	"net/http"
	"strings"

	"hostvra/api/internal/response"
)

type contextKey string

const (
	UserContextKey contextKey = "hostvra_user_claims"
)

func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
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
				tokenStr = r.URL.Query().Get("token")
				if tokenStr == "" {
					if c, err := r.Cookie("hostvra_token"); err == nil && c != nil {
						tokenStr = c.Value
					} else if c, err := r.Cookie("access_token"); err == nil && c != nil {
						tokenStr = c.Value
					}
				}
			}

			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header", nil, "")
				return
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
