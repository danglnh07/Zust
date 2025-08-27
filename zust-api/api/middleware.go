package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// claimsKey is a custom type to avoid context key collisions
type claimsKey string

// key is the key used to store and retrieve claims from the request context.
// Its value can be whatever
var key claimsKey = "claims"

// AuthMiddleware is a middleware that checks for a valid JWT token in the Authorization header
func (server *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the request header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			server.WriteError(w, http.StatusUnauthorized, "Missing request header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify token
		claims, err := server.jwtService.VerifyToken(tokenString, server.query)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				server.WriteError(w, http.StatusUnauthorized, "Access token expired")
				return
			}

			if errors.Is(err, jwt.ErrTokenMalformed) {
				server.WriteError(w, http.StatusBadRequest, "Invalid access token: token is malformed")
				return
			}

			server.WriteError(w, http.StatusBadRequest, "Invalid access token")
			return
		}

		// Extract the claims and put them in the request context
		r = r.WithContext(context.WithValue(r.Context(), key, claims))
		next.ServeHTTP(w, r)
	})
}
