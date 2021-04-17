package data

import (
	golog "log"
	"os"
	"path"
	"sync"

	"github.com/akrylysov/pogreb"
	"github.com/akrylysov/pogreb/fs"
	log "github.com/sirupsen/logrus"
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

	userLock sync.RWMutex
}

// Open opens the database with options and returns a DBService instance.
func Open(options pogreb.Options) (*DBService, error) {
	dbPath, ok := os.LookupEnv("DATABASE_DIR")
	if !ok {
		dbPath = path.Join(os.TempDir(), "nanorss")
	}
	log.WithField("dir", dbPath).WithField("inmemory", options.FileSystem == fs.Mem).Info("Opening database")
	db, err := pogreb.Open(dbPath, &options)
	if err != nil {
		return nil, err
	}
	return &DBService{db: db}, nil
}

// GC deletes expired items and attempts to perform a database cleanup.
func (service *DBService) GC() {
	service.deleteStaleFetchStatuses()
	service.deleteStaleReadStatuses()

	result, err := service.db.Compact()
	if err != nil {
		log.WithError(err).Error("Cleanup failed")
	}
	if result.CompactedSegments != 0 {
		log.WithField("ReclaimedBytes", result.ReclaimedBytes).
			WithField("ReclaimedRecords", result.ReclaimedRecords).
			WithField("CompactedSegments", result.CompactedSegments).
			Info("Cleanup reclaimed space")
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

// view will acquire a read lock on the database and execute txn.
// Returns the error returned by txn.
func (service *DBService) view(txn func() error) error {
	service.userLock.RLock()
	defer service.userLock.RUnlock()

	return txn()
}

// update will acquire a write lock on the database and execute txn.
// Returns the error returned by txn.
func (service *DBService) update(txn func() error) error {
	service.userLock.Lock()
	defer service.userLock.Unlock()

	return txn()
}
