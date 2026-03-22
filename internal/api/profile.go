package api

import (
	"net/http"

	"github.com/civiledcode/grxm-webapp/internal/config"
	"github.com/civiledcode/grxm-webapp/internal/iam"
	"github.com/civiledcode/grxm-webapp/internal/profile"
)

// ProfileRequired is a middleware that ensures a user has created a profile before proceeding.
// If the user has no profile, they are redirected to the /profile/create endpoint.
func ProfileRequired(next func(http.ResponseWriter, *http.Request, *iam.Identity, *profile.Profile)) func(http.ResponseWriter, *http.Request, *iam.Identity) {
	return func(w http.ResponseWriter, r *http.Request, ident *iam.Identity) {
		p, err := profile.Get(r.Context(), ident.UserID)
		if err != nil {
			if err == profile.ErrProfileNotFound {
				http.Redirect(w, r, "/profile/create", http.StatusSeeOther)
				return
			}
			http.Error(w, "Internal server error checking profile", http.StatusInternalServerError)
			return
		}

		next(w, r, ident, p)
	}
}

// ProfileCreateHandler provides the UI and form submission for creating a new profile.
func ProfileCreateHandler(cfg *config.AppConfig) func(http.ResponseWriter, *http.Request, *iam.Identity) {
	return func(w http.ResponseWriter, r *http.Request, ident *iam.Identity) {
		// If they already have a profile, redirect away from create page
		if _, err := profile.Get(r.Context(), ident.UserID); err == nil {
			http.Redirect(w, r, cfg.AuthedPath, http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html")
			tmpl := loadTemplate("profile_create", "./dynamic/profile_create.html")
			tmpl.Execute(w, nil)
			return
		}

		if r.Method == http.MethodPost {
			r.ParseForm()
			username := r.FormValue("username")

			_, err := profile.Create(r.Context(), cfg, ident.UserID, username)
			if err != nil {
				w.Header().Set("Content-Type", "text/html")
				tmpl := loadTemplate("profile_create", "./dynamic/profile_create.html")
				tmpl.Execute(w, map[string]string{"Error": err.Error()})
				return
			}

			http.Redirect(w, r, cfg.AuthedPath, http.StatusSeeOther)
		}
	}
}
