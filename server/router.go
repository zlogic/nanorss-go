package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func CreateRouter(s *Services) (*mux.Router, error) {
	r := mux.NewRouter()
	r.HandleFunc("/", RootHandler(s)).Methods("GET")
	r.HandleFunc("/login", HtmlLoginHandler(s)).Methods("GET")
	r.HandleFunc("/logout", LogoutHandler(s)).Methods("GET")
	r.HandleFunc("/feed", HtmlFeedHandler(s)).Methods("GET").Name("feed")
	r.HandleFunc("/settings", HtmlSettingsHandler(s)).Methods("GET").Name("settings")
	r.HandleFunc("/favicon.ico", FaviconHandler)
	fs := http.FileServer(staticResourceFileSystem{http.Dir("static")})
	r.PathPrefix("/static/").Handler(http.StripPrefix(strings.TrimRight("/static", "/"), fs))

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/login", LoginHandler(s)).Methods("POST")
	api.HandleFunc("/configuration", SettingsHandler(s)).Methods("GET", "POST")
	api.HandleFunc("/feed", FeedHandler(s)).Methods("GET")
	api.HandleFunc("/items/{key}", FeedItemHandler(s)).Methods("GET")
	api.HandleFunc("/refresh", RefreshHandler(s)).Methods("GET")
	return r, nil
}
