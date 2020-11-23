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
	m.On("GetFeeditems", mock.AnythingOfType("*data.User")).Return(feedItems, nil).Once()
	m.On("GetPages", mock.AnythingOfType("*data.User")).Return(pages, nil).Once()
}

func TestFeedListHelperEmptyList(t *testing.T) {
	dbMock := new(DBMock)
	feedListService := FeedListService{db: dbMock}
	user := &data.User{
		Opml:        defaultOpml,
		Pagemonitor: defaultPagemonitor,
	}

	dbMock.configureMockForFeedList([]*data.Feeditem{}, []*data.PagemonitorPage{})
	dbMock.On("GetFeeditemsReadStatus", user).Return(nil, nil).Once()
	dbMock.On("GetPagesReadStatus", user).Return(nil, nil).Once()

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
		{
			Origin:   "Site 2",
			SortDate: time.Date(2019, time.February, 18, 23, 3, 0, 0, time.UTC),
			FetchURL: "api/items/page/http:%252F%252Fsite1%252F2%2F%2F",
			IsRead:   false,
		},
		{
			Title:    "t2",
			Origin:   "Feed 1",
			SortDate: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
			FetchURL: "api/items/feeditem/http:%252F%252Fsite1%252Frss%2Fg2",
			IsRead:   false,
		},
		{
			Title:    "t1",
			Origin:   "Feed 1",
			SortDate: time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
			FetchURL: "api/items/feeditem/http:%252F%252Fsite1%252Frss%2Fg1",
			IsRead:   false,
		},
		{
			Title:    "t21",
			Origin:   "Feed 2",
			SortDate: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
			FetchURL: "api/items/feeditem/http:%252F%252Fsite2%252Frss%2Fg1",
			IsRead:   true,
		},
		{
			Origin:   "Site 1",
			SortDate: time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
			FetchURL: "api/items/page/http:%252F%252Fsite1%252F1%2Fm1%2Fr1",
			IsRead:   true,
		},
	}

	feedItems := []*data.Feeditem{
		{
			Title:   "t1",
			Key:     &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"},
			Date:    time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 1, 0, 0, time.UTC),
		},
		{
			Title:   "t21",
			Key:     &data.FeeditemKey{FeedURL: "http://site2/rss", GUID: "g1"},
			Date:    time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		},
		{
			Title:   "t2",
			Key:     &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g2"},
			Date:    time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 2, 0, 0, time.UTC),
		},
	}

	pages := []*data.PagemonitorPage{
		{
			Config:  &data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"},
			Updated: time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
		},
		{
			Config:  &data.UserPagemonitor{URL: "http://site1/2"},
			Updated: time.Date(2019, time.February, 18, 23, 3, 0, 0, time.UTC),
		},
	}

	readPages := []*data.UserPagemonitor{pages[0].Config}
	readFeeditems := []*data.FeeditemKey{feedItems[1].Key}

	dbMock.configureMockForFeedList(feedItems, pages)
	dbMock.On("GetFeeditemsReadStatus", user).Return(readFeeditems, nil).Once()
	dbMock.On("GetPagesReadStatus", user).Return(readPages, nil).Once()

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
		{
			Title:   "t1",
			Key:     &data.FeeditemKey{FeedURL: "http://site3/rss", GUID: "g1"},
			Date:    time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
			Updated: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
		},
	}

	pages := []*data.PagemonitorPage{
		{
			Config:  &data.UserPagemonitor{URL: "http://site1/1"},
			Updated: time.Date(2019, time.February, 16, 23, 3, 0, 0, time.UTC),
		},
		{
			Config:  &data.UserPagemonitor{URL: "http://site1/3"},
			Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		},
	}

	dbMock.configureMockForFeedList(feedItems, pages)
	dbMock.On("GetFeeditemsReadStatus", user).Return(nil, nil).Once()
	dbMock.On("GetPagesReadStatus", user).Return(nil, nil).Once()

	items, err := feedListService.GetAllItems(user)
	assert.NoError(t, err)
	assert.Empty(t, items)

	dbMock.AssertExpectations(t)
}
