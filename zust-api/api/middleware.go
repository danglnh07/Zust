package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

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

			server.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid access token: %s", err.Error()))
			return
		}

		// Check token type
		path := r.URL.Path
		if claims.TokenType == "refresh-token" && path == "/auth/token/refresh" ||
			claims.TokenType == "access-token" && path != "/auth/token/refresh" {
			// Extract the claims and put them in the request context
			r = r.WithContext(context.WithValue(r.Context(), clKey, claims))
			next.ServeHTTP(w, r)
			return
		}

		server.WriteError(w, http.StatusBadRequest, "Invalid access token: unsuitable token type for this request")

	})
}
