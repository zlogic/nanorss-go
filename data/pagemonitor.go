package data

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
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
func (page *PagemonitorPage) encode() (map[string]interface{}, error) {
	updated, err := page.Updated.MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "Error marshaling updated time")
	}
	return map[string]interface{}{
		"contents": page.Contents,
		"delta":    page.Delta,
		"updated":  string(updated),
	}, nil
}

// decodePagemonitorPage deserializes a PagemonitorPage.
func decodePagemonitorPage(pm *UserPagemonitor, res map[string]string) (*PagemonitorPage, error) {
	updated := time.Time{}
	err := updated.UnmarshalText([]byte(res["updated"]))
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling updated time")
	}
	return &PagemonitorPage{
		Config:   pm,
		Contents: res["contents"],
		Delta:    res["delta"],
		Updated:  updated,
	}, nil
}

// GetPage retrieves a PagemonitorPage for the UserPagemonitor configuration.
// If page doesn't exist, returns nil.
func (s *DBService) GetPage(pm *UserPagemonitor) (*PagemonitorPage, error) {
	pageMap, err := s.client.HGetAll(pm.CreateKey()).Result()
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot get page %v", pm)
	}

	if len(pageMap) == 0 {
		return nil, nil
	}

	page, err := decodePagemonitorPage(pm, pageMap)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot decode page %v", pm)
	}

	return page, nil
}

// SavePage saves a PagemonitorPage.
func (s *DBService) SavePage(page *PagemonitorPage) error {
	key := page.Config.CreateKey()
	if err := s.SetLastSeen(key); err != nil {
		return errors.Wrap(err, "Cannot set last seen time")
	}

	getPreviousPage := func(key string) (*PagemonitorPage, error) {
		previous, err := s.client.HGetAll(key).Result()
		if err != nil && err != redis.Nil {
			return nil, errors.Wrapf(err, "Failed to get previous page %v", key)
		} else if len(previous) > 0 {
			existingPage, err := decodePagemonitorPage(page.Config, previous)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to decode previous value of page %v %v", key, err)
			}
			return existingPage, nil
		}
		// Page doesn't exist
		return nil, nil
	}

	previousPage, err := getPreviousPage(key)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("Failed to read previous page")
	} else if previousPage != nil {
		previousPage.Config = page.Config
	}

	value, err := page.encode()
	if err != nil {
		return errors.Wrap(err, "Cannot marshal page")
	}

	if previousPage != nil && *previousPage == *page {
		// Avoid writing to the database if nothing has changed
		return nil
	}

	err = s.client.HMSet(key, value).Err()
	return err
}

// ReadAllPages reads all PagemonitorPage items from database and sends them to the provided channel.
func (s *DBService) ReadAllPages(ch chan *PagemonitorPage) error {
	defer close(ch)

	failed := false

	cursor := uint64(0)
	for haveData := true; haveData; {
		var keys []string
		var err error
		keys, cursor, err = s.client.Scan(cursor, PagemonitorKeyPrefix+"*", 100).Result()
		if err != nil {
			log.WithError(err).Error("Failed to get pages")
			failed = true
			continue
		}
		if cursor == 0 {
			haveData = false
		}

		for _, key := range keys {
			pm, err := DecodePagemonitorKey(key)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to decode key of page")
				failed = true
				continue
			}

			value, err := s.client.HGetAll(key).Result()
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed get value of page")
				failed = true
				continue
			}

			page, err := decodePagemonitorPage(pm, value)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to decode value of page")
				failed = true
				continue
			}
			ch <- page
		}
	}
	if failed {
		return fmt.Errorf("Failed to read at least one item")
	}
	return nil
}
