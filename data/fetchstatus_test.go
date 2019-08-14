package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetFetchStatusEmpty(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	key := "i1"
	fetchStatus, err := dbService.GetFetchStatus(key)
	assert.NoError(t, err)
	assert.Nil(t, fetchStatus)
}

func TestSaveGetFetchStatus(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	key := "i1"
	fetchStatus := &FetchStatus{LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)}
	err = dbService.SetFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err := dbService.GetFetchStatus(key)
	assert.NoError(t, err)
	assert.Equal(t, fetchStatus, dbFetchStatus)
}

func TestUpdateFetchStatus(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	key := "i1"
	fetchStatus := &FetchStatus{LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)}
	err = dbService.SetFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	fetchStatus = &FetchStatus{LastFailure: time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)}
	err = dbService.SetFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err := dbService.GetFetchStatus(key)
	assert.NoError(t, err)
	assert.Equal(t, &FetchStatus{
		LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastFailure: time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC),
	}, dbFetchStatus)

	fetchStatus = &FetchStatus{LastFailure: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC)}
	err = dbService.SetFetchStatus(key, fetchStatus)
	assert.NoError(t, err)

	dbFetchStatus, err = dbService.GetFetchStatus(key)
	assert.NoError(t, err)
	assert.Equal(t, &FetchStatus{
		LastSuccess: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		LastFailure: time.Date(2019, time.February, 16, 23, 2, 0, 0, time.UTC),
	}, dbFetchStatus)
}

func TestCleanupStaleFetchStatus(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	key1 := "i1"
	fetchStatus1 := &FetchStatus{LastSuccess: time.Now().In(time.UTC).Add(-itemTTL - time.Minute)}
	err = dbService.SetFetchStatus(key1, fetchStatus1)
	assert.NoError(t, err)

	key2 := "i2"
	fetchStatus2 := &FetchStatus{LastFailure: time.Now().In(time.UTC).Truncate(time.Millisecond)}
	err = dbService.SetFetchStatus(key2, fetchStatus2)
	assert.NoError(t, err)

	dbService.DeleteStaleFetchStatuses()

	dbFetchStatus1, err := dbService.GetFetchStatus(key1)
	assert.NoError(t, err)
	assert.Nil(t, dbFetchStatus1)

	dbFetchStatus2, err := dbService.GetFetchStatus(key2)
	assert.NoError(t, err)
	assert.Equal(t, fetchStatus2, dbFetchStatus2)
}
