package data

import (
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetValue(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

	err = dbService.db.Update(func(txn *badger.Txn) error {
		return txn.Set(createServerConfigKey("k1"), []byte("v1"))
	})
	assert.NoError(t, err)

	value, err := dbService.GetOrCreateConfigVariable("k1", func() (string, error) {
		assert.Fail(t, "Generator should not be called")
		return "", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "v1", value)
}

func TestGenerateValue(t *testing.T) {
	err := resetDb()
	assert.NoError(t, err)

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
	err := resetDb()
	assert.NoError(t, err)

	err = dbService.SetConfigVariable("k1", "v1")
	assert.NoError(t, err)

	err = dbService.SetConfigVariable("k2", "v2")
	assert.NoError(t, err)

	values, err := dbService.GetAllConfigVariables()
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, values)
}
