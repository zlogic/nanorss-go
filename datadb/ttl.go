package datadb

import (
	"database/sql"
	"fmt"
	"time"
)

var itemTTL = 14 * 24 * time.Hour

// deleteExpiredItems deletes all items which the Update function was not called at least itemTTL.
func (s *DBService) deleteExpiredItems() error {
	expires := time.Now().Add(-itemTTL)
	return s.updateTx(func(tx *sql.Tx) error {
		errors := make([]error, 0, 2)

		_, err := tx.Exec("DELETE FROM feeditems WHERE updated < $1", expires)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to clean up expired feed items %w", err))
		}

		_, err = tx.Exec("DELETE FROM pagemonitors WHERE updated < $1", expires)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to clean up expired pages %w", err))
		}

		if len(errors) > 0 {
			return fmt.Errorf("failed to delete expired items: %v", errors)
		}
		return nil
	})
}
