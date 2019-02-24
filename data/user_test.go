package data

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const defaultUsername = "default"

func TestGetUserEmpty(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()
	userService := dbService.NewUserService(defaultUsername)

	user, err := userService.Get()
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestCreateGetUser(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()
	userService := dbService.NewUserService(defaultUsername)
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

func TestReadAllUsers(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user1 := User{
		Password:    "pass1",
		Opml:        "opml1",
		Pagemonitor: "pagemonitor1",
	}
	user2 := User{
		Password:    "pass2",
		Opml:        "opml2",
		Pagemonitor: "pagemonitor2",
	}
	users := []User{user1, user2}
	for i, user := range users {
		err = dbService.NewUserService("user" + strconv.Itoa(i)).Save(&user)
		assert.NoError(t, err)
	}

	dbUsers := []User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, *user)
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
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := User{
		Password:    "pass1",
		Opml:        "opml1",
		Pagemonitor: "pagemonitor1",
	}
	users := []User{user}
	userService := dbService.NewUserService("user01")
	err = userService.Save(&user)
	assert.NoError(t, err)

	err = userService.SetUsername("user02")
	assert.NoError(t, err)
	assert.Equal(t, "user02", userService.Username)

	dbUser, err := userService.Get()
	assert.Equal(t, user, *dbUser)

	dbUsers := []User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, *user)
		}
		close(done)
	}()
	err = dbService.ReadAllUsers(ch)
	<-done
	assert.NoError(t, err)
	assert.EqualValues(t, users, dbUsers)
}

func TestSetUsernameAlreadyExists(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user1 := User{
		Password:    "pass1",
		Opml:        "opml1",
		Pagemonitor: "pagemonitor1",
	}
	user2 := User{
		Password:    "pass2",
		Opml:        "opml2",
		Pagemonitor: "pagemonitor2",
	}
	users := []User{user1, user2}
	userService1 := dbService.NewUserService("user01")
	userService2 := dbService.NewUserService("user02")
	err = userService1.Save(&user1)
	assert.NoError(t, err)
	err = userService2.Save(&user2)
	assert.NoError(t, err)

	err = userService1.SetUsername("user02")
	assert.Error(t, err)
	assert.Equal(t, "user01", userService1.Username)

	dbUser1, err := userService1.Get()
	assert.Equal(t, user1, *dbUser1)

	dbUser2, err := userService2.Get()
	assert.Equal(t, user2, *dbUser2)

	dbUsers := []User{}
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		for user := range ch {
			dbUsers = append(dbUsers, *user)
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
		UserPagemonitor{URL: "https://site1.com", Title: "Page 1", Match: "m1", Replace: "r1"},
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
