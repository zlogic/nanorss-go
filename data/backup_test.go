package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testBackupUsers = []*User{
	{
		Password:    "pass1",
		Opml:        "opml1",
		Pagemonitor: "pagemonitor1",
		username:    "user01",
	},
	{
		Password:    "pass2",
		Opml:        "opml2",
		Pagemonitor: "pagemonitor2",
		username:    "user02",
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
		Config:   &UserPagemonitor{URL: "http://site1", Match: "m1", Replace: "r1"},
	},
	{
		Contents: "p2",
		Delta:    "d2",
		Updated:  time.Date(2019, time.February, 16, 23, 4, 0, 0, time.UTC),
		Config:   &UserPagemonitor{URL: "http://site2"},
	},
}

const testBackupData = `{
  "Users": [
    {
      "Password": "pass1",
      "Opml": "opml1",
      "Pagemonitor": "pagemonitor1",
      "Username": "user01",
      "ReadItems": [
        "feeditem/aHR0cDovL2ZlZWQx/ZzE",
        "feeditem/aHR0cDovL2ZlZWQx/ZzI",
        "pagemonitor/aHR0cDovL3NpdGUx/bTE/cjE"
      ]
    },
    {
      "Password": "pass2",
      "Opml": "opml2",
      "Pagemonitor": "pagemonitor2",
      "Username": "user02",
      "ReadItems": [
        "feeditem/aHR0cDovL2ZlZWQx/ZzE",
        "feeditem/aHR0cDovL2ZlZWQy/ZzE",
        "pagemonitor/aHR0cDovL3NpdGUy//"
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
      "Title": "",
      "Match": "m1",
      "Replace": "r1"
    },
    {
      "Contents": "p2",
      "Delta": "d2",
      "Updated": "2019-02-16T23:04:00Z",
      "URL": "http://site2",
      "Title": "",
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
	assert.Equal(t, testBackupData, data)
}

func TestRestore(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	err = dbService.Restore(testBackupData)
	assert.NoError(t, err)

	done := make(chan bool)
	userChan := make(chan *User)
	dbUsers := make([]*User, 0)
	go func() {
		for user := range userChan {
			dbUsers = append(dbUsers, user)
		}
		done <- true
	}()
	err = dbService.ReadAllUsers(userChan)
	assert.NoError(t, err)
	<-done
	assert.Equal(t, testBackupUsers, dbUsers)

	user1ReadStatus, err := dbService.GetReadStatus(testBackupUsers[0])
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{
		testBackupFeeditems[0].Key.CreateKey(),
		testBackupFeeditems[1].Key.CreateKey(),
		testBackupPagemonitor[0].Config.CreateKey(),
	}, user1ReadStatus)

	user2ReadStatus, err := dbService.GetReadStatus(testBackupUsers[1])
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{
		testBackupFeeditems[0].Key.CreateKey(),
		testBackupFeeditems[2].Key.CreateKey(),
		testBackupPagemonitor[1].Config.CreateKey(),
	}, user2ReadStatus)

	feedChan := make(chan *Feeditem)
	dbFeeditems := make([]*Feeditem, 0)
	go func() {
		for feedItem := range feedChan {
			dbFeeditems = append(dbFeeditems, feedItem)
		}
		done <- true
	}()
	err = dbService.ReadAllFeedItems(feedChan)
	assert.NoError(t, err)
	<-done
	assert.Equal(t, testBackupFeeditems, dbFeeditems)

	pageChan := make(chan *PagemonitorPage)
	dbPages := make([]*PagemonitorPage, 0)
	go func() {
		for page := range pageChan {
			dbPages = append(dbPages, page)
		}
		close(done)
	}()
	err = dbService.ReadAllPages(pageChan)
	assert.NoError(t, err)
	<-done
	assert.Equal(t, testBackupPagemonitor, dbPages)

	values, err := dbService.GetAllConfigVariables()
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, values)
}
