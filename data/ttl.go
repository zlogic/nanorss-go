package data

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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

// DeleteExpiredItems deletes all items which SetLastSeen was not called at least itemTTL.
func (s *DBService) DeleteExpiredItems() error {
	now := time.Now()

	failed := false
	return s.db.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(LastSeenKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			k := item.Key()
			itemKey := DecodeLastSeenKey(k)

			v, err := item.Value()
			if err != nil {
				failed = true
				log.WithField("key", k).WithError(err).Error("Failed to get last seen time")
				continue
			}

			lastSeen := time.Time{}
			err = lastSeen.UnmarshalBinary(v)
			if err != nil {
				failed = true
				log.WithField("value", v).WithError(err).Error("Failed to unmarshal time")
				continue
			}

			expires := lastSeen.Add(itemTTL)
			if expires.After(now) {
				continue
			}

			err = txn.Delete(itemKey)
			if err != nil {
				failed = true
				log.WithField("key", itemKey).WithError(err).Error("Failed to delete expired item")
			}

			err = txn.Delete(k)
			if err != nil {
				failed = true
				log.WithField("key", k).WithError(err).Error("Failed to delete item expiration time")
			}
			return nil
		}
		if failed {
			return fmt.Errorf("Failed to delete at least one expired item")
		}
		return nil
	})
}
