package server

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zlogic/nanorss-go/data"
)

// Item is a generic item (RSS feed item or pagemonitor page).
type Item struct {
	Title    string
	Origin   string
	SortDate time.Time `json:"-"`
	FetchURL string
	IsRead   bool
}

// FeedListService is a service which gets feed items for a user.
type FeedListService struct {
	db DB
}

type itemsSortable []*Item

func escapeKeyForURL(key []byte) string {
	return strings.Replace(string(key), "/", "-", -1)
}

func (a itemsSortable) Len() int      { return len(a) }
func (a itemsSortable) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a itemsSortable) Less(i, j int) bool {
	if !a[i].IsRead && a[j].IsRead {
		return true
	}
	if a[i].IsRead && !a[j].IsRead {
		return false
	}
	return a[i].SortDate.After(a[j].SortDate)
}

// GetAllItems returns all Items for user.
func (h *FeedListService) GetAllItems(user *data.User) ([]*Item, error) {
	feeds, err := user.GetFeeds()
	if err != nil {
		return nil, err
	}
	pages, err := user.GetPages()
	if err != nil {
		return nil, err
	}

	readItems, err := h.db.GetReadStatus(user)
	if err != nil {
		return nil, err
	}

	items := make(itemsSortable, 0)

	findFeedTitle := func(feedURL string) (string, error) {
		for _, feed := range feeds {
			if feed.URL == feedURL {
				return feed.Title, nil
			}
		}
		return "", fmt.Errorf("Not found")
	}
	isRead := func(itemKey []byte) bool {
		for _, readItemKey := range readItems {
			if bytes.Equal(readItemKey, itemKey) {
				return true
			}
		}
		return false
	}
	feedItemsChan := make(chan *data.Feeditem)
	feedItemsDone := make(chan bool)
	go func() {
		for feedItem := range feedItemsChan {
			title, err := findFeedTitle(feedItem.Key.FeedURL)
			//TODO: this is not efficient for more than a couple of users
			if err != nil {
				// Probably an orphaned feed
				continue
			}
			item := &Item{
				Title:    feedItem.Title,
				Origin:   title,
				FetchURL: "api/items/" + escapeKeyForURL(feedItem.Key.CreateKey()),
				SortDate: feedItem.Date,
				IsRead:   isRead(feedItem.Key.CreateKey()),
			}
			items = append(items, item)
		}
		close(feedItemsDone)
	}()
	err = h.db.ReadAllFeedItems(feedItemsChan)
	if err != nil {
		return nil, err
	}
	<-feedItemsDone

	findPagemonitorTitle := func(key []byte) (string, error) {
		for _, page := range pages {
			if bytes.Equal(key, page.CreateKey()) {
				return page.Title, nil
			}
		}
		return "", fmt.Errorf("Not found")
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
				FetchURL: "api/items/" + escapeKeyForURL(page.Config.CreateKey()),
				SortDate: page.Updated,
				IsRead:   isRead(page.Config.CreateKey()),
			}
			items = append(items, item)
		}
		close(pagemonitorDone)
	}()
	err = h.db.ReadAllPages(pagemonitorPageChan)
	if err != nil {
		return nil, err
	}
	<-pagemonitorDone

	sort.Sort(items)

	return items, nil
}
