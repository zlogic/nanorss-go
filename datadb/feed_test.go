package datadb

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetItemEmpty(t *testing.T) {
	err := prepareFeedTests()
	assert.NoError(t, err)

	key := FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	item, err := dbService.GetFeeditem(&key)
	assert.NoError(t, err)
	assert.Nil(t, item)
}

func TestSaveReadItem(t *testing.T) {
	err := prepareFeedTests()
	assert.NoError(t, err)

	prepareFeeds(dbService, "http://feed1", "http://feed2")

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
	err := prepareFeedTests()
	assert.NoError(t, err)

	prepareFeeds(dbService, "http://feed1")

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
	err := prepareFeedTests()
	assert.NoError(t, err)

	prepareFeeds(dbService, "http://feed1")

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

func TestSaveReadItemTTLExpiredItem(t *testing.T) {
	err := prepareFeedTests()
	assert.NoError(t, err)

	prepareFeeds(dbService, "http://feed1")

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
	err = dbService.SaveFeeditems(item)
	assert.NoError(t, err)

	expiredTime := time.Now().Add(-time.Minute * 15).UTC().Truncate(time.Millisecond)
	_, err = dbService.db.Exec("UPDATE feeditems SET last_seen=$1 WHERE url='http://item1'", expiredTime)
	assert.NoError(t, err)

	err = dbService.deleteExpiredItems()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
}

func TestSaveReadItemTTLExpiredFeed(t *testing.T) {
	err := prepareFeedTests()
	assert.NoError(t, err)

	prepareFeeds(dbService, "http://feed1")
	_, err = dbService.db.Exec("UPDATE feeds SET last_success = NULL")
	assert.NoError(t, err)

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

	err = dbService.deleteExpiredItems()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Nil(t, dbItem)
}

func TestSaveReadItemTTLNotExpired(t *testing.T) {
	err := prepareFeedTests()
	assert.NoError(t, err)

	prepareFeeds(dbService, "http://feed1")

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

	err = dbService.deleteExpiredItems()
	assert.NoError(t, err)

	dbItem, err := dbService.GetFeeditem(item.Key)
	assert.NoError(t, err)
	assert.Equal(t, item, dbItem)
}

func TestSaveReadAllItems(t *testing.T) {
	err := prepareFeedTests()
	assert.NoError(t, err)

	user1 := &User{
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Site 1" title="Site 1" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`</body>` +
			`</opml>`,
		username: "user01",
	}
	err = dbService.SaveUser(user1)
	assert.NoError(t, err)
	user2 := &User{
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Site 1" title="Site 1" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`<outline text="Site 2" title="Site 2" type="rss" xmlUrl="http://feed2" htmlUrl="http://feed2"/>` +
			`</body>` +
			`</opml>`,
		username: "user02",
	}
	err = dbService.SaveUser(user2)
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
	err = dbService.SaveFeeditems(&item1, &item2, &item3)
	assert.NoError(t, err)

	items1 := []*Feeditem{&item1, &item2}
	dbItems1, err := dbService.GetFeeditems(user1)
	assert.NoError(t, err)
	assert.EqualValues(t, items1, dbItems1)

	items2 := []*Feeditem{&item1, &item2, &item3}
	dbItems2, err := dbService.GetFeeditems(user2)
	assert.NoError(t, err)
	assert.EqualValues(t, items2, dbItems2)
}

func prepareFeedTests() error {
	cleanDatabases := []string{"users", "feeds"}
	for _, table := range cleanDatabases {
		_, err := dbService.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return err
		}
	}
	return nil
}

func prepareFeeds(s *DBService, feedURLs ...string) error {
	return s.updateTx(func(tx *sql.Tx) error {
		for _, feedURL := range feedURLs {
			_, err := s.db.Exec("INSERT INTO feeds(url, last_success) VALUES($1, NOW())", feedURL)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
