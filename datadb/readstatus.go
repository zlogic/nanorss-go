package datadb

import (
	"fmt"
)

// GetPagesReadStatus returns the list of pages that are marked as read for user.
func (s *DBService) GetPagesReadStatus(user *User) ([]*UserPagemonitor, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	if user.id == nil {
		return nil, fmt.Errorf("user id is nil")
	}

	rows, err := s.db.Query(
		"SELECT pm.url, pm.match, pm.replace FROM user_read_pagemonitors urpm, pagemonitors pm WHERE urpm.pagemonitor_id = pm.id AND urpm.user_id = $1",
		*user.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pages := make([]*UserPagemonitor, 0)
	for rows.Next() {
		page := &UserPagemonitor{}
		err := rows.Scan(&page.URL, &page.Match, &page.Replace)
		if err != nil {
			return nil, fmt.Errorf("failed to read pagemonitor read status: %w", err)
		}

		pages = append(pages, page)
	}
	return pages, nil
}

// GetFeeditemsReadStatus returns the list of feed items that are marked as read for user.
func (s *DBService) GetFeeditemsReadStatus(user *User) ([]*FeeditemKey, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	if user.id == nil {
		return nil, fmt.Errorf("user id is nil")
	}

	rows, err := s.db.Query(
		"SELECT f.url, fi.guid FROM user_read_feeditems urfi, feeditems fi, feeds f WHERE urfi.feeditem_guid = fi.guid AND fi.feed_id = f.id AND urfi.user_id = $1",
		*user.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeditems := make([]*FeeditemKey, 0)
	for rows.Next() {
		feeditem := &FeeditemKey{}
		err := rows.Scan(&feeditem.FeedURL, &feeditem.GUID)
		if err != nil {
			return nil, fmt.Errorf("failed to read feed item read status: %w", err)
		}

		feeditems = append(feeditems, feeditem)
	}
	return feeditems, nil
}

// SetPageReadStatus sets the read status for a page, true for read, false for unread.
func (s *DBService) SetPageReadStatus(user *User, k *UserPagemonitor, read bool) error {
	if read {
		_, err := s.db.Exec(
			"INSERT INTO user_read_pagemonitors(user_id, pagemonitor_id) VALUES($1, (SELECT id FROM pagemonitors WHERE url=$2 AND match=$3 AND replace=$4))",
			user.id, k.URL, k.Match, k.Replace,
		)
		return err
	}
	_, err := s.db.Exec(
		"DELETE FROM user_read_pagemonitors WHERE user_id=$1 AND pagemonitor_id IN(SELECT id FROM pagemonitors WHERE url=$2 AND match=$3 AND replace=$4)",
		user.id, k.URL, k.Match, k.Replace,
	)
	return err
}

// SetFeeditemReadStatus sets the read status for a feed item, true for read, false for unread.
func (s *DBService) SetFeeditemReadStatus(user *User, k *FeeditemKey, read bool) error {
	if read {
		_, err := s.db.Exec(
			"INSERT INTO user_read_feeditems(user_id, feed_id, feeditem_guid) VALUES($1, (SELECT id FROM feeditems fi, feeds f WHERE fi.feed_id = f.id AND f.url=$2 AND fi.guid=$3), $3)",
			user.id, k.FeedURL, k.GUID,
		)
		return err
	}
	_, err := s.db.Exec(
		"DELETE FROM user_read_feeditems WHERE user_id=$1 AND feed_id IN(SELECT f.id FROM feeditems fi, feeds f WHERE fi.feed_id = f.id AND f.url=$2 AND fi.guid=$3) AND feeditem_guid=$3",
		user.id, k.FeedURL, k.GUID,
	)
	return err
}

// SetPageUnreadForAll sets the page as unread (for all users).
func (s *DBService) SetPageUnreadForAll(k *UserPagemonitor) error {
	_, err := s.db.Exec(
		"DELETE FROM user_read_pagemonitors WHERE pagemonitor_id IN(SELECT id FROM pagemonitors WHERE url=$1 AND match=$2 AND replace=$3)",
		k.URL, k.Match, k.Replace,
	)
	return err
}
