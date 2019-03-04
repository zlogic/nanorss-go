package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zlogic/nanorss-go/data"
)

const defaultOpml = `<opml version="1.0">` +
	`<body>` +
	`<outline title="Feed 1" type="rss" xmlUrl="http://site1/rss"/>` +
	`<outline title="Feed 2" type="rss" xmlUrl="http://site2/rss"/>` +
	`</body>` +
	`</opml>`

const defaultPagemonitor = `<pages>` +
	`<page url="http://site1/1" match="m1" replace="r1">Site 1</page>` +
	`<page url="http://site1/2">Site 2</page>` +
	`</pages>`

func (m *DBMock) configureMockForFeedList(feedItems []*data.Feeditem, pages []*data.PagemonitorPage) {
	m.On("ReadAllFeedItems", mock.AnythingOfType("chan *data.Feeditem")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			ch := args.Get(0).(chan *data.Feeditem)
			for _, feedItem := range feedItems {
				ch <- feedItem
			}
			close(ch)
		})
	m.On("ReadAllPages", mock.AnythingOfType("chan *data.PagemonitorPage")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			ch := args.Get(0).(chan *data.PagemonitorPage)
			for _, page := range pages {
				ch <- page
			}
			close(ch)
		})
}

func TestFeedListHelperEmptyList(t *testing.T) {
	dbMock := new(DBMock)
	feedListService := FeedListService{db: dbMock}
	user := &data.User{
		Opml:        defaultOpml,
		Pagemonitor: defaultPagemonitor,
	}

	dbMock.configureMockForFeedList([]*data.Feeditem{}, []*data.PagemonitorPage{})

	items, err := feedListService.GetAllItems(user)
	assert.NoError(t, err)
	assert.Empty(t, items)

	dbMock.AssertExpectations(t)
}

func TestFeedListHelperOrdering(t *testing.T) {
	dbMock := new(DBMock)
	feedListService := FeedListService{db: dbMock}
	user := &data.User{
		Opml:        defaultOpml,
		Pagemonitor: defaultPagemonitor,
	}

	expectedItems := []*Item{
		&Item{
			Origin:   "Site 2",
			SortDate: time.Date(2019, time.February, 18, 23, 3, 0, 0, time.UTC),
			FetchURL: "api/items/pagemonitor-aHR0cDovL3NpdGUxLzI--",
		},
		&Item{
			Title:    "t2",
			Origin:   "Feed 1",
			SortDate: time.Date(2019, time.February, 18, 23, 2, 0, 0, time.UTC),
			FetchURL: "api/items/feeditem-aHR0cDovL3NpdGUxL3Jzcw-ZzI",
		},
		&Item{
			Title:    "t1",
			Origin:   "Feed 1",
			SortDate: time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
			FetchURL: "api/items/feeditem-aHR0cDovL3NpdGUxL3Jzcw-ZzE",
		},
		&Item{
			Title:    "t21",
			Origin:   "Feed 2",
			SortDate: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
			FetchURL: "api/items/feeditem-aHR0cDovL3NpdGUyL3Jzcw-ZzE",
		},
		&Item{
			Origin:   "Site 1",
			SortDate: time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
			FetchURL: "api/items/pagemonitor-aHR0cDovL3NpdGUxLzE-bTE-cjE",
		},
	}

	feedItems := []*data.Feeditem{
		&data.Feeditem{
			Title:   "t1",
			Key:     &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"},
			Date:    time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
		},
		&data.Feeditem{
			Title:   "t21",
			Key:     &data.FeeditemKey{FeedURL: "http://site2/rss", GUID: "g1"},
			Date:    time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		},
		&data.Feeditem{
			Title:   "t2",
			Key:     &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g2"},
			Date:    time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 2, 0, 0, time.UTC),
		},
	}

	pages := []*data.PagemonitorPage{
		&data.PagemonitorPage{
			Config:  &data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"},
			Updated: time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
		},
		&data.PagemonitorPage{
			Config:  &data.UserPagemonitor{URL: "http://site1/2"},
			Updated: time.Date(2019, time.February, 18, 23, 3, 0, 0, time.UTC),
		},
	}

	dbMock.configureMockForFeedList(feedItems, pages)

	items, err := feedListService.GetAllItems(user)
	assert.NoError(t, err)
	assert.Equal(t, expectedItems, items)

	dbMock.AssertExpectations(t)
}

func TestFeedListHelperIgnoreUnknownItems(t *testing.T) {
	dbMock := new(DBMock)
	feedListService := FeedListService{db: dbMock}
	user := &data.User{
		Opml:        defaultOpml,
		Pagemonitor: defaultPagemonitor,
	}

	feedItems := []*data.Feeditem{
		&data.Feeditem{
			Title:   "t1",
			Key:     &data.FeeditemKey{FeedURL: "http://site3/rss", GUID: "g1"},
			Date:    time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		},
	}

	pages := []*data.PagemonitorPage{
		&data.PagemonitorPage{
			Config:  &data.UserPagemonitor{URL: "http://site1/1"},
			Updated: time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
		},
		&data.PagemonitorPage{
			Config:  &data.UserPagemonitor{URL: "http://site1/3"},
			Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		},
	}

	dbMock.configureMockForFeedList(feedItems, pages)

	items, err := feedListService.GetAllItems(user)
	assert.NoError(t, err)
	assert.Empty(t, items)

	dbMock.AssertExpectations(t)
}
