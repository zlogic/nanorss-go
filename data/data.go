package data

import (
	"log"
	"os"
	"path"

	"github.com/dgraph-io/badger"
)

func DefaultOptions() (opts badger.Options) {
	opts = badger.DefaultOptions
	if opts.Dir == "" || opts.ValueDir == "" {
		dbPath, ok := os.LookupEnv("DATABASE_DIR")
		if !ok {
			dbPath = path.Join(os.TempDir(), "nanorss")
		}
		opts.Dir = dbPath
		opts.ValueDir = dbPath
	}
	return
}

type Service struct {
	db          *badger.DB
	userService *UserService
}

func Open(options badger.Options) (*Service, error) {
	log.Print("Opening database in ", options.Dir)
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:          db,
		userService: newUserService(db),
	}, nil
}

func (service *Service) Close() {
	if service.db != nil {
		err := service.db.Close()
		if err != nil {
			log.Fatal(err)
		}
		service.db = nil
	}
}
