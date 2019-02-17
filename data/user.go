package data

import (
	"encoding/json"
	"encoding/xml"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Password    string
	Opml        string
	Pagemonitor string
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
	Flags   string `xml:"flags,attr"`
}

type UserFeed struct {
	URL   string `xml:"xmlUrl,attr"`
	Title string `xml:"title,attr"`
}

func (dbService *DBService) newUserService(username string) *UserService {
	return &UserService{
		DBService: *dbService,
		Username:  username,
	}
}

func (s *UserService) Get() (*User, error) {
	user := &User{}
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(user.CreateKey(s.Username))
		if err == badger.ErrKeyNotFound {
			user = nil
			return nil
		}

		value, err := item.Value()
		if err != nil {
			return err
		}
		err = json.Unmarshal(value, user)
		if err != nil {
			user = nil
		}
		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read User %v", s.Username)
	}
	return user, nil
}

func (s *UserService) Save(user *User) (err error) {
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}
	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(user.CreateKey(s.Username), value)
	})
	return err
}

func (user *User) SetPassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return nil
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
