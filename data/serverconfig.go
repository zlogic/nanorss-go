package data

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

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
