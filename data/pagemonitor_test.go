package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func checkPages(t *testing.T, dbService DBService, userPages *[]UserPagemonitor, pages *[]PagemonitorPage) {
	dbPages := make([]PagemonitorPage, 0, 2)
	ch := make(chan PagemonitorPage)
	done := make(chan bool)
	go func() {
		for page := range ch {
			dbPages = append(dbPages, page)
		}
		close(done)
	}()
	err := dbService.ReadAllPages(ch)
	<-done
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
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC), Config: userPage}
	err = dbService.SavePage(page)
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(userPage)
	assert.NoError(t, err)
	assert.Equal(t, page, dbPage)
}

func TestSavePage(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	userPage1 := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	userPage2 := UserPagemonitor{
		URL: "http://site1.com",
	}
	userPages := []UserPagemonitor{userPage1, userPage2}
	page1 := PagemonitorPage{Contents: "c1", Config: userPage1}
	page2 := PagemonitorPage{Contents: "c2", Config: userPage2}
	pages := []PagemonitorPage{page1, page2}

	//Empty pages
	err = dbService.SavePage(page1)
	assert.NoError(t, err)
	err = dbService.SavePage(page2)
	assert.NoError(t, err)
	checkPages(t, dbService, &userPages, &pages)

	//Update one page
	page1.Contents = "c1"
	page1.Delta = "d1"
	page1.Updated = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	pages[0] = page1
	err = dbService.SavePage(page1)
	assert.NoError(t, err)
	checkPages(t, dbService, &userPages, &pages)
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
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC), Config: userPage}
	err = dbService.SavePage(page)
	assert.NoError(t, err)

	itemTTL = time.Minute * 1
	err = dbService.DeleteExpiredItems()
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(userPage)
	assert.NoError(t, err)
	assert.Equal(t, page, dbPage)

	itemTTL = time.Nanosecond * 0
	err = dbService.DeleteExpiredItems()
	assert.NoError(t, err)

	dbPage, err = dbService.GetPage(userPage)
	assert.NoError(t, err)
	assert.Equal(t, PagemonitorPage{}, dbPage)
}

func TestSaveReadPageTTLNotExpired(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC), Config: userPage}
	err = dbService.SavePage(page)
	assert.NoError(t, err)

	err = dbService.DeleteExpiredItems()
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(userPage)
	assert.NoError(t, err)
	assert.Equal(t, page, dbPage)
}
