package data

import (
	"fmt"
	"strings"
	"time"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

// itemTTL specifies the TTL after which items expire.
var itemTTL = 14 * 24 * time.Hour

// skipUpdateTTL how old the item has to be for its "last seen" status to be updated.
// Prevents excessive writes to the database.
var skipUpdateTTL = itemTTL / 2

// SetLastSeen creates or updates the last seen value for key.
func (s *DBService) SetLastSeen(key []byte) error {
	currentTime := time.Now()
	lastSeenKey := createLastSeenKey(key)

	previous, err := s.db.Get(lastSeenKey)
	if err != nil {
		log.WithError(err).Error("Failed to get previous last seen time")
	}
	if previous != nil {
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
		return fmt.Errorf("error marshaling current time: %w", err)
	}
	if err := s.db.Put(lastSeenKey, value); err != nil {
		return fmt.Errorf("error saving last seen time: %w", err)
	}
	return nil
}

func (s *DBService) deleteExpiredItems(prefix []byte) error {
	now := time.Now()

	purgeItem := func(key []byte) {
		if err := s.db.Delete(key); err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to delete item")
		}

		key = createLastSeenKey(key)
		if err := s.db.Delete(key); err != nil {
			log.WithField("key", string(key)).WithError(err).Error("Failed to delete item last seen time")
		}
	}

	it := s.db.Items()
	for {
		// TODO: use an index here.
		k, _, err := it.Next()
		if err == pogreb.ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		if !strings.HasPrefix(string(k), string(prefix)) {
			continue
		}

		lastSeenKey := createLastSeenKey(k)
		lastSeenValue, err := s.db.Get(lastSeenKey)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get last seen time item")
			purgeItem(k)
			continue
		}

		lastSeen := time.Time{}
		if err := lastSeen.UnmarshalBinary(lastSeenValue); err != nil {
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
	err := s.deleteExpiredItems([]byte(feeditemKeyPrefix))
	if err != nil {
		failed = true
		log.WithError(err).Error("Failed to clean up expired feed items")
	}

	err = s.deleteExpiredItems([]byte(pagemonitorKeyPrefix))
	if err != nil {
		failed = true
		log.WithError(err).Error("Failed to clean up expired pages")
	}

	if failed {
		return fmt.Errorf("failed to delete at least one expired item")
	}
	return nil
}
