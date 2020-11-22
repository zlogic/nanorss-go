package datadb

import (
	"context"
	"fmt"
	"os"

	"database/sql"

	log "github.com/sirupsen/logrus"
)

// DBService provides services for reading and writing structs in the database.
type DBService struct {
	db *sql.DB
}

// Open opens the database and returns a DBService instance.
func Open() (*DBService, error) {
	databaseURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		return nil, fmt.Errorf("cannot determine database URL - DATABASE_URL is missing")
	}

	log.Info("Opening database")
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		defer db.Close()
		return nil, fmt.Errorf("failed to ping database %w", err)
	}

	log.Info("Applying database migrations")
	dbService := &DBService{db: db}
	err = dbService.updateTx(applyMigrations)
	if err != nil {
		defer db.Close()
		return nil, fmt.Errorf("failed to apply database migrations: %w", err)
	}
	return dbService, nil
}

// GC deletes expired items and attempts to perform a database cleanup.
func (s *DBService) GC() {
	s.deleteExpiredItems()
}

// Close closes the underlying database.
func (s *DBService) Close() {
	log.Info("Closing database")
	if s != nil && s.db != nil {
		err := s.db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// viewTx runs the function in a read-only transaction.
func (s *DBService) viewTx(fn func(tx *sql.Tx) error) error {
	return tx(s.db, fn, true)
}

// updateTx runs the function in a read-write transaction.
func (s *DBService) updateTx(fn func(tx *sql.Tx) error) error {
	return tx(s.db, fn, false)
}

// tx runs the function in a transaction.
// If the function doesn't return an error, the transaction is committed.
// Otherwise, the transaction will be rolled back.
func tx(db *sql.DB, fn func(tx *sql.Tx) error, readOnly bool) error {
	tx, err := db.BeginTx(context.TODO(), &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		return fmt.Errorf("failed to open transaction: %w", err)
	}

	if fnErr := fn(tx); fnErr != nil {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("transaction failed to rollback: %v when hadling error from: %w", err, fnErr)
		}
		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
