package data

import (
	"encoding/json"
	"log"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

type PagemonitorPage struct {
	Contents string
	Delta    string
	Error    string
}

type PagemonitorService struct {
	db *badger.DB
}

func (s *DBService) SavePages(user *User) (err error) {
	pages, err := user.GetPages()
	if err != nil {
		return errors.Wrap(err, "Cannot parse pagemonitor configuration")
	}
	if err != nil {
		return err
	}
	err = s.db.Update(func(txn *badger.Txn) error {
		for _, page := range pages {
			//Upsert
			key := page.CreateKey()
			_, err := txn.Get(key)
			if err == badger.ErrKeyNotFound {
				value, err := json.Marshal(page)
				if err != nil {
					return errors.Wrap(err, "Cannot marshal page")
				}
				err = txn.Set(key, value)
				if err != nil {
					return errors.Wrap(err, "Cannot save page")
				}
			} else if err != nil {
				return errors.Wrap(err, "Cannot read page")
			}
		}
		return nil
	})
	return err
}

func (s *DBService) SavePage(pm *UserPagemonitor, page *PagemonitorPage) error {
	key := pm.CreateKey()
	value, err := json.Marshal(page)
	if err != nil {
		return errors.Wrap(err, "Cannot marshal page")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *DBService) DeletePage(pm *UserPagemonitor) error {
	key := pm.CreateKey()
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
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
