package data

import (
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// FetchStatus keeps track of successful and failed fetches.
type FetchStatus struct {
	LastSuccess time.Time
	LastFailure time.Time
}

// encode serializes a FetchStatus.
func (fetchStatus *FetchStatus) encode() (map[string]interface{}, error) {
	lastSuccess, err := fetchStatus.LastSuccess.MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "Error marshaling last success time")
	}
	lastFailure, err := fetchStatus.LastFailure.MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "Error marshaling last failure time")
	}
	return map[string]interface{}{
		"lastSuccess": string(lastSuccess),
		"lastFailure": string(lastFailure),
	}, nil
}

// decodeFetchStatus deserializes a FetchStatus.
func decodeFetchStatus(res map[string]string) (*FetchStatus, error) {
	lastSuccess := time.Time{}
	err := lastSuccess.UnmarshalText([]byte(res["lastSuccess"]))
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling last success time")
	}
	lastFailure := time.Time{}
	err = lastFailure.UnmarshalText([]byte(res["lastFailure"]))
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling last failure time")
	}
	return &FetchStatus{
		LastSuccess: lastSuccess,
		LastFailure: lastFailure,
	}, nil
}

func (s *DBService) getFetchStatus(k string) (*FetchStatus, error) {
	fetchStatusMap, err := s.client.HGetAll(k).Result()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read fetch status %v", k)
	}

	if len(fetchStatusMap) == 0 {
		return nil, nil
	}

	fetchStatus, err := decodeFetchStatus(fetchStatusMap)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to decode fetch status %v", k)
	}
	return fetchStatus, nil
}

// GetFetchStatus returns the fetch status for key, or nil if the fetch status is unknown.
func (s *DBService) GetFetchStatus(key string) (*FetchStatus, error) {
	k := CreateFetchStatusKey(key)

	fetchStatus, err := s.getFetchStatus(k)
	if err != nil {
		return nil, err
	}
	return fetchStatus, nil
}

// SetFetchStatus creates or updates the fetch status for key.
func (s *DBService) SetFetchStatus(key string, fetchStatus *FetchStatus) error {
	key = CreateFetchStatusKey(key)

	previousFetchStatus, err := s.getFetchStatus(key)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("Failed to read existing fetch status")
		return err
	}

	newFetchStatus := FetchStatus{}
	if previousFetchStatus != nil {
		newFetchStatus = *previousFetchStatus
	}

	var emptyTime time.Time
	if fetchStatus.LastSuccess != emptyTime {
		newFetchStatus.LastSuccess = fetchStatus.LastSuccess
	}
	if fetchStatus.LastFailure != emptyTime {
		newFetchStatus.LastFailure = fetchStatus.LastFailure
	}

	value, err := newFetchStatus.encode()
	if err != nil {
		return errors.Wrap(err, "Cannot marshal fetch status")
	}

	_, err = s.client.HMSet(key, value).Result()
	if err != nil {
		return errors.Wrap(err, "Error saving fetch status")
	}
	return nil
}

// DeleteStaleFetchStatuses deletes all FetchStatus items which were not updated for itemTTL.
func (s *DBService) DeleteStaleFetchStatuses() {
	now := time.Now()

	cursor := uint64(0)
	for haveData := true; haveData; {
		var keys []string
		var err error
		keys, cursor, err = s.client.Scan(cursor, FetchStatusKeyPrefix+"*", 100).Result()
		if err != nil {
			log.WithError(err).Error("Failed to get fetch statuses")
		}
		if cursor == 0 {
			haveData = false
		}

		for _, key := range keys {
			value, err := s.client.HGetAll(key).Result()
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed get value of fetch status")
				continue
			}

			fetchStatus, err := decodeFetchStatus(value)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to decode value of fetch status")
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
				_, err := s.client.Del(key).Result()
				if err != nil {
					log.WithField("key", key).WithError(err).Error("Failed to delete fetch status")
				}
			}
		}
	}
}
