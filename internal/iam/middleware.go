package iam

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

// RequireAuth is an HTTP middleware that validates the JWT Bearer token
// using the public key previously fetched by the Client.
func (c *Client) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeJSONError(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		tokenString := parts[1]

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

		// Inject the UID into the request for downstream handlers to use
		r.Header.Set("X-User-ID", uid)

		next.ServeHTTP(w, r)
	}
}
