package fetcher

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/zlogic/nanorss-go/data"
)

type DB interface {
	GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error)
	SavePage(page *data.PagemonitorPage) error
	SaveFeeditems(feedItems ...*data.Feeditem) (err error)
	ReadAllUsers(chan *data.User) error
}

type Fetcher struct {
	DB     DB
	Client *http.Client
}

func NewFetcher(db DB) *Fetcher {
	return &Fetcher{DB: db}
}

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
