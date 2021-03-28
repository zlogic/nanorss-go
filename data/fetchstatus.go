package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

// FetchStatus keeps track of successful and failed fetches.
type FetchStatus struct {
	LastSuccess time.Time
	LastFailure time.Time
}

// decode deserializes a FetchStatus.
func (fetchStatus *FetchStatus) decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(fetchStatus)
}

// getFetchStatus returns a function to get the fetch status for a key.
func (s *DBService) getFetchStatus(k []byte) (*FetchStatus, error) {
	value, err := s.db.Get(k)
	if err != nil {
		log.WithField("key", k).WithError(err).Error("Failed to read fetch status")
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	fetchStatus := &FetchStatus{}
	if err := fetchStatus.decode(value); err != nil {
		log.WithField("key", k).WithError(err).Error("Failed to read value of fetch status")
		return nil, err
	}

	return fetchStatus, nil
}

// GetFetchStatus returns the fetch status for key, or nil if the fetch status is unknown.
func (s *DBService) GetFetchStatus(key []byte) (fetchStatus *FetchStatus, err error) {
	k := createFetchStatusKey(key)

	return s.getFetchStatus(k)
}

// SetFetchStatus creates or updates the fetch status for key.
func (s *DBService) SetFetchStatus(key []byte, fetchStatus *FetchStatus) error {
	k := createFetchStatusKey(key)

	previousFetchStatus, err := s.getFetchStatus(k)
	if err != nil {
		log.WithField("key", k).WithError(err).Error("Failed to read existing fetch status")
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

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(newFetchStatus); err != nil {
		return fmt.Errorf("failed to encode fetch status: %w", err)
	}

	if err := s.db.Put(k, value.Bytes()); err != nil {
		return fmt.Errorf("failed to save fetch status: %w", err)
	}
	return nil
}

// DeleteStaleFetchStatuses deletes all FetchStatus items which were not updated for itemTTL.
func (s *DBService) DeleteStaleFetchStatuses() error {
	now := time.Now()

	it := s.db.Items()
	for {
		// TODO: use an index here.
		k, value, err := it.Next()
		if err == pogreb.ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		if !isFetchStatusKey(k) {
			continue
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
			if err := s.db.Delete(k); err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to delete fetch status")
			}
		}
	}
	return nil
}
