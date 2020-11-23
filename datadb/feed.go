package datadb

import (
	"database/sql"
	"fmt"
	"time"
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

	LastSeen *time.Time

	Key *FeeditemKey `json:",omitempty"`
}

// GetFeeditem retrieves a Feeditem for the FeeditemKey.
// If item doesn't exist, returns nil.
func (s *DBService) GetFeeditem(key *FeeditemKey) (*Feeditem, error) {
	var feeditem *Feeditem
	err := s.viewTx(func(tx *sql.Tx) error {
		var err error
		feeditem, err = getFeeditem(key, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return feeditem, nil
}

func getFeeditem(key *FeeditemKey, tx *sql.Tx) (*Feeditem, error) {
	feeditem := Feeditem{Key: key}
	err := tx.QueryRow(
		"SELECT fi.title, fi.url, fi.date, fi.contents, fi.updated, fi.last_seen FROM feeditems fi, feeds f WHERE fi.feed_id = f.id AND f.url=$1 AND fi.guid=$2",
		key.FeedURL, key.GUID,
	).Scan(&feeditem.Title, &feeditem.URL, &feeditem.Date, &feeditem.Contents, &feeditem.Updated, &feeditem.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("cannot read feed item %v: %w", key, err)
	}
	return &feeditem, nil
}

// SaveFeeditems saves feedItems in the database.
func (s *DBService) SaveFeeditems(feedItems ...*Feeditem) (err error) {
	return s.updateTx(func(tx *sql.Tx) error {
		for _, feedItem := range feedItems {
			previousItem, err := getFeeditem(feedItem.Key, tx)
			if err != nil {
				return err
			}

			if previousItem != nil {
				feedItem.Date = feedItem.Date.In(previousItem.Date.Location())
				if previousItem.Contents == feedItem.Contents &&
					previousItem.Title == feedItem.Title &&
					previousItem.URL == feedItem.URL {
					feedItem.Contents = previousItem.Contents
					feedItem.Title = previousItem.Title
					feedItem.URL = previousItem.URL
					feedItem.Updated = previousItem.Updated
				}
			}

			if previousItem != nil {
				_, err := tx.Exec(
					"UPDATE feeditems fi SET title=$1, url=$2, date=$3, contents=$4, updated=$5, last_seen=$6 FROM feeds f WHERE fi.feed_id = f.id AND f.url = $7 AND fi.guid = $8",
					feedItem.Title, feedItem.URL, feedItem.Date, feedItem.Contents, feedItem.Updated, feedItem.LastSeen,
					feedItem.Key.FeedURL, feedItem.Key.GUID,
				)
				if err != nil {
					return err
				}
			} else {
				_, err = tx.Exec(
					"INSERT INTO feeditems(feed_id, guid, title, url, date, contents, updated, last_seen) VALUES((SELECT id FROM feeds WHERE url = $1), $2, $3, $4, $5, $6, $7, $8)",
					feedItem.Key.FeedURL, feedItem.Key.GUID,
					feedItem.Title, feedItem.URL, feedItem.Date, feedItem.Contents, feedItem.Updated, feedItem.LastSeen,
				)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func linkUserFeeds(user *User, tx *sql.Tx) error {
	if user.id == nil {
		return fmt.Errorf("user hasn't been created yet")
	}
	feeds, err := user.GetFeeds()
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		var id int
		err := tx.QueryRow("SELECT id FROM feeds WHERE url=$1", feed.URL).
			Scan(&id)
		if err == sql.ErrNoRows {
			err := tx.QueryRow("INSERT INTO feeds(url) VALUES($1) RETURNING id", feed.URL).Scan(&id)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		_, err = tx.Exec("INSERT INTO user_feeds(user_id, feed_id) VALUES($1, $2)", *user.id, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFeeditems reads all Feeditem items from database for a specific user.
func (s *DBService) GetFeeditems(user *User) ([]*Feeditem, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	if user.id == nil {
		return nil, fmt.Errorf("user id is nil")
	}

	rows, err := s.db.Query(
		"SELECT f.url, fi.guid, fi.title, fi.url, fi.date, fi.contents, fi.updated, fi.last_seen FROM feeditems fi, feeds f, user_feeds uf WHERE fi.feed_id = f.id AND f.id = uf.feed_id AND uf.user_id = $1",
		*user.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeditems := make([]*Feeditem, 0)
	for rows.Next() {
		feeditem := &Feeditem{Key: &FeeditemKey{}}
		err := rows.Scan(&feeditem.Key.FeedURL, &feeditem.Key.GUID, &feeditem.Title, &feeditem.URL, &feeditem.Date, &feeditem.Contents, &feeditem.Updated, &feeditem.LastSeen)
		if err != nil {
			return nil, fmt.Errorf("failed to read feed item: %w", err)
		}

		feeditems = append(feeditems, feeditem)
	}
	return feeditems, nil
}
