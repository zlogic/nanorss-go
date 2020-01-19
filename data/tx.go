package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

// TxOperation is the TxEntry operation type
type TxOperation string

// TxOperationPut is a Put operation.
const TxOperationPut = "put"

// TxOperationDelete is a Delete operation.
const TxOperationDelete = "del"

// retryInterval is the delay between retrying failed transactions.
const retryInterval = time.Second * 10

// TxEntry is an operation in a Tx transaction log.
type TxEntry struct {
	Op    TxOperation
	Key   []byte
	Value []byte
}

// Tx is a transaction in the transaction log.
type Tx struct {
	Items []*TxEntry
}

// addOp adds a TxEntry to Tx.
func (tx *Tx) addOp(key []byte, value []byte, op TxOperation) {
	tx.Items = append(tx.Items, &TxEntry{
		Key:   key,
		Value: value,
		Op:    op,
	})
}

// Put adds a Put operation to Tx.
func (tx *Tx) Put(key []byte, value []byte) {
	tx.addOp(key, value, TxOperationPut)
}

// Delete adds a Delete operation to Tx.
func (tx *Tx) Delete(key []byte) {
	tx.addOp(key, nil, TxOperationDelete)
}

// Apply writes Tx to the transaction log, then applies the transaction, then deletes Tx from the transaction log.
func (tx *Tx) Apply(db *pogreb.DB, txName []byte) error {
	txKey := CreateTxKey(txName)
	txExists, err := db.Has(txKey)
	if err != nil {
		return err
	}
	if txExists {
		return fmt.Errorf("Transaction already exists")
	}

	val, err := tx.Encode()
	if err != nil {
		return fmt.Errorf("Cannor encode transaction because of %w", err)
	}
	err = db.Put(txKey, val)
	if err != nil {
		return fmt.Errorf("Cannor write transaction to log because of %w", err)
	}

	for _, item := range tx.Items {
		switch item.Op {
		case TxOperationPut:
			err := db.Put(item.Key, item.Value)
			if err != nil {
				return err
			}
		case TxOperationDelete:
			err := db.Delete(item.Key)
			if err != nil {
				return err
			}
		}
	}

	return db.Delete(txKey)
}

// Encode serializes a Tx.
func (tx *Tx) Encode() ([]byte, error) {
	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(tx); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

// Decode deserializes a Tx.
func (tx *Tx) Decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(tx)
}

// CompleteTransactions attempts to complete transactions that were not completed before the last shutdown.
func (s *DBService) CompleteTransactions() error {
	prefix := []byte(TxKeyPrefix)
	it := s.db.Items()
	for {
		k, v, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read previous transactions statuses because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		txName, err := DecodeTxKey(k)
		if err != nil {
			return fmt.Errorf("Cannot decode previous transaction key because of %w", err)
		}

		tx := Tx{}
		err = tx.Decode(v)
		if err != nil {
			return fmt.Errorf("Cannot decode previous transaction because of %w", err)
		}

		err = tx.Apply(s.db, txName)
		if err != nil {
			return fmt.Errorf("Cannot apply previous transaction because of %w", err)
		}

		err = s.db.Delete(k)
		if err != nil {
			return fmt.Errorf("Cannot delete completed previous transaction because of %w", err)
		}
	}
	return nil
}

func txName(lockKeys [][]byte) []byte {
	if len(lockKeys) == 0 {
		return []byte{}
	}
	size := len(lockKeys) - 1
	for i := range lockKeys {
		size += len(lockKeys[i])
	}

	txName := make([]byte, 0, size)
	for i := range lockKeys {
		txName = append(txName, lockKeys[i]...)
		if i < len(lockKeys)-1 {
			txName = append(txName, []byte{0}...)
		}
	}
	return txName
}

// InTransaction locks lockKeys, runs fn and then performs a cleanup.
func (s *DBService) InTransaction(fn func(*Tx) error, lockKeys ...[]byte) error {
	s.tx.Lock(lockKeys...)
	tx := &Tx{}

	// Collect actions to run in transaction
	err := fn(tx)
	if err != nil {
		// Safe to unlock - no operations done
		s.tx.Unlock(lockKeys...)
		return err
	}

	txName := txName(lockKeys)
	for {
		err := tx.Apply(s.db, txName)
		if err == nil {
			break
		}
		// Write failed - need to retry
		log.WithError(err).Error("Failed to run transaction, retrying...")
		time.Sleep(retryInterval)
	}
	// Safe to unlock - operation was successfully completed
	s.tx.Unlock(lockKeys...)
	return nil
}
