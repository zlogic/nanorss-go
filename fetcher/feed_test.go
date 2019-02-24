package fetcher

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zlogic/nanorss-go/data"
	"gopkg.in/h2non/gock.v1"
)

const rssFeed = `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
<channel>
<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>
<item><title>Title 1</title><link>http://site1/link1</link><description>Text 1</description><pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate><guid>Item@1</guid></item>
<item><title>Title 2</title><link>http://site1/link2</link><description>Text 2</description><guid>Item@2</guid></item>
<item><title>Title 3</title><link>http://site1/link3</link><description>Text 3</description><pubDate>Tue, 07 Jun 2016 13:19:00 GMT</pubDate></item>
<item><title>Title 4</title><link>http://site1/link4</link><description>Text 4</description><content:encoded>Content 4</content:encoded><pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate><guid>Item@4</guid></item>
</channel>
</rss>`

var expectedRssFeedItems = []*data.Feeditem{
	&data.Feeditem{
		Title:    "Title 1",
		URL:      "http://site1/link1",
		Date:     time.Date(2016, time.June, 8, 10, 34, 0, 0, time.UTC),
		Contents: "Text 1",
		Key:      &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "Item@1"},
	},
	&data.Feeditem{
		Title:    "Title 2",
		URL:      "http://site1/link2",
		Contents: "Text 2",
		Key:      &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "Item@2"},
	},
	&data.Feeditem{
		Title:    "Title 3",
		URL:      "http://site1/link3",
		Date:     time.Date(2016, time.June, 7, 13, 19, 0, 0, time.UTC),
		Contents: "Text 3",
		Key:      &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "http://site1/link3"},
	},
	&data.Feeditem{
		Title:    "Title 4",
		URL:      "http://site1/link4",
		Date:     time.Date(2016, time.June, 8, 10, 34, 0, 0, time.UTC),
		Contents: "Text 4",
		Key:      &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "Item@4"},
	},
}

func TestFetchFeed(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/rss").Reply(200).
		BodyString(rssFeed)

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	feedURL := "http://site1/rss"
	beforeUpdate := time.Now()
	dbMock.On("SaveFeeditems", mock.AnythingOfType("[]*data.Feeditem")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			savedItems := args.Get(0).([]*data.Feeditem)
			if len(savedItems) == 4 {
				assertTimeBetween(t, beforeUpdate, time.Now(), savedItems[1].Date)
				savedItems[1].Date = time.Time{}
			}
			assert.Equal(t, expectedRssFeedItems, savedItems)
		})
	err := fetcher.FetchFeed(feedURL)
	assert.NoError(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchFeedError(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/rss").Reply(400)

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	feedURL := "http://site1/rss"
	err := fetcher.FetchFeed(feedURL)
	assert.Error(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchAllFeeds(t *testing.T) {
	defer gock.Off()

	secondRssFeed := `<rss version="2.0">` +
		`<channel>` +
		`<pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate>` +
		`<item><title>Title 21</title><link>http://site2/link1</link><description>Text 21</description><pubDate>Wed, 08 Jun 2016 10:34:00 GMT</pubDate><guid>Item@1</guid></item>` +
		`</channel>` +
		`</rss>`

	expectedSecondRssFeedItems := []*data.Feeditem{
		&data.Feeditem{
			Title:    "Title 21",
			URL:      "http://site2/link1",
			Date:     time.Date(2016, time.June, 8, 10, 34, 0, 0, time.UTC),
			Contents: "Text 21",
			Key:      &data.FeeditemKey{FeedURL: "http://site2/rss", GUID: "Item@1"},
		},
	}

	gock.New("http://site1").Get("/rss").Reply(200).
		BodyString(rssFeed)

	gock.New("http://site2").Get("/rss").Reply(200).
		BodyString(secondRssFeed)

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	dbMock.On("ReadAllUsers", mock.AnythingOfType("chan *data.User")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			ch := args.Get(0).(chan *data.User)
			defer close(ch)
			user := data.User{Opml: `<opml version="1.0">` +
				`<body>` +
				`<outline title="Feed 1" type="rss" xmlUrl="http://site1/rss"/>` +
				`<outline title="Feed 2" type="rss" xmlUrl="http://site2/rss"/>` +
				`</body>` +
				`</opml>`}
			ch <- &user
		})

	beforeUpdate := time.Now()
	dbSavedItems := make([][]*data.Feeditem, 0, 2)
	expectedSavedItems := [][]*data.Feeditem{expectedRssFeedItems, expectedSecondRssFeedItems}
	dbMock.On("SaveFeeditems", mock.AnythingOfType("[]*data.Feeditem")).Return(nil).Twice().
		Run(func(args mock.Arguments) {
			savedItems := args.Get(0).([]*data.Feeditem)
			if len(savedItems) == 4 {
				assertTimeBetween(t, beforeUpdate, time.Now(), savedItems[1].Date)
				savedItems[1].Date = time.Time{}
			}
			dbSavedItems = append(dbSavedItems, savedItems)
		})
	err := fetcher.FetchAllFeeds()
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedSavedItems, dbSavedItems)
	dbMock.AssertExpectations(t)
}
