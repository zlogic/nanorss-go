package data

import (
	"io/ioutil"
	"os"

	"github.com/dgraph-io/badger"
)

func createDb() (dbService *DBService, cleanupFunc func(), err error) {
	dir, err := ioutil.TempDir("", "nanorss")
	if err != nil {
		return nil, func() {}, err
	}

	var opts = badger.DefaultOptions
	opts.ValueLogFileSize = 1 << 20
	opts.SyncWrites = false
	opts.Dir = dir
	opts.ValueDir = dir

	dbService, err = Open(opts)
	if err != nil {
		return nil, func() {}, err
	}
	return dbService, func() {
		dbService.Close()
		os.RemoveAll(opts.Dir)
	}, nil
}
