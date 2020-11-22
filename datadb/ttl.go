package datadb

import (
	"database/sql"
	"fmt"
	"time"
)

var itemTTL = 14 * 24 * time.Hour

// deleteExpiredItems deletes all items which the Update function was not called at least itemTTL.
func (s *DBService) deleteExpiredItems() error {
	expires := time.Now().Add(-itemTTL).UTC()
	return s.updateTx(func(tx *sql.Tx) error {
		errors := make([]error, 0, 2)

		_, err := tx.Exec("DELETE FROM feeds WHERE (last_success < $1 OR last_success IS NULL) AND (last_failure < $1 OR last_failure IS NULL)", expires)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to clean up expired feeds %w", err))
		}

		_, err = tx.Exec("DELETE FROM feeditems WHERE (last_seen < $1 OR LAST_SEEN IS NULL)", expires)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to clean up expired feed items %w", err))
		}

		_, err = tx.Exec("DELETE FROM pagemonitors WHERE (last_success < $1 OR last_success IS NULL) AND (last_failure < $1 OR last_failure IS NULL)", expires)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to clean up expired pages %w", err))
		}

		if len(errors) > 0 {
			return fmt.Errorf("failed to delete expired items: %v", errors)
		}
		return nil
	})
}
