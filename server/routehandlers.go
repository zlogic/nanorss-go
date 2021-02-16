package server

import (
	"bytes"
	"embed"
	"net/http"
	"path"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/server/auth"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.WithError(err).Error("Error while handling request")
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

// PageAuthHandler checks to see if an HTML page is accessed by an authorized user,
// and redirects to the login page if the request is done by an unauthorized user.
func PageAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			http.Redirect(w, r, "login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loadTemplate(s *Services, pageName string) (*template.Template, error) {
	return template.ParseFS(s.templates, "layout.html", path.Join("pages", pageName+".html"))
}

type viewData struct {
	User     *data.User
	Username string
	Name     string
}

// RootHandler handles the root url.
// It redirects authenticated users to the default page and unauthenticated users to the login page.
func RootHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Light check for authentication cookie - prevent errors from liveness probe.
		var url string
		if !s.cookieHandler.HasAuthenticationCookie(r) {
			url = "login"
		} else {
			url = "feed"
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

// LogoutHandler logs out the user.
func LogoutHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.cookieHandler.SetCookieUsername(w, "", false)
		if err != nil {
			log.WithError(err).Error("Error while clearing the cookie during logout")
		}
		http.Redirect(w, r, "login", http.StatusSeeOther)
	}
}

//go:embed static/favicon.ico
var faviconFS embed.FS

const faviconFilename = "static/favicon.ico"

// FaviconHandler serves the favicon.
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	data, err := faviconFS.ReadFile(faviconFilename)
	if err != nil {
		handleError(w, r, err)
		return
	}

	f, err := faviconFS.Open(faviconFilename)
	if err != nil {
		handleError(w, r, err)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		handleError(w, r, err)
		return
	}

	http.ServeContent(w, r, "favicon.ico", stat.ModTime(), bytes.NewReader(data))
}

// HTMLLoginHandler serves the login page.
func HTMLLoginHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := loadTemplate(s, "login")
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{})
	}
}

// HTMLFeedHandler serves the feed (and page monitor) items page.
func HTMLFeedHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
			return
		}

		const templateName = "feed"
		t, err := loadTemplate(s, templateName)
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: user.GetUsername(), Name: templateName})
	}
}

// HTMLSettingsHandler serves the user settings page.
func HTMLSettingsHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
			return
		}

		const templateName = "settings"
		t, err := loadTemplate(s, templateName)
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: user.GetUsername(), Name: templateName})
	}
}

// HTMLStatusHandler serves the feed status page.
func HTMLStatusHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			// This should never happen.
			return
		}

		const templateName = "status"
		t, err := loadTemplate(s, templateName)
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: user.GetUsername(), Name: templateName})
	}
}
