package server

import (
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"
)

// NoCacheHeaderMiddlewareFunc creates a handler to disable caching.
func NoCacheHeaderMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private")
		next.ServeHTTP(w, r)
	})
}

// parseBoolEnv will try to parse the varName into a boolean.
// If varName is not set, will return defaultValue instead.
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

	r.Group(func(authorized chi.Router) {
		authorized.Use(s.cookieHandler.AuthHandlerFunc)
		authorized.Use(PageAuthHandler)
		authorized.Get("/logout", LogoutHandler(s))
		authorized.Get("/feed", HTMLFeedHandler(s))
		authorized.Get("/settings", HTMLSettingsHandler(s))
		authorized.Get("/status", HTMLStatusHandler(s))
	})
	r.HandleFunc("/favicon.ico", FaviconHandler)

	r.Route("/api", func(api chi.Router) {
		api.Use(NoCacheHeaderMiddlewareFunc)
		api.Post("/login", LoginHandler(s))
		api.Group(func(authorized chi.Router) {
			authorized.Use(s.cookieHandler.AuthHandlerFunc)
			authorized.Use(APIAuthHandler)
			authorized.Get("/configuration", SettingsHandler(s))
			authorized.Post("/configuration", SettingsHandler(s))
			authorized.Get("/feed", FeedHandler(s))
			authorized.Get("/items/{key}", FeedItemHandler(s))
			authorized.Post("/items/{key}", FeedItemHandler(s))
			authorized.Get("/refresh", RefreshHandler(s))
			authorized.Get("/status", StatusHandler(s))
		})
	})
	return r, nil
}
