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

func assertTimeBetween(t *testing.T, before, after time.Time, check time.Time) {
	before, err := time.Parse(time.RFC3339Nano, before.Format(time.RFC3339Nano))
	assert.NoError(t, err)
	after, err = time.Parse(time.RFC3339Nano, after.Format(time.RFC3339Nano))
	assert.NoError(t, err)

	assert.True(t, before == check || before.Before(check))
	assert.True(t, after == check || after.After(check))
}

func assertItemEqual(t *testing.T, expected *Feeditem, actual *Feeditem) {
	actual.Updated = expected.Updated
	assert.Equal(t, expected, actual)
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
	timeStarted1 := time.Now()
	err = dbService.SaveFeeditems(&item1)
	timeSaved1 := time.Now()
	assert.NoError(t, err)

	key2 := FeeditemKey{FeedURL: "http://feed2", GUID: "g1"}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item2/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Key:      &key2,
	}
	timeStarted2 := time.Now()
	err = dbService.SaveFeeditems(&item2)
	timeSaved2 := time.Now()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key1)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assertTimeBetween(t, timeStarted1, timeSaved1, dbItem.Updated)
	assertItemEqual(t, &item1, dbItem)

	dbItem, err = dbService.GetFeeditem(&key2)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assertTimeBetween(t, timeStarted2, timeSaved2, dbItem.Updated)
	assertItemEqual(t, &item2, dbItem)
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
	timeStarted := time.Now()
	err = dbService.SaveFeeditems(&item)
	timeSaved := time.Now()
	assert.NoError(t, err)

	item.Title = "t2"
	item.URL = "http://item2"
	item.Date = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	item.Contents = "c2"
	err = dbService.SaveFeeditems(&item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.NotNil(t, dbItem)
	assertTimeBetween(t, timeStarted, timeSaved, dbItem.Updated)
	assertItemEqual(t, &item, dbItem)
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
	err = dbService.SaveFeeditems(&item)
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
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
	timeStarted := time.Now()
	err = dbService.SaveFeeditems(&item1, &item2, &item3)
	timeSaved := time.Now()
	assert.NoError(t, err)

	dbItems := []Feeditem{}
	ch := make(chan *Feeditem)
	done := make(chan bool)
	go func() {
		for item := range ch {
			assertTimeBetween(t, timeStarted, timeSaved, item.Updated)
			item.Updated = time.Time{}
			dbItems = append(dbItems, *item)
		}
		close(done)
	}()
	err = dbService.ReadAllFeedItems(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, items, dbItems)
}
