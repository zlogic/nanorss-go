package data

import (
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var itemTTL = 14 * 24 * time.Hour

var skipUpdateTTL = itemTTL / 2

// SetLastSeen creates or updates the last seen value for key.
func (s *DBService) SetLastSeen(key string) error {
	currentTime := time.Now()
	lastSeenKey := CreateLastSeenKey(key)

	previous, err := s.client.Get(lastSeenKey).Result()
	if err == nil {
		previousLastSeen := time.Time{}
		if err := previousLastSeen.UnmarshalText([]byte(previous)); err != nil {
			log.WithError(err).Error("Failed to read previous last seen time")
		} else {
			if currentTime.Before(previousLastSeen.Add(skipUpdateTTL)) {
				// Add a safe barrier to previousLastSeen;
				// If current time hasn't reached the half-life of previousLastSeen, skip update
				return nil
			}
		}
	} else if err != redis.Nil {
		log.WithError(err).Error("Failed to get previous last seen time")
	}

	value, err := currentTime.MarshalText()
	if err != nil {
		return errors.Wrap(err, "Error marshaling current time")
	}
	err = s.client.Set(lastSeenKey, value, 0).Err()
	if err != nil {
		return errors.Wrap(err, "Error saving last seen time")
	}
	return nil
}

func (s *DBService) deleteExpiredItems(prefix string) {
	expired := time.Now().Add(-itemTTL)

	purgeItem := func(key string) {
		lastSeenKey := CreateLastSeenKey(key)
		_, err := s.client.Del(lastSeenKey).Result()
		if err != nil {
			log.WithField("key", string(key)).WithError(err).Error("Failed to delete item")
		}
		_, err = s.client.Del(key).Result()
		if err != nil {
			log.WithField("key", string(key)).WithError(err).Error("Failed to delete item last seen time")
		}
	}

	cursor := uint64(0)
	for haveData := true; haveData; {
		var keys []string
		var err error
		keys, cursor, err = s.client.Scan(cursor, prefix+"*", 100).Result()
		if err != nil {
			log.WithField("prefix", prefix).WithError(err).Error("Failed to get expired keys")
		}
		if cursor == 0 {
			haveData = false
		}

		for _, key := range keys {
			lastSeenStr, err := s.client.Get(CreateLastSeenKey(key)).Result()
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to get last seen time item")
				purgeItem(key)
				continue
			}

			lastSeen := time.Time{}
			err = lastSeen.UnmarshalText([]byte(lastSeenStr))
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to unmarshal last seen time value")
				purgeItem(key)
				continue
			}

			if lastSeen.Before(expired) || lastSeen.Equal(expired) {
				log.Debug("Deleting expired item")
				purgeItem(key)
			}
		}
	}
}

// DeleteExpiredItems deletes all items which SetLastSeen was not called at least itemTTL.
func (s *DBService) DeleteExpiredItems() {
	s.deleteExpiredItems(FeeditemKeyPrefix)
	s.deleteExpiredItems(PagemonitorKeyPrefix)
}
