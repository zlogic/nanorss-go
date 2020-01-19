package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/akrylysov/pogreb"
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

// Encode serializes a Feeditem.
func (feedItem *Feeditem) Encode() ([]byte, error) {
	key := feedItem.Key
	defer func() { feedItem.Key = key }()
	feedItem.Key = nil

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(feedItem); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

// Decode deserializes a Feeditem.
func (feedItem *Feeditem) Decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(feedItem)
}

// GetFeeditem retrieves a Feeditem for the FeeditemKey.
// If item doesn't exist, returns nil.
func (s *DBService) GetFeeditem(key *FeeditemKey) (*Feeditem, error) {
	feeditem := &Feeditem{Key: key}
	value, err := s.db.Get(key.CreateKey())
	if err != nil {
		feeditem = nil
		return nil, fmt.Errorf("Cannot read feed item %v because of %w", key, err)
	}

	if value == nil {
		return nil, nil
	}

	if err := feeditem.Decode(value); err != nil {
		feeditem = nil
		return nil, fmt.Errorf("Cannot decode feed item %v because of %w", key, err)
	}
	return feeditem, nil
}

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	getPreviousItem := func(key []byte) (*Feeditem, error) {
		value, err := s.db.Get(key)
		if err != nil {
			return nil, fmt.Errorf("Failed to get previous feed item %v because of %w", string(key), err)
		}
		if value == nil {
			return nil, nil
		}
		existingFeedItem := &Feeditem{}
		if err := existingFeedItem.Decode(value); err != nil {
			return nil, fmt.Errorf("Failed to read previous value of feed item %v because of %w", string(key), err)
		}
		return existingFeedItem, nil
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

		value, err := feedItem.Encode()
		if err != nil {
			return fmt.Errorf("Cannot marshal feed item because of %w", err)
		}

		if err := s.SetLastSeen(key); err != nil {
			return fmt.Errorf("Cannot set last seen time because of %w", err)
		}

		if previousItem != nil && *feedItem == *previousItem {
			// Avoid writing to the database if nothing has changed
			continue
		} else if previousItem != nil {
			log.WithField("previousItem", previousItem).WithField("feedItem", feedItem).Debug("Item has changed")
		}

		if err := s.db.Put(key, value); err != nil {
			return fmt.Errorf("Cannot save feed item because of %w", err)
		}
	}
	return nil
}

// ReadAllFeedItems reads all Feeditem items from database and sends them to the provided channel.
func (s *DBService) ReadAllFeedItems(ch chan *Feeditem) (err error) {
	defer close(ch)
	prefix := []byte(FeeditemKeyPrefix)
	it := s.db.Items()
	for {
		k, v, err := it.Next()
		if err != nil {
			if err != pogreb.ErrIterationDone {
				return fmt.Errorf("Cannot read feed items because of %w", err)
			}
			break
		}
		if !validForPrefix(prefix, k) {
			continue
		}

		key, err := DecodeFeeditemKey(k)
		if err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to decode key of item")
			continue
		}

		feedItem := &Feeditem{Key: key}
		if err := feedItem.Decode(v); err != nil {
			log.WithField("key", k).WithError(err).Error("Failed to read value of item")
			continue
		}
		ch <- feedItem
	}
	return nil
}
