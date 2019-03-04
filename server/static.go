package server

import (
	"net/http"
	"os"
)

type staticResourceFileSystem struct {
	base http.FileSystem
}

func (fs staticResourceFileSystem) Open(name string) (http.File, error) {
	f, err := fs.base.Open(name)

	if err != nil {
		return nil, err
	}

	if d, err := f.Stat(); err == nil {
		if d.IsDir() {
			defer f.Close()
			return nil, os.ErrNotExist
		}
	}

	return f, nil
}
