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
}

type PagemonitorService struct {
	db *badger.DB
}

func (s *DBService) GetPage(pm *UserPagemonitor) (*PagemonitorPage, error) {
	page := &PagemonitorPage{}
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

func (s *DBService) SavePage(pm *UserPagemonitor, page *PagemonitorPage) error {
	key := pm.CreateKey()
	value, err := json.Marshal(page)
	if err != nil {
		return errors.Wrap(err, "Cannot marshal page")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.SetWithTTL(key, value, itemTTL)
	})
}

func (s *DBService) ReadAllPages(handler func(*UserPagemonitor, *PagemonitorPage)) (err error) {
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
			page := &PagemonitorPage{}
			err = json.Unmarshal(v, &page)
			if err != nil {
				log.Printf("Failed to unmarshal value of item %v because of %v", k, err)
				continue
			}
			handler(pm, page)
		}
		return nil
	})
	return err
}
