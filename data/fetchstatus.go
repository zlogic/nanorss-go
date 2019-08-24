package data

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// FetchStatus keeps track of successful and failed fetches.
type FetchStatus struct {
	LastSuccess time.Time
	LastFailure time.Time
}

// Decode deserializes a FetchStatus.
func (fetchStatus *FetchStatus) Decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(fetchStatus)
}

func getFetchStatus(k []byte) func(*badger.Txn) (FetchStatus, error) {
	return func(txn *badger.Txn) (FetchStatus, error) {
		item, err := txn.Get(k)
		if err == badger.ErrKeyNotFound {
			return FetchStatus{}, nil
		}
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to read fetch status")
			return FetchStatus{}, err
		}

		fetchStatus := FetchStatus{}
		if err := item.Value(fetchStatus.Decode); err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to read value of fetch status")
			return FetchStatus{}, err
		}

		return fetchStatus, nil
	}
}

// GetFetchStatus returns the fetch status for key, or nil if the fetch status is unknown.
func (s DBService) GetFetchStatus(key []byte) (fetchStatus FetchStatus, err error) {
	fetchStatus = FetchStatus{}
	k := CreateFetchStatusKey(key)

	err = s.db.View(func(txn *badger.Txn) error {
		var err error
		fetchStatus, err = getFetchStatus(k)(txn)
		return err
	})
	if err != nil {
		return FetchStatus{}, err
	}
	return
}

// SetFetchStatus creates or updates the fetch status for key.
func (s DBService) SetFetchStatus(key []byte, fetchStatus FetchStatus) error {
	k := CreateFetchStatusKey(key)

	return s.db.Update(func(txn *badger.Txn) error {
		previousFetchStatus, err := getFetchStatus(k)(txn)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to read existing fetch status")
			return err
		}

		newFetchStatus := FetchStatus{}
		if newFetchStatus != previousFetchStatus {
			newFetchStatus = previousFetchStatus
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
			return errors.Wrap(err, "Error encoding fetch status")
		}

		if err := txn.Set(k, value.Bytes()); err != nil {
			return errors.Wrap(err, "Error saving fetch status")
		}
		return nil
	})
}

// DeleteStaleFetchStatuses deletes all FetchStatus items which were not updated for itemTTL.
func (s DBService) DeleteStaleFetchStatuses() error {
	return s.db.Update(func(txn *badger.Txn) error {
		now := time.Now()

		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(FetchStatusKeyPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)

			fetchStatus := FetchStatus{}
			if err := item.Value(fetchStatus.Decode); err != nil {
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
				if err := txn.Delete(k); err != nil {
					log.WithField("key", k).WithError(err).Error("Failed to delete fetch status")
				}
			}
		}
		return nil
	})
}
