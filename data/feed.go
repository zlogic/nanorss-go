package data

import (
	"bytes"
	"encoding/gob"
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
	Key      *FeeditemKey `json:",omitempty"`
}

var itemTTL = 14 * 24 * time.Hour

func (feedItem *Feeditem) Encode() ([]byte, error) {
	key := feedItem.Key
	defer func() { feedItem.Key = key }()
	feedItem.Key = nil

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(feedItem); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

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
		err = gob.NewDecoder(bytes.NewBuffer(value)).Decode(feeditem)
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
	return s.db.Update(func(txn *badger.Txn) error {
		failed := false

		ls, err := NewLastSeen(s, txn)
		if err != nil {
			failed = true
			return err
		}

		getPreviousValue := func(key []byte) ([]byte, error) {
			item, err := txn.Get(key)
			if err != nil && err != badger.ErrKeyNotFound {
				return nil, errors.Wrapf(err, "Failed to get feed item %v", string(key))
			}
			if err == nil {
				value, err := item.Value()
				if err != nil {
					return nil, errors.Wrapf(err, "Failed to read previous value of feed item %v %v", string(key), err)
				}
				return value, nil
			}
			return nil, nil
		}
		getPreviousUpdatedTime := func(previousValue []byte) (*time.Time, error) {
			if previousValue == nil {
				return nil, nil
			}
			existingFeedItem := &Feeditem{}
			err := gob.NewDecoder(bytes.NewBuffer(previousValue)).Decode(existingFeedItem)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to get unmarshal value of feed item")
			}
			return &existingFeedItem.Updated, nil
		}

		for _, feedItem := range feedItems {
			key := feedItem.Key.CreateKey()

			previousValue, err := getPreviousValue(key)
			if err != nil {
				log.Printf("Cannot get previous value for item %v %v", key, err)
			}

			previousTimeUpdated, err := getPreviousUpdatedTime(previousValue)
			if err != nil {
				log.Printf("Failed to read previous updated time %v %v", key, err)
			} else if previousTimeUpdated != nil {
				feedItem.Updated = *previousTimeUpdated
			}

			value, err := feedItem.Encode()
			if err != nil {
				return errors.Wrap(err, "Cannot marshal feed item")
			}

			if err := ls.SetLastSeen(key); err != nil {
				return errors.Wrap(err, "Cannot set last seen time")
			}

			if bytes.Equal(value, previousValue) {
				// Avoid writing to the database if nothing has changed
				return nil
			}

			if err := txn.Set(key, value); err != nil {
				return errors.Wrap(err, "Cannot save feed item")
			}
		}
		if failed {
			return fmt.Errorf("At least one feed failed to save properly")
		}
		return nil
	})
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
			err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(feedItem)
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