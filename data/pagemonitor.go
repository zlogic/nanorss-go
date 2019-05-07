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

		value, err := item.Value()
		if err != nil {
			return err
		}
		err = gob.NewDecoder(bytes.NewBuffer(value)).Decode(&page)
		if err != nil {
			page = nil
		}
		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read page %v", page)
	}
	return page, nil
}

// SavePage saves a PagemonitorPage.
func (s *DBService) SavePage(page *PagemonitorPage) error {
	key := page.Config.CreateKey()
	value, err := page.Encode()
	if err != nil {
		return errors.Wrap(err, "Cannot marshal page")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		if err := s.SetLastSeen(key)(txn); err != nil {
			return errors.Wrap(err, "Cannot set last seen time")
		}

		getPreviousValue := func() ([]byte, error) {
			item, err := txn.Get(key)
			if err != nil && err != badger.ErrKeyNotFound {
				return nil, errors.Wrapf(err, "Failed to get page %v", string(key))
			} else if err == nil {
				value, err := item.Value()
				if err != nil {
					return nil, errors.Wrapf(err, "Failed to read previous value of page %v %v", string(key), err)
				}
				return value, nil
			}
			return nil, nil
		}

		previousValue, err := getPreviousValue()
		if err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to read previous value of page")
		}
		if bytes.Equal(value, previousValue) {
			// Avoid writing to the database if nothing has changed
			return nil
		}
		return txn.SetWithDiscard(key, value, 0)
	})
}

// ReadAllPages reads all PagemonitorPage items from database and sends them to the provided channel.
func (s *DBService) ReadAllPages(ch chan *PagemonitorPage) (err error) {
	defer close(ch)
	err = s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(PagemonitorKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			k := item.Key()
			pm, err := DecodePagemonitorKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode key of item")
				continue
			}

			v, err := item.Value()
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to read value of item")
				continue
			}
			page := &PagemonitorPage{Config: pm}
			err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(page)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to unmarshal value of item")
				continue
			}
			ch <- page
		}
		return nil
	})
	return err
}
