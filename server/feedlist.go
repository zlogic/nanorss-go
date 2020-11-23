package server

import (
	"encoding/base64"
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

const (
	feeditemURLPrefix = "feeditem"
	pageURLPrefix     = "page"
)

func escapeKeyForURL(attributes ...string) string {
	safeAttributes := make([]string, len(attributes))
	for i := range attributes {
		safeAttributes[i] = base64.RawStdEncoding.EncodeToString([]byte(attributes[i]))
	}
	return strings.Join(safeAttributes, ".")
}

func escapeFeeditemKeyForURL(key *data.FeeditemKey) string {
	return escapeKeyForURL(key.FeedURL, key.GUID)
}

func escapePagemonitorKeyForURL(key *data.UserPagemonitor) string {
	return escapeKeyForURL(key.URL, key.Match, key.Replace)
}

func decodeKeyFromURL(key string) ([]string, error) {
	parts := strings.Split(key, ".")
	for i := range parts {
		part, err := base64.RawStdEncoding.DecodeString(parts[i])
		if err != nil {
			return nil, err
		}
		parts[i] = string(part)
	}
	return parts, nil
}

func decodeFeeditemKeyFromURL(key string) (*data.FeeditemKey, error) {
	parts, err := decodeKeyFromURL(key)
	if err != nil {
		return nil, err
	} else if len(parts) != 2 {
		return nil, fmt.Errorf("unsupported feed item key format: %v", parts)
	}
	return &data.FeeditemKey{FeedURL: parts[0], GUID: parts[1]}, nil
}

func decodePagemonitorKeyFromURL(key string) (*data.UserPagemonitor, error) {
	parts, err := decodeKeyFromURL(key)
	if err != nil {
		return nil, err
	} else if len(parts) != 3 {
		return nil, fmt.Errorf("unsupported pagemonitor pagekey format: %v", parts)
	}
	return &data.UserPagemonitor{URL: parts[0], Match: parts[1], Replace: parts[2]}, nil
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
	userFeeds, err := user.GetFeeds()
	if err != nil {
		return nil, err
	}
	userPages, err := user.GetPages()
	if err != nil {
		return nil, err
	}

	readFeeditems, err := h.db.GetFeeditemsReadStatus(user)
	if err != nil {
		return nil, err
	}

	readPages, err := h.db.GetPagesReadStatus(user)
	if err != nil {
		return nil, err
	}

	items := make(itemsSortable, 0)

	findFeedTitle := func(key *data.FeeditemKey) (string, error) {
		for _, feed := range userFeeds {
			if feed.URL == key.FeedURL {
				return feed.Title, nil
			}
		}
		return "", fmt.Errorf("Not found")
	}
	isReadFeeditem := func(feedURL string) bool {
		for _, readFeeditem := range readFeeditems {
			if readFeeditem.FeedURL == feedURL {
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
		title, err := findFeedTitle(feedItem.Key)
		if err != nil {
			// Probably an orphaned feed
			continue
		}
		item := &Item{
			Title:    feedItem.Title,
			Origin:   title,
			FetchURL: "api/items/feeditem/" + escapeFeeditemKeyForURL(feedItem.Key),
			SortDate: feedItem.Date,
			IsRead:   isReadFeeditem(feedItem.Key.FeedURL),
		}
		items = append(items, item)
	}

	findPagemonitorTitle := func(key *data.UserPagemonitor) (string, error) {
		for _, page := range userPages {
			if page.URL == key.URL && page.Match == key.Match && page.Replace == key.Replace {
				return page.Title, nil
			}
		}
		return "", fmt.Errorf("Not found")
	}
	isReadPage := func(key *data.UserPagemonitor) bool {
		for _, readPageKey := range readPages {
			if readPageKey.URL == key.URL && readPageKey.Match == key.Match && readPageKey.Replace == key.Replace {
				return true
			}
		}
		return false
	}

	pages, err := h.db.GetPages(user)
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		title, err := findPagemonitorTitle(page.Config)
		if err != nil {
			// Probably an orphaned feed
			continue
		}
		item := &Item{
			Title:    "",
			Origin:   title,
			FetchURL: "api/items/page/" + escapePagemonitorKeyForURL(page.Config),
			SortDate: page.Updated,
			IsRead:   isReadPage(page.Config),
		}
		items = append(items, item)
	}

	sort.Sort(items)

	return items, nil
}
