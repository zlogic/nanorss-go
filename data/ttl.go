package data

import (
	"fmt"
	"time"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

var itemTTL = 14 * 24 * time.Hour

var skipUpdateTTL = itemTTL / 2

// SetLastSeen creates or updates the last seen value for key.
func (s *DBService) SetLastSeen(key []byte) error {
	currentTime := time.Now()
	lastSeenKey := CreateLastSeenKey(key)

	previous, err := s.db.Get(lastSeenKey)
	if err != nil {
		log.WithError(err).Error("Failed to get previous last seen time")
	} else if previous == nil {
		log.Debug("Previous seen time doesn't exist")
	} else if err == nil {
		previousLastSeen := time.Time{}
		if err := previousLastSeen.UnmarshalBinary(previous); err != nil {
			log.WithError(err).Error("Failed to read previous last seen time")
		} else {
			if currentTime.Before(previousLastSeen.Add(skipUpdateTTL)) {
				// Add a safe barrier to previousLastSeen;
				// If current time hasn't reached the half-life of previousLastSeen, skip update
				return nil
			}
		}
	}

	value, err := currentTime.MarshalBinary()
	if err != nil {
		return fmt.Errorf("Error marshaling current time (%w)", err)
	}
	if err := s.db.Put(lastSeenKey, value); err != nil {
		return fmt.Errorf("Error saving last seen time (%w)", err)
	}
	return nil
}

func (s *DBService) deleteExpiredItems(prefix []byte) error {
	now := time.Now()

	purgeItem := func(key []byte) {
		if err := s.db.Delete(key); err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to delete item")
		}

		key = CreateLastSeenKey(key)
		if err := s.db.Delete(key); err != nil {
			log.WithField("key", string(key)).WithError(err).Error("Failed to delete item last seen time")
		}
	}

	it := s.db.Items()
	for {
		k, _, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read expired items because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		lastSeenKey := CreateLastSeenKey(k)
		lastSeenValue, err := s.db.Get(lastSeenKey)

		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get last seen time item")
			purgeItem(k)
			continue
		}

		lastSeen := time.Time{}
		err = lastSeen.UnmarshalBinary(lastSeenValue)
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

// DeleteExpiredItems deletes all items which SetLastSeen was not called at least itemTTL.
func (s *DBService) DeleteExpiredItems() error {
	failed := false
	err := s.deleteExpiredItems([]byte(FeeditemKeyPrefix))
	if err != nil {
		failed = true
		log.WithError(err).Error("Failed to clean up expired feed items")
	}

	err = s.deleteExpiredItems([]byte(PagemonitorKeyPrefix))
	if err != nil {
		failed = true
		log.WithError(err).Error("Failed to clean up expired pages")
	}

	if failed {
		return fmt.Errorf("Failed to delete at least one expired item")
	}
	return nil
}
