package datadb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func checkPages(t *testing.T, dbService *DBService, userPages *[]UserPagemonitor, pages *[]PagemonitorPage) {
	dbPages := make([]PagemonitorPage, 0, 2)

	rows, err := dbService.db.Query("SELECT url, match, replace, contents, delta, updated FROM pagemonitors")
	assert.NoError(t, err)
	for rows.Next() {
		page := PagemonitorPage{Config: &UserPagemonitor{}}
		err = rows.Scan(&page.Config.URL, &page.Config.Match, &page.Config.Replace, &page.Contents, &page.Delta, &page.Updated)
		assert.NoError(t, err)
		dbPages = append(dbPages, page)
	}

	assert.ElementsMatch(t, *pages, dbPages)
}

func TestGetPage(t *testing.T) {
	dbService, err := preparePagemonitorTests()
	assert.NoError(t, err)
	defer dbService.Close()

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
	dbService, err := preparePagemonitorTests()
	assert.NoError(t, err)
	defer dbService.Close()

	userPage1 := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	userPage2 := UserPagemonitor{
		URL: "http://site1.com",
	}
	userPages := []UserPagemonitor{userPage1, userPage2}
	page1 := PagemonitorPage{Config: &userPage1}
	page2 := PagemonitorPage{Config: &userPage2}
	pages := []PagemonitorPage{page1, page2}

	//Empty pages
	err = dbService.SavePage(&page1)
	assert.NoError(t, err)
	err = dbService.SavePage(&page2)
	assert.NoError(t, err)
	checkPages(t, dbService, &userPages, &pages)

	//Update one page
	page1.Contents = "c1"
	page1.Delta = "d1"
	page1.Updated = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	pages[0] = page1
	err = dbService.SavePage(&page1)
	assert.NoError(t, err)
	checkPages(t, dbService, &userPages, &pages)
}

func TestSaveReadPageTTLExpired(t *testing.T) {
	dbService, err := preparePagemonitorTests()
	assert.NoError(t, err)
	defer dbService.Close()
	var oldTTL = itemTTL
	defer func() { itemTTL = oldTTL }()

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	expiredTime := time.Now().Add(-time.Minute * 15).UTC().Truncate(time.Millisecond)
	page := PagemonitorPage{
		Contents: "c1", Delta: "d1",
		Updated:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastSuccess: &expiredTime,
		Config:      &userPage,
	}
	err = dbService.SavePage(&page)
	assert.NoError(t, err)

	itemTTL = time.Hour * 1
	err = dbService.deleteExpiredItems()
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Equal(t, &page, dbPage)

	itemTTL = time.Minute * 1
	err = dbService.deleteExpiredItems()
	assert.NoError(t, err)

	dbPage, err = dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Nil(t, dbPage)
}

func TestSaveReadPageTTLNotExpired(t *testing.T) {
	dbService, err := preparePagemonitorTests()
	assert.NoError(t, err)
	defer dbService.Close()

	userPage := UserPagemonitor{
		URL:     "http://site1.com",
		Match:   "m1",
		Replace: "r1",
	}
	lastSuccess := time.Now().UTC().Truncate(time.Millisecond)
	page := PagemonitorPage{
		Contents: "c1", Delta: "d1",
		Updated:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastSuccess: &lastSuccess,
		Config:      &userPage,
	}
	err = dbService.SavePage(&page)
	assert.NoError(t, err)

	err = dbService.deleteExpiredItems()
	assert.NoError(t, err)

	dbPage, err := dbService.GetPage(&userPage)
	assert.NoError(t, err)
	assert.Equal(t, &page, dbPage)
}

func preparePagemonitorTests() (*DBService, error) {
	useRealDatabase()
	dbService, err := Open()
	if err != nil {
		return nil, err
	}
	_, err = dbService.db.Exec("DELETE FROM pagemonitors")
	if err != nil {
		dbService.Close()
		return nil, err
	}
	return dbService, nil
}
