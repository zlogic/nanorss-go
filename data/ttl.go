package data

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type LastSeen struct {
	dbService *DBService
	txn       *badger.Txn
	lastSeen  []byte
}

func NewLastSeen(dbService *DBService, txn *badger.Txn) (*LastSeen, error) {
	timeValue, err := time.Now().MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "Error marshaling current time")
	}
	return &LastSeen{
		dbService: dbService,
		txn:       txn,
		lastSeen:  timeValue,
	}, nil
}

func (ls *LastSeen) SetLastSeen(key []byte) error {
	lastSeenKey := CreateLastSeenKey(key)
	if err := ls.txn.Set(lastSeenKey, ls.lastSeen); err != nil {
		return errors.Wrap(err, "Error saving last seen time")
	}
	return nil
}

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
