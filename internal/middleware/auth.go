// Package middleware contains HTTP middleware used by the ticket service.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ticket-system/internal/auth"
	"ticket-system/internal/utils"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// Auth returns middleware that requires a valid "Authorization: Bearer <token>"
// header, validates the JWT against secret, and injects the authenticated
// user's ID into the request context for downstream handlers.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				utils.WriteError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				utils.WriteError(w, http.StatusUnauthorized, "Authorization header must use Bearer scheme")
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
			if token == "" {
				utils.WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := auth.ParseToken(token, secret)
			if err != nil {
				if errors.Is(err, auth.ErrExpiredToken) {
					utils.WriteError(w, http.StatusUnauthorized, "token expired")
					return
				}
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user's ID set by Auth.
// The bool result is false if no user ID is present (should not happen
// for routes wrapped in Auth middleware).
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDContextKey).(string)
	return id, ok
}
