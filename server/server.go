package server

import (
	"io/fs"
	"net/http"

	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/fetcher"
	"github.com/zlogic/nanorss-go/server/auth"
	"github.com/zlogic/nanorss-go/server/templates"
)

// DB provides functions to read and write items in the database.
type DB interface {
	GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error)
	GetUser(username string) (*data.User, error)
	SaveUser(*data.User) error
	GetFeeditem(*data.FeeditemKey) (*data.Feeditem, error)
	GetFeeditems(*data.User) ([]*data.Feeditem, error)
	GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error)
	GetPages(*data.User) ([]*data.PagemonitorPage, error)
	GetReadStatus(user *data.User, itemKey []byte) (bool, error)
	SetReadStatus(user *data.User, itemKey []byte, read bool) error
	GetFetchStatus(key []byte) (*data.FetchStatus, error)
}

// Fetcher provides a method to refresh all feeds.
type Fetcher interface {
	Refresh()
}

// FeedListHelper returns all feed (and page monitor) items for a user.
type FeedListHelper interface {
	GetAllItems(*data.User) ([]*Item, error)
}

// AuthHandler handles authentication and authentication cookies.
type AuthHandler interface {
	SetCookieUsername(w http.ResponseWriter, username string, rememberMe bool) error
	AuthHandlerFunc(next http.Handler) http.Handler
	HasAuthenticationCookie(r *http.Request) bool
}

// Services keeps references to all services needed by handlers.
type Services struct {
	db             DB
	cookieHandler  AuthHandler
	fetcher        Fetcher
	feedListHelper FeedListHelper
	templates      fs.FS
}

// CreateServices creates a Services instance with db and default implementations of other services.
func CreateServices(db *data.DBService) (*Services, error) {
	cookieHandler, err := auth.NewCookieHandler(db)
	if err != nil {
		return nil, err
	}
	return &Services{
		db:             db,
		cookieHandler:  cookieHandler,
		fetcher:        fetcher.NewFetcher(db),
		feedListHelper: &FeedListService{db: db},
		templates:      templates.Templates,
	}, nil
}
