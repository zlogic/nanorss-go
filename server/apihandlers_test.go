package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zlogic/nanorss-go/data"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type FeedListHelperMock struct {
	mock.Mock
}

func (m *FeedListHelperMock) GetAllItems(user *data.User) ([]*Item, error) {
	args := m.Called(user)
	return args.Get(0).([]*Item), args.Error(1)
}

type FetcherMock struct {
	mock.Mock
}

func (m *FetcherMock) Refresh() {
	m.Called()
}

func TestLoginHandlerSuccessful(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader("username=user01&password=pass"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "OK", string(res.Body.Bytes()))
	cookies := res.Result().Cookies()
	assert.Equal(t, 1, len(cookies))
	if len(cookies) > 0 {
		decodedCookie := UserCookie{}
		err := cookieHandler.secureCookie.Decode(AuthorizationCookie, cookies[0].Value, &decodedCookie)
		assert.NoError(t, err)
		assert.Equal(t, "user01", decodedCookie.Username)
	}

	dbMock.AssertExpectations(t)
}

func TestLoginHandlerIncorrectPassword(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader("username=user01&password=accessdenied"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))
	assert.Empty(t, res.Result().Cookies())

	dbMock.AssertExpectations(t)
}

func TestLoginHandlerUnknownUsername(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user02").Return(nil, nil).Once()

	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader("username=user02&password=pass"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))
	assert.Empty(t, res.Result().Cookies())

	dbMock.AssertExpectations(t)
}

func TestFeedHandlerAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: cookieHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	feedListHelper.On("GetAllItems", user).Return([]*Item{
		&Item{
			Origin:   "Site 1",
			SortDate: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
			FetchURL: "fetchurl1",
		},
		&Item{
			Title:    "t2",
			Origin:   "Feed 1",
			SortDate: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
			FetchURL: "fetchurl2",
		},
	}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `[{"Title":"","Origin":"Site 1","FetchURL":"fetchurl1"},{"Title":"t2","Origin":"Feed 1","FetchURL":"fetchurl2"}]`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedHandlerAuthorizedNoItemsFound(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: cookieHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	feedListHelper.On("GetAllItems", user).Return([]*Item{}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "[]\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedHandlerNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: cookieHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedHandlerUserDoesNotExist(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: cookieHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user01").Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedItemAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}
	item := &data.Feeditem{
		Title:    "Title 1",
		URL:      "http://site1/link1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "Text 1",
		Key:      key,
	}

	dbMock.On("GetFeeditem", key).Return(item, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(key.CreateKey()), nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"URL":"http://site1/link1","Contents":"Text 1","Date":"2019-02-16T23:00:00Z","Plaintext":false}`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestPageAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	config := &data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}
	page := &data.PagemonitorPage{
		Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Delta:   "Text 1",
		Config:  config,
	}

	dbMock.On("GetPage", config).Return(page, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(config.CreateKey()), nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"URL":"http://site1/1","Contents":"Text 1","Date":"2019-02-16T23:00:00Z","Plaintext":true}`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestFeedItemAuthorizedNotFound(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}

	dbMock.On("GetFeeditem", key).Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(key.CreateKey()), nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "Not found\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestPageAuthorizedNotFound(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	config := &data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}

	dbMock.On("GetPage", config).Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(config.CreateKey()), nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "Not found\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestGetUnknownItemTypeAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/magic", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "Not found\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestGetItemNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/items/feeditem--", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestGetItemUserDoesNotExist(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user01").Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/feeditem--", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestGetSettingsAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("GET", "/api/configuration", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user01","Opml":"opml1","Pagemonitor":"pagemonitor1"}`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestGetSettingsNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/configuration", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestGetSettingsUserDoesNotExist(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user01").Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/configuration", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestSaveSettingsAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Twice()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	saveUser := *user
	saveUser.Opml = "opml2"
	saveUser.Pagemonitor = "pagemonitor2"
	dbMock.On("SaveUser", &saveUser).Return(nil).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user01","Opml":"opml2","Pagemonitor":"pagemonitor2"}`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestSaveSettingsChangePasswordAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Twice()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Password=newpass&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	dbMock.On("SaveUser", mock.AnythingOfType("*data.User")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			saveUser := args.Get(0).(*data.User)
			assert.NoError(t, saveUser.ValidatePassword("newpass"))
			assert.Equal(t, "opml2", saveUser.Opml)
			assert.Equal(t, "pagemonitor2", saveUser.Pagemonitor)
		})

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user01","Opml":"opml2","Pagemonitor":"pagemonitor2"}`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestSaveSettingsChangeUsernameAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user02&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	saveUser := *user
	saveUser.Opml = "opml2"
	saveUser.Pagemonitor = "pagemonitor2"
	dbMock.On("SetUsername", &saveUser, "user02").Return(nil).Once()

	dbMock.On("GetUser", "user02").Return(&saveUser, nil).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user02","Opml":"opml2","Pagemonitor":"pagemonitor2"}`+"\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestSaveSettingsChangeUsernameFailedAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user02&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	saveUser := *user
	saveUser.Opml = "opml2"
	saveUser.Pagemonitor = "pagemonitor2"
	dbMock.On("SetUsername", &saveUser, "user02").Return(fmt.Errorf("Username already in use")).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, "Internal server error\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestSaveSettingsUnauthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestSaveSettingsUserDoesNotExist(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	services := &Services{db: dbMock, cookieHandler: cookieHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user01").Return(nil, nil).Once()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
}

func TestRefreshAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	fetcherMock := new(FetcherMock)
	fetcherMock.On("Refresh").Once()

	services := &Services{db: dbMock, cookieHandler: cookieHandler, fetcher: fetcherMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("GET", "/api/refresh", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "OK", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	fetcherMock.AssertExpectations(t)
}

func TestRefreshNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	fetcherMock := new(FetcherMock)

	services := &Services{db: dbMock, cookieHandler: cookieHandler, fetcher: fetcherMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/refresh", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	fetcherMock.AssertExpectations(t)
}

func TestRefreshUserDoesNotExist(t *testing.T) {
	dbMock := new(DBMock)
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	fetcherMock := new(FetcherMock)

	services := &Services{db: dbMock, cookieHandler: cookieHandler, fetcher: fetcherMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user01").Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/refresh", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", string(res.Body.Bytes()))

	dbMock.AssertExpectations(t)
	fetcherMock.AssertExpectations(t)
}
