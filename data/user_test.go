package data

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
)

const defaultUsername = "default"

func createDb() (dbService *DBService, cleanupFunc func(), err error) {
	dir, err := ioutil.TempDir("", "nanorss")
	if err != nil {
		return nil, func() {}, err
	}

	var opts = badger.DefaultOptions
	opts.ValueLogFileSize = 1 << 20
	opts.SyncWrites = false
	opts.Dir = dir
	opts.ValueDir = dir

	dbService, err = Open(opts)
	if err != nil {
		return nil, func() {}, err
	}
	return dbService, func() {
		dbService.Close()
		os.RemoveAll(opts.Dir)
	}, nil
}

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
