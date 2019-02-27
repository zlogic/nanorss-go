package data

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

type FeeditemKey struct {
	FeedURL string
	GUID    string
}

type Feeditem struct {
	Title    string
	URL      string
	Date     time.Time
	Contents string
	Updated  time.Time
	Key      *FeeditemKey `json:"-"`
}

var itemTTL = 14 * 24 * time.Hour

func (s *DBService) GetFeeditem(key *FeeditemKey) (*Feeditem, error) {
	feeditem := &Feeditem{Key: key}
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key.CreateKey())
		if err == badger.ErrKeyNotFound {
			feeditem = nil
			return nil
		}

		value, err := item.Value()
		if err != nil {
			return err
		}
		err = json.Unmarshal(value, feeditem)
		if err != nil {
			feeditem = nil
		}
		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read feed item %v", key)
	}
	return feeditem, nil
}

func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	err = s.db.Update(func(txn *badger.Txn) error {
		failed := false

		getPreviousUpdatedTime := func(key []byte) (*time.Time, error) {
			item, err := txn.Get(key)
			if err != nil && err != badger.ErrKeyNotFound {
				return nil, errors.Wrapf(err, "Failed to get feed item %v", string(key))
			}
			if err == nil {
				value, err := item.Value()
				if err != nil {
					return nil, errors.Wrapf(err, "Failed to get read value of feed item %v", string(key))
				}
				existingFeedItem := Feeditem{}
				err = json.Unmarshal(value, &existingFeedItem)
				if err != nil {
					return nil, errors.Wrapf(err, "Failed to get unmarshal value of feed item %v", string(key))
				}
				return &existingFeedItem.Updated, nil
			}
			return nil, nil
		}

		for _, feedItem := range feedItems {
			key := feedItem.Key.CreateKey()

			previousTimeUpdated, err := getPreviousUpdatedTime(key)
			if err != nil {
				log.Printf("Failed to read previous updated time %v", err)
			} else if previousTimeUpdated != nil {
				feedItem.Updated = *previousTimeUpdated
			}

			value, err := json.Marshal(feedItem)
			if err != nil {
				return errors.Wrap(err, "Cannot marshal feed item")
			}

			err = txn.SetWithTTL(key, value, itemTTL)
			if err != nil {
				return errors.Wrap(err, "Cannot save feed item")
			}
		}
		if failed {
			return fmt.Errorf("At least one feed failed to save properly")
		}
		return nil
	})
	return err
}

func (s *DBService) ReadAllFeedItems(ch chan *Feeditem) (err error) {
	defer close(ch)
	err = s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(FeeditemKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			k := item.Key()
			key, err := DecodeFeeditemKey(k)
			if err != nil {
				log.Printf("Failed to decode key of item %v because of %v", k, err)
				continue
			}

			v, err := item.Value()
			if err != nil {
				log.Printf("Failed to read value of item %v because of %v", k, err)
				continue
			}
			feedItem := &Feeditem{Key: key}
			err = json.Unmarshal(v, &feedItem)
			if err != nil {
				log.Printf("Failed to unmarshal value of item %v because of %v", k, err)
				continue
			}
			ch <- feedItem
		}
		return nil
	})
	return
}
