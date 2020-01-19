package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/akrylysov/pogreb"
	log "github.com/sirupsen/logrus"
)

// PagemonitorPage keeps the state and diff for a web page monitored by Pagemonitor.
type PagemonitorPage struct {
	Contents string
	Delta    string
	Updated  time.Time
	Config   *UserPagemonitor `json:",omitempty"`
}

// Encode serializes a PagemonitorPage.
func (page *PagemonitorPage) Encode() ([]byte, error) {
	config := page.Config
	defer func() { page.Config = config }()
	page.Config = nil

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(page); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

// Decode deserializes a PagemonitorPage.
func (page *PagemonitorPage) Decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(page)
}

// GetPage retrieves a PagemonitorPage for the UserPagemonitor configuration.
// If page doesn't exist, returns nil.
func (s *DBService) GetPage(pm *UserPagemonitor) (*PagemonitorPage, error) {
	page := &PagemonitorPage{Config: pm}
	item, err := s.db.Get(pm.CreateKey())
	if err != nil {
		return nil, fmt.Errorf("Cannot read page %v because of %w", page, err)
	}
	if item == nil {
		return nil, nil
	}

	err = page.Decode(item)
	if err != nil {
		return nil, fmt.Errorf("Cannot decode page %v because of %w", page, err)
	}
	return page, nil
}

// SavePage saves a PagemonitorPage.
func (s *DBService) SavePage(page *PagemonitorPage) error {
	key := page.Config.CreateKey()
	if err := s.SetLastSeen(key); err != nil {
		return fmt.Errorf("Cannot set last seen time because of %w", err)
	}

	getPreviousPage := func(key []byte) (*PagemonitorPage, error) {
		item, err := s.db.Get(key)
		if err != nil {
			return nil, fmt.Errorf("Failed to get previous page %v because of %w", string(key), err)
		}
		if item == nil {
			return nil, nil
		}
		existingPage := &PagemonitorPage{}
		if err := existingPage.Decode(item); err != nil {
			return nil, fmt.Errorf("Failed to read previous value of page %v because of %w", string(key), err)
		}
		return existingPage, nil
	}

	previousPage, err := getPreviousPage(key)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("Failed to read previous page")
	} else if previousPage != nil {
		previousPage.Config = page.Config
	}

	value, err := page.Encode()
	if err != nil {
		return fmt.Errorf("Cannot marshal page because of %w", err)
	}

	if previousPage != nil && *previousPage == *page {
		// Avoid writing to the database if nothing has changed
		return nil
	}

	return s.db.Put(key, value)
}

// ReadAllPages reads all PagemonitorPage items from database and sends them to the provided channel.
func (s *DBService) ReadAllPages(ch chan *PagemonitorPage) error {
	defer close(ch)
	prefix := []byte(PagemonitorKeyPrefix)
	it := s.db.Items()
	for {
		k, v, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read pages because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		pm, err := DecodePagemonitorKey(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to decode key of page")
			continue
		}

		page := &PagemonitorPage{Config: pm}
		if err := page.Decode(v); err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to read value of page")
			continue
		}
		ch <- page
	}
	return nil
}
