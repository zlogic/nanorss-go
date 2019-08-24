package server

import (
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

var staticFileBytes = []byte("Test data")

const staticFileName = "test.txt"

func prepareStaticTestFile(tempDir string) error {
	return prepareTestFile(path.Join(tempDir, "static"), staticFileName, staticFileBytes)
}

func TestGetStaticResource(t *testing.T) {
	// Prepare resources dir
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareStaticTestFile(tempDir)
	assert.NoError(t, err)

	dbMock := new(DBMock)
	services := Services{db: dbMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/static/test.txt", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, staticFileBytes, res.Body.Bytes())

	dbMock.AssertExpectations(t)
}

func TestListingNotAllowed(t *testing.T) {
	tempDir, recover, err := prepareTempDir()
	defer func() {
		if recover != nil {
			recover()
		}
	}()
	assert.NoError(t, err)
	err = prepareStaticTestFile(tempDir)
	assert.NoError(t, err)

	dbMock := new(DBMock)
	services := Services{db: dbMock}
	router, err := CreateRouter(services)
	assert.NoError(t, err)

	for _, url := range []string{"/static", "/static/"} {
		req, _ := http.NewRequest("GET", url, nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotFound, res.Code)
		assert.Equal(t, "404 page not found\n", string(res.Body.Bytes()))
	}

	dbMock.AssertExpectations(t)
}
