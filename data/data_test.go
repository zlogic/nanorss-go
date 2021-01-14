package data

import (
	"github.com/dgraph-io/badger/v3"
	log "github.com/sirupsen/logrus"
)

var dbService *DBService

func resetDb() (err error) {
	var opts = badger.DefaultOptions("")
	opts.Logger = log.New()
	opts.ValueLogFileSize = 1 << 20
	opts.InMemory = true

	dbService, err = Open(opts)
	return
}
