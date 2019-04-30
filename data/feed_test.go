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
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &key1,
	}
	err = dbService.SaveFeeditems(&item1)
	assert.NoError(t, err)

	key2 := FeeditemKey{FeedURL: "http://feed2", GUID: "g1"}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item2/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Updated:  time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
		Key:      &key2,
	}
	err = dbService.SaveFeeditems(&item2)
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
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &key,
	}
	err = dbService.SaveFeeditems(&item)
	assert.NoError(t, err)

	item.Title = "t2"
	item.URL = "http://item2"
	item.Date = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	item.Contents = "c2"
	item.Updated = time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC)
	err = dbService.SaveFeeditems(&item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, &item, dbItem)
}

func TestUpdateReadItemUnchanged(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	key := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item := Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &key,
	}
	err = dbService.SaveFeeditems(&item)
	assert.NoError(t, err)

	item.Updated = time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC)
	err = dbService.SaveFeeditems(&item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assert.Equal(t, time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC), dbItem.Updated)
	item.Updated = time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC)
	assert.Equal(t, &item, dbItem)
}

func TestSaveReadItemTTLExpired(t *testing.T) {
	var oldTTL = itemTTL
	itemTTL = time.Nanosecond * 0
	defer func() { itemTTL = oldTTL }()
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	item := &Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	err = dbService.SaveFeeditems(item)
	assert.NoError(t, err)

	err = dbService.DeleteExpiredItems()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
}

func TestSaveReadItemTTLNotExpired(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	item := &Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	err = dbService.SaveFeeditems(item)
	assert.NoError(t, err)

	err = dbService.DeleteExpiredItems()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Equal(t, item, dbItem)
}

func TestSaveReadAllItems(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	item1 := Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item1/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Updated:  time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g2"},
	}
	item3 := Feeditem{
		Title:    "t3",
		URL:      "http://item2",
		Date:     time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
		Contents: "c3",
		Updated:  time.Date(2019, time.February, 18, 23, 2, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed2", GUID: "g2"},
	}
	items := []Feeditem{item1, item2, item3}
	err = dbService.SaveFeeditems(&item1, &item2, &item3)
	assert.NoError(t, err)

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
