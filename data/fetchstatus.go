package data

import (
	"database/sql"
	"fmt"
	"time"
)

// FetchStatus keeps track of successful and failed fetches.
type FetchStatus struct {
	LastSuccess      time.Time
	LastFailure      time.Time
	LastFailureError string
}

// GetFeedFetchStatus returns the fetch status for a feed, or nil if the fetch status is unknown.
func (s *DBService) GetFeedFetchStatus(feedURL string) (*FetchStatus, error) {
	var lastSuccess, lastFailure *time.Time
	var lastFailureError *string

	err := s.db.QueryRow("SELECT last_success, last_failure, last_failure_error FROM feeds WHERE url=$1", feedURL).
		Scan(&lastSuccess, &lastFailure, &lastFailureError)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get fetch staths: %w", err)
	}

	fetchStatus := FetchStatus{}
	if lastSuccess != nil {
		fetchStatus.LastSuccess = *lastSuccess
	}
	if lastFailure != nil {
		fetchStatus.LastFailure = *lastFailure
	}
	if lastFailureError != nil {
		fetchStatus.LastFailureError = *lastFailureError
	}

	return &fetchStatus, nil
}

// GetPageFetchStatus returns the fetch status for a page, or nil if the fetch status is unknown.
func (s *DBService) GetPageFetchStatus(key *UserPagemonitor) (*FetchStatus, error) {
	var lastSuccess, lastFailure *time.Time
	var lastFailureError *string
	err := s.db.QueryRow(
		"SELECT last_success, last_failure, last_failure_error FROM pagemonitors WHERE url=$1 AND match=$2 AND replace=$3",
		key.URL, key.Match, key.Replace).
		Scan(&lastSuccess, &lastFailure, &lastFailureError)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get fetch staths: %w", err)
	}

	fetchStatus := FetchStatus{}
	if lastSuccess != nil {
		fetchStatus.LastSuccess = *lastSuccess
	}
	if lastFailure != nil {
		fetchStatus.LastFailure = *lastFailure
	}
	if lastFailureError != nil {
		fetchStatus.LastFailureError = *lastFailureError
	}

	return &fetchStatus, nil
}

// SetFeedFetchStatus creates or updates the fetch status for a feed.
func (s *DBService) SetFeedFetchStatus(feedURL string, fetchStatus *FetchStatus) error {
	if fetchStatus.LastSuccess != (time.Time{}) {
		_, err := s.db.Exec("UPDATE feeds SET last_success=$1 WHERE url = $2", fetchStatus.LastSuccess, feedURL)
		return err
	}
	if fetchStatus.LastFailure != (time.Time{}) {
		_, err := s.db.Exec("UPDATE feeds SET last_failure=$1, last_failure_error=$2 WHERE url = $3", fetchStatus.LastFailure, fetchStatus.LastFailureError, feedURL)
		return err
	}
	return nil
}

// SetPageFetchStatus creates or updates the fetch status for a page.
func (s *DBService) SetPageFetchStatus(key *UserPagemonitor, fetchStatus *FetchStatus) error {
	if fetchStatus.LastSuccess != (time.Time{}) {
		_, err := s.db.Exec(
			"UPDATE pagemonitors SET last_success=$1 WHERE url = $2 AND match = $3 AND replace=$4",
			fetchStatus.LastSuccess, key.URL, key.Match, key.Replace,
		)
		return err
	}
	if fetchStatus.LastFailure != (time.Time{}) {
		_, err := s.db.Exec(
			"UPDATE pagemonitors SET last_failure=$1, last_failure_error=$2 WHERE url = $3 AND match = $4 AND replace=$5",
			fetchStatus.LastFailure, fetchStatus.LastFailureError, key.URL, key.Match, key.Replace,
		)
		return err
	}
	return nil
}
