package data

import (
	"fmt"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

type itemKey = []byte

// GetReadStatus returns the read status for keys and returns the list of items which are marked as read for user.
func (s *DBService) GetReadStatus(user *User) ([]itemKey, error) {
	items := make([]itemKey, 0)
	prefix := []byte(user.CreateReadStatusPrefix())
	it := s.db.Items()
	for {
		k, _, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return nil, fmt.Errorf("Cannot read read statuses because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		itemKey, err := DecodeReadStatusKey(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to decode item key")
			return nil, err
		}
		items = append(items, itemKey)
	}
	return items, nil
}

// SetReadStatus sets the read status for item, true for read, false for unread.
func (s *DBService) SetReadStatus(user *User, k itemKey, read bool) error {
	readStatusKey := user.CreateReadStatusKey(k)
	userKey := CreateUserKey(user.username)
	return s.InTransaction(func(tx *Tx) error {
		if read {
			return s.db.Put(readStatusKey, []byte{})
		}
		return s.db.Delete(readStatusKey)
	}, userKey)
}

// SetReadStatusForAll sets the read status for item (for all users), true for read, false for unread.
func (s *DBService) SetReadStatusForAll(k itemKey, read bool) error {
	prefix := []byte(UserKeyPrefix)
	it := s.db.Items()

	for {
		userKey, _, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read users because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, userKey) {
			continue
		}

		username, err := DecodeUserKey(userKey)
		if err != nil {
			log.WithField("key", userKey).WithError(err).Error("Failed to decode username of user")
			continue
		}

		user := &User{username: *username}
		readStatusKey := user.CreateReadStatusKey(k)

		s.InTransaction(func(tx *Tx) error {
			if read {
				if err := s.db.Put(readStatusKey, []byte{}); err != nil {
					log.WithField("key", readStatusKey).WithError(err).Error("Failed to set read status for all users")
					return err
				}
				return nil
			}
			if err := s.db.Delete(readStatusKey); err != nil {
				log.WithField("key", readStatusKey).WithError(err).Error("Failed to set unread status for all users")
				return err
			}
			return nil
		}, userKey)
	}
	return nil
}

// renameReadStatus updates read status items for user to the new username.
func (s *DBService) renameReadStatus(user *User, tx *Tx) error {
	newUser := &User{username: user.newUsername}
	prefix := []byte(user.CreateReadStatusPrefix())
	it := s.db.Items()

	for {
		k, _, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read read statuses because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		itemKey, err := DecodeReadStatusKey(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to decode item key")
			return err
		}

		tx.Delete(k)

		newK := newUser.CreateReadStatusKey(itemKey)
		tx.Put(newK, []byte{})
	}
	return nil
}

// DeleteStaleReadStatuses deletes all read statuses which are referring to items which no longer exist.
func (s *DBService) DeleteStaleReadStatuses() error {
	prefix := []byte(ReadStatusPrefix)
	it := s.db.Items()
	for {
		k, _, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read read statuses because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		itemKey, err := DecodeReadStatusKey(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to decode key of read status")
			continue
		}

		referencedItem, err := s.db.Get(itemKey)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get item referenced by read status")
			continue
		}
		if referencedItem == nil {
			log.WithField("item", referencedItem).Debug("Deleting invalid read status")
			if err := s.db.Delete(k); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to delete read status")
				continue
			}
		}
	}
	return nil
}
