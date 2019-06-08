package data

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// User keeps configuration for a user.
type User struct {
	Password    string
	Opml        string
	Pagemonitor string
	username    string
	newUsername string
}

// UserService wraps a DBService and makes sure that only data for Username is returned.
type UserService struct {
	DBService
	Username string
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
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(UserKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			k := item.Key()

			username, err := DecodeUserKey(k)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to decode username of user")
				continue
			}

			v, err := item.Value()
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to read value of user")
				continue
			}

			user := &User{username: *username}
			err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(&user)
			if err != nil {
				log.WithField("key", k).WithError(err).Error("Failed to unmarshal value of user")
				continue
			}
			ch <- user
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "Cannot read users")
	}
	return nil
}

// GetUser returns the User by username.
// If user doesn't exist, returns nil.
func (s *DBService) GetUser(username string) (*User, error) {
	user := &User{username: username}
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(user.CreateKey())
		if err == badger.ErrKeyNotFound {
			user = nil
			return nil
		}

		value, err := item.Value()
		if err != nil {
			return err
		}
		err = gob.NewDecoder(bytes.NewBuffer(value)).Decode(&user)
		if err != nil {
			user = nil
		}
		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read User %v", username)
	}
	return user, nil
}

// SaveUser saves the user in the database.
func (s *DBService) SaveUser(user *User) (err error) {
	if user.newUsername == "" {
		user.newUsername = user.username
	}
	key := CreateUserKey(user.newUsername)

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(user); err != nil {
		return errors.Wrap(err, "Cannot marshal user")
	}
	err = s.db.Update(func(txn *badger.Txn) error {
		if user.newUsername != user.username {
			existingUser, err := txn.Get(key)
			if existingUser != nil || (err != nil && err != badger.ErrKeyNotFound) {
				return fmt.Errorf("New username %v is already in use", key)
			}

			oldUserKey := CreateUserKey(user.username)
			if err := txn.Delete(oldUserKey); err != nil {
				return err
			}
			if err := s.renameReadStatus(user)(txn); err != nil {
				return err
			}
		}
		return txn.Set(key, value.Bytes())
	})
	if err == nil {
		user.username = user.newUsername
		user.newUsername = ""
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
		return fmt.Errorf("Cannot set username to an empty string")
	}
	user.newUsername = newUsername
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
	type UserPages struct {
		XMLName xml.Name          `xml:"pages"`
		Pages   []UserPagemonitor `xml:"page"`
	}
	items := &UserPages{}
	err := xml.Unmarshal([]byte(user.Pagemonitor), items)
	if err != nil {
		err = errors.Wrap(err, "Cannot parse pagemonitor xml")
		return nil, err
	}
	return items.Pages, nil
}

// GetFeeds parses user's configuration and returns all UserFeed configuration items.
func (user *User) GetFeeds() ([]UserFeed, error) {
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
		err = errors.Wrap(err, "Cannot parse opml xml")
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
