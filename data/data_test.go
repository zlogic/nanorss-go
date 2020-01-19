package data

import (
	"os"

	"github.com/akrylysov/pogreb"
	"github.com/akrylysov/pogreb/fs"
)

var dbService *DBService

func resetDb() (err error) {
	if dbService != nil {
		dbService.Close()
		err = fs.Mem.Remove("nanorss")
		err = fs.Mem.Remove("nanorss.index")
		err = fs.Mem.Remove("nanorss.lock")
	}
	os.Setenv("DATABASE_DIR", "nanorss")
	dbService, err = Open(pogreb.Options{
		FileSystem: fs.Mem,
	})
	return
}
