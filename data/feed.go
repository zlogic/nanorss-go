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

	contents, err := s.db.Get(key.createContentsKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get contents of previous feed item %v: %w", key, err)
	}
	if contents != nil {
		feeditem.Contents = string(contents)
	}

	return feeditem, nil
}

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	for _, feedItem := range feedItems {
		if err := s.addReferencedKey(feedItem.Key.createIndexKey(), []byte(feedItem.Key.GUID)); err != nil {
			return fmt.Errorf("failed to add feed item %v to feed index: %w", feedItem.Key, err)
		}

		key := feedItem.Key.CreateKey()
		saveFeedItem := Feeditem{
			Title:   feedItem.Title,
			URL:     feedItem.URL,
			Date:    feedItem.Date,
			Key:     feedItem.Key,
			Updated: feedItem.Updated,
		}

		previousItem, err := s.GetFeeditem(feedItem.Key)
		if err != nil {
			log.WithField("key", feedItem.Key).WithError(err).Error("Failed to read previous item")
		} else if previousItem != nil {
			saveFeedItem.Date = feedItem.Date.In(previousItem.Date.Location())
		}

		if err := s.SetLastSeen(key); err != nil {
			return fmt.Errorf("cannot set last seen time: %w", err)
		}

		if previousItem != nil &&
			feedItem.Title == previousItem.Title &&
			feedItem.URL == previousItem.URL &&
			saveFeedItem.Date == previousItem.Date &&
			feedItem.Contents == previousItem.Contents {
			// Avoid writing to the database if nothing has changed.
			continue
		} else if previousItem != nil {
			log.WithField("previousItem", previousItem).WithField("feedItem", feedItem).Debug("Item has changed")
		}

		value, err := saveFeedItem.encode()
		if err != nil {
			return fmt.Errorf("cannot marshal feed item: %w", err)
		}

		if err := s.db.Put(key, value); err != nil {
			return fmt.Errorf("cannot save feed item: %w", err)
		}

		contentsKey := feedItem.Key.createContentsKey()
		if err := s.db.Put(contentsKey, []byte(feedItem.Contents)); err != nil {
			return fmt.Errorf("cannot save feed item contents: %w", err)
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
