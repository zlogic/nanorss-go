package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

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
func (feedItem *Feeditem) encode() ([]byte, error) {
	key := feedItem.Key
	defer func() { feedItem.Key = key }()
	feedItem.Key = nil

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(feedItem); err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

// decode deserializes a Feeditem.
func (feedItem *Feeditem) decode(val []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(val)).Decode(feedItem)
}

// GetFeeditem retrieves a Feeditem for the FeeditemKey.
// If item doesn't exist, returns nil.
func (s *DBService) GetFeeditem(key *FeeditemKey) (*Feeditem, error) {
	feeditem := &Feeditem{Key: key}
	value, err := s.db.Get(key.CreateKey())
	if err != nil {
		return nil, fmt.Errorf("cannot get feed item %v: %w", key, err)
	}
	if value == nil {
		return nil, nil
	}

	if err := feeditem.decode(value); err != nil {
		return nil, fmt.Errorf("cannot decode feed item %v: %w", key, err)
	}

	return feeditem, nil
}

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	getPreviousItem := func(key []byte) (*Feeditem, error) {
		value, err := s.db.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous feed item %v: %w", string(key), err)
		}
		if value == nil {
			// Item doesn't exist.
			return nil, nil
		}
		existingFeedItem := &Feeditem{}
		if err := existingFeedItem.decode(value); err != nil {
			return nil, fmt.Errorf("failed to read previous value of feed item %v: %w", string(key), err)
		}
		return existingFeedItem, nil
	}

	for _, feedItem := range feedItems {
		if err := s.addReferencedKey(feedItem.Key.createIndexKey(), []byte(feedItem.Key.GUID)); err != nil {
			return fmt.Errorf("failed to add feed item %v to feed index: %w", feedItem.Key, err)
		}

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
			return fmt.Errorf("cannot marshal feed item: %w", err)
		}

		if err := s.SetLastSeen(key); err != nil {
			return fmt.Errorf("cannot set last seen time: %w", err)
		}

		if previousItem != nil && *feedItem == *previousItem {
			// Avoid writing to the database if nothing has changed
			continue
		} else if previousItem != nil {
			log.WithField("previousItem", previousItem).WithField("feedItem", feedItem).Debug("Item has changed")
		}

		if err := s.db.Put(key, value); err != nil {
			return fmt.Errorf("cannot save feed item: %w", err)
		}
	}
	return nil
}

// GetFeeditems returns all Feeditem items for user.
func (s *DBService) GetFeeditems(user *User) ([]*Feeditem, error) {
	feeds, err := user.GetFeeds()
	if err != nil {
		return nil, err
	}

	feedItems := make([]*Feeditem, 0)
	for i := range feeds {
		feed := feeds[i]

		indexItems, err := s.getReferencedKeys(feed.createItemsIndexKey())
		if err != nil {
			log.WithField("feed", feed.URL).WithError(err).Error("Failed to get index for items of a feed")
			continue
		}
		for j := range indexItems {
			itemKey := FeeditemKey{
				FeedURL: feed.URL,
				GUID:    string(indexItems[j]),
			}

			value, err := s.db.Get(itemKey.CreateKey())
			if err != nil {
				log.WithField("key", itemKey).WithError(err).Error("Failed to read value of item")
			}

			feedItem := &Feeditem{Key: &itemKey}
			if err := feedItem.decode(value); err != nil {
				log.WithField("key", itemKey).WithError(err).Error("Failed to decode value of item")
				continue
			}

			feedItems = append(feedItems, feedItem)
		}
	}
	return feedItems, nil
}
