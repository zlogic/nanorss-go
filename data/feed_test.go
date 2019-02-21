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

	key := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.Nil(t, item)
}

func TestSaveReadItem(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	key1 := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item1 := Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Key:      &key1,
	}
	err = dbService.SaveFeeditem(&item1)
	assert.NoError(t, err)

	key2 := FeeditemKey{FeedURL: "http://feed2", GUID: "g1"}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item2/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Key:      &key2,
	}
	err = dbService.SaveFeeditem(&item2)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key1)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, &item1, dbItem)

	dbItem, err = dbService.GetFeeditem(&key2)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, &item2, dbItem)
}

func TestUpdateReadItem(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	key := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item := Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Key:      &key,
	}
	err = dbService.SaveFeeditem(&item)
	assert.NoError(t, err)

	item.Title = "t2"
	item.URL = "http://item2"
	item.Date = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	item.Contents = "c2"
	err = dbService.SaveFeeditem(&item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key)
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

	key := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item := Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Key:      &key,
	}
	err = dbService.SaveFeeditem(&item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
}

func TestReadAllItems(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	item1 := Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item1/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g2"},
	}
	item3 := Feeditem{
		Title:    "t3",
		URL:      "http://item2",
		Date:     time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
		Contents: "c3",
		Key:      &FeeditemKey{FeedURL: "http://feed2", GUID: "g2"},
	}
	items := []Feeditem{item1, item2, item3}
	for _, item := range items {
		err = dbService.SaveFeeditem(&item)
		assert.NoError(t, err)
	}

	dbItems := []Feeditem{}
	ch := make(chan *Feeditem)
	done := make(chan bool)
	go func() {
		for item := range ch {
			dbItems = append(dbItems, *item)
		}
		close(done)
	}()
	err = dbService.ReadAllFeedItems(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, items, dbItems)
}
