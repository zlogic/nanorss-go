package datadb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const defaultUsername = "default"

func TestNewUser(t *testing.T) {
	user := NewUser("user01")
	assert.Equal(t, &User{username: "user01"}, user)
}

func TestGetUserEmpty(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user, err := dbService.GetUser(defaultUsername)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestCreateGetUser(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user := &User{
		Password:    "password",
		Opml:        "<opml></opml>",
		Pagemonitor: "<pages></pages>",
		username:    "user01",
	}
	err = dbService.SaveUser(user)
	assert.NoError(t, err)

	user, err = dbService.GetUser("user01")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "password", user.Password)
	assert.Equal(t, "<opml></opml>", user.Opml)
	assert.Equal(t, "<pages></pages>", user.Pagemonitor)
}

func TestReadAllUsers(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user1 := User{
		Password:    "pass1",
		Opml:        "<opml><!-- 1 --></opml>",
		Pagemonitor: "<pages><!-- 1 --></pages>",
		username:    "user01",
	}
	user2 := User{
		Password:    "pass2",
		Opml:        "<opml><!-- 2 --></opml>",
		Pagemonitor: "<pages><!-- 2 --></pages>",
		username:    "user02",
	}
	users := []*User{&user1, &user2}
	for _, user := range users {
		err = dbService.SaveUser(user)
		assert.NoError(t, err)
	}

	dbUsers := []*User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, user)
		}
		close(done)
	}()
	err = dbService.ReadAllUsers(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, users, dbUsers)
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

func TestSetUsername(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user := User{
		Password:    "pass1",
		Opml:        "<opml><!-- 1 --></opml>",
		Pagemonitor: "<pages><!-- 1 --></pages>",
		username:    "user01",
	}
	users := []*User{&user}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	err = user.SetUsername("user02")
	assert.NoError(t, err)
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)
	assert.Equal(t, "user02", user.username)

	dbUser, err := dbService.GetUser(user.username)
	assert.Equal(t, user, *dbUser)

	dbUsers := []*User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, user)
		}
		close(done)
	}()
	err = dbService.ReadAllUsers(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, users, dbUsers)
}

func TestSetUsernameAndOtherFields(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user := User{
		Password:    "pass1",
		Opml:        "<opml><!-- 1 --></opml>",
		Pagemonitor: "<pages><!-- 1 --></pages>",
		username:    "user01",
	}
	users := []*User{&user}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	user.Password = "pass1new"
	user.Opml = "<opml><!-- 1 new --></opml>"
	user.Pagemonitor = "<pages><!-- 1 new --></pages>"
	err = user.SetUsername("user02")
	assert.NoError(t, err)

	assert.NoError(t, err)
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)
	assert.Equal(t, "user02", user.username)

	dbUser, err := dbService.GetUser(user.username)
	assert.Equal(t, user, *dbUser)

	dbUsers := []*User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, user)
		}
		close(done)
	}()
	err = dbService.ReadAllUsers(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, users, dbUsers)
}

func TestSetUsernameAlreadyExists(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user1 := User{
		Password:    "pass1",
		Opml:        "<opml><!-- 1 --></opml>",
		Pagemonitor: "<pages><!-- 1 --></pages>",
		username:    "user01",
	}
	user2 := User{
		Password:    "pass2",
		Opml:        "<opml><!-- 2 --></opml>",
		Pagemonitor: "<pages><!-- 2 --></pages>",
		username:    "user02",
	}
	users := []*User{&user1, &user2}
	for _, user := range users {
		err = dbService.SaveUser(user)
		assert.NoError(t, err)
	}

	err = user1.SetUsername("user02")
	assert.NoError(t, err)
	err = dbService.SaveUser(&user1)
	assert.Error(t, err)
	assert.Equal(t, "user01", user1.username)

	dbUser1, err := dbService.GetUser("user01")
	assert.Equal(t, user1, *dbUser1)

	dbUser2, err := dbService.GetUser("user02")
	assert.Equal(t, user2, *dbUser2)

	dbUsers := []*User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, user)
		}
		close(done)
	}()
	err = dbService.ReadAllUsers(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, users, dbUsers)
}

func TestSetUsernameEmptyString(t *testing.T) {
	err := prepareUserTests()
	assert.NoError(t, err)

	user := User{
		Password:    "pass1",
		Opml:        "<opml><!-- 1 --></opml>",
		Pagemonitor: "<pages><!-- 1 --></pages>",
		username:    "user01",
	}
	users := []*User{&user}
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	err = user.SetUsername("  ")
	assert.Error(t, err)
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	dbUser, err := dbService.GetUser(user.username)
	assert.Equal(t, user, *dbUser)

	dbUsers := []*User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, user)
		}
		close(done)
	}()
	err = dbService.ReadAllUsers(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, users, dbUsers)
}

func TestParsePagemonitor(t *testing.T) {
	user := &User{Pagemonitor: `<pages>` +
		`<page url="https://site1.com" match="m1" replace="r1">Page 1</page>` +
		`<page url="http://site2.com">Page 2</page>` +
		`</pages>`}
	items, err := user.GetPages()
	assert.NoError(t, err)
	assert.NotNil(t, items)
	assert.Equal(t, []UserPagemonitor{
		{URL: "https://site1.com", Title: "Page 1", Match: "m1", Replace: "r1"},
		{URL: "http://site2.com", Title: "Page 2"},
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
		{URL: "http://sites-site1.com", Title: "Site 1"},
		{URL: "http://updates-site2.com", Title: "Site 2"},
		{URL: "http://updates-site3.com", Title: "Site 3"},
	}, items)
}

func prepareUserTests() error {
	cleanDatabases := []string{"users", "pagemonitors"}
	for _, table := range cleanDatabases {
		_, err := dbService.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return err
		}
	}
	return nil
}
