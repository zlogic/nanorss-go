package datadb

import (
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
		return nil, fmt.Errorf("failed to ping database %w", err)
	}
	return &DBService{db: db}, nil
}

// GC deletes expired items and attempts to perform a database cleanup.
func (service *DBService) GC() {
	//service.DeleteExpiredItems()
	//service.DeleteStaleFetchStatuses()
	//service.DeleteStaleReadStatuses()
}

// Close closes the underlying database.
func (service *DBService) Close() {
	log.Info("Closing database")
	if service != nil && service.db != nil {
		err := service.db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}
