package data

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// User keeps configuration for a user.
type User struct {
	Password    string
	Opml        string
	Pagemonitor string
	username    string

	id *int
}

// UserPagemonitor is a deserialized copy of a page from the Pagemonitor.
type UserPagemonitor struct {
	URL     string `xml:"url,attr"`
	Title   string `xml:",chardata"`
	Match   string `xml:"match,attr"`
	Replace string `xml:"replace,attr"`
}

// UserFeed is a deserialized copy of a page from OPML.
type UserFeed struct {
	URL   string `xml:"xmlUrl,attr"`
	Title string `xml:"title,attr"`
}

// NewUser creates an instance of User with the provided username.
func NewUser(username string) *User {
	return &User{username: username}
}

// ReadAllUsers reads all users from database and sends them to the provided channel.
func (s *DBService) ReadAllUsers(ch chan *User) error {
	defer close(ch)
	rows, err := s.db.Query("SELECT id, username, password, opml, pagemonitor FROM users")
	if err != nil {
		return fmt.Errorf("failed to read users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		user := &User{id: &id}
		err := rows.Scan(&user.id, &user.username, &user.Password, &user.Opml, &user.Pagemonitor)

		if err != nil {
			return fmt.Errorf("failed to read user: %w", err)
		}

		ch <- user
	}
	return nil
}

// GetUser returns the User by username.
// If user doesn't exist, returns nil.
func (s *DBService) GetUser(username string) (*User, error) {
	var id int
	user := User{id: &id}
	err := s.db.QueryRow("SELECT id, username, password, opml, pagemonitor FROM users WHERE username=$1", username).
		Scan(&user.id, &user.username, &user.Password, &user.Opml, &user.Pagemonitor)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read user %v: %w", username, err)
	}

	return &user, nil
}

// SaveUser saves the user in the database.
func (s *DBService) SaveUser(user *User) (err error) {
	var id int
	userExists := user.id != nil
	err = s.updateTx(func(tx *sql.Tx) error {
		if userExists && user.id != nil {
			id = *user.id
			_, err := tx.Exec("UPDATE users SET username=$1, password=$2, opml=$3, pagemonitor=$4 WHERE id=$5", user.username, user.Password, user.Opml, user.Pagemonitor, id)
			if err != nil {
				return err
			}
		} else {
			err := tx.QueryRow("INSERT INTO users(username, password, opml, pagemonitor) VALUES($1, $2, $3, $4) RETURNING id", user.username, user.Password, user.Opml, user.Pagemonitor).Scan(&id)
			if err != nil {
				return err
			}
		}

		if user.id == nil {
			user.id = &id
		}

		err = linkUserPages(user, tx)
		if err != nil {
			return err
		}
		return linkUserFeeds(user, tx)
	})
	if err != nil && userExists {
		revertErr := s.db.QueryRow("SELECT username, password, opml, pagemonitor FROM users WHERE id=$1", user.id).
			Scan(&user.username, &user.Password, &user.Opml, &user.Pagemonitor)
		if revertErr != nil {
			return fmt.Errorf("failed to reload user details (%v) on a failed SaveUser: %w", revertErr, err)
		}
	}
	if err != nil && !userExists {
		user.id = nil
	}
	return err
}

// GetUsername returns the user's current username.
func (user *User) GetUsername() string {
	return user.username
}

// SetUsername sets a new username for User which will be updated when SaveUser is called.
func (user *User) SetUsername(newUsername string) error {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return fmt.Errorf("cannot set username to an empty string")
	}
	user.username = newUsername
	return nil
}

// SetPassword sets a new password for user. The password is hashed and salted with bcrypt.
func (user *User) SetPassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return nil
}

// ValidatePassword checks if password matches the user's password.
func (user *User) ValidatePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

// GetPages parses user's configuration and returns all UserPagemonitor configuration items.
func (user *User) GetPages() ([]UserPagemonitor, error) {
	if user.Pagemonitor == "" {
		// Empty pagemonitor config - treat as empty XML.
		return []UserPagemonitor{}, nil
	}

	type UserPages struct {
		XMLName xml.Name          `xml:"pages"`
		Pages   []UserPagemonitor `xml:"page"`
	}
	items := &UserPages{}
	err := xml.Unmarshal([]byte(user.Pagemonitor), items)
	if err != nil {
		err = fmt.Errorf("cannot parse pagemonitor xml: %w", err)
		return nil, err
	}
	return items.Pages, nil
}

// GetFeeds parses user's configuration and returns all UserFeed configuration items.
func (user *User) GetFeeds() ([]UserFeed, error) {
	if user.Opml == "" {
		// Empty opml - treat as empty XML.
		return []UserFeed{}, nil
	}

	type UserOPMLOutline struct {
		UserFeed
		Children []UserOPMLOutline `xml:"outline"`
	}
	type UserOPML struct {
		XMLName xml.Name          `xml:"opml"`
		Feeds   []UserOPMLOutline `xml:"body>outline"`
	}
	items := &UserOPML{}
	err := xml.Unmarshal([]byte(user.Opml), items)
	if err != nil {
		err = fmt.Errorf("cannot parse opml xml: %w", err)
		return nil, err
	}
	feeds := []UserFeed{}
	var findFeeds func([]UserOPMLOutline)
	findFeeds = func(outlines []UserOPMLOutline) {
		for _, outline := range outlines {
			if outline.URL != "" {
				feeds = append(feeds, outline.UserFeed)
			}
			findFeeds(outline.Children)
		}
	}
	findFeeds(items.Feeds)
	return feeds, nil
}
