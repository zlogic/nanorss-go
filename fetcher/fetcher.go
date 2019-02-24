package fetcher

import (
	"log"
	"net/http"

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
		log.Println("Failed to fetch at least one page")
	} else {
		log.Println("Pages fetched successfully")
	}
	errFeed := fetcher.FetchAllFeeds()
	if errFeed != nil {
		log.Println("Failed to fetch at least one feed")
	} else {
		log.Println("Feeds fetched successfully")
	}
}
