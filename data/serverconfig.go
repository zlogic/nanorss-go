package data

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

// GetOrCreateConfigVariable returns the value for the varName ServerConfig variable, or if there's no entry, uses generator to create and save a value.
func (s *DBService) GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error) {
	varValue := ""
	varKey := CreateServerConfigKey(varName)
	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(varKey)
		if err == badger.ErrKeyNotFound {
			varValue, err = generator()
			if err != nil {
				varValue = ""
				return err
			}
			if varValue == "" {
				return nil
			}
			return txn.Set(varKey, []byte(varValue))
		}

		value, err := item.Value()
		if err != nil {
			return err
		}
		varValue = string(value)
		return nil
	})
	if err != nil {
		return "", errors.Wrapf(err, "Cannot read config key %v", varName)
	}
	return varValue, nil
}

// SetConfigVariable returns the value for the varName ServerConfig variable, or nil if no value is saved.
func (s *DBService) SetConfigVariable(varName, varValue string) error {
	varKey := CreateServerConfigKey(varName)
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(varKey, []byte(varValue))
	})
	if err != nil {
		return errors.Wrapf(err, "Cannot write config key %v", varName)
	}
	return nil
}

// GetAllConfigVariables returns all ServerConfig variables in a key-value map.
func (s *DBService) GetAllConfigVariables() (map[string]string, error) {
	vars := make(map[string]string)
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(ServerConfigKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()

			key, err := DecodeServerConfigKey(k)
			if err != nil {
				return errors.Wrapf(err, "Error reading config key %v", k)
			}

			value, err := item.Value()
			if err != nil {
				return errors.Wrapf(err, "Error reading config value %v", k)
			}
			vars[key] = string(value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return vars, nil
}
