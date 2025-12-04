package middleware

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJWTAuth(t *testing.T) {
	secret := []byte("secret")

	newToken := func() string {
		claims := jwt.MapClaims{"sub": "u"}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		s, _ := tok.SignedString(secret)
		return s
	}

	tests := []struct {
		name         string
		secret       []byte
		token        string
		expectStatus int
		expectNext   bool
	}{
		{
			name:         "empty secret passthrough",
			secret:       nil,
			expectStatus: http.StatusOK,
			expectNext:   true,
		},
		{
			name:         "missing token",
			secret:       secret,
			expectStatus: http.StatusForbidden,
			expectNext:   false,
		},
		{
			name:         "valid token",
			secret:       secret,
			token:        newToken(),
			expectStatus: http.StatusOK,
			expectNext:   true,
		},
		{
			name:         "invalid token",
			secret:       secret,
			token:        "deadbeef",
			expectStatus: http.StatusUnauthorized,
			expectNext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			mw := JWTAuth(tt.secret)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			rr := httptest.NewRecorder()
			mw.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectStatus, rr.Code)
			assert.Equal(t, tt.expectNext, nextCalled)
		})
	}
}
