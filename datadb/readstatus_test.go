package datadb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetReadStatusEmpty(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user := User{username: "user01"}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	readStatusesFeeditems, err := dbService.GetFeeditemsReadStatus(&user)
	assert.NoError(t, err)
	assert.Empty(t, readStatusesFeeditems)

	readStatusesPages, err := dbService.GetPagesReadStatus(&user)
	assert.NoError(t, err)
	assert.Empty(t, readStatusesPages)
}

func TestSaveGetFeeditemsReadStatus(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user := User{
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Site 1" title="Site 1" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`</body>` +
			`</opml>`,
		username: "user01",
	}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	item1 := Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Time{},
		Contents: "c1",
		Updated:  time.Time{},
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item1/1",
		Date:     time.Time{},
		Contents: "c2",
		Updated:  time.Time{},
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g2"},
	}
	err = dbService.SaveFeeditems(&item1, &item2)
	assert.NoError(t, err)

	err = dbService.SetFeeditemReadStatus(&user, item1.Key, true)
	assert.NoError(t, err)

	err = dbService.SetFeeditemReadStatus(&user, item2.Key, true)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetFeeditemsReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, []*FeeditemKey{item1.Key, item2.Key}, readStatuses)
}

func TestSaveGetPagesReadStatus(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user := User{
		Pagemonitor: `<pages>` +
			`<page url="https://site1.com" match="m1" replace="r1">Page 1</page>` +
			`<page url="http://site2.com">Page 2</page>` +
			`</pages>`,
		username: "user01",
	}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	key1 := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}
	key2 := &UserPagemonitor{URL: "http://site2.com"}

	err = dbService.SetPageReadStatus(&user, key1, true)
	assert.NoError(t, err)

	err = dbService.SetPageReadStatus(&user, key2, true)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetPagesReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, []*UserPagemonitor{key1, key2}, readStatuses)
}

func TestRemoveGetFeeditemReadStatus(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user := User{
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Site 1" title="Site 1" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`</body>` +
			`</opml>`,
		username: "user01",
	}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	item1 := Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Time{},
		Contents: "c1",
		Updated:  time.Time{},
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	}
	item2 := Feeditem{
		Title:    "t2",
		URL:      "http://item1/1",
		Date:     time.Time{},
		Contents: "c2",
		Updated:  time.Time{},
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g2"},
	}
	err = dbService.SaveFeeditems(&item1, &item2)
	assert.NoError(t, err)

	err = dbService.SetFeeditemReadStatus(&user, item1.Key, true)
	assert.NoError(t, err)

	err = dbService.SetFeeditemReadStatus(&user, item2.Key, true)
	assert.NoError(t, err)

	err = dbService.SetFeeditemReadStatus(&user, item1.Key, false)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetFeeditemsReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, []*FeeditemKey{item2.Key}, readStatuses)
}

func TestRemoveGetPageReadStatus(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user := User{
		Pagemonitor: `<pages>` +
			`<page url="https://site1.com" match="m1" replace="r1">Page 1</page>` +
			`<page url="http://site2.com">Page 2</page>` +
			`</pages>`,
		username: "user01",
	}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	key1 := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}
	key2 := &UserPagemonitor{URL: "http://site2.com"}

	err = dbService.SetPageReadStatus(&user, key1, true)
	assert.NoError(t, err)

	err = dbService.SetPageReadStatus(&user, key2, true)
	assert.NoError(t, err)

	err = dbService.SetPageReadStatus(&user, key1, false)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetPagesReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, []*UserPagemonitor{key2}, readStatuses)
}

func TestSetStatusoesntExist(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user := User{username: "user01"}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	feeditemKey := &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"}
	err = dbService.SetFeeditemReadStatus(&user, feeditemKey, false)
	assert.NoError(t, err)

	readStatusesFeeditems, err := dbService.GetFeeditemsReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, []*FeeditemKey{}, readStatusesFeeditems)

	pagemonitor := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}
	err = dbService.SetPageReadStatus(&user, pagemonitor, false)
	assert.NoError(t, err)

	readStatusesPages, err := dbService.GetPagesReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, []*UserPagemonitor{}, readStatusesPages)
}

func TestSetUnreadStatusForAll(t *testing.T) {
	err := prepareReadstatusTests()
	assert.NoError(t, err)

	user1 := User{
		Pagemonitor: `<pages>` +
			`<page url="https://site1.com" match="m1" replace="r1">Page 1</page>` +
			`<page url="http://site2.com">Page 2</page>` +
			`</pages>`,
		username: "user01",
	}
	err = dbService.SaveUser(&user1)
	assert.NoError(t, err)

	user2 := User{
		Pagemonitor: `<pages>` +
			`<page url="https://site1.com" match="m1" replace="r1">Page 1</page>` +
			`<page url="http://site2.com">Page 2</page>` +
			`</pages>`,
		username: "user02",
	}
	err = dbService.SaveUser(&user2)
	assert.NoError(t, err)

	key1 := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}
	key2 := &UserPagemonitor{URL: "http://site2.com"}

	err = dbService.SetPageReadStatus(&user1, key1, true)
	assert.NoError(t, err)

	err = dbService.SetPageReadStatus(&user1, key2, true)
	assert.NoError(t, err)

	err = dbService.SetPageReadStatus(&user2, key1, true)
	assert.NoError(t, err)

	err = dbService.SetPageUnreadForAll(key1)

	readStatuses, err := dbService.GetPagesReadStatus(&user1)
	assert.NoError(t, err)
	assert.Equal(t, []*UserPagemonitor{key2}, readStatuses)

	readStatuses, err = dbService.GetPagesReadStatus(&user2)
	assert.NoError(t, err)
	assert.Equal(t, []*UserPagemonitor{}, readStatuses)
}

func prepareReadstatusTests() error {
	cleanDatabases := []string{"users", "pagemonitors", "feeds"}
	for _, table := range cleanDatabases {
		_, err := dbService.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return err
		}
	}
	return nil
}
