package data

import (
	golog "log"
	"os"
	"path"

	"github.com/akrylysov/pogreb"
	"github.com/akrylysov/pogreb/fs"
	log "github.com/sirupsen/logrus"
	"github.com/zlogic/rst"
)

func init() {
	pogrebLog := golog.New(log.New().Writer(), "", 0)
	pogreb.SetLogger(pogrebLog)
}

// DefaultOptions returns default options for the database, customized based on environment variables.
func DefaultOptions() pogreb.Options {
	return pogreb.Options{
		FileSystem: fs.OS,
	}
}

// DBService provides services for reading and writing structs in the database.
type DBService struct {
	db *pogreb.DB
	tx *rst.KeyLocker
}

// Open opens the database with options and returns a DBService instance.
func Open(options pogreb.Options) (*DBService, error) {
	dbPath, ok := os.LookupEnv("DATABASE_DIR")
	if !ok {
		dbPath = path.Join(os.TempDir(), "nanorss")
	}
	log.WithField("dir", dbPath).Info("Opening database")
	db, err := pogreb.Open(dbPath, &options)
	if err != nil {
		return nil, err
	}
	s := DBService{db: db, tx: rst.New()}
	err = s.CompleteTransactions()
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GC deletes expired items and attempts to perform a database cleanup.
func (service *DBService) GC() {
	service.DeleteExpiredItems()
	service.DeleteStaleFetchStatuses()
	service.DeleteStaleReadStatuses()
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
