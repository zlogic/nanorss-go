package data

import (
	"log"
	"os"
	"path"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
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
	opts.ValueLogFileSize = badger.LSMOnlyOptions.ValueLogFileSize
	opts.ValueLogLoadingMode = options.FileIO
	opts.MaxTableSize = 1 << 24
	//opts.Truncate = true
	return opts
}

type DBService struct {
	db *badger.DB
}

func Open(options badger.Options) (*DBService, error) {
	log.Print("Opening database in ", options.Dir)
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	return &DBService{db: db}, nil
}

func (service *DBService) GC() {
	service.db.RunValueLogGC(0.5)
}

func (service *DBService) Close() {
	log.Println("Closing database")
	if service != nil && service.db != nil {
		err := service.db.Close()
		if err != nil {
			log.Fatal(err)
		}
		service.db = nil
	}
}
