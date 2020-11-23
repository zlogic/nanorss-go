package server

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"

	"github.com/stretchr/testify/mock"
	"github.com/zlogic/nanorss-go/data"
)

type DBMock struct {
	mock.Mock
}

func (m *DBMock) GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error) {
	args := m.Called(varName, generator)
	return args.Get(0).(string), args.Error(1)
}

func (m *DBMock) GetUser(username string) (*data.User, error) {
	args := m.Called(username)
	user := args.Get(0)
	var returnUser *data.User
	if user != nil {
		returnUser = user.(*data.User)
	}
	return returnUser, args.Error(1)
}

func (m *DBMock) SaveUser(user *data.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *DBMock) SetUsername(user *data.User, newUsername string) error {
	args := m.Called(user, newUsername)
	return args.Error(0)
}

func (m *DBMock) GetFeeditem(key *data.FeeditemKey) (*data.Feeditem, error) {
	args := m.Called(key)
	feedItem := args.Get(0)
	var returnFeeditem *data.Feeditem
	if feedItem != nil {
		returnFeeditem = feedItem.(*data.Feeditem)
	}
	return returnFeeditem, args.Error(1)
}

func (m *DBMock) GetFeeditems(user *data.User) ([]*data.Feeditem, error) {
	args := m.Called(user)
	feedItems := args.Get(0)
	var returnFeeditems []*data.Feeditem
	if feedItems != nil {
		returnFeeditems = feedItems.([]*data.Feeditem)
	}
	return returnFeeditems, args.Error(1)
}

func (m *DBMock) GetPage(pm *data.UserPagemonitor) (*data.PagemonitorPage, error) {
	args := m.Called(pm)
	page := args.Get(0)
	var returnPage *data.PagemonitorPage
	if page != nil {
		returnPage = page.(*data.PagemonitorPage)
	}
	return returnPage, args.Error(1)
}

func (m *DBMock) GetPages(user *data.User) ([]*data.PagemonitorPage, error) {
	args := m.Called(user)
	pages := args.Get(0)
	var returnPages []*data.PagemonitorPage
	if pages != nil {
		returnPages = pages.([]*data.PagemonitorPage)
	}
	return returnPages, args.Error(1)
}

func (m *DBMock) ReadAllPages(ch chan *data.PagemonitorPage) error {
	args := m.Called(ch)
	return args.Error(0)
}

func (m *DBMock) GetFeeditemsReadStatus(user *data.User) ([]*data.FeeditemKey, error) {
	args := m.Called(user)
	readItems := args.Get(0)
	var returnReadItems []*data.FeeditemKey
	if readItems != nil {
		returnReadItems = readItems.([]*data.FeeditemKey)
	}
	return returnReadItems, args.Error(1)
}

func (m *DBMock) GetPagesReadStatus(user *data.User) ([]*data.UserPagemonitor, error) {
	args := m.Called(user)
	readItems := args.Get(0)
	var returnReadItems []*data.UserPagemonitor
	if readItems != nil {
		returnReadItems = readItems.([]*data.UserPagemonitor)
	}
	return returnReadItems, args.Error(1)
}

func (m *DBMock) SetFeeditemReadStatus(user *data.User, k *data.FeeditemKey, read bool) error {
	args := m.Called(user, k, read)
	return args.Error(0)
}

func (m *DBMock) SetPageReadStatus(user *data.User, k *data.UserPagemonitor, read bool) error {
	args := m.Called(user, k, read)
	return args.Error(0)
}

func (m *DBMock) GetFeedFetchStatus(feedURL string) (*data.FetchStatus, error) {
	args := m.Called(feedURL)
	fetchStatus := args.Get(0)
	var returnFetchStatus *data.FetchStatus
	if fetchStatus != nil {
		returnFetchStatus = fetchStatus.(*data.FetchStatus)
	}
	return returnFetchStatus, args.Error(1)
}

func (m *DBMock) GetPageFetchStatus(key *data.UserPagemonitor) (*data.FetchStatus, error) {
	args := m.Called(key)
	fetchStatus := args.Get(0)
	var returnFetchStatus *data.FetchStatus
	if fetchStatus != nil {
		returnFetchStatus = fetchStatus.(*data.FetchStatus)
	}
	return returnFetchStatus, args.Error(1)
}

func createTestCookieHandler() (*CookieHandler, error) {
	signKey := base64.StdEncoding.EncodeToString(generateRandomKey(64))
	dbMock := new(DBMock)

	dbMock.On("GetOrCreateConfigVariable", "cookie-sign-key", mock.AnythingOfType("func() (string, error)")).Return(signKey, nil).Once()
	return NewCookieHandler(dbMock)
}

func prepareTempDir() (string, func(), error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	recover := func() {
		os.Chdir(currentDir)
	}
	tempDir, err := ioutil.TempDir("", "nanorss")
	if err != nil {
		return currentDir, recover, err
	}
	recover = func() {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
	}
	err = os.Chdir(tempDir)
	return tempDir, recover, err
}

func prepareTestFile(dir, fileName string, data []byte) error {
	err := os.Mkdir(dir, 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(dir, fileName), data, 0644)
}
