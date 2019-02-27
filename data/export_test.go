package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var exportUsers = []*User{
	&User{
		Password:    "pass1",
		Opml:        "opml1",
		Pagemonitor: "pagemonitor1",
		Username:    "user01",
	},
	&User{
		Password:    "pass2",
		Opml:        "opml2",
		Pagemonitor: "pagemonitor2",
		Username:    "user02",
	},
}

var exportFeeditems = []*Feeditem{
	&Feeditem{
		Title:    "t1",
		URL:      "http://item1/1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "c1",
		Updated:  time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g1"},
	},
	&Feeditem{
		Title:    "t2",
		URL:      "http://item1/2",
		Date:     time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
		Contents: "c2",
		Updated:  time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed1", GUID: "g2"},
	},
	&Feeditem{
		Title:    "t3",
		URL:      "http://item2/1",
		Date:     time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
		Contents: "c3",
		Updated:  time.Date(2019, time.February, 18, 23, 2, 0, 0, time.UTC),
		Key:      &FeeditemKey{FeedURL: "http://feed2", GUID: "g1"},
	},
}

var exportPagemonitor = []*PagemonitorPage{
	&PagemonitorPage{
		Contents: "p1",
		Delta:    "d1",
		Updated:  time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
		Config:   &UserPagemonitor{URL: "http://site1", Match: "m1", Replace: "r1"},
	},
	&PagemonitorPage{
		Contents: "p2",
		Delta:    "d2",
		Updated:  time.Date(2019, time.February, 16, 23, 4, 0, 0, time.UTC),
		Config:   &UserPagemonitor{URL: "http://site2"},
	},
}

const exportData = `{
  "Users": [
    {
      "Password": "pass1",
      "Opml": "opml1",
      "Pagemonitor": "pagemonitor1",
      "Username": "user01"
    },
    {
      "Password": "pass2",
      "Opml": "opml2",
      "Pagemonitor": "pagemonitor2",
      "Username": "user02"
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
  ]
}`

func TestExport(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	for _, user := range exportUsers {
		dbService.NewUserService(user.Username).Save(user)
	}
	dbService.SaveFeeditems(exportFeeditems...)
	for _, page := range exportPagemonitor {
		dbService.SavePage(page)
	}

	data, err := dbService.Export()
	assert.NoError(t, err)
	assert.Equal(t, exportData, data)
}

func TestImport(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	err = dbService.Import(exportData)
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
	assert.Equal(t, exportUsers, dbUsers)

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
	assert.Equal(t, exportFeeditems, dbFeeditems)

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
	assert.Equal(t, exportPagemonitor, dbPages)
}
