package middleware

import (
	"context"
	"net/http"
	"strings"

	"eventy-api/internal/http/responses"
	"eventy-api/internal/platform/jwt"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"

type AuthMiddleware struct {
	tokenManager *jwt.Manager
}

func NewAuthMiddleware(tokenManager *jwt.Manager) *AuthMiddleware {
	return &AuthMiddleware{tokenManager: tokenManager}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			responses.WriteError(w, http.StatusUnauthorized, "authorization header is required")
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(header, bearerPrefix) {
			responses.WriteError(w, http.StatusUnauthorized, "authorization header must use bearer token")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
		if token == "" {
			responses.WriteError(w, http.StatusUnauthorized, "bearer token is required")
			return
		}

		claims, err := m.tokenManager.Parse(token)
		if err != nil {
			responses.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ClaimsFromContext(ctx context.Context) (*jwt.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*jwt.Claims)
	return claims, ok
}
