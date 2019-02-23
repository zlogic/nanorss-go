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

func TestFetchPageFirstTime(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/1").Reply(200).
		BodyString("Hello World<br>First page")

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	pageConfig := data.UserPagemonitor{
		URL:   "http://site1/1",
		Title: "Site 1",
	}
	beforeUpdate := time.Now()
	dbMock.On("GetPage", &pageConfig).Return(nil, nil).Once()
	dbMock.On("SavePage", &pageConfig, mock.AnythingOfType("*data.PagemonitorPage")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			savedPage := args.Get(1).(*data.PagemonitorPage)
			assert.Equal(t, "Hello World\nFirst page", savedPage.Contents)
			assert.Equal(t, "@@ -1 +1,2 @@\n-\n+Hello World\n+First page\n", savedPage.Delta)
			assertTimeBetween(t, beforeUpdate, time.Now(), savedPage.Updated)
		})
	err := fetcher.FetchPage(&pageConfig)
	assert.NoError(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchPageNoChange(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/1").Reply(200).
		BodyString("Hello World<br>First page")

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	pageConfig := data.UserPagemonitor{
		URL:   "http://site1/1",
		Title: "Site 1",
	}
	existingResult := data.PagemonitorPage{
		Contents: "Hello World\nFirst page",
		Delta:    "+Hello World%0AFirst page",
		Updated:  time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
	}
	dbMock.On("GetPage", &pageConfig).Return(&existingResult, nil)
	err := fetcher.FetchPage(&pageConfig)
	assert.NoError(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchPageChanged(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/1").Reply(200).
		BodyString("Hello World<br>Updated page")

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	pageConfig := data.UserPagemonitor{
		URL:   "http://site1/1",
		Title: "Site 1",
	}
	existingResult := data.PagemonitorPage{
		Contents: "Hello World\nFirst page",
		Delta:    "+Hello World%0AFirst page",
		Updated:  time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
	}
	beforeUpdate := time.Now()
	dbMock.On("GetPage", &pageConfig).Return(&existingResult, nil)
	dbMock.On("SavePage", &pageConfig, mock.AnythingOfType("*data.PagemonitorPage")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			savedPage := args.Get(1).(*data.PagemonitorPage)
			assert.Equal(t, "Hello World\nUpdated page", savedPage.Contents)
			assert.Equal(t, "@@ -1,2 +1,2 @@\n Hello World\n-First page\n+Updated page\n", savedPage.Delta)
			assertTimeBetween(t, beforeUpdate, time.Now(), savedPage.Updated)
		})
	err := fetcher.FetchPage(&pageConfig)
	assert.NoError(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchPageMatchReplace(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/1").Reply(200).
		BodyString("Hello World<br>Updated page<br>New Line")

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	pageConfig := data.UserPagemonitor{
		URL:     "http://site1/1",
		Title:   "Site 1",
		Match:   "(?msi)^.*(hello .* page).*$",
		Replace: "$1",
	}
	existingResult := data.PagemonitorPage{
		Contents: "Hello World\nFirst page",
		Delta:    "+Hello World%0AFirst page",
		Updated:  time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
	}
	beforeUpdate := time.Now()
	dbMock.On("GetPage", &pageConfig).Return(&existingResult, nil)
	dbMock.On("SavePage", &pageConfig, mock.AnythingOfType("*data.PagemonitorPage")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			savedPage := args.Get(1).(*data.PagemonitorPage)
			assert.Equal(t, "Hello World\nUpdated page\nNew Line", savedPage.Contents)
			assert.Equal(t, "@@ -1,2 +1,2 @@\n Hello World\n-First page\n+Updated page\n", savedPage.Delta)
			assertTimeBetween(t, beforeUpdate, time.Now(), savedPage.Updated)
		})
	err := fetcher.FetchPage(&pageConfig)
	assert.NoError(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchPageError(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/1").Reply(400)

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	pageConfig := data.UserPagemonitor{
		URL:   "http://site1/1",
		Title: "Site 1",
	}
	existingResult := data.PagemonitorPage{
		Contents: "Hello World\nFirst page",
		Delta:    "+Hello World%0AFirst page",
		Updated:  time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
	}
	dbMock.On("GetPage", &pageConfig).Return(&existingResult, nil)
	err := fetcher.FetchPage(&pageConfig)
	assert.Error(t, err)
	dbMock.AssertExpectations(t)
}

func TestFetchTwoPages(t *testing.T) {
	defer gock.Off()

	gock.New("http://site1").Get("/1").Reply(200).
		BodyString("Hello World<br>Updated page 1")
	gock.New("http://site1/").Get("/2").Reply(200).
		BodyString("Hello World<br>Updated page 2")

	dbMock := new(DBMock)
	fetcher := Fetcher{
		DB:     dbMock,
		Client: &http.Client{},
	}

	pageConfig1 := data.UserPagemonitor{URL: "http://site1/1", Title: "Site 1"}
	pageConfig2 := data.UserPagemonitor{URL: "http://site1/2", Title: "Site 2"}
	existingResult1 := data.PagemonitorPage{
		Contents: "Hello World\nFirst page 1",
		Updated:  time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
	}
	existingResult2 := data.PagemonitorPage{
		Contents: "Hello World\nFirst page 2",
		Updated:  time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
	}
	beforeUpdate := time.Now()
	dbMock.On("ReadAllUsers", mock.AnythingOfType("func(string, *data.User)")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			handler := args.Get(0).(func(string, *data.User))
			user := data.User{Pagemonitor: `<pages>` +
				`<page url="http://site1/1">Site 1</page>` +
				`<page url="http://site1/2">Site 2</page>` +
				`</pages>`}
			handler("user01", &user)
		})
	dbMock.On("GetPage", &pageConfig1).Return(&existingResult1, nil)
	dbMock.On("GetPage", &pageConfig2).Return(&existingResult2, nil)
	dbMock.On("SavePage", &pageConfig1, mock.AnythingOfType("*data.PagemonitorPage")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			savedPage := args.Get(1).(*data.PagemonitorPage)
			assert.Equal(t, "Hello World\nUpdated page 1", savedPage.Contents)
			assert.Equal(t, "@@ -1,2 +1,2 @@\n Hello World\n-First page 1\n+Updated page 1\n", savedPage.Delta)
			assertTimeBetween(t, beforeUpdate, time.Now(), savedPage.Updated)
		})
	dbMock.On("SavePage", &pageConfig2, mock.AnythingOfType("*data.PagemonitorPage")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			savedPage := args.Get(1).(*data.PagemonitorPage)
			assert.Equal(t, "Hello World\nUpdated page 2", savedPage.Contents)
			assert.Equal(t, "@@ -1,2 +1,2 @@\n Hello World\n-First page 2\n+Updated page 2\n", savedPage.Delta)
			assertTimeBetween(t, beforeUpdate, time.Now(), savedPage.Updated)
		})
	err := fetcher.FetchAllPages()
	assert.NoError(t, err)
	dbMock.AssertExpectations(t)
}
