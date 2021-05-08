package server

import (
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
	feedTitles, err := getFeedTitles(user)
	if err != nil {
		return nil, err
	}
	pageTitles, err := getPageTitles(user)
	if err != nil {
		return nil, err
	}

	items := make(itemsSortable, 0)

	readStatuses, err := h.getReadStatuses(user)
	if err != nil {
		return nil, err
	}

	feedItems, err := h.db.GetFeeditems(user)
	if err != nil {
		return nil, err
	}
	for _, feedItem := range feedItems {
		title, ok := feedTitles[feedItem.Key.FeedURL]
		if !ok {
			// Probably an orphaned feed.
			continue
		}
		isRead := readStatuses[string(feedItem.Key.CreateKey())]
		item := &Item{
			Title:    feedItem.Title,
			Origin:   title,
			FetchURL: "api/items/" + escapeKeyForURL(feedItem.Key.CreateKey()),
			SortDate: feedItem.Date,
			IsRead:   isRead,
		}
		items = append(items, item)
	}

	pages, err := h.db.GetPages(user)
	for _, page := range pages {
		title, ok := pageTitles[string(page.Config.CreateKey())]
		if !ok {
			// Probably an orphaned feed.
			continue
		}
		isRead := readStatuses[string(page.Config.CreateKey())]
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

// getFeedTitles returns a map of user's feed titles.
func getFeedTitles(user *data.User) (map[string]string, error) {
	feeds, err := user.GetFeeds()
	if err != nil {
		return nil, err
	}

	feedTitles := make(map[string]string, len(feeds))
	for i := range feeds {
		url := feeds[i].URL
		feedTitles[url] = feeds[i].Title
	}
	return feedTitles, nil
}

// getFeedTitles returns a map of user's page titles.
func getPageTitles(user *data.User) (map[string]string, error) {
	userPages, err := user.GetPages()
	if err != nil {
		return nil, err
	}

	pageTitles := make(map[string]string, len(userPages))
	for i := range userPages {
		url := string(userPages[i].CreateKey())
		pageTitles[url] = userPages[i].Title
	}
	return pageTitles, nil
}

// getFeedTitles returns a map with item read statuses.
func (h *FeedListService) getReadStatuses(user *data.User) (map[string]bool, error) {
	userReadItems, err := h.db.GetReadItems(user)
	if err != nil {
		return nil, err
	}

	readStatuses := make(map[string]bool, len(userReadItems))
	for i := range userReadItems {
		itemKey := string(userReadItems[i])
		readStatuses[itemKey] = true
	}

	return readStatuses, nil
}
