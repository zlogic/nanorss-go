package data

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var itemTTL = 14 * 24 * time.Hour

// SetLastSeen creates or updates the last seen value for key.
func (s *DBService) SetLastSeen(key []byte) func(*badger.Txn) error {
	return func(txn *badger.Txn) error {
		lastSeen, err := time.Now().MarshalBinary()
		if err != nil {
			return errors.Wrap(err, "Error marshaling current time")
		}
		lastSeenKey := CreateLastSeenKey(key)
		if err := txn.Set(lastSeenKey, lastSeen); err != nil {
			return errors.Wrap(err, "Error saving last seen time")
		}
		return nil
	}
}

func (s *DBService) deleteExpiredItems(prefix []byte) func(*badger.Txn) error {
	return func(txn *badger.Txn) error {
		now := time.Now()

		purgeItem := func(key []byte) {
			if err := txn.Delete(key); err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to delete item")
			}

			key = CreateLastSeenKey(key)
			if err := txn.Delete(key); err != nil {
				log.WithField("key", string(key)).WithError(err).Error("Failed to delete item last seen time")
			}
		}

		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)

			lastSeenKey := CreateLastSeenKey(k)
			lastSeenItem, err := txn.Get(lastSeenKey)

			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to get last seen time item")
				purgeItem(k)
				continue
			}

			lastSeen := time.Time{}
			err = lastSeenItem.Value(func(val []byte) error {
				return lastSeen.UnmarshalBinary(val)
			})
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to get last seen time value")
				purgeItem(k)
				continue
			}

			expires := lastSeen.Add(itemTTL)
			if now.After(expires) || now.Equal(expires) {
				log.Debug("Deleting expired item")
				purgeItem(k)
			}
		}
		return nil
	}
}

// DeleteExpiredItems deletes all items which SetLastSeen was not called at least itemTTL.
func (s *DBService) DeleteExpiredItems() error {
	failed := false
	return s.db.Update(func(txn *badger.Txn) error {
		err := s.deleteExpiredItems([]byte(FeeditemKeyPrefix))(txn)
		if err != nil {
			failed = true
			log.WithError(err).Error("Failed to clean up expired feed items")
		}

		err = s.deleteExpiredItems([]byte(PagemonitorKeyPrefix))(txn)
		if err != nil {
			failed = true
			log.WithError(err).Error("Failed to clean up expired pages")
		}

		if failed {
			return fmt.Errorf("Failed to delete at least one expired item")
		}
		return nil
	})
}
