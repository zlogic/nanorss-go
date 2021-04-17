package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

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

	if err := s.addReferencedKey([]byte(fetchStatusKeyPrefix), key); err != nil {
		return fmt.Errorf("failed to add fetch status to index: %w", err)
	}

	if err := s.db.Put(k, value.Bytes()); err != nil {
		return fmt.Errorf("failed to save fetch status: %w", err)
	}
	return nil
}
