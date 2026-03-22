package api

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/civiledcode/grxm-webapp/internal/config"
	"github.com/civiledcode/grxm-webapp/internal/iam"
	"github.com/civiledcode/grxm-webapp/internal/profile"
)

var (
	adminTemplate      *template.Template
	adminUsersTemplate *template.Template
	loginTemplate      *template.Template
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
	adminUsersTemplate = loadTemplate("admin_users", "./dynamic/admin_users.html")
	loginTemplate = loadTemplate("login", "./dynamic/login.html")
}

// LoginHandler serves the dynamic login page, injecting configuration.
func LoginHandler(cfg *config.AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			AuthedPath string
		}{
			AuthedPath: cfg.AuthedPath,
		}
		w.Header().Set("Content-Type", "text/html")
		loginTemplate.Execute(w, data)
	}
}

// APIResponse represents the JSON structure for the Hello endpoint.
type APIResponse struct {
	Message   string `json:"message"`
	UID       string `json:"uid,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

// ServeStatic returns an http.HandlerFunc that serves a specific static file.
func ServeStatic(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filePath)
	}
}

// ServeStaticAuthed returns an authed handler that serves a specific static file.
func ServeStaticAuthed(filePath string) func(http.ResponseWriter, *http.Request, *iam.Identity, *profile.Profile) {
	return func(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
		http.ServeFile(w, r, filePath)
	}
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

// AdminDashboardHandler handles the protected admin dashboard route.
func AdminDashboardHandler(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
	v := AdminValues{
		User: p.Username + " (" + ident.UserID + ")",
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	adminTemplate.Execute(w, v)
}

// AdminUsersHandler renders the users management UI.
func AdminUsersHandler(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
	count, _ := profile.Count(r.Context())
	
	data := struct {
		User  string
		Count int64
	}{
		User:  p.Username,
		Count: count,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	adminUsersTemplate.Execute(w, data)
}

// AdminSearchUsersAPI handles searching profiles.
func AdminSearchUsersAPI(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*profile.Profile{})
		return
	}

	profiles, err := profile.Search(r.Context(), query)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

// AdminBanUserAPI bans a user via IAM Authority.
func AdminBanUserAPI(iamClient *iam.Client) func(http.ResponseWriter, *http.Request, *iam.Identity, *profile.Profile) {
	return func(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := iamClient.BanUser(req.UserID, req.Reason); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

// AdminPromoteUserAPI adds a role to a user via IAM Authority.
func AdminPromoteUserAPI(iamClient *iam.Client) func(http.ResponseWriter, *http.Request, *iam.Identity, *profile.Profile) {
	return func(w http.ResponseWriter, r *http.Request, ident *iam.Identity, p *profile.Profile) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := iamClient.AddRole(req.UserID, req.Role); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}