package fetcher

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/zlogic/nanorss-go/data"
)

// DB provides functions to read and write items in the database.
type DB interface {
	GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error)
	SavePage(page *data.PagemonitorPage) error
	SaveFeeditems(feedItems ...*data.Feeditem) (err error)
	ReadAllUsers(chan *data.User) error
}

// Fetcher contains services needed to fetch items and save them into a database.
type Fetcher struct {
	DB     DB
	Client *http.Client
}

// NewFetcher creates a new Fetcher instance with db.
func NewFetcher(db DB) *Fetcher {
	return &Fetcher{DB: db}
}

// Refresh performs a fetch of all monitored items.
func (fetcher *Fetcher) Refresh() {
	if fetcher.Client == nil {
		fetcher.Client = &http.Client{}
	}
	errPagemonitor := fetcher.FetchAllPages()
	if errPagemonitor != nil {
		log.Error("Failed to fetch at least one page")
	} else {
		log.Info("Pages fetched successfully")
	}
	errFeed := fetcher.FetchAllFeeds()
	if errFeed != nil {
		log.Error("Failed to fetch at least one feed")
	} else {
		log.Info("Feeds fetched successfully")
	}
}
