package fetcher

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zlogic/nanorss-go/data"
)

// FetchFeed fetches a feed from feedURL and saves it into the database if fetching was successful.
func (fetcher *Fetcher) FetchFeed(feedURL string) error {
	resp, err := fetcher.Client.Get(feedURL)
	if err == nil {
		defer resp.Body.Close()
	}

	if err == nil && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Cannot GET feed (status code %v)", resp.StatusCode)
	}
	if err != nil {
		return errors.Wrapf(err, "Cannot GET feed %v", feedURL)
	}

	items, err := fetcher.ParseFeed(feedURL, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Cannot parse feed %v", feedURL)
	}

	if len(items) == 0 {
		return errors.Wrapf(err, "Feed has no items %v", feedURL)
	}

	for _, item := range items {
		item.Updated = time.Now()
	}
	return fetcher.DB.SaveFeeditems(items...)
}

// FetchAllFeeds calls FetchFeed for all feeds for all users.
func (fetcher *Fetcher) FetchAllFeeds() error {
	failed := false
	ch := make(chan *data.User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			feeds, err := user.GetFeeds()
			if err != nil {
				log.WithError(err).Error("Failed to get feeds")
				failed = true
				continue
			}
			countFeeds := len(feeds)
			completed := make(chan int)
			for i, feed := range feeds {
				go func(config data.UserFeed, index int) {
					err := fetcher.FetchFeed(config.URL)
					if err != nil {
						log.WithField("feed", config).WithError(err).Error("Failed to get feed")
						failed = true
					}
					completed <- index
				}(feed, i)
			}
			for i := 0; i < countFeeds; i++ {
				<-completed
			}
		}
		close(done)
	}()
	err := fetcher.DB.ReadAllUsers(ch)
	<-done
	if err != nil {
		return err
	}
	if failed {
		return fmt.Errorf("At least one feed failed to fetch properly")
	}
	return nil
}
