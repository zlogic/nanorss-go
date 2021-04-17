package data

import (
	log "github.com/sirupsen/logrus"
)

type itemKey = []byte

// GetReadStatus returns true if itemKey is read, otherwise returns false.
func (s *DBService) GetReadStatus(user *User, key itemKey) (bool, error) {
	k := user.createReadStatusKey(key)

	var readStatus = false
	err := s.view(func() error {
		value, err := s.db.Get(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get read status key")
			return err
		}
		readStatus = value != nil
		return nil
	})

	return readStatus, err
}

// setReadStatus sets the read status for item, true for read, false for unread.
func (s *DBService) setReadStatus(user *User, k itemKey, read bool) error {
	readStatusPrefix := user.createReadStatusPrefix()
	readStatusKey := user.createReadStatusKey(k)
	if read {
		if err := s.addReferencedKey([]byte(readStatusPrefix), k); err != nil {
			return err
		}
		return s.db.Put(readStatusKey, []byte{})
	}
	if err := s.db.Delete(readStatusKey); err != nil {
		return err
	}
	return s.deleteReferencedKey([]byte(readStatusPrefix), k)
}

// SetReadStatus sets the read status for item, true for read, false for unread.
func (s *DBService) SetReadStatus(user *User, k itemKey, read bool) error {
	return s.view(func() error {
		return s.setReadStatus(user, k, read)
	})
}

// SetReadStatusForAll sets the read status for item (for all users), true for read, false for unread.
func (s *DBService) SetReadStatusForAll(k itemKey, read bool) error {
	return s.view(func() error {
		indexKeys, err := s.getReferencedKeys([]byte(userKeyPrefix))
		if err != nil {
			log.WithError(err).Error("Failed to decode list of usernames")
			return err
		}
		for i := range indexKeys {
			userKey := indexKeys[i]
			username := string(userKey)

			user := &User{username: username}

			if err := s.setReadStatus(user, k, read); err != nil {
				log.WithField("key", string(k)).WithField("username", username).WithError(err).Error("Failed to set read status for user")
			}
		}
		return nil
	})
}

// renameReadStatus updates read status items for user to the new username.
func (s *DBService) renameReadStatus(user *User) error {
	oldReadStatusIndexKey := []byte(user.createReadStatusPrefix())
	newUser := &User{username: user.newUsername}
	newReadStatusIndexKey := []byte(newUser.createReadStatusPrefix())

	readItemsIndex, err := s.getReferencedKeys(oldReadStatusIndexKey)
	if err != nil {
		log.WithField("username", user.username).WithError(err).Error("Failed to get read status index")
		return err
	}

	for i := range readItemsIndex {
		k := readItemsIndex[i]

		if err := s.deleteReferencedKey(oldReadStatusIndexKey, k); err != nil {
			log.WithField("key", k).WithField("user", user.username).WithError(err).Error("Failed to delete read status from old username index")
			return err
		}

		itemKey := user.createReadStatusKey(k)
		if err := s.db.Delete(itemKey); err != nil {
			log.WithField("key", k).WithField("user", user.username).WithError(err).Error("Failed to delete read status from old username")
			return err
		}

		if err := s.addReferencedKey(newReadStatusIndexKey, k); err != nil {
			log.WithField("key", k).WithField("user", newUser.username).WithError(err).Error("Failed to create read status to index for new username")
			return err
		}

		newItemKey := newUser.createReadStatusKey(k)
		if err := s.db.Put(newItemKey, []byte{}); err != nil {
			log.WithField("key", k).WithField("user", newUser.username).WithError(err).Error("Failed to create read status for new username")
			return err
		}
	}
	return nil
}

// deleteStaleReadStatuses deletes all read statuses which are referring to items which no longer exist.
func (s *DBService) deleteStaleReadStatuses() error {
	return s.view(func() error {
		userIndexKeys, err := s.getReferencedKeys([]byte(userKeyPrefix))
		if err != nil {
			log.WithError(err).Error("Failed to decode list of usernames")
			return err
		}
		for i := range userIndexKeys {
			username := string(userIndexKeys[i])
			user := User{username: username}

			readItemsIndex, err := s.getReferencedKeys([]byte(user.createReadStatusPrefix()))
			if err != nil {
				log.WithField("username", username).WithError(err).Error("Failed to get read status index")
				continue
			}

			for j := range readItemsIndex {
				k := readItemsIndex[j]

				exists, err := s.db.Has(k)
				if err != nil {
					log.WithField("key", k).WithError(err).Error("Failed to get item referenced by read status")
					continue
				}
				if !exists {
					log.Debug("Deleting invalid read status")

					if err := s.setReadStatus(&user, k, false); err != nil {
						log.WithField("key", string(k)).WithError(err).Error("Failed to delete read status")
						continue
					}
				}
			}
		}
		return nil
	})
}
