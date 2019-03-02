package server

import (
	"net/http"
	"strings"

	"github.com/zlogic/nanorss-go/data"

	"github.com/gorilla/mux"
)

type services struct {
	db            *data.DBService
	cookieHandler *CookieHandler
}

func CreateRouter(db *data.DBService) (*mux.Router, error) {
	r := mux.NewRouter()
	cookieHandler, err := NewCookieHandler(db)
	if err != nil {
		return nil, err
	}
	services := services{db: db, cookieHandler: cookieHandler}
	r.HandleFunc("/", RootHandler(&services)).Methods("GET")
	r.HandleFunc("/login", HtmlLoginHandler(&services)).Methods("GET")
	r.HandleFunc("/logout", LogoutHandler(&services)).Methods("GET")
	r.HandleFunc("/feed", HtmlFeedHandler(&services)).Methods("GET").Name("feed")
	r.HandleFunc("/settings", HtmlSettingsHandler(&services)).Methods("GET").Name("settings")
	//r.PathPrefix("/app", RootHandler)
	r.HandleFunc("/favicon.ico", FaviconHandler)
	fs := http.FileServer(staticResourceFileSystem{http.Dir("static")})
	r.PathPrefix("/static/").Handler(http.StripPrefix(strings.TrimRight("/static", "/"), fs))

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/login", LoginHandler(&services)).Methods("POST")
	api.HandleFunc("/configuration", SettingsHandler(&services)).Methods("GET", "POST")
	api.HandleFunc("/feed", FeedHandler(&services)).Methods("GET")
	api.HandleFunc("/items/{key}", FeedItemHandler(&services)).Methods("GET")
	api.HandleFunc("/refresh", RefreshHandler(&services)).Methods("GET")
	return r, nil
}
