package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetItemEmpty(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	key := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.Nil(t, item)
}

func TestSaveReadItem(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

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
	err := resetDb()
	assert.NoError(t, err)

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
	err := resetDb()
	assert.NoError(t, err)

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
	err := resetDb()
	assert.NoError(t, err)
	var oldTTL = itemTTL
	itemTTL = time.Nanosecond * 0
	defer func() { itemTTL = oldTTL }()

	item := &Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	feedKey := &UserFeed{URL: item.Key.FeedURL}
	err = dbService.SaveFeeditems(item)
	assert.NoError(t, err)

	err = dbService.SetFetchStatus(feedKey.CreateKey(), &FetchStatus{LastSuccess: time.Time{}})
	assert.NoError(t, err)

	err = dbService.deleteStaleFetchStatuses()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
}

func TestSaveReadItemTTLNotExpired(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	item := &Feeditem{
		Title:    "t1",
		URL:      "http://item1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	feedKey := &UserFeed{URL: item.Key.FeedURL}
	err = dbService.SaveFeeditems(item)
	assert.NoError(t, err)

	err = dbService.SetFetchStatus(feedKey.CreateKey(), &FetchStatus{LastSuccess: time.Time{}})
	assert.NoError(t, err)

	err = dbService.deleteStaleFetchStatuses()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Equal(t, item, dbItem)
}

func TestSaveReadAllItems(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

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
	items := []*Feeditem{&item1, &item2, &item3}
	err = dbService.SaveFeeditems(items...)
	assert.NoError(t, err)

	feedKey1 := UserFeed{URL: "http://feed1"}
	err = dbService.SetFetchStatus(feedKey1.CreateKey(), &FetchStatus{LastSuccess: time.Time{}})
	assert.NoError(t, err)
	feedKey2 := UserFeed{URL: "http://feed2"}
	err = dbService.SetFetchStatus(feedKey2.CreateKey(), &FetchStatus{LastSuccess: time.Time{}})
	assert.NoError(t, err)

	user := User{
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Updates" title="Updates">` +
			`<outline text="Site 2" title="Site 2" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`<outline text="Site 3" title="Site 3" type="rss" xmlUrl="http://feed2" htmlUrl="http://feed2"/>` +
			`</outline>` +
			`</body>` +
			`</opml>`,
	}
	dbItems, err := getFeedItems(&user)
	assert.NoError(t, err)
	assert.EqualValues(t, items, dbItems)
}
