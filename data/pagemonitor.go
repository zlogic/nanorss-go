package data

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// PagemonitorPage keeps the state and diff for a web page monitored by Pagemonitor.
type PagemonitorPage struct {
	Contents string
	Delta    string
	Updated  time.Time
	Config   *UserPagemonitor `json:",omitempty"`
}

// Encode serializes a PagemonitorPage.
func (page *PagemonitorPage) Encode() ([]byte, error) {
	config := page.Config
	defer func() { page.Config = config }()
	page.Config = nil

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(page); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

// Decode deserializes a PagemonitorPage.
func (page *PagemonitorPage) Decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(page)
}

// GetPage retrieves a PagemonitorPage for the UserPagemonitor configuration.
// If page doesn't exist, returns nil.
func (s *DBService) GetPage(pm *UserPagemonitor) (*PagemonitorPage, error) {
	page := &PagemonitorPage{Config: pm}
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(pm.CreateKey())
		if err == badger.ErrKeyNotFound {
			page = nil
			return nil
		}

		return item.Value(page.Decode)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read page %v", page)
	}
	return page, nil
}

// SavePage saves a PagemonitorPage.
func (s *DBService) SavePage(page *PagemonitorPage) error {
	key := page.Config.CreateKey()
	return s.db.Update(func(txn *badger.Txn) error {
		if err := s.SetLastSeen(key)(txn); err != nil {
			return errors.Wrap(err, "Cannot set last seen time")
		}

		getPreviousPage := func(key []byte) (*PagemonitorPage, error) {
			item, err := txn.Get(key)
			if err != nil && err != badger.ErrKeyNotFound {
				return nil, errors.Wrapf(err, "Failed to get previous page %v", string(key))
			}
			if err == nil {
				existingPage := &PagemonitorPage{}
				if err := item.Value(existingPage.Decode); err != nil {
					return nil, errors.Wrapf(err, "Failed to read previous value of page %v %v", string(key), err)
				}
				return existingPage, nil
			}
			// Page doesn't exist
			return nil, nil
		}

		previousPage, err := getPreviousPage(key)
		if err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to read previous page")
		} else if previousPage != nil {
			previousPage.Config = page.Config
		}

		value, err := page.Encode()
		if err != nil {
			return errors.Wrap(err, "Cannot marshal page")
		}

		if err := s.SetLastSeen(key)(txn); err != nil {
			return errors.Wrap(err, "Cannot set last seen time")
		}

		if previousPage != nil && *previousPage == *page {
			// Avoid writing to the database if nothing has changed
			return nil
		}

		return txn.Set(key, value)
	})
}

// ReadAllPages reads all PagemonitorPage items from database and sends them to the provided channel.
func (s *DBService) ReadAllPages(ch chan *PagemonitorPage) (err error) {
	defer close(ch)
	err = s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(PagemonitorKeyPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			k := item.Key()
			pm, err := DecodePagemonitorKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode key of page")
				continue
			}

			page := &PagemonitorPage{Config: pm}
			if err := item.Value(page.Decode); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to read value of page")
				continue
			}
			ch <- page
		}
		return nil
	})
	return err
}
