package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReadStatusEmpty(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := User{username: "user01"}

	readStatuses, err := dbService.GetReadStatus(&user)
	assert.NoError(t, err)
	assert.Empty(t, readStatuses)
}

func TestSaveGetReadStatus(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := User{username: "user01"}

	key1 := []byte("i1")
	key2 := []byte("i2")

	err = dbService.SetReadStatus(&user, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user, key2, true)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{key1, key2}, readStatuses)
}

func TestRemoveGetReadStatus(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := User{username: "user01"}

	key1 := []byte("i1")
	key2 := []byte("i2")

	err = dbService.SetReadStatus(&user, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user, key2, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user, key1, false)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{key2}, readStatuses)
}

func TestSetStatusDoesntExist(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := User{username: "user01"}

	key := []byte("i1")
	err = dbService.SetReadStatus(&user, key, false)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{}, readStatuses)
}

func TestRenameUserTransferReadStatusSuccess(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user := User{username: "user01"}

	key1 := []byte("i1")
	key2 := []byte("i2")
	err = dbService.SetReadStatus(&user, key1, true)
	assert.NoError(t, err)
	err = dbService.SetReadStatus(&user, key2, true)
	assert.NoError(t, err)

	user.SetUsername("user02")
	err = dbService.SaveUser(&user)
	assert.NoError(t, err)

	readStatuses, err := dbService.GetReadStatus(&user)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{key1, key2}, readStatuses)

	oldUser := User{username: "user01"}
	readStatuses, err = dbService.GetReadStatus(&oldUser)
	assert.NoError(t, err)
	assert.Empty(t, readStatuses)
}

func TestRenameUserTransferReadStatusAlreadyExists(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user1 := User{username: "user01"}

	key1 := []byte("i1")
	key2 := []byte("i2")
	err = dbService.SetReadStatus(&user1, key1, true)
	assert.NoError(t, err)
	err = dbService.SetReadStatus(&user1, key2, true)
	assert.NoError(t, err)

	user2 := User{username: "user02"}
	err = dbService.SaveUser(&user2)
	assert.NoError(t, err)

	user1.SetUsername("user02")
	err = dbService.SaveUser(&user1)
	assert.Error(t, err)

	readStatuses, err := dbService.GetReadStatus(&user1)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{key1, key2}, readStatuses)

	readStatuses, err = dbService.GetReadStatus(&user2)
	assert.NoError(t, err)
	assert.Empty(t, readStatuses)
}

func TestCleanupStaleReadStatus(t *testing.T) {
	dbService, cleanup, err := createDb()
	assert.NoError(t, err)
	defer cleanup()

	user1 := User{username: "user01"}
	err = dbService.SaveUser(&user1)
	assert.NoError(t, err)

	user2 := User{username: "user02"}
	err = dbService.SaveUser(&user2)
	assert.NoError(t, err)

	feedItem1 := &Feeditem{Key: &FeeditemKey{FeedURL: "http://site1", GUID: "g1"}}
	feedItem2 := &Feeditem{Key: &FeeditemKey{FeedURL: "http://site2", GUID: "g2"}}
	err = dbService.SaveFeeditems(feedItem1)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user1, feedItem1.Key.CreateKey(), true)
	assert.NoError(t, err)
	err = dbService.SetReadStatus(&user1, feedItem2.Key.CreateKey(), true)
	assert.NoError(t, err)
	err = dbService.SetReadStatus(&user2, feedItem1.Key.CreateKey(), true)
	assert.NoError(t, err)

	dbReadStatus1, err := dbService.GetReadStatus(&user1)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{feedItem1.Key.CreateKey(), feedItem2.Key.CreateKey()}, dbReadStatus1)

	dbReadStatus2, err := dbService.GetReadStatus(&user2)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{feedItem1.Key.CreateKey()}, dbReadStatus2)

	err = dbService.DeleteStaleReadStatuses()
	assert.NoError(t, err)

	dbReadStatus1, err = dbService.GetReadStatus(&user1)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{feedItem1.Key.CreateKey()}, dbReadStatus1)

	dbReadStatus2, err = dbService.GetReadStatus(&user2)
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{feedItem1.Key.CreateKey()}, dbReadStatus2)
}
