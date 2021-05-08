package data

import (
	"fmt"
	"time"

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

	if err := s.addReferencedKey([]byte(lastSeenKeyPrefix), key); err != nil {
		return fmt.Errorf("error adding last seen time to index: %w", err)
	}

	if err := s.db.Put(lastSeenKey, value); err != nil {
		return fmt.Errorf("error saving last seen time: %w", err)
	}
	return nil
}

// deleteExpiredItems will delete all feedItems for feedKey that have expired.
func (s *DBService) deleteExpiredItems(feedKey []byte) error {
	prefix := append(feedKey, []byte(separator)...)

	now := time.Now()

	purgeItem := func(key []byte) {
		contentsKey := append(key, []byte(feedContentsSuffix)...)
		if err := s.db.Delete(contentsKey); err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to delete item contents")
		}

		if err := s.db.Delete(key); err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to delete item")
		}

		lastSeenKey := createLastSeenKey(key)
		if err := s.db.Delete(lastSeenKey); err != nil {
			log.WithField("key", string(lastSeenKey)).WithError(err).Error("Failed to delete item last seen time")
		}

		if err := s.deleteReferencedKey(prefix, key); err != nil {
			log.WithField("key", string(key)).WithError(err).Error("Failed to delete item from index")
		}
	}

	indexKeys, err := s.getReferencedKeys(prefix)
	if err != nil {
		return fmt.Errorf("cannot get index for feed key %v: %w", string(feedKey), err)
	}
	for _, k := range indexKeys {
		itemGUID := encodePart(string(k))

		itemKey := append(prefix, []byte(itemGUID)...)
		lastSeenKey := createLastSeenKey(itemKey)
		lastSeenValue, err := s.db.Get(lastSeenKey)
		if err != nil {
			log.WithField("key", string(itemKey)).WithError(err).Error("Failed to get last seen time item")
			purgeItem(itemKey)
			continue
		}

		lastSeen := time.Time{}
		if err := lastSeen.UnmarshalBinary(lastSeenValue); err != nil {
			log.WithField("key", string(itemKey)).WithError(err).Error("Failed to get last seen time value")
			purgeItem(itemKey)
			continue
		}

		expires := lastSeen.Add(itemTTL)
		if now.After(expires) || now.Equal(expires) {
			log.Debug("Deleting expired item")
			purgeItem(itemKey)
		}
	}
	return nil
}

// deleteStaleFetchStatuses deletes all FetchStatus items which were not updated for itemTTL.
func (s *DBService) deleteStaleFetchStatuses() error {
	now := time.Now()

	indexKeys, err := s.getReferencedKeys([]byte(fetchStatusKeyPrefix))
	if err != nil {
		return err
	}

	for _, k := range indexKeys {
		fetchStatusKey := createFetchStatusKey(k)
		value, err := s.db.Get(fetchStatusKey)
		if err != nil {
			return err
		}

		fetchStatus := &FetchStatus{}
		if err := fetchStatus.decode(value); err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to get fetch status value")
			continue
		}

		var lastUpdated time.Time
		if fetchStatus.LastFailure.After(lastUpdated) {
			lastUpdated = fetchStatus.LastFailure
		}
		if fetchStatus.LastSuccess.After(lastUpdated) {
			lastUpdated = fetchStatus.LastSuccess
		}

		expires := lastUpdated.Add(itemTTL)
		if now.After(expires) {
			log.Debug("Deleting expired fetch status")

			if IsFeeditemKey(k) {
				if err := s.deleteExpiredItems(k); err != nil {
					log.WithField("key", k).WithError(err).Error("Failed to remove expired feed items")
					continue
				}
			}

			if err := s.db.Delete(k); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to delete fetch status item")
				continue
			}

			if err := s.db.Delete(fetchStatusKey); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to delete fetch status key")
				continue
			}

			if err := s.deleteReferencedKey([]byte(fetchStatusKeyPrefix), k); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to remove fetch status from index")
				continue
			}
		}
	}
	return nil
}
