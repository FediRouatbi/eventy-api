package middleware

import (
	"net/http"

	"eventy-api/internal/http/responses"
)

func (m *AuthMiddleware) RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if _, allowedRole := allowed[claims.Role]; !allowedRole {
				responses.WriteError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
