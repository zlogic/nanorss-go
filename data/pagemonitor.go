package data

import (
	"database/sql"
	"fmt"
	"time"
)

// PagemonitorPage keeps the state and diff for a web page monitored by Pagemonitor.
type PagemonitorPage struct {
	Contents string
	Delta    string
	Updated  time.Time

	Config *UserPagemonitor `json:",omitempty"`
}

// GetPage retrieves a PagemonitorPage for the UserPagemonitor configuration.
// If page doesn't exist, returns nil.
func (s *DBService) GetPage(pm *UserPagemonitor) (*PagemonitorPage, error) {
	var page *PagemonitorPage
	err := s.viewTx(func(tx *sql.Tx) error {
		var err error
		page, err = getPage(pm, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}

func getPage(pm *UserPagemonitor, tx *sql.Tx) (*PagemonitorPage, error) {
	page := PagemonitorPage{Config: pm}
	err := tx.QueryRow("SELECT contents, delta, updated FROM pagemonitors WHERE url=$1 AND match=$2 AND replace=$3",
		pm.URL, pm.Match, pm.Replace,
	).Scan(&page.Contents, &page.Delta, &page.Updated)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("cannot read page %v: %w", page, err)
	}
	return &page, nil
}

// SavePage saves a PagemonitorPage.
func (s *DBService) SavePage(page *PagemonitorPage) error {
	return s.updateTx(func(tx *sql.Tx) error {
		previousPage, err := getPage(page.Config, tx)
		if err != nil {
			return err
		}

		if previousPage != nil {
			_, err := tx.Exec(
				"UPDATE pagemonitors SET contents=$1, delta=$2, updated=$3 WHERE url=$4 AND match=$5 AND replace=$6",
				page.Contents, page.Delta, page.Updated,
				page.Config.URL, page.Config.Match, page.Config.Replace,
			)
			return err
		}

		_, err = tx.Exec(
			"INSERT INTO pagemonitors(url, match, replace, contents, delta, updated) VALUES($1, $2, $3, $4, $5, $6)",
			page.Config.URL, page.Config.Match, page.Config.Replace,
			page.Contents, page.Delta, page.Updated,
		)
		return err
	})
}

func linkUserPages(user *User, tx *sql.Tx) error {
	if user.id == nil {
		return fmt.Errorf("user hasn't been created yet")
	}
	pages, err := user.GetPages()
	if err != nil {
		return err
	}

	for _, page := range pages {
		var id int
		err := tx.QueryRow("SELECT id FROM pagemonitors WHERE url=$1 AND match=$2 AND replace=$3", page.URL, page.Match, page.Replace).
			Scan(&id)
		if err == sql.ErrNoRows {
			err := tx.QueryRow(
				"INSERT INTO pagemonitors(url, match, replace, contents, delta, updated) VALUES($1, $2, $3, '', '', $4) RETURNING id",
				page.URL, page.Match, page.Replace, time.Time{},
			).Scan(&id)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		var existingUserPage int
		err = tx.QueryRow("SELECT 1 FROM user_pagemonitors WHERE user_id=$1 AND pagemonitor_id=$2", *user.id, id).
			Scan(&existingUserPage)
		if err == sql.ErrNoRows {
			_, err = tx.Exec("INSERT INTO user_pagemonitors(user_id, pagemonitor_id) VALUES($1, $2)", *user.id, id)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

	}
	return nil
}

// GetPages reads all PagemonitorPage items from database for a specific user.
func (s *DBService) GetPages(user *User) ([]*PagemonitorPage, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	if user.id == nil {
		return nil, fmt.Errorf("user id is nil")
	}

	rows, err := s.db.Query(
		"SELECT pm.url, pm.match, pm.replace, pm.contents, pm.delta, pm.updated FROM pagemonitors pm, user_pagemonitors upm WHERE pm.id = upm.pagemonitor_id AND upm.user_id = $1",
		*user.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pages := make([]*PagemonitorPage, 0)
	for rows.Next() {
		page := &PagemonitorPage{Config: &UserPagemonitor{}}
		err := rows.Scan(&page.Config.URL, &page.Config.Match, &page.Config.Replace, &page.Contents, &page.Delta, &page.Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to read pagemonitor: %w", err)
		}

		pages = append(pages, page)
	}
	return pages, nil
}
