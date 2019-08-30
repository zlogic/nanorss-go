package server

import (
	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/fetcher"
)

// DB provides functions to read and write items in the database.
type DB interface {
	GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error)
	GetUser(username string) (*data.User, error)
	SaveUser(*data.User) error
	GetFeeditem(*data.FeeditemKey) (*data.Feeditem, error)
	ReadAllFeedItems(chan *data.Feeditem) error
	GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error)
	ReadAllPages(chan *data.PagemonitorPage) error
	GetReadStatus(*data.User) ([][]byte, error)
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

// Services keeps references to all services needed by handlers.
type Services struct {
	db             DB
	cookieHandler  *CookieHandler
	fetcher        Fetcher
	feedListHelper FeedListHelper
}

// CreateServices creates a Services instance with db and default implementations of other services.
func CreateServices(db *data.DBService) (*Services, error) {
	cookieHandler, err := NewCookieHandler(db)
	if err != nil {
		return nil, err
	}
	return &Services{
		db:             db,
		cookieHandler:  cookieHandler,
		fetcher:        fetcher.NewFetcher(db),
		feedListHelper: &FeedListService{db: db},
	}, nil
}
