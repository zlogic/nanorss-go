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
	userPages, err := user.GetPages()
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
		return "", fmt.Errorf("title for feedURL %v not found", feedURL)
	}

	readStatuses, err := h.db.GetReadItems(user)
	if err != nil {
		return nil, err
	}

	isRead := func(itemKey []byte) bool {
		for i := range readStatuses {
			if bytes.Equal(readStatuses[i], itemKey) {
				return true
			}
		}
		return false
	}

	feedItems, err := h.db.GetFeeditems(user)
	if err != nil {
		return nil, err
	}
	for _, feedItem := range feedItems {
		title, err := findFeedTitle(feedItem.Key.FeedURL)
		if err != nil {
			// Probably an orphaned feed.
			continue
		}
		isRead := isRead(feedItem.Key.CreateKey())
		item := &Item{
			Title:    feedItem.Title,
			Origin:   title,
			FetchURL: "api/items/" + escapeKeyForURL(feedItem.Key.CreateKey()),
			SortDate: feedItem.Date,
			IsRead:   isRead,
		}
		items = append(items, item)
	}

	findPagemonitorTitle := func(key []byte) (string, error) {
		for _, userPage := range userPages {
			if bytes.Equal(key, userPage.CreateKey()) {
				return userPage.Title, nil
			}
		}
		return "", fmt.Errorf("title for page %v not found", string(key))
	}
	pages, err := h.db.GetPages(user)
	for _, page := range pages {
		title, err := findPagemonitorTitle(page.Config.CreateKey())
		if err != nil {
			// Probably an orphaned feed.
			continue
		}
		isRead := isRead(page.Config.CreateKey())
		item := &Item{
			Title:    "",
			Origin:   title,
			FetchURL: "api/items/" + escapeKeyForURL(page.Config.CreateKey()),
			SortDate: page.Updated,
			IsRead:   isRead,
		}
		items = append(items, item)
	}
	if err != nil {
		return nil, err
	}

	sort.Sort(items)

	return items, nil
}
