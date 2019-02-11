package data

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
)

func createOpts() (badger.Options, error) {
	dir, err := ioutil.TempDir("", "nanorss")
	if err != nil {
		return badger.Options{}, err
	}

	var opts = badger.DefaultOptions
	opts.ValueLogFileSize = 1 << 20
	opts.SyncWrites = false
	opts.Dir = dir
	opts.ValueDir = dir
	return opts, nil
}

func cleanupTestDb(s *Service, opts badger.Options) {
	s.Close()
	os.RemoveAll(opts.Dir)
}

func (s *Service) clearDb() error {
	return s.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().Key()

			if err := txn.Delete(k); err != nil {
				return err
			}
		}

		return nil
	})
}

func TestGetUserEmpty(t *testing.T) {
	opts, err := createOpts()
	assert.NoError(t, err)
	dbService, err := Open(opts)
	assert.NoError(t, err)
	defer cleanupTestDb(dbService, opts)
	err = dbService.clearDb()
	assert.NoError(t, err)

	user, err := dbService.userService.Get("Hello")
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestCreateGetUser(t *testing.T) {
	opts, err := createOpts()
	assert.NoError(t, err)
	dbService, err := Open(opts)
	defer cleanupTestDb(dbService, opts)
	assert.NoError(t, err)
	err = dbService.clearDb()
	assert.NoError(t, err)

	user := &User{
		Password:    "password",
		Opml:        "opml",
		Pagemonitor: "pagemonitor",
	}
	err = dbService.userService.Save(user)
	assert.NoError(t, err)

	user, err = dbService.userService.Get("default")
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
