package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func checkPages(t *testing.T, dbService *DBService, userPages *[]UserPagemonitor, pages *[]PagemonitorPage) {
	dbUserPages := make([]UserPagemonitor, 0, 2)
	dbPages := make([]PagemonitorPage, 0, 2)
	err := dbService.ReadAllPages(func(pm *UserPagemonitor, page *PagemonitorPage) {
		dbUserPages = append(dbUserPages, *pm)
		dbPages = append(dbPages, *page)
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, *userPages, dbUserPages)
	assert.ElementsMatch(t, *pages, dbPages)
}

func TestGetPage(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
		Flags:   "f1",
	}
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Error: "e1"}
	err = dbService.SavePage(&userPage, &page)
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Equal(t, &page, dbPage)
}
func TestSavePage(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	userPage1 := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
		Flags:   "f1",
	}
	userPage2 := UserPagemonitor{
		URL: "http://site1.com",
	}
	userPages := []UserPagemonitor{userPage1, userPage2}
	page1 := PagemonitorPage{}
	page2 := PagemonitorPage{}
	pages := []PagemonitorPage{page1, page2}

	//Empty pages
	err = dbService.SavePage(&userPage1, &page1)
	assert.NoError(t, err)
	err = dbService.SavePage(&userPage2, &page2)
	assert.NoError(t, err)
	checkPages(t, dbService, &userPages, &pages)

	//Update one page
	page1.Contents = "c1"
	page1.Delta = "d1"
	page1.Error = "e1"
	pages[0] = page1
	err = dbService.SavePage(&userPage1, &page1)
	assert.NoError(t, err)
	checkPages(t, dbService, &userPages, &pages)
}

func TestSaveReadPageTTL(t *testing.T) {
	var oldTTL = itemTTL
	itemTTL = time.Nanosecond * 1
	defer func() { itemTTL = oldTTL }()
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
		Flags:   "f1",
	}
	page := PagemonitorPage{Contents: "c1", Delta: "d1", Error: "e1"}
	err = dbService.SavePage(&userPage, &page)
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Nil(t, dbPage)
}
