package server

import (
	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/fetcher"
)

type DB interface {
	GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error)
	GetUser(username string) (*data.User, error)
	SaveUser(*data.User) error
	SetUsername(user *data.User, newUsername string) error
	GetFeeditem(*data.FeeditemKey) (*data.Feeditem, error)
	ReadAllFeedItems(chan *data.Feeditem) error
	GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error)
	ReadAllPages(chan *data.PagemonitorPage) error
}

type Fetcher interface {
	Refresh()
}

type FeedListHelper interface {
	GetAllItems(*data.User) ([]*Item, error)
}

type Services struct {
	db             DB
	cookieHandler  *CookieHandler
	fetcher        Fetcher
	feedListHelper FeedListHelper
}

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
