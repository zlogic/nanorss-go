package server

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"

	"github.com/stretchr/testify/mock"

	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/server/auth"
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

func (m *DBMock) ReadAllFeedItems(ch chan *data.Feeditem) error {
	args := m.Called(ch)
	return args.Error(0)
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

func (m *DBMock) ReadAllPages(ch chan *data.PagemonitorPage) error {
	args := m.Called(ch)
	return args.Error(0)
}

func (m *DBMock) GetReadStatus(user *data.User) ([][]byte, error) {
	args := m.Called(user)
	readItems := args.Get(0)
	var returnReadItems [][]byte
	if readItems != nil {
		returnReadItems = readItems.([][]byte)
	}
	return returnReadItems, args.Error(1)
}

func (m *DBMock) SetReadStatus(user *data.User, itemKey []byte, read bool) error {
	args := m.Called(user, itemKey, read)
	return args.Error(0)
}

func (m *DBMock) GetFetchStatus(key []byte) (*data.FetchStatus, error) {
	args := m.Called(key)
	fetchStatus := args.Get(0)
	var returnFetchStatus *data.FetchStatus
	if fetchStatus != nil {
		returnFetchStatus = fetchStatus.(*data.FetchStatus)
	}
	return returnFetchStatus, args.Error(1)
}

var testAuthCookie = "testusername"

type AuthHandlerMock struct {
	mock.Mock
	authUser *data.User
}

func (m *AuthHandlerMock) SetCookieUsername(w http.ResponseWriter, username string, rememberMe bool) error {
	args := m.Called(w, username, rememberMe)
	return args.Error(0)
}

func (m *AuthHandlerMock) AuthHandlerFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if m.authUser != nil {
			ctx = context.WithValue(ctx, auth.UserContextKey, m.authUser)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthHandlerMock) HasAuthenticationCookie(r *http.Request) bool {
	args := m.Called(r)
	return args.Get(0).(bool)
}

func (m *AuthHandlerMock) AllowUser(user *data.User) *http.Cookie {
	m.authUser = user
	return nil
}

// testRecorder fixes go-chi support in httptest.ResponseRecorder.
type testRecorder struct {
	*httptest.ResponseRecorder
}

func (rec *testRecorder) ReadFrom(r io.Reader) (n int64, err error) {
	return io.Copy(rec.ResponseRecorder, r)
}

func newRecorder() *testRecorder {
	return &testRecorder{ResponseRecorder: httptest.NewRecorder()}
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
