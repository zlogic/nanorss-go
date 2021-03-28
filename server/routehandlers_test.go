package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/zlogic/nanorss-go/data"
)

const layoutTemplate = `{{ define "layout" }}User {{ .User }}
Name {{ .Name }}
Content {{ template "content" . }}{{ end }}`

func prepareTemplate(pageName, tmpl string) fs.FS {
	files := fstest.MapFS{
		"layout.html":                 &fstest.MapFile{Data: []byte(layoutTemplate)},
		"pages/" + pageName + ".html": &fstest.MapFile{Data: []byte(tmpl)},
	}
	return files
}

func TestRootHandlerNotLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()

	authHandler.On("HasAuthenticationCookie", mock.Anything).
		Return(false).Once()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/login", res.Header().Get("Location"))

	authHandler.AssertExpectations(t)
}

func TestRootHandlerLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()

	authHandler.On("HasAuthenticationCookie", mock.Anything).
		Return(true).Once()
	authHandler.AllowUser(&data.User{})

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/feed", res.Header().Get("Location"))

	authHandler.AssertExpectations(t)
}

func TestLogoutHandler(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/logout", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/login", res.Header().Get("Location"))

	assert.Empty(t, res.Result().Cookies())

	authHandler.AssertExpectations(t)
}

func TestFaviconHandler(t *testing.T) {
	faviconBytes, err := faviconFS.ReadFile(faviconFilename)
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	router, err := CreateRouter(&Services{cookieHandler: &authHandler})
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	res := newRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, faviconBytes, res.Body.Bytes())

	authHandler.AssertExpectations(t)
}

func TestHtmlLoginHandlerNotLoggedIn(t *testing.T) {
	templates := prepareTemplate("login", `{{ define "content" }}loginpage{{ end }}`)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler, templates: templates}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/login", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User <nil>\nName \nContent loginpage", res.Body.String())

	authHandler.AssertExpectations(t)
}

func TestHtmlLoginHandlerAlreadyLoggedIn(t *testing.T) {
	templates := prepareTemplate("login", `{{ define "content" }}loginpage{{ end }}`)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler, templates: templates}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/login", nil)
	res := httptest.NewRecorder()

	authHandler.AllowUser(&data.User{})

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User <nil>\nName \nContent loginpage", res.Body.String())

	authHandler.AssertExpectations(t)
}

func TestHtmlFeedHandlerLoggedIn(t *testing.T) {
	templates := prepareTemplate("feed", `{{ define "content" }}feedpage{{ end }}`)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler, templates: templates}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/feed", nil)
	res := httptest.NewRecorder()

	authHandler.AllowUser(&data.User{})

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User {    }\nName feed\nContent feedpage", res.Body.String())

	authHandler.AssertExpectations(t)
}

func TestHtmlFeedHandlerNotLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/feed", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/login", res.Header().Get("Location"))

	authHandler.AssertExpectations(t)
}

func TestHtmlSettingsHandlerLoggedIn(t *testing.T) {
	templates := prepareTemplate("settings", `{{ define "content" }}settingspage{{ end }}`)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler, templates: templates}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/settings", nil)
	res := httptest.NewRecorder()

	authHandler.AllowUser(&data.User{})

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User {    }\nName settings\nContent settingspage", res.Body.String())

	authHandler.AssertExpectations(t)
}

func TestHtmlSettingsHandlerNotLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/settings", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/login", res.Header().Get("Location"))

	authHandler.AssertExpectations(t)
}

func TestHtmlStatusHandlerLoggedIn(t *testing.T) {
	templates := prepareTemplate("status", `{{ define "content" }}feedpage{{ end }}`)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler, templates: templates}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/status", nil)
	res := httptest.NewRecorder()

	authHandler.AllowUser(&data.User{})

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User {    }\nName status\nContent feedpage", res.Body.String())

	authHandler.AssertExpectations(t)
}

func TestHtmlStatusHandlerNotLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/status", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/login", res.Header().Get("Location"))

	authHandler.AssertExpectations(t)
}
