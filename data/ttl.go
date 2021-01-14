package data

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
	log "github.com/sirupsen/logrus"
)

// itemTTL specifies the TTL after which items expire.
var itemTTL = 14 * 24 * time.Hour

// skipUpdateTTL how old the item has to be for its "last seen" status to be updated.
// Prevents excessive writes to the database.
var skipUpdateTTL = itemTTL / 2

// SetLastSeen creates or updates the last seen value for key.
func (s *DBService) SetLastSeen(key []byte) func(*badger.Txn) error {
	return func(txn *badger.Txn) error {
		currentTime := time.Now()
		lastSeenKey := createLastSeenKey(key)

		previous, err := txn.Get(lastSeenKey)
		if err == nil {
			previousLastSeen := time.Time{}
			if err := previous.Value(previousLastSeen.UnmarshalBinary); err != nil {
				log.WithError(err).Error("Failed to read previous last seen time")
			} else {
				if currentTime.Before(previousLastSeen.Add(skipUpdateTTL)) {
					// Add a safe barrier to previousLastSeen;
					// If current time hasn't reached the half-life of previousLastSeen, skip update
					return nil
				}
			}
		} else if err != badger.ErrKeyNotFound {
			log.WithError(err).Error("Failed to get previous last seen time")
		}

		value, err := currentTime.MarshalBinary()
		if err != nil {
			return fmt.Errorf("error marshaling current time: %w", err)
		}
		if err := txn.Set(lastSeenKey, value); err != nil {
			return fmt.Errorf("error saving last seen time: %w", err)
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

			key = createLastSeenKey(key)
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

			lastSeenKey := createLastSeenKey(k)
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
		err := s.deleteExpiredItems([]byte(feeditemKeyPrefix))(txn)
		if err != nil {
			failed = true
			log.WithError(err).Error("Failed to clean up expired feed items")
		}

		err = s.deleteExpiredItems([]byte(pagemonitorKeyPrefix))(txn)
		if err != nil {
			failed = true
			log.WithError(err).Error("Failed to clean up expired pages")
		}

		if failed {
			return fmt.Errorf("failed to delete at least one expired item")
		}
		return nil
	})
}
