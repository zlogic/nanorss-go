package data

import (
	"fmt"

	"github.com/akrylysov/pogreb"
)

// GetOrCreateConfigVariable returns the value for the varName ServerConfig variable,
// or if there's no entry, uses generator to create and save a value.
func (s *DBService) GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error) {
	varKey := createServerConfigKey(varName)
	value, err := s.db.Get(varKey)
	if err != nil {
		return "", fmt.Errorf("cannot get config key %v: %w", varName, err)
	}
	if value != nil {
		return string(value), nil
	}
	varValue, err := generator()
	if err != nil {
		varValue = ""
		return "", err
	}
	if varValue == "" {
		return "", nil
	}
	err = s.db.Put(varKey, []byte(varValue))
	if err != nil {
		return "", err
	}
	return varValue, nil
}

// SetConfigVariable returns the value for the varName ServerConfig variable, or nil if no value is saved.
func (s *DBService) SetConfigVariable(varName, varValue string) error {
	varKey := createServerConfigKey(varName)
	if err := s.db.Put(varKey, []byte(varValue)); err != nil {
		return fmt.Errorf("cannot write config key %v: %w", varName, err)
	}
	return nil
}

// GetAllConfigVariables returns all ServerConfig variables in a key-value map.
func (s *DBService) GetAllConfigVariables() (map[string]string, error) {
	vars := make(map[string]string)
	it := s.db.Items()
	for {
		// TODO: use an index here.
		k, value, err := it.Next()
		if err == pogreb.ErrIterationDone {
			break
		} else if err != nil {
			return nil, err
		}
		if !isServerConfigKey(k) {
			continue
		}

		key, err := decodeServerConfigKey(k)
		if err != nil {
			return nil, fmt.Errorf("error reading config key %v: %w", string(k), err)
		}

		vars[key] = string(value)
	}
	return vars, nil
}
