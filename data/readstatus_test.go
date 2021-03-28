package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReadStatusEmpty(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user := User{username: "user01"}

	readStatus, err := dbService.GetReadStatus(&user, []byte{})
	assert.NoError(t, err)
	assert.False(t, readStatus)
}

func TestSaveGetReadStatus(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user := User{username: "user01"}

	key1 := []byte("i1")
	key2 := []byte("i2")

	err = dbService.SetReadStatus(&user, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user, key2, true)
	assert.NoError(t, err)

	readStatus, err := dbService.GetReadStatus(&user, key1)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user, key2)
	assert.NoError(t, err)
	assert.True(t, readStatus)
}

func TestRemoveGetReadStatus(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user := User{username: "user01"}

	key1 := []byte("i1")
	key2 := []byte("i2")

	err = dbService.SetReadStatus(&user, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user, key2, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user, key1, false)
	assert.NoError(t, err)

	readStatus, err := dbService.GetReadStatus(&user, key1)
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user, key2)
	assert.NoError(t, err)
	assert.True(t, readStatus)
}

func TestSetStatusDoesntExist(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user := User{username: "user01"}

	key := []byte("i1")
	err = dbService.SetReadStatus(&user, key, false)
	assert.NoError(t, err)

	readStatus, err := dbService.GetReadStatus(&user, key)
	assert.NoError(t, err)
	assert.False(t, readStatus)
}

func TestRenameUserTransferReadStatusSuccess(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

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

	readStatus, err := dbService.GetReadStatus(&user, key1)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user, key2)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	oldUser := User{username: "user01"}
	readStatus, err = dbService.GetReadStatus(&oldUser, key1)
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&oldUser, key2)
	assert.NoError(t, err)
	assert.False(t, readStatus)
}

func TestRenameUserTransferReadStatusAlreadyExists(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

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

	readStatus, err := dbService.GetReadStatus(&user1, key1)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user1, key2)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, key1)
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, key2)
	assert.NoError(t, err)
	assert.False(t, readStatus)
}

func TestCleanupStaleReadStatus(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

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

	readStatus, err := dbService.GetReadStatus(&user1, feedItem1.Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user1, feedItem2.Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, feedItem1.Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, feedItem2.Key.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)

	err = dbService.DeleteStaleReadStatuses()
	assert.NoError(t, err)

	readStatus, err = dbService.GetReadStatus(&user1, feedItem1.Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user1, feedItem2.Key.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, feedItem1.Key.CreateKey())
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, feedItem2.Key.CreateKey())
	assert.NoError(t, err)
	assert.False(t, readStatus)
}

func TestSetUnreadStatusForAll(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user1 := User{username: "user01"}
	err = dbService.SaveUser(&user1)
	assert.NoError(t, err)

	user2 := User{username: "user02"}
	err = dbService.SaveUser(&user2)
	assert.NoError(t, err)

	key1 := []byte("i1")
	key2 := []byte("i2")

	err = dbService.SetReadStatus(&user1, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user1, key2, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user2, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatusForAll(key1, false)
	assert.NoError(t, err)

	readStatus, err := dbService.GetReadStatus(&user1, key1)
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user1, key2)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, key1)
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, key2)
	assert.NoError(t, err)
	assert.False(t, readStatus)
}

func TestSetReadStatusForAll(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user1 := User{username: "user01"}
	err = dbService.SaveUser(&user1)
	assert.NoError(t, err)

	user2 := User{username: "user02"}
	err = dbService.SaveUser(&user2)
	assert.NoError(t, err)

	key1 := []byte("i1")
	key2 := []byte("i2")

	err = dbService.SetReadStatus(&user1, key1, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatus(&user2, key2, true)
	assert.NoError(t, err)

	err = dbService.SetReadStatusForAll(key1, true)
	assert.NoError(t, err)

	readStatus, err := dbService.GetReadStatus(&user1, key1)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user1, key2)
	assert.NoError(t, err)
	assert.False(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, key1)
	assert.NoError(t, err)
	assert.True(t, readStatus)

	readStatus, err = dbService.GetReadStatus(&user2, key2)
	assert.NoError(t, err)
	assert.True(t, readStatus)
}
