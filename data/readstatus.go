package data

import (
	"github.com/dgraph-io/badger"
	log "github.com/sirupsen/logrus"
)

type itemKey = []byte

// GetReadStatus returns the read status for keys and returns the list of items which are marked as read for user.
func (s DBService) GetReadStatus(user User) ([]itemKey, error) {
	items := make([]itemKey, 0)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := IteratorDoNotPrefetchOptions()
		opts.Prefix = []byte(user.CreateReadStatusPrefix())
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			k := item.Key()

			itemKey, err := DecodeReadStatusKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode item key")
				return err
			}
			items = append(items, itemKey)
		}
		return nil
	})
	if err != nil {
		return []itemKey{}, err
	}
	return items, nil
}

// SetReadStatus sets the read status for item, true for read, false for unread.
func (s DBService) SetReadStatus(user User, k itemKey, read bool) error {
	readStatusKey := user.CreateReadStatusKey(k)
	return s.db.Update(func(txn *badger.Txn) error {
		if read {
			return txn.Set(readStatusKey, nil)
		}
		return txn.Delete(readStatusKey)
	})
}

// SetReadStatusForAll sets the read status for item (for all users), true for read, false for unread.
func (s DBService) SetReadStatusForAll(k itemKey, read bool) error {
	return s.db.Update(func(txn *badger.Txn) error {
		opts := IteratorDoNotPrefetchOptions()
		opts.Prefix = []byte(UserKeyPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			userKey := item.Key()

			username, err := DecodeUserKey(userKey)
			if err != nil {
				log.WithField("key", userKey).WithError(err).Error("Failed to decode username of user")
				continue
			}

			user := User{username: username}
			readStatusKey := user.CreateReadStatusKey(k)

			if read {
				if err := txn.Set(readStatusKey, nil); err != nil {
					log.WithField("key", readStatusKey).WithError(err).Error("Failed to set read status for all users")
					return err
				}
				continue
			}
			if err := txn.Delete(readStatusKey); err != nil {
				log.WithField("key", readStatusKey).WithError(err).Error("Failed to set unread status for all users")
				return err
			}
		}
		return nil
	})
}

// RenameReadStatus updates read status items for user to the new username.
func (s DBService) renameReadStatus(user User) func(*badger.Txn) error {
	newUser := User{username: user.newUsername}
	return func(txn *badger.Txn) error {
		opts := IteratorDoNotPrefetchOptions()
		opts.Prefix = []byte(user.CreateReadStatusPrefix())
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)

			itemKey, err := DecodeReadStatusKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode item key")
				return err
			}

			err = txn.Delete(k)
			if err != nil {
				log.WithField("key", k).WithField("user", user.username).WithError(err).Error("Failed to delete read status from old username")
				return err
			}

			newK := newUser.CreateReadStatusKey(itemKey)
			err = txn.Set(newK, nil)
			if err != nil {
				log.WithField("key", newK).WithField("user", newUser.username).WithError(err).Error("Failed to create read status for new username")
				return err
			}
		}
		return nil
	}
}

// DeleteStaleReadStatuses deletes all read statuses which are referring to items which no longer exist.
func (s DBService) DeleteStaleReadStatuses() error {
	return s.db.Update(func(txn *badger.Txn) error {
		opts := IteratorDoNotPrefetchOptions()
		opts.Prefix = []byte(ReadStatusPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)

			itemKey, err := DecodeReadStatusKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode key of read status")
				continue
			}

			referencedItem, err := txn.Get(itemKey)
			if err == badger.ErrKeyNotFound {
				log.WithField("item", referencedItem).Debug("Deleting invalid read status")
				if err := txn.Delete(k); err != nil {
					log.WithField("key", k).WithError(err).Error("Failed to delete read status")
					continue
				}
			} else if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to get item referenced by read status")
				continue
			}
		}
		return nil
	})
}
