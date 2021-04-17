package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func checkPages(t *testing.T, dbService *DBService, pages *[]PagemonitorPage) {
	usernames, err := dbService.GetUsers()
	assert.NoError(t, err)

	dbPages := make([]PagemonitorPage, 0)
	for _, username := range usernames {
		user, err := dbService.GetUser(username)
		assert.NoError(t, err)

		userPages, err := dbService.GetPages(user)
		assert.NoError(t, err)

		for _, dbPage := range userPages {
			dbPages = append(dbPages, *dbPage)
		}
	}

	assert.NoError(t, err)
	assert.ElementsMatch(t, *pages, dbPages)
}

func TestGetPage(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC), Config: &userPage}
	err = dbService.SavePage(&page)
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Equal(t, &page, dbPage)
}

func TestSavePage(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	userPage1 := UserPagemonitor{
		Title:   "Page 1 (match/replace)",
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	userPage2 := UserPagemonitor{
		Title: "Page 1",
		URL:   "http://site1.com",
	}
	page1 := PagemonitorPage{Config: &userPage1}
	page2 := PagemonitorPage{Config: &userPage2}
	pages := []PagemonitorPage{page1, page2}

	user := User{
		Pagemonitor: `<pages>` +
			`<page url="http://site1.com" match="m1" replace="r1">Page 1 (match/replace)</page>` +
			`<page url="http://site1.com">Page 1</page>` +
			`</pages>`,
		username: "user01",
	}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	// Empty pages.
	err = dbService.SavePage(&page1)
	assert.NoError(t, err)
	err = dbService.SavePage(&page2)
	assert.NoError(t, err)
	checkPages(t, dbService, &pages)

	// Update one page.
	page1.Contents = "c1"
	page1.Delta = "d1"
	page1.Updated = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	pages[0] = page1
	err = dbService.SavePage(&page1)
	assert.NoError(t, err)
	checkPages(t, dbService, &pages)
}

func TestSaveReadPageTTLExpired(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)
	var oldTTL = itemTTL
	defer func() { itemTTL = oldTTL }()

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC), Config: &userPage}
	err = dbService.SavePage(&page)
	assert.NoError(t, err)
	err = dbService.SetFetchStatus(userPage.CreateKey(), &FetchStatus{LastSuccess: time.Now()})
	assert.NoError(t, err)

	itemTTL = time.Minute * 1
	err = dbService.deleteStaleFetchStatuses()
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Equal(t, &page, dbPage)

	itemTTL = time.Nanosecond * 0
	err = dbService.deleteStaleFetchStatuses()
	assert.NoError(t, err)

	dbPage, err = dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Nil(t, dbPage)
}

func TestSaveReadPageTTLNotExpired(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC), Config: &userPage}
	err = dbService.SavePage(&page)
	assert.NoError(t, err)

	err = dbService.SetFetchStatus(userPage.CreateKey(), &FetchStatus{LastSuccess: time.Now().Add(time.Minute * 1)})
	assert.NoError(t, err)
	err = dbService.deleteStaleFetchStatuses()
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Equal(t, &page, dbPage)
}
