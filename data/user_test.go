package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const defaultUsername = "default"

func TestGetUserEmpty(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()
	userService := dbService.newUserService(defaultUsername)

	user, err := userService.Get()
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestCreateGetUser(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()
	userService := dbService.newUserService(defaultUsername)
	assert.NoError(t, err)

	user := &User{
		Password:    "password",
		Opml:        "opml",
		Pagemonitor: "pagemonitor",
	}
	err = userService.Save(user)
	assert.NoError(t, err)

	user, err = userService.Get()
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "password", user.Password)
	assert.Equal(t, "opml", user.Opml)
	assert.Equal(t, "pagemonitor", user.Pagemonitor)
}

func TestSetUserPassword(t *testing.T) {
	user := &User{}
	err := user.SetPassword("hello")
	assert.NoError(t, err)
	assert.NotNil(t, user.Password)
	assert.NotEqual(t, "password", user.Password)

	err = user.ValidatePassword("hello")
	assert.NoError(t, err)

	err = user.ValidatePassword("hellow")
	assert.Error(t, err)
}

func TestParsePagemonitor(t *testing.T) {
	user := &User{Pagemonitor: `<pages>` +
		`<page url="https://site1.com" match="m1" replace="r1" flags="f1">Page 1</page>` +
		`<page url="http://site2.com">Page 2</page>` +
		`</pages>`}
	items, err := user.GetPages()
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Equal(t, []UserPagemonitor{
		UserPagemonitor{URL: "https://site1.com", Title: "Page 1", Match: "m1", Replace: "r1", Flags: "f1"},
		UserPagemonitor{URL: "http://site2.com", Title: "Page 2"},
	}, items)
}

func TestParseOPML(t *testing.T) {
	user := &User{Opml: `<opml version="1.0">` +
		`<head><title>My OPML list</title></head>` +
		`<body>` +
		`<outline text="Sites" title="Sites"><outline text="Site 1" title="Site 1" type="rss" xmlUrl="http://sites-site1.com" htmlUrl="http://sites-site1.com"/></outline>` +
		`<outline text="Updates" title="Updates">` +
		`<outline text="Site 2" title="Site 2" type="rss" xmlUrl="http://updates-site2.com" htmlUrl="http://updates-site2.com"/>` +
		`<outline text="Site 3" title="Site 3" type="rss" xmlUrl="http://updates-site3.com" htmlUrl="http://updates-site3.com"/>` +
		`</outline>` +
		`</body>` +
		`</opml>`}
	items, err := user.GetFeeds()
	assert.NoError(t, err)
	assert.NotNil(t, items)

	assert.Equal(t, []UserFeed{
		UserFeed{URL: "http://sites-site1.com", Title: "Site 1"},
		UserFeed{URL: "http://updates-site2.com", Title: "Site 2"},
		UserFeed{URL: "http://updates-site3.com", Title: "Site 3"},
	}, items)
}
