package data

import (
	"fmt"

	"github.com/akrylysov/pogreb"
)

// GetOrCreateConfigVariable returns the value for the varName ServerConfig variable, or if there's no entry, uses generator to create and save a value.
func (s *DBService) GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error) {
	varKey := CreateServerConfigKey(varName)
	val, err := s.db.Get(varKey)
	if err != nil {
		return "", fmt.Errorf("Cannot read config key %v because of %w", varName, err)
	}
	varValue := string(val)
	if val == nil {
		varValue, err = generator()
		if err != nil {
			varValue = ""
			return "", err
		}
		if varValue == "" {
			return "", nil
		}
		err := s.db.Put(varKey, []byte(varValue))
		if err != nil {
			return "", fmt.Errorf("Cannot save config key %v because of %w", varName, err)
		}
	}

	return varValue, nil
}

// SetConfigVariable returns the value for the varName ServerConfig variable, or nil if no value is saved.
func (s *DBService) SetConfigVariable(varName, varValue string) error {
	varKey := CreateServerConfigKey(varName)
	err := s.db.Put(varKey, []byte(varValue))
	if err != nil {
		return fmt.Errorf("Cannot write config key %v because of %w", varName, err)
	}
	return nil
}

// GetAllConfigVariables returns all ServerConfig variables in a key-value map.
func (s *DBService) GetAllConfigVariables() (map[string]string, error) {
	vars := make(map[string]string)
	prefix := []byte(ServerConfigKeyPrefix)

	it := s.db.Items()
	for {
		k, v, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return nil, fmt.Errorf("Cannot read config variables because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		key, err := DecodeServerConfigKey(k)
		if err != nil {
			return nil, fmt.Errorf("Error reading config key %v because of %w", k, err)
		}

		vars[key] = string(v)
	}
	return vars, nil
}
