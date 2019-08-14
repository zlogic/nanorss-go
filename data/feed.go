package data

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// FeeditemKey is used to uniquely identify a Feeditem.
type FeeditemKey struct {
	FeedURL string
	GUID    string
}

// Feeditem keeps an item from an RSS feed.
type Feeditem struct {
	Title    string
	URL      string
	Date     time.Time
	Contents string
	Updated  time.Time
	Key      *FeeditemKey `json:",omitempty"`
}

// encode serializes a Feeditem.
func (feedItem Feeditem) encode() (map[string]interface{}, error) {
	updated, err := feedItem.Updated.MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "Error marshaling updated time")
	}
	date, err := feedItem.Date.MarshalText()
	if err != nil {
		return nil, errors.Wrap(err, "Error marshaling item date")
	}
	return map[string]interface{}{
		"title":    feedItem.Title,
		"url":      feedItem.URL,
		"date":     string(date),
		"contents": feedItem.Contents,
		"updated":  string(updated),
	}, nil
}

// decodeFeeditem deserializes a Feeditem.
func decodeFeeditem(key *FeeditemKey, res map[string]string) (*Feeditem, error) {
	updated := time.Time{}
	err := updated.UnmarshalText([]byte(res["updated"]))
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling updated time")
	}
	date := time.Time{}
	err = date.UnmarshalText([]byte(res["date"]))
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling item date")
	}
	return &Feeditem{
		Title:    res["title"],
		URL:      res["url"],
		Date:     date,
		Contents: res["contents"],
		Updated:  updated,
		Key:      key,
	}, nil
}

// GetFeeditem retrieves a Feeditem for the FeeditemKey.
// If item doesn't exist, returns nil.
func (s *DBService) GetFeeditem(key *FeeditemKey) (*Feeditem, error) {
	feeditemMap, err := s.client.HGetAll(key.CreateKey()).Result()
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot get feed item %v", key)
	}

	if len(feeditemMap) == 0 {
		return nil, nil
	}

	feeditem, err := decodeFeeditem(key, feeditemMap)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot decode feed item %v", key)
	}

	return feeditem, nil
}

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	getPreviousItem := func(key string) (*Feeditem, error) {
		previous, err := s.client.HGetAll(key).Result()
		if err != nil && err != redis.Nil {
			return nil, errors.Wrapf(err, "Failed to get previous feed item %v", key)
		} else if len(previous) > 0 {
			existingFeeditem, err := decodeFeeditem(nil, previous)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to decode previous value of feed item %v %v", key, err)
			}
			return existingFeeditem, nil
		}
		// Page doesn't exist
		return nil, nil
	}

	for _, feedItem := range feedItems {
		key := feedItem.Key.CreateKey()

		previousItem, err := getPreviousItem(key)
		if err != nil {
			log.WithField("key", key).WithError(err).Error("Failed to read previous item")
		} else if previousItem != nil {
			feedItem.Date = feedItem.Date.In(previousItem.Date.Location())
			previousItem.Updated = feedItem.Updated
			previousItem.Key = feedItem.Key
		}

		value, err := feedItem.encode()
		if err != nil {
			return errors.Wrap(err, "Cannot marshal feed item")
		}

		if err := s.SetLastSeen(key); err != nil {
			return errors.Wrap(err, "Cannot set last seen time")
		}

		if previousItem != nil && *feedItem == *previousItem {
			// Avoid writing to the database if nothing has changed
			continue
		} else if previousItem != nil {
			log.WithField("previousItem", previousItem).WithField("feedItem", feedItem).Debug("Item has changed")
		}

		err = s.client.HMSet(key, value).Err()
		if err != nil {
			return errors.Wrap(err, "Cannot save feed item")
		}
	}
	return nil
}

// ReadAllFeedItems reads all Feeditem items from database and sends them to the provided channel.
func (s *DBService) ReadAllFeedItems(ch chan *Feeditem) error {
	defer close(ch)

	failed := false

	cursor := uint64(0)
	for haveData := true; haveData; {
		var keys []string
		var err error
		keys, cursor, err = s.client.Scan(cursor, FeeditemKeyPrefix+"*", 100).Result()
		if err != nil {
			log.WithError(err).Error("Failed to get pages")
			failed = true
			continue
		}
		if cursor == 0 {
			haveData = false
		}

		for _, key := range keys {
			feeditemKey, err := DecodeFeeditemKey(key)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to decode key of feed item")
				failed = true
				continue
			}

			value, err := s.client.HGetAll(key).Result()
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed get value of feed item")
				failed = true
				continue
			}

			feeditem, err := decodeFeeditem(feeditemKey, value)
			if err != nil {
				log.WithField("key", key).WithError(err).Error("Failed to decode value of feed item")
				failed = true
				continue
			}
			ch <- feeditem
		}
	}
	if failed {
		return fmt.Errorf("Failed to read at least one item")
	}
	return nil
}
