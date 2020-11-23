package data

import (
	"database/sql"
	"fmt"
)

// GetOrCreateConfigVariable returns the value for the varName ServerConfig variable, or if there's no entry, uses generator to create and save a value.
func (s *DBService) GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error) {
	varValue := ""
	err := s.updateTx(func(tx *sql.Tx) error {
		err := tx.QueryRow("SELECT value FROM serverconfig WHERE key=$1", varName).Scan(&varValue)
		if err == sql.ErrNoRows {
			varValue, err = generator()
			if err != nil {
				varValue = ""
				return err
			}
			if varValue == "" {
				return nil
			}
			_, err := tx.Exec("INSERT INTO serverconfig(key, value) VALUES($1, $2)", varName, varValue)
			return err
		}

		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("cannot read config key %v because of %w", varName, err)
	}
	return varValue, nil
}

// SetConfigVariable returns the value for the varName ServerConfig variable, or nil if no value is saved.
func (s *DBService) SetConfigVariable(varName, varValue string) error {
	err := s.updateTx(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO serverconfig(key, value) VALUES($1, $2)", varName, varValue)
		return err
	})
	if err != nil {
		return fmt.Errorf("cannot write config key %v because of %w", varName, err)
	}
	return nil
}

// GetAllConfigVariables returns all ServerConfig variables in a key-value map.
func (s *DBService) GetAllConfigVariables() (map[string]string, error) {
	vars := make(map[string]string)

	rows, err := s.db.Query("SELECT key, value FROM serverconfig")
	if err != nil {
		return nil, fmt.Errorf("error reading config keys because of %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("error reading config entry: %w", err)
		}
		vars[key] = value
	}

	return vars, nil
}
