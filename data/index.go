package data

import (
	"bytes"
	"encoding/gob"
)

// getReferencedKeys will return a list of keys referenced by an index key.
func (service *DBService) getReferencedKeys(prefix []byte) ([][]byte, error) {
	index, err := service.db.Get(prefix)
	if err != nil {
		return nil, err
	}
	if len(index) == 0 {
		return [][]byte{}, nil
	}

	indexKeys := make([][]byte, 0)
	if err := gob.NewDecoder(bytes.NewBuffer(index)).Decode(&indexKeys); err != nil {
		return nil, err
	}
	return indexKeys, nil
}

// addReferencedKey will add key to the prefix index.
// If lessFn is provided, this function will be used to sort values.
func (service *DBService) addReferencedKey(prefix, key []byte) error {
	index, err := service.db.Get(prefix)
	if err != nil {
		return err
	}

	indexKeys := [][]byte{}
	if len(index) > 0 {
		if err := gob.NewDecoder(bytes.NewBuffer(index)).Decode(&indexKeys); err != nil {
			return err
		}
	}

	// Check if key already exists in index.
	for i := range indexKeys {
		if bytes.Equal(key, indexKeys[i]) {
			return nil
		}
	}
	indexKeys = append(indexKeys, key)

	var updatedValue bytes.Buffer
	if err := gob.NewEncoder(&updatedValue).Encode(indexKeys); err != nil {
		return err
	}
	return service.db.Put(prefix, updatedValue.Bytes())
}

// deleteReferencedKey will remove key from the prefix index.
func (service *DBService) deleteReferencedKey(prefix, key []byte) error {
	index, err := service.db.Get(prefix)
	if err != nil {
		return err
	}

	indexKeys := [][]byte{}
	if len(index) > 0 {
		if err := gob.NewDecoder(bytes.NewBuffer(index)).Decode(&indexKeys); err != nil {
			return err
		}
	}

	updatedIndex := make([][]byte, 0, len(index))
	for i := range indexKeys {
		indexKey := indexKeys[i]
		if bytes.Equal(key, indexKey) {
			continue
		}
		updatedIndex = append(updatedIndex, indexKey)
	}

	var updatedValue bytes.Buffer
	if err := gob.NewEncoder(&updatedValue).Encode(updatedIndex); err != nil {
		return err
	}
	return service.db.Put(prefix, updatedValue.Bytes())
}
