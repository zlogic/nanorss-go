package data

import (
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

// GetReadStatus returns the read status for keys and returns the list of items which are marked as read for user.
func (s *DBService) GetReadStatus(user *User) ([]string, error) {
	res, err := s.client.SMembers(user.CreateReadStatusKey()).Result()
	if err == redis.Nil {
		return []string{}, nil
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

// SetReadStatus sets the read status for item, true for read, false for unread.
func (s DBService) SetReadStatus(user *User, itemKey string, read bool) error {
	if read {
		_, err := s.client.SAdd(user.CreateReadStatusKey(), itemKey).Result()
		return err
	}
	_, err := s.client.SRem(user.CreateReadStatusKey(), itemKey).Result()
	return err
}

// SetReadStatusForAll sets the read status for item (for all users), true for read, false for unread.
func (s *DBService) SetReadStatusForAll(itemKey string, read bool) error {
	ch := make(chan *User)
	done := make(chan error)
	go func() {
		defer close(done)
		for user := range ch {
			err := s.SetReadStatus(user, itemKey, read)

			if err != nil {
				log.WithField("user", user).WithField("item", itemKey).WithError(err).Error("Failed to set read status")
				done <- err
				return
			}
		}
	}()
	err := s.ReadAllUsers(ch)
	if err != nil {
		return err
	}
	err = <-done
	return err
}

// DeleteStaleReadStatuses deletes all read statuses which are referring to items which no longer exist.
func (s DBService) DeleteStaleReadStatuses() {
	ch := make(chan *User)
	done := make(chan bool)
	go func() {
		defer close(done)
		for user := range ch {
			readItems, err := s.GetReadStatus(user)

			if err != nil {
				log.WithField("key", user).WithError(err).Error("Failed to get read status")
				continue
			}

			for _, key := range readItems {
				res, err := s.client.Exists(key).Result()

				if err != nil {
					log.WithField("user", user).WithField("key", key).WithError(err).Error("Failed to check existance of item referenced by read status")
					continue
				}

				if res == 0 {
					err := s.client.SRem(user.CreateReadStatusKey(), key).Err()
					if err != nil {
						log.WithField("user", user).WithField("key", key).WithError(err).Error("Failed to remove stale read status")
						continue
					}
				}
			}
		}
	}()
	s.ReadAllUsers(ch)
	<-done
}
