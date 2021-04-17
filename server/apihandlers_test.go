package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/zlogic/nanorss-go/data"
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
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	authHandler.On("SetCookieUsername", mock.Anything, "user01", false).
		Run(func(args mock.Arguments) {
			w := args.Get(0).(http.ResponseWriter)
			http.SetCookie(w, &http.Cookie{Name: testAuthCookie})
		}).
		Return(nil).Once()

	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader("username=user01&password=pass"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "OK", res.Body.String())
	cookies := res.Result().Cookies()
	assert.Equal(t, 1, len(cookies))
	if len(cookies) > 0 {
		assert.Equal(t, testAuthCookie, cookies[0].Name)
	}

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestLoginHandlerIncorrectPassword(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
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
	assert.Equal(t, "Bad credentials\n", res.Body.String())
	assert.Empty(t, res.Result().Cookies())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestLoginHandlerUnknownUsername(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	dbMock.On("GetUser", "user02").Return(nil, nil).Once()

	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader("username=user02&password=pass"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())
	assert.Empty(t, res.Result().Cookies())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestFeedHandlerAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: &authHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	feedListHelper.On("GetAllItems", user).Return([]*Item{
		{
			Origin:   "Site 1",
			SortDate: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
			FetchURL: "fetchurl1",
			IsRead:   true,
		},
		{
			Title:    "t2",
			Origin:   "Feed 1",
			SortDate: time.Date(2019, time.February, 18, 23, 0, 0, 0, time.UTC),
			FetchURL: "fetchurl2",
		},
	}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `[{"Title":"","Origin":"Site 1","FetchURL":"fetchurl1","IsRead":true},{"Title":"t2","Origin":"Feed 1","FetchURL":"fetchurl2","IsRead":false}]`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedHandlerAuthorizedNoItemsFound(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: &authHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	feedListHelper.On("GetAllItems", user).Return([]*Item{}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "[]\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedHandlerNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	feedListHelper := new(FeedListHelperMock)

	services := &Services{db: dbMock, cookieHandler: &authHandler, feedListHelper: feedListHelper}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/feed", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
	feedListHelper.AssertExpectations(t)
}

func TestFeedItemAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}
	item := &data.Feeditem{
		Title:    "Title 1",
		URL:      "http://site1/link1",
		Date:     time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Contents: "Text 1",
		Key:      key,
	}

	dbMock.On("GetFeeditem", key).Return(item, nil).Once()
	dbMock.On("SetReadStatus", user, key.CreateKey(), true).Return(nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(key.CreateKey()), nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"URL":"http://site1/link1","Contents":"Text 1","Date":"2019-02-16T23:00:00Z","Plaintext":false,"MarkUnreadURL":"api/items/feed-aHR0cDovL3NpdGUxL3Jzcw-ZzE"}`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestPageAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	config := &data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}
	page := &data.PagemonitorPage{
		Updated: time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC),
		Delta:   "Text 1",
		Config:  config,
	}

	dbMock.On("GetPage", config).Return(page, nil).Once()
	dbMock.On("SetReadStatus", user, config.CreateKey(), true).Return(nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(config.CreateKey()), nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"URL":"http://site1/1","Contents":"Text 1","Date":"2019-02-16T23:00:00Z","Plaintext":true,"MarkUnreadURL":"api/items/pagemonitor-aHR0cDovL3NpdGUxLzE-bTE-cjE"}`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestFeedItemAuthorizedNotFound(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}

	dbMock.On("GetFeeditem", key).Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(key.CreateKey()), nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "Not found\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestPageAuthorizedNotFound(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	config := &data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}

	dbMock.On("GetPage", config).Return(nil, nil).Once()

	req, _ := http.NewRequest("GET", "/api/items/"+escapeKeyForURL(config.CreateKey()), nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "Not found\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetUnknownItemTypeAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	req, _ := http.NewRequest("GET", "/api/items/magic", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "Not found\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetItemNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/items/feeditem--", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSetUnreadAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}

	dbMock.On("SetReadStatus", user, key.CreateKey(), false).Return(nil).Once()

	req, _ := http.NewRequest("POST", "/api/items/"+escapeKeyForURL(key.CreateKey()), strings.NewReader("Read=false"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "OK", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestPostItemUnsupportedOperationAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}

	req, _ := http.NewRequest("POST", "/api/items/"+escapeKeyForURL(key.CreateKey()), strings.NewReader("Read=true"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, "Internal server error\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSetUnreadNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	key := &data.FeeditemKey{FeedURL: "http://site1/rss", GUID: "g1"}

	req, _ := http.NewRequest("POST", "/api/items/"+escapeKeyForURL(key.CreateKey()), strings.NewReader("Read=false"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetSettingsAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	req, _ := http.NewRequest("GET", "/api/configuration", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user01","Opml":"opml1","Pagemonitor":"pagemonitor1"}`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetSettingsNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/configuration", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSaveSettingsAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")

	authHandler.AllowUser(user)
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	saveUser := *user
	saveUser.Opml = "opml2"
	saveUser.Pagemonitor = "pagemonitor2"
	err = saveUser.SetUsername("user01")
	assert.NoError(t, err)
	dbMock.On("SaveUser", &saveUser).Return(nil).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user01","Opml":"opml2","Pagemonitor":"pagemonitor2"}`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSaveSettingsChangePasswordAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")

	authHandler.AllowUser(user)
	dbMock.On("GetUser", "user01").Return(user, nil).Once()

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Password=newpass&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	dbMock.On("SaveUser", mock.AnythingOfType("*data.User")).Return(nil).Once().
		Run(func(args mock.Arguments) {
			saveUser := args.Get(0).(*data.User)
			assert.NoError(t, saveUser.ValidatePassword("newpass"))
			assert.Equal(t, "opml2", saveUser.Opml)
			assert.Equal(t, "pagemonitor2", saveUser.Pagemonitor)
		})

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user01","Opml":"opml2","Pagemonitor":"pagemonitor2"}`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSaveSettingsChangeUsernameAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user02&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	saveUser := *user
	saveUser.Opml = "opml2"
	saveUser.Pagemonitor = "pagemonitor2"
	err = saveUser.SetUsername("user02")
	assert.NoError(t, err)

	getUpdatedUser := data.NewUser("user02")
	getUpdatedUser.Opml = "opml2"
	getUpdatedUser.Pagemonitor = "pagemonitor2"

	dbMock.On("SaveUser", &saveUser).Return(nil).Once().
		Run(func(args mock.Arguments) {
			userArg := args.Get(0).(*data.User)
			*userArg = *getUpdatedUser
		})
	dbMock.On("GetUser", "user02").Return(getUpdatedUser, nil).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"Username":"user02","Opml":"opml2","Pagemonitor":"pagemonitor2"}`+"\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSaveSettingsChangeUsernameFailedAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = "opml1"
	user.Pagemonitor = "pagemonitor1"
	user.SetPassword("pass")

	authHandler.AllowUser(user)

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user02&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	saveUser := *user
	saveUser.Opml = "opml2"
	saveUser.Pagemonitor = "pagemonitor2"
	err = saveUser.SetUsername("user02")
	assert.NoError(t, err)
	dbMock.On("SaveUser", &saveUser).Return(fmt.Errorf("Username already in use")).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, "Internal server error\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestSaveSettingsUnauthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("POST", "/api/configuration", strings.NewReader("Username=user01&Opml=opml2&Pagemonitor=pagemonitor2"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestRefreshAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	fetcherMock := new(FetcherMock)
	fetcherMock.On("Refresh").Once()

	services := &Services{db: dbMock, cookieHandler: &authHandler, fetcher: fetcherMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")

	authHandler.AllowUser(user)

	req, _ := http.NewRequest("GET", "/api/refresh", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "OK", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
	fetcherMock.AssertExpectations(t)
}

func TestRefreshNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	fetcherMock := new(FetcherMock)

	services := &Services{db: dbMock, cookieHandler: &authHandler, fetcher: fetcherMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/refresh", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
	fetcherMock.AssertExpectations(t)
}

func TestGetStatusAuthorizedSuccess(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = defaultOpml
	user.Pagemonitor = defaultPagemonitor

	authHandler.AllowUser(user)

	date1 := time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)
	date2 := time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	dbMock.On("GetFetchStatus", (&data.UserFeed{URL: "http://site1/rss"}).CreateKey()).Return(&data.FetchStatus{LastSuccess: date2, LastFailure: date1}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserFeed{URL: "http://site2/rss"}).CreateKey()).Return(&data.FetchStatus{LastSuccess: date1}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}).CreateKey()).Return(&data.FetchStatus{LastSuccess: date2, LastFailure: date1}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserPagemonitor{URL: "http://site1/2"}).CreateKey()).Return(&data.FetchStatus{LastSuccess: date2}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/status", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "["+`{"Name":"Feed 1","Success":true,"LastFailure":"2019-02-16T23:00:00Z","LastSuccess":"2019-02-16T23:01:00Z"},`+
		`{"Name":"Feed 2","Success":true,"LastSuccess":"2019-02-16T23:00:00Z"},`+
		`{"Name":"Site 1","Success":true,"LastFailure":"2019-02-16T23:00:00Z","LastSuccess":"2019-02-16T23:01:00Z"},`+
		`{"Name":"Site 2","Success":true,"LastSuccess":"2019-02-16T23:01:00Z"}`+
		"]\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetStatusAuthorizedFailure(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = defaultOpml
	user.Pagemonitor = defaultPagemonitor

	authHandler.AllowUser(user)

	date1 := time.Date(2019, time.February, 16, 23, 0, 0, 0, time.UTC)
	date2 := time.Date(2019, time.February, 16, 23, 1, 0, 0, time.UTC)
	dbMock.On("GetFetchStatus", (&data.UserFeed{URL: "http://site1/rss"}).CreateKey()).Return(&data.FetchStatus{LastSuccess: date1, LastFailure: date2}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserFeed{URL: "http://site2/rss"}).CreateKey()).Return(&data.FetchStatus{LastFailure: date1}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}).CreateKey()).Return(&data.FetchStatus{LastSuccess: date1, LastFailure: date2}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserPagemonitor{URL: "http://site1/2"}).CreateKey()).Return(&data.FetchStatus{LastFailure: date2}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/status", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "["+`{"Name":"Feed 1","Success":false,"LastFailure":"2019-02-16T23:01:00Z","LastSuccess":"2019-02-16T23:00:00Z"},`+
		`{"Name":"Feed 2","Success":false,"LastFailure":"2019-02-16T23:00:00Z"},`+
		`{"Name":"Site 1","Success":false,"LastFailure":"2019-02-16T23:01:00Z","LastSuccess":"2019-02-16T23:00:00Z"},`+
		`{"Name":"Site 2","Success":false,"LastFailure":"2019-02-16T23:01:00Z"}`+
		"]\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetStatusAuthorizedUnknown(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	user := data.NewUser("user01")
	user.Opml = defaultOpml
	user.Pagemonitor = defaultPagemonitor

	authHandler.AllowUser(user)

	dbMock.On("GetFetchStatus", (&data.UserFeed{URL: "http://site1/rss"}).CreateKey()).Return(&data.FetchStatus{}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserFeed{URL: "http://site2/rss"}).CreateKey()).Return(&data.FetchStatus{}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserPagemonitor{URL: "http://site1/1", Match: "m1", Replace: "r1"}).CreateKey()).Return(&data.FetchStatus{}, nil).Once()
	dbMock.On("GetFetchStatus", (&data.UserPagemonitor{URL: "http://site1/2"}).CreateKey()).Return(&data.FetchStatus{}, nil).Once()

	req, _ := http.NewRequest("GET", "/api/status", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "["+`{"Name":"Feed 1","Success":false},`+
		`{"Name":"Feed 2","Success":false},`+
		`{"Name":"Site 1","Success":false},`+
		`{"Name":"Site 2","Success":false}`+
		"]\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}

func TestGetStatusNotAuthorized(t *testing.T) {
	dbMock := new(DBMock)
	authHandler := AuthHandlerMock{}

	services := &Services{db: dbMock, cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/status", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.Equal(t, "Bad credentials\n", res.Body.String())

	dbMock.AssertExpectations(t)
	authHandler.AssertExpectations(t)
}
