package fetcher

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/microcosm-cc/bluemonday"
	"github.com/zlogic/nanorss-go/data"
)

// DB provides functions to read and write items in the database.
type DB interface {
	GetPage(data.UserPagemonitor) (data.PagemonitorPage, error)
	SavePage(data.PagemonitorPage) error
	SaveFeeditems(...data.Feeditem) (err error)
	SetFetchStatus([]byte, data.FetchStatus) error
	SetReadStatusForAll(k []byte, read bool) error
	ReadAllUsers(chan data.User) error
}

// Fetcher contains services needed to fetch items and save them into a database.
type Fetcher struct {
	DB         DB
	Client     *http.Client
	TagsPolicy *bluemonday.Policy
}

// NewFetcher creates a new Fetcher instance with db.
func NewFetcher(db DB) Fetcher {
	policy := bluemonday.UGCPolicy()
	return Fetcher{DB: db, TagsPolicy: policy}
}

// Refresh performs a fetch of all monitored items.
func (fetcher Fetcher) Refresh() {
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
