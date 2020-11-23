package server

import (
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	log "github.com/sirupsen/logrus"
)

// NoCacheHeaderMiddlewareFunc creates a handler to disable caching.
func NoCacheHeaderMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private")
		next.ServeHTTP(w, r)
	})
}

func parseBoolEnv(varName string, defaultValue bool) bool {
	valueStr, _ := os.LookupEnv(varName)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.WithField("variable", varName).WithField("value", value).WithError(err).Error("Cannot parse environment value")
		return defaultValue
	}
	return value
}

// CreateRouter returns a router and all handlers.
func CreateRouter(s *Services) (*chi.Mux, error) {
	logRequests := parseBoolEnv("LOG_REQUESTS", true)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if logRequests {
		r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.New(), NoColor: true}))
	}
	r.Use(middleware.Recoverer)

	r.Get("/", RootHandler(s))
	r.Get("/login", HTMLLoginHandler(s))
	r.Get("/logout", LogoutHandler(s))
	r.Get("/feed", HTMLFeedHandler(s))
	r.Get("/settings", HTMLSettingsHandler(s))
	r.Get("/status", HTMLStatusHandler(s))
	r.HandleFunc("/favicon.ico", FaviconHandler)
	fs := http.FileServer(staticResourceFileSystem{http.Dir("static")})
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	r.Route("/api", func(api chi.Router) {
		api.Use(NoCacheHeaderMiddlewareFunc)
		api.Post("/login", LoginHandler(s))
		api.Get("/configuration", SettingsHandler(s))
		api.Post("/configuration", SettingsHandler(s))
		api.Get("/feed", FeedHandler(s))
		api.Get("/items/{type}/{key}", FeedItemHandler(s))
		api.Post("/items/{type}/{key}", FeedItemHandler(s))
		api.Get("/refresh", RefreshHandler(s))
		api.Get("/status", StatusHandler(s))
	})
	return r, nil
}
