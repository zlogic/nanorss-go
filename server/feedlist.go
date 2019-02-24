package server

import (
	"bytes"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/zlogic/nanorss-go/data"
)

type Item struct {
	Title    string
	Origin   string
	SortDate time.Time
	FetchURL string
}

type itemsSortable []*Item

func (a itemsSortable) Len() int           { return len(a) }
func (a itemsSortable) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a itemsSortable) Less(i, j int) bool { return a[i].SortDate.After(a[j].SortDate) }

func GetAllItems(userService *data.UserService) ([]*Item, error) {
	user, err := userService.Get()
	if err != nil {
		return nil, err
	}

	feeds, err := user.GetFeeds()
	if err != nil {
		return nil, err
	}
	pages, err := user.GetPages()
	if err != nil {
		return nil, err
	}

	items := make(itemsSortable, 0)

	escapeKeyForUrl := func(key []byte) string {
		return strings.Replace(string(key), "/", "-", -1)
	}
	findFeedTitle := func(feedURL string) (string, error) {
		for _, feed := range feeds {
			if feed.URL == feedURL {
				return feed.Title, nil
			}
		}
		return "", errors.New("Not found")
	}
	feedItemsChan := make(chan *data.Feeditem)
	feedItemsDone := make(chan bool)
	go func() {
		for feedItem := range feedItemsChan {
			title, err := findFeedTitle(feedItem.Key.FeedURL)
			if err != nil {
				// Probably an orphaned feed
				continue
			}
			item := &Item{
				Title:    feedItem.Title,
				Origin:   title,
				FetchURL: "api/items/" + escapeKeyForUrl(feedItem.Key.CreateKey()),
				SortDate: feedItem.Updated,
			}
			items = append(items, item)
		}
		close(feedItemsDone)
	}()
	err = userService.ReadAllFeedItems(feedItemsChan)
	if err != nil {
		return nil, err
	}

	findPagemonitorTitle := func(key []byte) (string, error) {
		for _, page := range pages {
			if bytes.Equal(key, page.CreateKey()) {
				return page.Title, nil
			}
		}
		return "", errors.New("Not found")
	}
	pagemonitorPageChan := make(chan *data.PagemonitorPage)
	pagemonitorDone := make(chan bool)
	go func() {
		for page := range pagemonitorPageChan {
			title, err := findPagemonitorTitle(page.Config.CreateKey())
			if err != nil {
				// Probably an orphaned feed
				continue
			}
			item := &Item{
				Title:    "",
				Origin:   title,
				FetchURL: "api/items/" + escapeKeyForUrl(page.Config.CreateKey()),
				SortDate: page.Updated,
			}
			items = append(items, item)
		}
		close(pagemonitorDone)
	}()
	err = userService.ReadAllPages(pagemonitorPageChan)
	if err != nil {
		return nil, err
	}

	<-feedItemsDone
	<-pagemonitorDone

	sort.Sort(items)

	return items, nil
}
