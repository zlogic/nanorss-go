package data

import (
	"os"
	"path"

	"github.com/dgraph-io/badger"
	log "github.com/sirupsen/logrus"
)

func DefaultOptions() badger.Options {
	opts := badger.DefaultOptions
	dbPath, ok := os.LookupEnv("DATABASE_DIR")
	if !ok {
		dbPath = path.Join(os.TempDir(), "nanorss")
	}
	opts.Dir = dbPath
	opts.ValueDir = dbPath
	// Optimize options for low memory and disk usage
	opts.MaxTableSize = 1 << 20
	return opts
}

type DBService struct {
	db *badger.DB
}

func Open(options badger.Options) (*DBService, error) {
	log.WithField("dir", options.Dir).Info("Opening database")
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	return &DBService{db: db}, nil
}

func (service *DBService) GC() {
	service.DeleteExpiredItems()
	err := service.db.RunValueLogGC(0.5)
	log.WithField("result", err).Info("Cleaned up database")
}

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
