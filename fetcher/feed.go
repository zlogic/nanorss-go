package fetcher

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/zlogic/nanorss-go/data"
)

// FetchFeed fetches a feed from feedURL and saves it into the database if fetching was successful.
func (fetcher *Fetcher) FetchFeed(feedURL string) error {
	err := func() error {
		resp, err := fetcher.Client.Get(feedURL)
		if err == nil {
			defer resp.Body.Close()
		}

		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("cannot GET feed (status code %v)", resp.StatusCode)
		}
		if err != nil {
			return fmt.Errorf("cannot GET feed %v: %w", feedURL, err)
		}

		items, err := fetcher.ParseFeed(feedURL, resp.Body)
		if err != nil {
			return fmt.Errorf("cannot parse feed %v: %w", feedURL, err)
		}

		if len(items) == 0 {
			return fmt.Errorf("feed %v has no items", feedURL)
		}

		for _, item := range items {
			item.Updated = time.Now()
		}
		return fetcher.DB.SaveFeeditems(items...)
	}()

	fetchStatus := &data.FetchStatus{}
	if err != nil {
		log.WithField("feed", feedURL).WithError(err).Error("Failed to get feed")
		fetchStatus.LastFailure = time.Now()
	} else {
		fetchStatus.LastSuccess = time.Now()
	}

	fetchStatusKey := (&data.UserFeed{URL: feedURL}).CreateKey()
	if err := fetcher.DB.SetFetchStatus(fetchStatusKey, fetchStatus); err != nil {
		log.WithField("feed", feedURL).WithError(err).Error("Failed to save fetch status for feed")
	}
	return err
}

// FetchAllFeeds calls FetchFeed for all feeds for all users.
func (fetcher *Fetcher) FetchAllFeeds() error {
	usernames, err := fetcher.DB.GetUsers()
	if err != nil {
		log.WithError(err).Error("Failed to get list of users")
		return err
	}
	for _, username := range usernames {
		user, err := fetcher.DB.GetUser(username)
		if err != nil {
			log.WithField("username", username).WithError(err).Error("Failed to get user")
			return err
		}

		feeds, err := user.GetFeeds()
		if err != nil {
			log.WithError(err).WithField("user", user).Error("Failed to get feeds for user")
			continue
		}
		countFeeds := len(feeds)
		completed := make(chan int)
		for i, feed := range feeds {
			go func(config data.UserFeed, index int) {
				// TODO: skip this page if it was already fetched this round.
				fetcher.FetchFeed(config.URL)
				completed <- index
			}(feed, i)
		}
		for i := 0; i < countFeeds; i++ {
			<-completed
		}
	}
	return nil
}
