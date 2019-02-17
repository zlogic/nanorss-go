package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetItemEmpty(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	feedURL, guid := "http://feed1", "g1"
	item, err := dbService.GetFeeditem(feedURL, guid)
	assert.NoError(t, err)
	assert.Nil(t, item)
}

func TestSaveReadItem(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	feedURL1, guid1 := "http://feed1", "g1"
	item1 := Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
	}
	err = dbService.SaveFeeditem(feedURL1, guid1, &item1)
	assert.NoError(t, err)

	feedURL2, guid2 := "http://feed2", "g1"
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item2/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
	}
	err = dbService.SaveFeeditem(feedURL2, guid2, &item2)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(feedURL1, guid1)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, &item1, dbItem)

	dbItem, err = dbService.GetFeeditem(feedURL2, guid2)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, &item2, dbItem)
}

func TestUpdateReadItem(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	feedURL, guid := "http://feed1", "g1"
	item := Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
	}
	err = dbService.SaveFeeditem(feedURL, guid, &item)
	assert.NoError(t, err)

	item.Title = "t2"
	item.URL = "http://item2"
	item.Date = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	item.Contents = "c2"
	err = dbService.SaveFeeditem(feedURL, guid, &item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(feedURL, guid)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, &item, dbItem)
}

func TestSaveReadItemTTL(t *testing.T) {
	var oldTTL = itemTTL
	itemTTL = time.Nanosecond * 1
	defer func() { itemTTL = oldTTL }()
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	feedURL, guid := "http://feed1", "g1"
	item := Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
	}
	err = dbService.SaveFeeditem(feedURL, guid, &item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(feedURL, guid)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
}

func TestReadAllItems(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	type ItemWithKey struct {
		feedURL string
		guid    string
		item    Feeditem
	}
	item1 := ItemWithKey{
		feedURL: "http://feed1",
		guid:    "g1",
		item: Feeditem{
			Title:    "t1",
			URL:      "http://item1/1",
			Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
			Contents: "c1",
		},
	}
	item2 := ItemWithKey{
		feedURL: "http://feed1",
		guid:    "g2",
		item: Feeditem{
			Title:    "t2",
			URL:      "http://item1/2",
			Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
			Contents: "c2",
		},
	}
	item3 := ItemWithKey{
		feedURL: "http://feed2",
		guid:    "g1",
		item: Feeditem{
			Title:    "t3",
			URL:      "http://item2",
			Date:     time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
			Contents: "c3",
		},
	}
	items := []ItemWithKey{item1, item2, item3}
	for _, item := range items {
		err = dbService.SaveFeeditem(item.feedURL, item.guid, &item.item)
		assert.NoError(t, err)
	}

	dbItems := []ItemWithKey{}
	dbService.ReadAllFeedItems(func(feedURL, guid string, item *Feeditem) {
		dbItems = append(dbItems, ItemWithKey{
			feedURL: feedURL,
			guid:    guid,
			item:    *item,
		})
	})
	assert.EqualValues(t, items, dbItems)
}
