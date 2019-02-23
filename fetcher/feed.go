package fetcher

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/pkg/errors"
	"github.com/zlogic/nanorss-go/data"
)

func (fetcher *Fetcher) FetchFeed(feedURL string) error {
	resp, err := fetcher.Client.Get(feedURL)
	if err != nil {
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
		dbItem := &data.Feeditem{
			Title:    item.Title,
			URL:      item.Link,
			Date:     date,
			Contents: item.Description,
			Key:      key,
		}
		saveItems = append(saveItems, dbItem)
	}
	return fetcher.DB.SaveFeeditems(saveItems...)
}

func (fetcher *Fetcher) FetchAllFeeds() error {
	failed := false
	ch := make(chan *data.User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			feeds, err := user.GetFeeds()
			if err != nil {
				log.Printf("Failed to get feeds for user %v %v", user, err)
				failed = true
				continue
			}
			countFeeds := len(feeds)
			completed := make(chan int)
			for i, feed := range feeds {
				go func(config data.UserFeed, index int) {
					err := fetcher.FetchFeed(config.URL)
					if err != nil {
						log.Printf("Failed to get feed %v %v", config, err)
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
