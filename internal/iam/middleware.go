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

// validateRequest extracts and validates the JWT token from the request,
// checks the Redis denylist, and returns the claims if valid. It also
// writes an appropriate HTTP JSON error response and returns nil claims if validation fails.
func (c *Client) validateRequest(w http.ResponseWriter, r *http.Request) jwt.MapClaims {
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
		return nil
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
		return nil
	}

	// Extract the primary UID claim required by zero-trust design
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Invalid token claims")
		return nil
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Token missing 'uid' claim")
		return nil
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
			return nil
		} else if err != redis.Nil {
			// Some other error occurred when querying Redis
			log.Printf("Warning: Failed to check Redis denylist for user %s: %v", uid, err)
		}
	}

	return claims
}

// AuthRequired is an HTTP middleware that validates the JWT Bearer token
// using the public key previously fetched by the Client and checks the Redis denylist.
func (c *Client) AuthRequired(next func(http.ResponseWriter, *http.Request, *Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := c.validateRequest(w, r)
		if claims == nil {
			return // Response already written by validateRequest
		}

		uid := claims["uid"].(string)

		var roles []string
		if rolesClaim, ok := claims["roles"]; ok {
			if rolesArr, ok := rolesClaim.([]interface{}); ok {
				for _, rClaim := range rolesArr {
					if rStr, ok := rClaim.(string); ok {
						roles = append(roles, rStr)
					}
				}
			}
		}

		ident := &Identity{
			UserID: uid,
			Roles:  roles,
		}

		// Inject the UID into the request for downstream handlers to use
		r.Header.Set("X-User-ID", uid)

		next(w, r, ident)
	}
}

// RoleRequired is an HTTP middleware that validates the JWT Bearer token,
// checks the Redis denylist, and ensures the user has the specified role.
func (c *Client) RoleRequired(role string, next func(http.ResponseWriter, *http.Request, *Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := c.validateRequest(w, r)
		if claims == nil {
			return // Response already written by validateRequest
		}

		var roles []string
		hasRole := false
		if rolesClaim, ok := claims["roles"]; ok {
			if rolesArr, ok := rolesClaim.([]interface{}); ok {
				for _, rClaim := range rolesArr {
					if rStr, ok := rClaim.(string); ok {
						roles = append(roles, rStr)
						if rStr == role {
							hasRole = true
						}
					}
				}
			}
		}

		if !hasRole {
			writeJSONError(w, http.StatusForbidden, "Insufficient permissions")
			return
		}

		uid := claims["uid"].(string)

		ident := &Identity{
			UserID: uid,
			Roles:  roles,
		}

		// Inject the UID into the request for downstream handlers to use
		r.Header.Set("X-User-ID", uid)

		next(w, r, ident)
	}
}
