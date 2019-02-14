package data

import (
	"testing"

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

func TestCreatePages(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := &User{Pagemonitor: `<pages>` +
		`<page url="https://site1.com" match="m1" replace="r1" flags="f1">Page 1</page>` +
		`<page url="http://site2.com">Page 2</page>` +
		`</pages>`}
	err = dbService.SavePages(user)
	assert.NoError(t, err)

	// Ignore title in comparison
	pages, err := user.GetPages()
	assert.NoError(t, err)
	for i := 0; i < len(pages); i++ {
		pages[i].Title = ""
	}

	dbPages := make([]UserPagemonitor, 0, 2)
	err = dbService.ReadAllPages(func(pm *UserPagemonitor, page *PagemonitorPage) {
		assert.Equal(t, &PagemonitorPage{}, page)
		dbPages = append(dbPages, *pm)
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, pages, dbPages)
}

func TestReadAllPages(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := &User{Pagemonitor: `<pages>` +
		`<page url="https://site1.com" match="m1" replace="r1" flags="f1">Page 1</page>` +
		`<page url="http://site2.com">Page 2</page>` +
		`</pages>`}
	err = dbService.SavePages(user)
	assert.NoError(t, err)

	// Ignore title in comparison
	pages, err := user.GetPages()
	assert.NoError(t, err)
	for i := 0; i < len(pages); i++ {
		pages[i].Title = ""
	}

	dbPages := make([]UserPagemonitor, 0, 2)
	err = dbService.ReadAllPages(func(pm *UserPagemonitor, page *PagemonitorPage) {
		assert.Equal(t, &PagemonitorPage{}, page)
		dbPages = append(dbPages, *pm)
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, pages, dbPages)
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

func TestDeleteNonExistingPage(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	userPage := UserPagemonitor{
		URL: "http://site1.com",
	}
	page := PagemonitorPage{}

	err = dbService.SavePage(&userPage, &page)
	assert.NoError(t, err)

	userPageNonExisting := UserPagemonitor{
		URL: "http://site2.com",
	}
	err = dbService.DeletePage(&userPageNonExisting)
	assert.NoError(t, err)
	userPages := []UserPagemonitor{userPage}
	pages := []PagemonitorPage{page}
	checkPages(t, dbService, &userPages, &pages)
}

func TestDeleteExistingPage(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	userPage := UserPagemonitor{
		URL: "http://site1.com",
	}
	page := PagemonitorPage{}

	err = dbService.SavePage(&userPage, &page)
	assert.NoError(t, err)

	err = dbService.DeletePage(&userPage)
	assert.NoError(t, err)
	userPages := []UserPagemonitor{}
	pages := []PagemonitorPage{}
	checkPages(t, dbService, &userPages, &pages)
}
