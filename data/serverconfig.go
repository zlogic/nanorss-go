package data

import (
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

// GetOrCreateConfigVariable returns the value for the varName ServerConfig variable, or if there's no entry, uses generator to create and save a value.
func (s *DBService) GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error) {
	varValue, err := s.client.HGet(ServerConfigKey, varName).Result()
	if err == redis.Nil {
		varValue = ""
	} else if err != nil {
		return "", errors.Wrapf(err, "Cannot read config key %v", varName)
	} else {
		return varValue, nil
	}
	varValue, err = generator()
	if err != nil {
		return "", err
	}
	err = s.client.HSetNX(ServerConfigKey, varName, varValue).Err()
	if err != nil {
		return "", err
	}
	return varValue, nil
}

// SetConfigVariable returns the value for the varName ServerConfig variable, or nil if no value is saved.
func (s *DBService) SetConfigVariable(varName, varValue string) error {
	_, err := s.client.HSet(ServerConfigKey, varName, varValue).Result()
	if err != nil {
		return errors.Wrapf(err, "Cannot write config key %v", varName)
	}
	return nil
}

// GetAllConfigVariables returns all ServerConfig variables in a key-value map.
func (s *DBService) GetAllConfigVariables() (map[string]string, error) {
	return s.client.HGetAll(ServerConfigKey).Result()
}
