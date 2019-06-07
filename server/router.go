package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// NoCacheHeaderMiddlewareFunc creates a handler to disable caching.
func NoCacheHeaderMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private")
		next.ServeHTTP(w, r)
	})
}

// CreateRouter returns a router and all handlers.
func CreateRouter(s *Services) (*mux.Router, error) {
	r := mux.NewRouter()
	r.HandleFunc("/", RootHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/login", HTMLLoginHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/logout", LogoutHandler(s)).Methods(http.MethodGet)
	r.HandleFunc("/feed", HTMLFeedHandler(s)).Methods(http.MethodGet).Name("feed")
	r.HandleFunc("/settings", HTMLSettingsHandler(s)).Methods(http.MethodGet).Name("settings")
	r.HandleFunc("/status", HTMLStatusHandler(s)).Methods(http.MethodGet).Name("status")
	r.HandleFunc("/favicon.ico", FaviconHandler)
	fs := http.FileServer(staticResourceFileSystem{http.Dir("static")})
	r.PathPrefix("/static/").Handler(http.StripPrefix(strings.TrimRight("/static", "/"), fs))

	api := r.PathPrefix("/api").Subrouter()
	api.Use(NoCacheHeaderMiddlewareFunc)
	api.HandleFunc("/login", LoginHandler(s)).Methods(http.MethodPost)
	api.HandleFunc("/configuration", SettingsHandler(s)).Methods(http.MethodGet, http.MethodPost)
	api.HandleFunc("/feed", FeedHandler(s)).Methods(http.MethodGet)
	api.HandleFunc("/items/{key}", FeedItemHandler(s)).Methods(http.MethodGet)
	api.HandleFunc("/refresh", RefreshHandler(s)).Methods(http.MethodGet)
	api.HandleFunc("/status", StatusHandler(s)).Methods(http.MethodGet)
	return r, nil
}
