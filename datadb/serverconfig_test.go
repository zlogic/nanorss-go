package datadb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetValue(t *testing.T) {
	dbService, err := prepareServerConfigTests()
	assert.NoError(t, err)
	defer dbService.Close()

	_, err = dbService.db.Exec("INSERT INTO serverconfig(key, value) VALUES ($1, $2)", "k1", "v1")
	assert.NoError(t, err)

	value, err := dbService.GetOrCreateConfigVariable("k1", func() (string, error) {
		assert.Fail(t, "Generator should not be called")
		return "", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "v1", value)
}

func TestGenerateValue(t *testing.T) {
	dbService, err := prepareServerConfigTests()
	assert.NoError(t, err)
	defer dbService.Close()

	value, err := dbService.GetOrCreateConfigVariable("k1", func() (string, error) {
		return "v1", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "v1", value)

	value, err = dbService.GetOrCreateConfigVariable("k1", func() (string, error) {
		assert.Fail(t, "Generator should not be called the second time")
		return "", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "v1", value)
}

func TestGetAllValues(t *testing.T) {
	dbService, err := prepareServerConfigTests()
	assert.NoError(t, err)
	defer dbService.Close()

	err = dbService.SetConfigVariable("k1", "v1")
	assert.NoError(t, err)

	err = dbService.SetConfigVariable("k2", "v2")
	assert.NoError(t, err)

	values, err := dbService.GetAllConfigVariables()
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, values)
}

func prepareServerConfigTests() (*DBService, error) {
	useRealDatabase()
	dbService, err := Open()
	if err != nil {
		return nil, err
	}
	_, err = dbService.db.Exec("DELETE FROM serverconfig")
	if err != nil {
		dbService.Close()
		return nil, err
	}
	return dbService, nil
}
