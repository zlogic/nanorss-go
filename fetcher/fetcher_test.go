package fetcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/zlogic/nanorss-go/data"
)

type DBMock struct {
	mock.Mock
}

func (m *DBMock) GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error) {
	args := m.Called(pm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*data.PagemonitorPage), args.Error(1)
}

func (m *DBMock) SavePage(page *data.PagemonitorPage) error {
	args := m.Called(page)
	return args.Error(0)
}

func (m *DBMock) SetReadStatusForAll(k []byte, read bool) error {
	args := m.Called(k, read)
	return args.Error(0)
}

func (m *DBMock) GetUsers() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *DBMock) GetUser(username string) (*data.User, error) {
	args := m.Called(username)
	return args.Get(0).(*data.User), args.Error(1)
}

func (m *DBMock) SetFetchStatus(key []byte, fetchStatus *data.FetchStatus) error {
	args := m.Called(key, fetchStatus)
	return args.Error(0)
}

func (m *DBMock) SaveFeeditems(feedItems ...*data.Feeditem) error {
	args := m.Called(feedItems)
	return args.Error(0)
}

func assertTimeBetween(t *testing.T, before, after time.Time, check time.Time) {
	assert.True(t, before.Equal(check) || before.Before(check), "%v >= %v", check, before)
	assert.True(t, after.Equal(check) || after.After(check), "%v <= %v", check, after)
}
