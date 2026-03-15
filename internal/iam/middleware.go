package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

// AuthRequired is an HTTP middleware that validates the JWT Bearer token
// using the public key previously fetched by the Client and checks the Redis denylist.
func (c *Client) AuthRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// 1. Try to get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		// 2. Fallback to getting token from configured cookie
		cookieName := c.config.CookieName
		if cookieName == "" {
			cookieName = "grxm-token" // Failsafe default
		}

		if tokenString == "" {
			cookie, err := r.Cookie(cookieName)
			if err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			writeJSONError(w, http.StatusUnauthorized, "Missing authentication token")
			return
		}

		// Parse and validate the token signature using the Client's dynamic PublicKey
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			if c.PublicKey == nil {
				return nil, fmt.Errorf("IAM public key is not loaded")
			}
			return c.PublicKey, nil
		})

		if err != nil || !token.Valid {
			log.Printf("JWT validation failed: %v", err)
			writeJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Extract the primary UID claim required by zero-trust design
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		uid, ok := claims["uid"].(string)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Token missing 'uid' claim")
			return
		}

		// Check the Redis denylist
		// If the user ID is found in Redis, they are banned/denied.
		if c.Redis != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
			defer cancel()
			if err := c.Redis.Get(ctx, uid).Err(); err == nil {
				// The key exists in Redis, meaning the user is on the denylist
				log.Printf("User %s is on the denylist, rejecting request.", uid)
				writeJSONError(w, http.StatusUnauthorized, "User session revoked or banned")
				return
			} else if err != redis.Nil {
				// Some other error occurred when querying Redis
				log.Printf("Warning: Failed to check Redis denylist for user %s: %v", uid, err)
			}
		}

		// Inject the UID into the request for downstream handlers to use
		r.Header.Set("X-User-ID", uid)

		next.ServeHTTP(w, r)
	}
}
