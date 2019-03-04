package server

import (
	"log"
	"net/http"
	"path"
	"text/template"

	"github.com/gorilla/mux"

	"github.com/zlogic/nanorss-go/data"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Error while handling request %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func validateUser(w http.ResponseWriter, r *http.Request, s *Services) string {
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
	User     *data.User
	Username string
	Name     string
}

func RootHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
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

func LogoutHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie := s.cookieHandler.NewCookie()
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "login", http.StatusSeeOther)
	}
}

func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join("static", "favicon.ico"))
}

func HtmlLoginHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
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

func HtmlFeedHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
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

func HtmlSettingsHandler(s *Services) func(w http.ResponseWriter, r *http.Request) {
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
