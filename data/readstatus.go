package data

import (
	"strings"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

type itemKey = []byte

// GetReadStatus returns true if itemKey is read, otherwise returns false.
func (s *DBService) GetReadStatus(user *User, key itemKey) (bool, error) {
	s.userLock.RLock()
	defer s.userLock.RUnlock()

	k := user.createReadStatusKey(key)
	value, err := s.db.Get(k)
	if err != nil {
		log.WithField("key", k).WithError(err).Error("Failed to get read status key")
		return false, err
	}

	readStatus := value != nil
	return readStatus, nil
}

// SetReadStatus sets the read status for item, true for read, false for unread.
func (s *DBService) SetReadStatus(user *User, k itemKey, read bool) error {
	readStatusKey := user.createReadStatusKey(k)
	if read {
		return s.db.Put(readStatusKey, []byte{})
	}
	return s.db.Delete(readStatusKey)
}

// SetReadStatusForAll sets the read status for item (for all users), true for read, false for unread.
func (s *DBService) SetReadStatusForAll(k itemKey, read bool) error {
	s.userLock.RLock()
	defer s.userLock.RUnlock()

	it := s.db.Items()
	for {
		// TODO: use an index here.
		userKey, _, err := it.Next()
		if err == pogreb.ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		if !isUserKey(userKey) {
			continue
		}

		username, err := decodeUserKey(userKey)
		if err != nil {
			log.WithField("key", userKey).WithError(err).Error("Failed to decode username of user")
			continue
		}

		user := &User{username: *username}
		readStatusKey := user.createReadStatusKey(k)

		if read {
			if err := s.db.Put(readStatusKey, []byte{}); err != nil {
				log.WithField("key", readStatusKey).WithError(err).Error("Failed to set read status for all users")
				return err
			}
			continue
		}
		if err := s.db.Delete(readStatusKey); err != nil {
			log.WithField("key", readStatusKey).WithError(err).Error("Failed to set unread status for all users")
			return err
		}
	}
	return nil
}

// renameReadStatus updates read status items for user to the new username.
func (s *DBService) renameReadStatus(user *User) error {
	newUser := &User{username: user.newUsername}
	prefix := user.createReadStatusPrefix()
	it := s.db.Items()
	for {
		// TODO: use an index here.
		k, _, err := it.Next()
		if err == pogreb.ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		if !strings.HasPrefix(string(k), prefix) {
			continue
		}

		itemKey, err := decodeReadStatusKey(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to decode item key")
			return err
		}

		err = s.db.Delete(k)
		if err != nil {
			log.WithField("key", k).WithField("user", user.username).WithError(err).Error("Failed to delete read status from old username")
			return err
		}

		newK := newUser.createReadStatusKey(itemKey)
		err = s.db.Put(newK, []byte{})
		if err != nil {
			log.WithField("key", newK).WithField("user", newUser.username).WithError(err).Error("Failed to create read status for new username")
			return err
		}
	}
	return nil
}

// DeleteStaleReadStatuses deletes all read statuses which are referring to items which no longer exist.
func (s *DBService) DeleteStaleReadStatuses() error {
	s.userLock.RLock()
	defer s.userLock.RUnlock()

	it := s.db.Items()
	for {
		// TODO: use an index here.
		k, _, err := it.Next()
		if err == pogreb.ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		if !isReadStatusKey(k) {
			continue
		}

		itemKey, err := decodeReadStatusKey(k)
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
