package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testBackupUsers = []*User{
	{
		Password: "pass1",
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Updates" title="Updates">` +
			`<outline text="Site 2" title="Site 2" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`<outline text="Site 3" title="Site 3" type="rss" xmlUrl="http://feed2" htmlUrl="http://feed2"/>` +
			`</outline>` +
			`</body>` +
			`</opml>`,
		Pagemonitor: `<pages>` +
			`<page url="http://site1" match="m1" replace="r1">Site 1</page>` +
			`<page url="http://site2">Site 2</page>` +
			`</pages>`,
		username: "user01",
	},
	{
		Password: "pass2",
		Opml: `<opml version="1.0">` +
			`<body>` +
			`<outline text="Updates" title="Updates">` +
			`<outline text="Site 2" title="Site 2" type="rss" xmlUrl="http://feed1" htmlUrl="http://feed1"/>` +
			`<outline text="Site 3" title="Site 3" type="rss" xmlUrl="http://feed2" htmlUrl="http://feed2"/>` +
			`</outline>` +
			`</body>` +
			`</opml>`,
		Pagemonitor: `<pages>` +
			`<page url="http://site1" match="m1" replace="r1">Site 1</page>` +
			`</pages>`,
		username: "user02",
	},
}

var testBackupFeeditems = []*Feeditem{
	{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	},
	{
		Title:    "t2",
		URL:      "http://item1/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Updated:  time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g2"},
	},
	{
		Title:    "t3",
		URL:      "http://item2/1",
		Date:     time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
		Contents: "c3",
		Updated:  time.Date(2019, time.February, 18, 23, 2, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed2", GUID: "g1"},
	},
}

var testBackupPagemonitor = []*PagemonitorPage{
	{
		Contents: "p1",
		Delta:    "d1",
		Updated:  time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
		Config:   &UserPagemonitor{Title: "Site 1", URL: "http://site1", Match: "m1", Replace: "r1"},
	},
	{
		Contents: "p2",
		Delta:    "d2",
		Updated:  time.Date(2019, time.February, 16, 23, 4, 0, 0, time.UTC),
		Config:   &UserPagemonitor{Title: "Site 2", URL: "http://site2"},
	},
}

const testBackupData = `{
  "Users": [
    {
      "Password": "pass1",
      "Opml": "<opml version=\"1.0\"><body><outline text=\"Updates\" title=\"Updates\"><outline text=\"Site 2\" title=\"Site 2\" type=\"rss\" xmlUrl=\"http://feed1\" htmlUrl=\"http://feed1\"/><outline text=\"Site 3\" title=\"Site 3\" type=\"rss\" xmlUrl=\"http://feed2\" htmlUrl=\"http://feed2\"/></outline></body></opml>",
      "Pagemonitor": "<pages><page url=\"http://site1\" match=\"m1\" replace=\"r1\">Site 1</page><page url=\"http://site2\">Site 2</page></pages>",
      "Username": "user01",
      "ReadItems": [
        "pagemonitor/aHR0cDovL3NpdGUx/bTE/cjE",
        "feed/aHR0cDovL2ZlZWQx/ZzE",
        "feed/aHR0cDovL2ZlZWQx/ZzI"
      ]
    },
    {
      "Password": "pass2",
      "Opml": "<opml version=\"1.0\"><body><outline text=\"Updates\" title=\"Updates\"><outline text=\"Site 2\" title=\"Site 2\" type=\"rss\" xmlUrl=\"http://feed1\" htmlUrl=\"http://feed1\"/><outline text=\"Site 3\" title=\"Site 3\" type=\"rss\" xmlUrl=\"http://feed2\" htmlUrl=\"http://feed2\"/></outline></body></opml>",
      "Pagemonitor": "<pages><page url=\"http://site1\" match=\"m1\" replace=\"r1\">Site 1</page></pages>",
      "Username": "user02",
      "ReadItems": [
        "pagemonitor/aHR0cDovL3NpdGUy//",
        "feed/aHR0cDovL2ZlZWQx/ZzE",
        "feed/aHR0cDovL2ZlZWQy/ZzE"
      ]
    }
  ],
  "Feeds": [
    {
      "Title": "t1",
      "URL": "http://item1/1",
      "Date": "2019-02-16T23:00:00Z",
      "Contents": "c1",
      "Updated": "2019-02-18T23:00:00Z",
      "FeedURL": "http://feed1",
      "GUID": "g1"
    },
    {
      "Title": "t2",
      "URL": "http://item1/2",
      "Date": "2019-02-16T23:01:00Z",
      "Contents": "c2",
      "Updated": "2019-02-18T23:01:00Z",
      "FeedURL": "http://feed1",
      "GUID": "g2"
    },
    {
      "Title": "t3",
      "URL": "http://item2/1",
      "Date": "2019-02-16T23:02:00Z",
      "Contents": "c3",
      "Updated": "2019-02-18T23:02:00Z",
      "FeedURL": "http://feed2",
      "GUID": "g1"
    }
  ],
  "Pagemonitor": [
    {
      "Contents": "p1",
      "Delta": "d1",
      "Updated": "2019-02-16T23:03:00Z",
      "URL": "http://site1",
      "Title": "Site 1",
      "Match": "m1",
      "Replace": "r1"
    },
    {
      "Contents": "p2",
      "Delta": "d2",
      "Updated": "2019-02-16T23:04:00Z",
      "URL": "http://site2",
      "Title": "Site 2",
      "Match": "",
      "Replace": ""
    }
  ],
  "ServerConfig": {
    "k1": "v1",
    "k2": "v2"
  }
}`

func TestBackup(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	for _, user := range testBackupUsers {
		dbService.SaveUser(user)
	}
	dbService.SaveFeeditems(testBackupFeeditems...)
	for _, page := range testBackupPagemonitor {
		dbService.SavePage(page)
	}

	dbService.SetReadStatus(testBackupUsers[0], testBackupFeeditems[0].Key.CreateKey(), true)
	dbService.SetReadStatus(testBackupUsers[0], testBackupFeeditems[1].Key.CreateKey(), true)
	dbService.SetReadStatus(testBackupUsers[0], testBackupPagemonitor[0].Config.CreateKey(), true)
	dbService.SetReadStatus(testBackupUsers[1], testBackupFeeditems[0].Key.CreateKey(), true)
	dbService.SetReadStatus(testBackupUsers[1], testBackupFeeditems[2].Key.CreateKey(), true)
	dbService.SetReadStatus(testBackupUsers[1], testBackupPagemonitor[1].Config.CreateKey(), true)

	dbService.SetConfigVariable("k1", "v1")
	dbService.SetConfigVariable("k2", "v2")

	data, err := dbService.Backup()
	assert.NoError(t, err)
	assert.JSONEq(t, testBackupData, data)
}

func TestRestore(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	err = dbService.Restore(testBackupData)
	assert.NoError(t, err)

	dbUsers, err := getAllUsers(dbService)
	assert.NoError(t, err)
	assert.Equal(t, testBackupUsers, dbUsers)

	readStatus, err := dbService.GetReadStatus(testBackupUsers[0], testBackupFeeditems[0].Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[0], testBackupFeeditems[1].Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[0], testBackupFeeditems[2].Key.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[0], testBackupPagemonitor[0].Config.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[0], testBackupPagemonitor[1].Config.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[1], testBackupFeeditems[0].Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[1], testBackupFeeditems[1].Key.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[1], testBackupFeeditems[2].Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[1], testBackupPagemonitor[0].Config.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(testBackupUsers[1], testBackupPagemonitor[1].Config.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	user := &User{username: "user01", Opml: testBackupUsers[0].Opml, Pagemonitor: testBackupUsers[0].Pagemonitor}
	dbFeeditems, err := dbService.GetFeeditems(user)
	assert.NoError(t, err)
	assert.Equal(t, testBackupFeeditems, dbFeeditems)

	dbPages, err := dbService.GetPages(user)
	assert.NoError(t, err)
	assert.Equal(t, testBackupPagemonitor, dbPages)

	values, err := dbService.GetAllConfigVariables()
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, values)
}
