package data

import (
	"os"
	"path"

	"github.com/dgraph-io/badger/v2"
	log "github.com/sirupsen/logrus"
)

// DefaultOptions returns default options for the database, customized based on environment variables.
func DefaultOptions() badger.Options {
	dbPath, ok := os.LookupEnv("DATABASE_DIR")
	if !ok {
		dbPath = path.Join(os.TempDir(), "nanorss")
	}
	opts := badger.DefaultOptions(dbPath)
	// Add a logger
	opts.Logger = log.New()
	// Optimize options for low memory usage
	opts.MaxTableSize = 1 << 20
	opts.BlockCacheSize = 0
	opts.IndexCacheSize = 0
	// Allow GC of value log
	opts.ValueLogFileSize = 4 << 20
	opts.ValueLogMaxEntries = 10000
	return opts
}

// DBService provides services for reading and writing structs in the database.
type DBService struct {
	db *badger.DB
}

// Open opens the database with options and returns a DBService instance.
func Open(options badger.Options) (*DBService, error) {
	log.WithField("dir", options.Dir).Info("Opening database")
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	return &DBService{db: db}, nil
}

// GC deletes expired items and attempts to perform a database cleanup.
func (service *DBService) GC() {
	service.DeleteExpiredItems()
	service.DeleteStaleFetchStatuses()
	service.DeleteStaleReadStatuses()
	for {
		err := service.db.RunValueLogGC(0.5)
		if err == badger.ErrNoRewrite {
			log.WithField("result", err).Debug("Cleanup didn't cause a log file rewrite")
		} else if err != nil {
			log.WithField("result", err).Info("Cleanup completed")
		}
		if err != nil {
			break
		}
		log.Info("Cleanup reclaimed space")
	}
}

// Close closes the underlying database.
func (service *DBService) Close() {
	log.Info("Closing database")
	if service != nil && service.db != nil {
		err := service.db.Close()
		if err != nil {
			log.Fatal(err)
		}
		service.db = nil
	}
}

// IteratorDoNotPrefetchOptions returns Badger iterator options with PrefetchValues = false.
func IteratorDoNotPrefetchOptions() badger.IteratorOptions {
	options := badger.DefaultIteratorOptions
	options.PrefetchValues = false
	return options
}
