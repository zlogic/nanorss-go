package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewCookieHandlerGenerateNewKey(t *testing.T) {
	dbMock := new(DBMock)

	var signKey string
	dbMock.On("GetOrCreateConfigVariable", "cookie-sign-key", mock.AnythingOfType("func() (string, error)")).Return(signKey, nil).Once().
		Run(func(args mock.Arguments) {
			generator := args.Get(1).(func() (string, error))
			key, err := generator()
			assert.NoError(t, err)
			assert.NotEmpty(t, key)
			signKey = key
		})
	handler, err := NewCookieHandler(dbMock)
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	dbMock.AssertExpectations(t)
}

func TestGetEmptyCookie(t *testing.T) {
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/", nil)
	res := httptest.NewRecorder()

	username := cookieHandler.GetUsername(res, req)
	assert.Equal(t, "", username)
}

func TestGetInvalidCookie(t *testing.T) {
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/", nil)
	res := httptest.NewRecorder()

	req.AddCookie(cookieHandler.NewCookie())

	username := cookieHandler.GetUsername(res, req)
	assert.Equal(t, "", username)
}

func TestGetValidCookie(t *testing.T) {
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/", nil)
	res := httptest.NewRecorder()

	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	username := cookieHandler.GetUsername(res, req)
	assert.Equal(t, "user01", username)
}

func TestGetExpiredCookie(t *testing.T) {
	cookieHandler, err := createTestCookieHandler()
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/api/", nil)
	res := httptest.NewRecorder()

	cookieHandler.cookieExpires = -1 * time.Hour
	cookie := cookieHandler.NewCookie()
	cookieHandler.SetCookieUsername(cookie, "user01")
	req.AddCookie(cookie)

	username := cookieHandler.GetUsername(res, req)
	assert.Equal(t, "", username)
}
