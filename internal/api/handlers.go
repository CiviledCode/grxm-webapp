package api

import (
	"encoding/json"
	"net/http"
)

// APIResponse represents the JSON structure for the Hello endpoint.
type APIResponse struct {
	Message   string `json:"message"`
	UID       string `json:"uid,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

// HelloHandler handles the protected "/api/hello" route and returns authenticated user data.
// It acts as a closure to capture the public key needed for the template display.
func HelloHandler(pubKeyPEM string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the secure user ID injected by the IAM authentication middleware
		uid := r.Header.Get("X-User-ID")

		response := APIResponse{
			Message:   "Hello, World! This is a protected endpoint.",
			UID:       uid,
			PublicKey: pubKeyPEM,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// HealthHandler returns a simple status message to indicate the server is running.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
