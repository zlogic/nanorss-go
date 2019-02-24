package data

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/dgraph-io/badger"
)

func DefaultOptions() badger.Options {
	opts := badger.DefaultOptions
	dbPath, ok := os.LookupEnv("DATABASE_DIR")
	if !ok {
		dbPath = path.Join(os.TempDir(), "nanorss")
	}
	opts.Dir = dbPath
	opts.ValueDir = dbPath
	valueLogFileSize, ok := os.LookupEnv("DATABASE_VALUE_LOG_FILE_SIZE")
	if ok {
		valueLogFileSizeInt, err := strconv.ParseInt(valueLogFileSize, 0, 64)
		if err != nil {
			fmt.Printf("Cannot parse DATABASE_VALUE_LOG_FILE_SIZE %v %v", valueLogFileSize, err)
		}
		opts.ValueLogFileSize = valueLogFileSizeInt
	}
	//opts.Truncate = true
	opts.NumVersionsToKeep = 1
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

func (service *DBService) Close() {
	log.Println("Closing database")
	if service != nil && service.db != nil {
		service.db.RunValueLogGC(0.5)
		err := service.db.Close()
		if err != nil {
			log.Fatal(err)
		}
		service.db = nil
	}
}
