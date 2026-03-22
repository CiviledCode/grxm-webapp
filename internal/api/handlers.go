package api

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/civiledcode/grxm-webapp/internal/iam"
	"github.com/civiledcode/grxm-webapp/internal/profile"
)

var (
	adminTemplate *template.Template
)

func loadTemplate(name, path string) *template.Template {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	content, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	t := template.New(name)
	val, err := t.Parse(string(content))
	if err != nil {
		panic(err)
	}

	return val
}

func init() {
	adminTemplate = loadTemplate("admin_dashboard", "./dynamic/admin.html")
}

// APIResponse represents the JSON structure for the Hello endpoint.
type APIResponse struct {
	Message   string `json:"message"`
	UID       string `json:"uid,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

// HelloHandler handles the protected "/api/hello" route and returns authenticated user data.
// It acts as a closure to capture the public key needed for the template display.
func HelloHandler(pubKeyPEM string) func(http.ResponseWriter, *http.Request, *iam.Identity, *profile.Profile) {
	return func(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
		// We can now use ident.UserID instead of fetching it from headers
		response := APIResponse{
			Message:   "Hello, World! This is a protected endpoint.",
			UID:       p.Username + " (" + ident.UserID + ")",
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
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "alive",
		"database": "ok",
		"time":     time.Now().Format(time.RFC3339),
	})
}

type AdminValues struct {
	User string
}

// AdminHandler handles the protected admin route.
func AdminHandler(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
	v := AdminValues{
		User: p.Username + " (" + ident.UserID + ")",
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	adminTemplate.Execute(w, v)
}
