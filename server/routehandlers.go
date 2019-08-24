package server

import (
	"net/http"
	"path"
	"text/template"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/zlogic/nanorss-go/data"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.WithError(err).Error("Error while handling request")
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func validateUser(w http.ResponseWriter, r *http.Request, s Services) string {
	username := s.cookieHandler.GetUsername(w, r)
	if username == "" {
		http.Redirect(w, r, "login", http.StatusSeeOther)
	}
	return username
}

func loadTemplate(pageName string) (*template.Template, error) {
	return template.ParseFiles(path.Join("templates", "layout.html"), path.Join("templates", "pages", pageName+".html"))
}

type viewData struct {
	User     data.User
	Username string
	Name     string
}

// RootHandler handles the root url.
// It redirects authenticated users to the default page and unauthenticated users to the login page.
func RootHandler(s Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := s.cookieHandler.GetUsername(w, r)
		var url string
		if username == "" {
			url = "login"
		} else {
			url = "feed"
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

// LogoutHandler logs out the user.
func LogoutHandler(s Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie := s.cookieHandler.NewCookie()
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "login", http.StatusSeeOther)
	}
}

// FaviconHandler serves the favicon.
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join("static", "favicon.ico"))
}

// HTMLLoginHandler serves the login page.
func HTMLLoginHandler(s Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := s.cookieHandler.GetUsername(w, r)
		if username != "" {
			http.Redirect(w, r, "feed", http.StatusSeeOther)
			return
		}
		t, err := loadTemplate("login")
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{})
	}
}

// HTMLFeedHandler serves the feed (and page monitor) items page.
func HTMLFeedHandler(s Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := validateUser(w, r, s)
		if username == "" {
			return
		}
		user, err := s.db.GetUser(username)
		if err != nil {
			handleError(w, r, err)
			return
		}

		t, err := loadTemplate("feed")
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: username, Name: mux.CurrentRoute(r).GetName()})
	}
}

// HTMLSettingsHandler serves the user settings page.
func HTMLSettingsHandler(s Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := validateUser(w, r, s)
		if username == "" {
			return
		}
		user, err := s.db.GetUser(username)
		if err != nil {
			handleError(w, r, err)
			return
		}

		t, err := loadTemplate("settings")
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: username, Name: mux.CurrentRoute(r).GetName()})
	}
}

// HTMLStatusHandler serves the feed status page.
func HTMLStatusHandler(s Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		username := validateUser(w, r, s)
		if username == "" {
			return
		}
		user, err := s.db.GetUser(username)
		if err != nil {
			handleError(w, r, err)
			return
		}

		t, err := loadTemplate("status")
		if err != nil {
			handleError(w, r, err)
			return
		}
		t.ExecuteTemplate(w, "layout", &viewData{User: user, Username: username, Name: mux.CurrentRoute(r).GetName()})
	}
}
