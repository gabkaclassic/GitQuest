package middleware

import (
	"context"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

func JWTAuth(secret []byte) func(next http.Handler) http.Handler {

	if secret == nil || len(secret) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if tokenStr == "" {
				apiErr := api.Forbidden("no token")
				api.RespondError(w, apiErr)
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				return secret, nil
			})
			if err != nil || !token.Valid {
				apiErr := api.Unauthorized("invalid token")
				api.RespondError(w, apiErr)
				return
			}

			ctx := context.WithValue(r.Context(), "claims", token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
