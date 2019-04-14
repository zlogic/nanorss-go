package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// CreateRouter returns a router and all handlers.
func CreateRouter(s *Services) (*mux.Router, error) {
	r := mux.NewRouter()
	r.HandleFunc("/", RootHandler(s)).Methods("GET")
	r.HandleFunc("/login", HTMLLoginHandler(s)).Methods("GET")
	r.HandleFunc("/logout", LogoutHandler(s)).Methods("GET")
	r.HandleFunc("/feed", HTMLFeedHandler(s)).Methods("GET").Name("feed")
	r.HandleFunc("/settings", HTMLSettingsHandler(s)).Methods("GET").Name("settings")
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
