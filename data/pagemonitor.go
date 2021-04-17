package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// PagemonitorPage keeps the state and diff for a web page monitored by Pagemonitor.
type PagemonitorPage struct {
	Contents string
	Delta    string
	Updated  time.Time
	Config   *UserPagemonitor `json:",omitempty"`
}

// encode serializes a PagemonitorPage.
func (page *PagemonitorPage) encode() ([]byte, error) {
	config := page.Config
	defer func() { page.Config = config }()
	page.Config = nil

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(page); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

// decode deserializes a PagemonitorPage.
func (page *PagemonitorPage) decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(page)
}

// GetPage retrieves a PagemonitorPage for the UserPagemonitor configuration.
// If page doesn't exist, returns nil.
func (s *DBService) GetPage(pm *UserPagemonitor) (*PagemonitorPage, error) {
	page := &PagemonitorPage{Config: pm}
	value, err := s.db.Get(pm.CreateKey())

	if err != nil {
		return nil, fmt.Errorf("cannot get page %v: %w", pm, err)
	}
	if value == nil {
		return nil, nil
	}

	if err := page.decode(value); err != nil {
		return nil, fmt.Errorf("cannot decode page %v: %w", pm, err)
	}

	return page, nil
}

// SavePage saves a PagemonitorPage.
func (s *DBService) SavePage(page *PagemonitorPage) error {
	key := page.Config.CreateKey()

	getPreviousPage := func(key []byte) (*PagemonitorPage, error) {
		value, err := s.db.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous page %v: %w", string(key), err)
		}
		if value == nil {
			// Item doesn't exist.
			return nil, nil
		}
		existingPage := &PagemonitorPage{}
		if err := existingPage.decode(value); err != nil {
			return nil, fmt.Errorf("failed to read previous value of page %v: %w", string(key), err)
		}
		return existingPage, nil
	}

	previousPage, err := getPreviousPage(key)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("Failed to read previous page")
	} else if previousPage != nil {
		previousPage.Config = page.Config
	}

	value, err := page.encode()
	if err != nil {
		return fmt.Errorf("cannot marshal page: %w", err)
	}

	if previousPage != nil && *previousPage == *page {
		// Avoid writing to the database if nothing has changed
		return nil
	}

	return s.db.Put(key, value)
}

// GetPages returns all PagemonitorPage items ffor user.
func (s *DBService) GetPages(user *User) ([]*PagemonitorPage, error) {
	userPages, err := user.GetPages()
	if err != nil {
		return nil, err
	}

	pages := make([]*PagemonitorPage, 0)
	for i := range userPages {
		pm := userPages[i]
		value, err := s.db.Get(pm.CreateKey())
		if err != nil {
			log.WithField("key", pm).WithError(err).Error("Failed to read value of page")
			continue
		}

		if value == nil {
			continue
		}

		page := &PagemonitorPage{Config: &pm}
		if err := page.decode(value); err != nil {
			log.WithField("key", pm).WithError(err).Error("Failed to decode value of page")
			continue
		}
		pages = append(pages, page)
	}
	return pages, nil
}
