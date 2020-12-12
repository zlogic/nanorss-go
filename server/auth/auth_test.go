package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	user, _ := args.Get(0).(*data.User)
	return user, args.Error(1)
}

func createTestCookieHandler() (*CookieHandler, error) {
	signKey := base64.StdEncoding.EncodeToString(generateRandomKey(64))
	dbMock := DBMock{}

	dbMock.On("GetOrCreateConfigVariable", "cookie-sign-key", mock.AnythingOfType("func() (string, error)")).
		Return(signKey, nil).
		Once()
	return NewCookieHandler(&dbMock)
}

func createTestEmptyCookie() *http.Cookie {
	return &http.Cookie{
		Name:    authenticationCookie,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  0,

		HttpOnly: true,
	}
}

func createTestCookie(handler *CookieHandler, username string, expires time.Duration) (*http.Cookie, error) {
	cookie := createTestEmptyCookie()

	currentExpires := handler.cookieExpires
	defer func() { handler.cookieExpires = currentExpires }()
	if expires != 0 {
		handler.cookieExpires = expires
	}

	value, err := handler.getUsernameToken(username)
	if err != nil {
		return nil, err
	}
	cookie.Value = value

	return cookie, nil
}

func TestNewCookieHandlerGenerateNewKey(t *testing.T) {
	dbMock := new(DBMock)

	dbMock.On("GetOrCreateConfigVariable", "cookie-sign-key", mock.AnythingOfType("func() (string, error)")).
		Run(func(args mock.Arguments) {
			generator := args.Get(1).(func() (string, error))
			key, err := generator()
			assert.NoError(t, err)
			assert.NotEmpty(t, key)
		}).
		Return("", nil).
		Once()

	handler, err := NewCookieHandler(dbMock)
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	dbMock.AssertExpectations(t)
}

func TestGetUsername(t *testing.T) {
	cookieHandler, err := createTestCookieHandler()
	if err != nil {
		t.Fatalf("failed to create cookie handler: %v", err)
	}

	validCookie, err := createTestCookie(cookieHandler, "user01", 0)
	if err != nil {
		t.Fatalf("failed to create test cookie: %v", err)
	}

	expiredCookie, err := createTestCookie(cookieHandler, "user01", -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create test cookie: %v", err)
	}

	tests := map[string]struct {
		Cookie         *http.Cookie
		ExpectUsername string
	}{
		"missing cookie": {
			ExpectUsername: "",
		},
		"invalid (empty) cookie": {
			Cookie:         createTestEmptyCookie(),
			ExpectUsername: "",
		},
		"valid cookie": {
			Cookie:         validCookie,
			ExpectUsername: "user01",
		},
		"expired cookie": {
			Cookie:         expiredCookie,
			ExpectUsername: "",
		},
	}

	for tName, test := range tests {
		t.Run(tName, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/", nil)
			res := httptest.NewRecorder()

			if test.Cookie != nil {
				req.AddCookie(test.Cookie)
			}

			username, err := cookieHandler.getUsername(res, req)
			assert.Equal(t, test.ExpectUsername, username)
			assert.NoError(t, err)
		})
	}
}

func TestAuthHandlerFunc(t *testing.T) {
	cookieHandler, err := createTestCookieHandler()
	if err != nil {
		t.Fatalf("failed to create cookie handler: %v", err)
	}
	dbMock, ok := cookieHandler.db.(*DBMock)
	if !ok {
		t.Fatalf("failed to parse db mock: %v", err)
	}

	validCookie, err := createTestCookie(cookieHandler, "user01", 0)
	if err != nil {
		t.Fatalf("failed to create test cookie: %v", err)
	}

	tests := map[string]struct {
		Cookie             *http.Cookie
		ExpectGetUser      string
		ReturnGetUserError error
		ReturnUser         *data.User
	}{
		"empty cookie": {
			Cookie: nil,
		},
		"valid cookie and user exists": {
			Cookie:        validCookie,
			ExpectGetUser: "user01",
			ReturnUser:    &data.User{Opml: "opml", Password: "pass"},
		},
		"valid cookie but user doesn't exist": {
			Cookie:        validCookie,
			ExpectGetUser: "user01",
		},
		"error getting user": {
			Cookie:             validCookie,
			ExpectGetUser:      "user01",
			ReturnGetUserError: fmt.Errorf("generic error"),
		},
	}

	for tName, test := range tests {
		t.Run(tName, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/", nil)
			res := httptest.NewRecorder()

			if test.Cookie != nil {
				req.AddCookie(test.Cookie)
			}

			if test.ExpectGetUser != "" {
				dbMock.On("GetUser", test.ExpectGetUser).
					Return(test.ReturnUser, test.ReturnGetUserError).
					Once()
			}

			var receivedUser *data.User
			cookieHandler.AuthHandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedUser = GetUser(r.Context())
			})).ServeHTTP(res, req)

			assert.Equal(t, test.ReturnUser, receivedUser)

			dbMock.AssertExpectations(t)
		})
	}
}
