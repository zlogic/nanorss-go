package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReadStatusEmpty(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user := User{username: "user01"}

	dbReadItems, err := dbService.GetReadItems(&user)
	assert.NoError(t, err)
	assert.Empty(t, dbReadItems)
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

	readItems := [][]byte{key1, key2}
	dbReadItems, err := dbService.GetReadItems(&user)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)
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

	readItems := [][]byte{key2}
	dbReadItems, err := dbService.GetReadItems(&user)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)
}

func TestSetStatusDoesntExist(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	user := User{username: "user01"}

	key := []byte("i1")
	err = dbService.SetReadStatus(&user, key, false)
	assert.NoError(t, err)

	dbReadItems, err := dbService.GetReadItems(&user)
	assert.NoError(t, err)
	assert.Empty(t, dbReadItems)
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

	readItems := [][]byte{key1, key2}
	dbReadItems, err := dbService.GetReadItems(&user)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	oldUser := User{username: "user01"}

	dbReadItems, err = dbService.GetReadItems(&oldUser)
	assert.NoError(t, err)
	assert.Empty(t, dbReadItems)
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

	readItems := [][]byte{key1, key2}
	dbReadItems, err := dbService.GetReadItems(&user1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	dbReadItems, err = dbService.GetReadItems(&user2)
	assert.NoError(t, err)
	assert.Empty(t, dbReadItems)
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

	readItems := [][]byte{feedItem1.Key.CreateKey(), feedItem2.Key.CreateKey()}
	dbReadItems, err := dbService.GetReadItems(&user1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	readItems = [][]byte{feedItem1.Key.CreateKey()}
	dbReadItems, err = dbService.GetReadItems(&user2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	err = dbService.deleteStaleReadStatuses()
	assert.NoError(t, err)

	readItems = [][]byte{feedItem1.Key.CreateKey()}
	dbReadItems, err = dbService.GetReadItems(&user1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	readItems = [][]byte{feedItem1.Key.CreateKey()}
	dbReadItems, err = dbService.GetReadItems(&user2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)
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

	readItems := [][]byte{key2}
	dbReadItems, err := dbService.GetReadItems(&user1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	dbReadItems, err = dbService.GetReadItems(&user2)
	assert.NoError(t, err)
	assert.Empty(t, dbReadItems)
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

	readItems := [][]byte{key1}
	dbReadItems, err := dbService.GetReadItems(&user1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)

	readItems = [][]byte{key1, key2}
	dbReadItems, err = dbService.GetReadItems(&user2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, readItems, dbReadItems)
}
