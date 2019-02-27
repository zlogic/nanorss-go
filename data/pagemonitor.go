package data

import (
	"encoding/json"
	"log"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

type PagemonitorPage struct {
	Contents string
	Delta    string
	Updated  time.Time
	Config   *UserPagemonitor `json:"-"`
}

type PagemonitorService struct {
	db *badger.DB
}

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
		err = json.Unmarshal(value, page)
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

func (s *DBService) SavePage(page *PagemonitorPage) error {
	key := page.Config.CreateKey()
	value, err := json.Marshal(page)
	if err != nil {
		return errors.Wrap(err, "Cannot marshal page")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		ls, err := NewLastSeen(s, txn)
		if err != nil {
			return err
		}
		if err := ls.SetLastSeen(key); err != nil {
			return errors.Wrap(err, "Cannot set last seen time")
		}
		return txn.Set(key, value)
	})
}

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
				log.Printf("Failed to decode key of item %v because of %v", k, err)
				continue
			}

			v, err := item.Value()
			if err != nil {
				log.Printf("Failed to read value of item %v because of %v", k, err)
				continue
			}
			page := &PagemonitorPage{Config: pm}
			err = json.Unmarshal(v, &page)
			if err != nil {
				log.Printf("Failed to unmarshal value of item %v because of %v", k, err)
				continue
			}
			ch <- page
		}
		return nil
	})
	return err
}
