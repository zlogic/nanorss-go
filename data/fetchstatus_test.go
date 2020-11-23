package data

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetFetchStatusEmpty(t *testing.T) {
	err := prepareFetchstatusTests()
	assert.NoError(t, err)

	fetchStatus, err := dbService.GetFeedFetchStatus("http://feed1")
	assert.NoError(t, err)
	assert.Nil(t, fetchStatus)

	key := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}
	fetchStatus, err = dbService.GetPageFetchStatus(key)
	assert.NoError(t, err)
	assert.Nil(t, fetchStatus)
}

func TestSaveGetFeedFetchStatus(t *testing.T) {
	err := prepareFetchstatusTests()
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

	fetchStatus := &FetchStatus{LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)}
	err = dbService.SetFeedFetchStatus("http://feed1", fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err := dbService.GetFeedFetchStatus("http://feed1")
	assert.NoError(t, err)
	assert.Equal(t, fetchStatus, dbFetchStatus)
}

func TestSaveGetPageFetchStatus(t *testing.T) {
	err := prepareFetchstatusTests()
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

	key := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}
	fetchStatus := &FetchStatus{LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)}
	err = dbService.SetPageFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err := dbService.GetPageFetchStatus(key)
	assert.NoError(t, err)
	assert.Equal(t, fetchStatus, dbFetchStatus)
}

func TestUpdateFeedFetchStatus(t *testing.T) {
	err := prepareFetchstatusTests()
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

	fetchStatus := &FetchStatus{LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)}
	err = dbService.SetFeedFetchStatus("http://feed1", fetchStatus)
	assert.NoError(t, err)

	fetchStatus = &FetchStatus{LastFailure: time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC), LastFailureError: "random error"}
	err = dbService.SetFeedFetchStatus("http://feed1", fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err := dbService.GetFeedFetchStatus("http://feed1")
	assert.NoError(t, err)
	assert.Equal(t, &FetchStatus{
		LastSuccess:      time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastFailure:      time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		LastFailureError: "random error",
	}, dbFetchStatus)

	fetchStatus = &FetchStatus{LastFailure: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC)}
	err = dbService.SetFeedFetchStatus("http://feed1", fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err = dbService.GetFeedFetchStatus("http://feed1")
	assert.NoError(t, err)
	assert.Equal(t, &FetchStatus{
		LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastFailure: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
	}, dbFetchStatus)
}

func TestUpdatePageFetchStatus(t *testing.T) {
	err := prepareFetchstatusTests()
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

	key := &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"}

	fetchStatus := &FetchStatus{LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)}
	err = dbService.SetPageFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	fetchStatus = &FetchStatus{LastFailure: time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC), LastFailureError: "random error"}
	err = dbService.SetPageFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err := dbService.GetPageFetchStatus(key)
	assert.NoError(t, err)
	assert.Equal(t, &FetchStatus{
		LastSuccess:      time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastFailure:      time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		LastFailureError: "random error",
	}, dbFetchStatus)

	fetchStatus = &FetchStatus{LastFailure: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC)}
	err = dbService.SetPageFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err = dbService.GetPageFetchStatus(key)
	assert.NoError(t, err)
	assert.Equal(t, &FetchStatus{
		LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastFailure: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
	}, dbFetchStatus)
}

func prepareFetchstatusTests() error {
	cleanDatabases := []string{"users", "pagemonitors", "feeds"}
	for _, table := range cleanDatabases {
		_, err := dbService.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return err
		}
	}
	return nil
}
