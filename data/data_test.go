package data

import (
	"github.com/akrylysov/pogreb"
	"github.com/akrylysov/pogreb/fs"
)

var dbService *DBService

func resetDb() (err error) {
	if dbService != nil {
		it := dbService.db.Items()
		for {
			k, _, err := it.Next()
			if err == pogreb.ErrIterationDone {
				break
			} else if err != nil {
				return err
			}
			err = dbService.db.Delete(k)
			if err != nil {
				return err
			}
		}
		return
	}
	opts := pogreb.Options{FileSystem: fs.Mem}

	dbService, err = Open(opts)
	return
}
