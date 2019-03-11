package data

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Password    string
	Opml        string
	Pagemonitor string
	username    string
}

type UserService struct {
	DBService
	Username string
}

type UserPagemonitor struct {
	URL     string `xml:"url,attr"`
	Title   string `xml:",chardata"`
	Match   string `xml:"match,attr"`
	Replace string `xml:"replace,attr"`
}

type UserFeed struct {
	URL   string `xml:"xmlUrl,attr"`
	Title string `xml:"title,attr"`
}

func NewUser(username string) *User {
	return &User{username: username}
}

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

func (s *DBService) SaveUser(user *User) (err error) {
	key := user.CreateKey()

	var value bytes.Buffer
	if err := gob.NewEncoder(&value).Encode(user); err != nil {
		return errors.Wrap(err, "Cannot marshal user")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value.Bytes())
	})
}

func (user *User) SetPassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return nil
}

func (s *DBService) SetUsername(user *User, newUsername string) error {
	newUser := *user
	newUser.username = newUsername
	err := s.db.Update(func(txn *badger.Txn) error {
		oldUserKey := user.CreateKey()
		item, err := txn.Get(oldUserKey)
		if err != nil {
			return err
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		newUserKey := newUser.CreateKey()
		existingUser, err := txn.Get(newUserKey)
		if existingUser != nil || (err != nil && err != badger.ErrKeyNotFound) {
			return fmt.Errorf("New username %v is already in use", newUsername)
		}
		err = txn.Set(newUserKey, value)
		if err != nil {
			return err
		}
		return txn.Delete(oldUserKey)
	})
	if err == nil {
		user.username = newUser.username
	}
	return err
}

func (user *User) ValidatePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

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
