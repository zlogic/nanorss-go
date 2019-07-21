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

// Decode deserializes a Feeditem.
func (feedItem *Feeditem) Decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(feedItem)
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

		if err := item.Value(feeditem.Decode); err != nil {
			feeditem = nil
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read feed item %v", key)
	}
	return feeditem, nil
}

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	return s.db.Update(func(txn *badger.Txn) error {
		getPreviousItem := func(key []byte) (*Feeditem, error) {
			item, err := txn.Get(key)
			if err != nil && err != badger.ErrKeyNotFound {
				return nil, errors.Wrapf(err, "Failed to get feed item %v", string(key))
			}
			if err == nil {
				existingFeedItem := &Feeditem{}
				if err := item.Value(existingFeedItem.Decode); err != nil {
					return nil, errors.Wrapf(err, "Failed to read previous value of feed item %v %v", string(key), err)
				}
				return existingFeedItem, nil
			}
			// Item doesn't exist
			return nil, nil
		}

		for _, feedItem := range feedItems {
			key := feedItem.Key.CreateKey()

			previousItem, err := getPreviousItem(key)
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
				continue
			} else if previousItem != nil {
				log.WithField("previousItem", previousItem).WithField("feedItem", feedItem).Debug("Item has changed")
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
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(FeeditemKeyPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			k := item.Key()
			key, err := DecodeFeeditemKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode key of item")
				continue
			}

			feedItem := &Feeditem{Key: key}
			if err := item.Value(feedItem.Decode); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to read value of item")
				continue
			}
			ch <- feedItem
		}
		return nil
	})
	return
}
