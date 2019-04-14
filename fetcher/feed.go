package fetcher

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
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

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Cannot parse feed %v", feedURL)
	}

	if feed.Items == nil {
		return errors.Wrapf(err, "Feed has no items %v", feedURL)
	}

	saveItems := make([]*data.Feeditem, 0, len(feed.Items))
	for _, item := range feed.Items {
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		key := &data.FeeditemKey{
			FeedURL: feedURL,
			GUID:    guid,
		}
		date := time.Now()
		if item.UpdatedParsed != nil {
			date = *item.UpdatedParsed
		} else if item.PublishedParsed != nil {
			date = *item.PublishedParsed
		}
		contents := item.Description
		if contents == "" {
			contents = item.Content
		}
		dbItem := &data.Feeditem{
			Title:    item.Title,
			URL:      item.Link,
			Date:     date,
			Contents: contents,
			Updated:  time.Now(),
			Key:      key,
		}
		saveItems = append(saveItems, dbItem)
	}
	return fetcher.DB.SaveFeeditems(saveItems...)
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
