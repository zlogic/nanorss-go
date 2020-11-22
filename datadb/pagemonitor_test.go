package datadb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func checkPages(t *testing.T, dbService *DBService, pages *[]PagemonitorPage) {
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
	page1 := PagemonitorPage{Config: &userPage1}
	page2 := PagemonitorPage{Config: &userPage2}
	pages := []PagemonitorPage{page1, page2}

	//Empty pages
	err = dbService.SavePage(&page1)
	assert.NoError(t, err)
	err = dbService.SavePage(&page2)
	assert.NoError(t, err)
	checkPages(t, dbService, &pages)

	//Update one page
	page1.Contents = "c1"
	page1.Delta = "d1"
	page1.Updated = time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	pages[0] = page1
	err = dbService.SavePage(&page1)
	assert.NoError(t, err)
	checkPages(t, dbService, &pages)
}

func TestGetPages(t *testing.T) {
	dbService, err := preparePagemonitorTests()
	assert.NoError(t, err)
	defer dbService.Close()

	user1 := &User{
		Pagemonitor: `<pages>` +
			`<page url="https://site1.com" match="m1" replace="r1">Page 1</page>` +
			`<page url="http://site2.com">Page 2</page>` +
			`</pages>`,
		username: "user01",
	}
	err = dbService.SaveUser(user1)
	assert.NoError(t, err)
	user2 := &User{
		Pagemonitor: `<pages>` +
			`<page url="https://site1.com">Page 1</page>` +
			`<page url="http://site2.com">Page 2</page>` +
			`</pages>`,
		username: "user02",
	}
	err = dbService.SaveUser(user2)
	assert.NoError(t, err)

	page1 := PagemonitorPage{
		Config: &UserPagemonitor{URL: "https://site1.com", Match: "m1", Replace: "r1"},
	}
	page2 := PagemonitorPage{
		Config: &UserPagemonitor{URL: "http://site2.com"},
	}
	page3 := PagemonitorPage{
		Config: &UserPagemonitor{URL: "https://site1.com"},
	}
	extraPage := PagemonitorPage{
		Contents: "c1", Delta: "d1",
		Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Config:  &UserPagemonitor{URL: "https://site3.com", Match: "m1", Replace: "r1"},
	}
	err = dbService.SavePage(&extraPage)
	assert.NoError(t, err)

	pages1 := []*PagemonitorPage{&page1, &page2}
	dbPages1, err := dbService.GetPages(user1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, pages1, dbPages1)

	pages2 := []*PagemonitorPage{&page2, &page3}
	dbPages2, err := dbService.GetPages(user2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, pages2, dbPages2)

	pages := []PagemonitorPage{page1, page2, page3, extraPage}
	checkPages(t, dbService, &pages)
}

func TestGetPagesEmptyList(t *testing.T) {
	dbService, err := preparePagemonitorTests()
	assert.NoError(t, err)
	defer dbService.Close()

	user := &User{
		Pagemonitor: "<pages></pages>",
		username:    "user01",
	}
	err = dbService.SaveUser(user)
	assert.NoError(t, err)

	extraPage := PagemonitorPage{
		Contents: "c1", Delta: "d1",
		Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Config:  &UserPagemonitor{URL: "https://site3.com", Match: "m1", Replace: "r1"},
	}
	err = dbService.SavePage(&extraPage)
	assert.NoError(t, err)

	dbPages, err := dbService.GetPages(user)
	assert.NoError(t, err)
	assert.Empty(t, dbPages)
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
	cleanDatabases := []string{"users", "pagemonitors"}
	for _, table := range cleanDatabases {
		_, err = dbService.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			dbService.Close()
			return nil, err
		}
	}
	return dbService, nil
}
