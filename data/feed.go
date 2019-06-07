package data

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// FeeditemKey is used to uniquely identify a Feeditem.
type FeeditemKey struct {
	FeedURL string
	GUID    string
}

// Feeditem keeps an item from an RSS feed.
type Feeditem struct {
	Title    string
	URL      string
	Date     time.Time
	Contents string
	Updated  time.Time
	Key      *FeeditemKey `json:",omitempty"`
}

// Encode serializes a Feeditem.
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

// GetFeeditem retrieves a Feeditem for the FeeditemKey.
// If item doesn't exist, returns nil.
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

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	return s.db.Update(func(txn *badger.Txn) error {
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
		getPreviousItem := func(previousValue []byte) (*Feeditem, error) {
			if previousValue == nil {
				return nil, nil
			}
			existingFeedItem := &Feeditem{}
			err := gob.NewDecoder(bytes.NewBuffer(previousValue)).Decode(existingFeedItem)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to get unmarshal value of feed item")
			}
			return existingFeedItem, nil
		}

		for _, feedItem := range feedItems {
			key := feedItem.Key.CreateKey()

			previousValue, err := getPreviousValue(key)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Cannot get previous value for item")
			}

			previousItem, err := getPreviousItem(previousValue)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to read previous updated time")
			} else if previousItem != nil {
				feedItem.Date = feedItem.Date.In(previousItem.Date.Location())
				previousItem.Updated = feedItem.Updated
				previousItem.Key = feedItem.Key
			}

			value, err := feedItem.Encode()
			if err != nil {
				return errors.Wrap(err, "Cannot marshal feed item")
			}

			if err := s.SetLastSeen(key)(txn); err != nil {
				return errors.Wrap(err, "Cannot set last seen time")
			}

			if previousItem != nil && *feedItem == *previousItem {
				// Avoid writing to the database if nothing has changed
				return nil
			} else if previousItem != nil {
				log.WithField("previousItem", previousItem).WithField("feedItem", feedItem).Info("Item has changed")
			}

			if err := txn.Set(key, value); err != nil {
				return errors.Wrap(err, "Cannot save feed item")
			}
		}
		return nil
	})
}

// ReadAllFeedItems reads all Feeditem items from database and sends them to the provided channel.
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
				log.WithField("key", k).WithError(err).Error("Failed to decode key of item")
				continue
			}

			v, err := item.Value()
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to read value of item")
				continue
			}
			feedItem := &Feeditem{Key: key}
			err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(feedItem)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to unmarshal value of item")
				continue
			}
			ch <- feedItem
		}
		return nil
	})
	return
}
