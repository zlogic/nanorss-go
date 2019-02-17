package fetcher

import (
	"log"
	"net/http"

	"github.com/zlogic/nanorss-go/data"
)

type DB interface {
	GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error)
	SavePage(pm *data.UserPagemonitor, page *data.PagemonitorPage) error
	ReadAllUsers(handler func(username string, user *data.User)) error
}

type Fetcher struct {
	DB     DB
	Client *http.Client
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
}
