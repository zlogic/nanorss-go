package server

import (
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zlogic/nanorss-go/data"
)

const layoutTemplate = `{{ define "layout" }}User {{ .User }}
Name {{ .Name }}
Content {{ template "content" . }}{{ end }}`

func prepareLayoutTemplateTestFile(tempDir string) error {
	return prepareTestFile(path.Join(tempDir, "templates"), "layout.html", []byte(layoutTemplate))
}

func TestRootHandlerNotLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/login", res.Header().Get("Location"))
}

func TestRootHandlerLoggedIn(t *testing.T) {
	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()

	req.AddCookie(authHandler.AllowCookie(&data.User{}))

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/feed", res.Header().Get("Location"))
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
}

func TestFaviconHandler(t *testing.T) {
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	faviconBytes := []byte("i am a favicon")
	err = prepareTestFile(path.Join(tempDir, "static"), "favicon.ico", faviconBytes)
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	router, err := CreateRouter(&Services{cookieHandler: &authHandler})
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, faviconBytes, res.Body.Bytes())
}

func TestHtmlLoginHandlerNotLoggedIn(t *testing.T) {
	loginTemplate := []byte(`{{ define "content" }}loginpage{{ end }}`)
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareLayoutTemplateTestFile(tempDir)
	assert.NoError(t, err)
	err = prepareTestFile(path.Join(tempDir, "templates", "pages"), "login.html", []byte(loginTemplate))
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/login", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User <nil>\nName \nContent loginpage", string(res.Body.Bytes()))
}

func TestHtmlLoginHandlerAlreadyLoggedIn(t *testing.T) {
	loginTemplate := []byte(`{{ define "content" }}loginpage{{ end }}`)
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareLayoutTemplateTestFile(tempDir)
	assert.NoError(t, err)
	err = prepareTestFile(path.Join(tempDir, "templates", "pages"), "login.html", []byte(loginTemplate))
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/login", nil)
	res := httptest.NewRecorder()

	req.AddCookie(authHandler.AllowCookie(&data.User{}))

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User <nil>\nName \nContent loginpage", string(res.Body.Bytes()))
}

func TestHtmlFeedHandlerLoggedIn(t *testing.T) {
	loginTemplate := []byte(`{{ define "content" }}feedpage{{ end }}`)
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareLayoutTemplateTestFile(tempDir)
	assert.NoError(t, err)
	err = prepareTestFile(path.Join(tempDir, "templates", "pages"), "feed.html", []byte(loginTemplate))
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/feed", nil)
	res := httptest.NewRecorder()

	req.AddCookie(authHandler.AllowCookie(&data.User{}))

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User {    }\nName feed\nContent feedpage", string(res.Body.Bytes()))
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
}

func TestHtmlSettingsHandlerLoggedIn(t *testing.T) {
	loginTemplate := []byte(`{{ define "content" }}settingspage{{ end }}`)
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareLayoutTemplateTestFile(tempDir)
	assert.NoError(t, err)
	err = prepareTestFile(path.Join(tempDir, "templates", "pages"), "settings.html", []byte(loginTemplate))
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/settings", nil)
	res := httptest.NewRecorder()

	req.AddCookie(authHandler.AllowCookie(&data.User{}))

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User {    }\nName settings\nContent settingspage", string(res.Body.Bytes()))
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
}

func TestHtmlStatusHandlerLoggedIn(t *testing.T) {
	loginTemplate := []byte(`{{ define "content" }}feedpage{{ end }}`)
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareLayoutTemplateTestFile(tempDir)
	assert.NoError(t, err)
	err = prepareTestFile(path.Join(tempDir, "templates", "pages"), "status.html", []byte(loginTemplate))
	assert.NoError(t, err)

	authHandler := AuthHandlerMock{}

	services := &Services{cookieHandler: &authHandler}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/status", nil)
	res := httptest.NewRecorder()

	req.AddCookie(authHandler.AllowCookie(&data.User{}))

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "User {    }\nName status\nContent feedpage", string(res.Body.Bytes()))
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
}
