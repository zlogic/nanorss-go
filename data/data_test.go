package data

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v2"
	log "github.com/sirupsen/logrus"
)

var dbService *DBService

func TestMain(m *testing.M) {
	dir, err := ioutil.TempDir("", "nanorss")
	if err != nil {
		panic(fmt.Sprintf("cannot create tempdir %v", err))
	}
	err = createDb(dir)
	if err != nil {
		panic(fmt.Sprintf("cannot open database %v", err))
	}

	code := m.Run()
	destroyDb(dir)
	os.Exit(code)
}

func createDb(dir string) (err error) {
	var opts = badger.DefaultOptions(dir)
	opts.Logger = log.New()
	opts.ValueLogFileSize = 1 << 20
	opts.SyncWrites = false
	opts.CompactL0OnClose = false

	dbService, err = Open(opts)
	return
}

func resetDb() error {
	return dbService.db.DropAll()
}

func destroyDb(dir string) {
	dbService.Close()
	os.RemoveAll(dir)
}
