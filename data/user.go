package data

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/go-redis/redis"
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

// decode deserializes a User.
func decodeUser(username string, res map[string]string) *User {
	return &User{
		username:    username,
		Password:    res["password"],
		Opml:        res["opml"],
		Pagemonitor: res["pagemonitor"],
	}
}

// encode serializes a User.
func (user *User) encode() map[string]interface{} {
	return map[string]interface{}{
		"password":    user.Password,
		"opml":        user.Opml,
		"pagemonitor": user.Pagemonitor,
	}
}

// ReadAllUsers reads all users from database and sends them to the provided channel.
func (s *DBService) ReadAllUsers(ch chan *User) error {
	defer close(ch)
	// TODO: use Scan to avoid loading all keys to RAM
	userKeys, err := s.client.Keys(UserKeyPrefix + "*").Result()
	if err != nil {
		log.WithError(err).Error("Failed to get list of users")
	}
	for _, userKey := range userKeys {
		username, err := DecodeUserKey(userKey)

		user, err := s.GetUser(username)
		if err != nil {
			log.WithError(err).Errorf("Failed to get user %v", username)
			continue
		}

		ch <- user
	}
	if err != nil {
		return errors.Wrapf(err, "Cannot read users")
	}
	return nil
}

// GetUser returns the User by username.
// If user doesn't exist, returns nil.
func (s *DBService) GetUser(username string) (*User, error) {
	userMap, err := s.client.HGetAll(CreateUserKey(username)).Result()
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read User %v", username)
	}

	if len(userMap) == 0 {
		return nil, nil
	}
	user := decodeUser(username, userMap)
	return user, nil
}

// SaveUser saves the user in the database.
func (s *DBService) SaveUser(user *User) error {
	if user.newUsername == "" {
		user.newUsername = user.username
	}
	if user.username != user.newUsername {
		userKey := CreateUserKey(user.username)
		newUserKey := CreateUserKey(user.newUsername)
		readStatusKey := CreateReadStatusKey(user.username)
		newReadStatusKey := CreateReadStatusKey(user.newUsername)
		var renameUser *redis.BoolCmd
		err := s.client.Watch(func(tx *redis.Tx) error {
			existsUser, err := tx.Exists(newUserKey).Result()
			if err != nil {
				return err
			}
			if existsUser != 0 {
				return fmt.Errorf("Failed to rename user to %v is already in use", user.newUsername)
			}

			existsReadStatus, err := tx.Exists(readStatusKey).Result()
			if err != nil {
				return err
			}
			_, err = tx.Pipelined(func(pipe redis.Pipeliner) error {

				renameUser = pipe.RenameNX(userKey, newUserKey)
				if existsReadStatus != 0 {
					pipe.RenameNX(readStatusKey, newReadStatusKey)
				}
				return nil
			})
			return err
		}, userKey, newUserKey)
		//renameReadStatus := pipe.RenameNX(oldKey, newKey)
		if err != nil {
			return errors.Wrapf(err, "Failed to rename user from %v to %v", user.username, user.newUsername)
		}
		if !renameUser.Val() {
			return fmt.Errorf("Failed to rename user to %v is already in use", user.newUsername)
		}
	}
	user.username = user.newUsername
	user.newUsername = ""
	err := s.client.HMSet(user.CreateKey(), user.encode()).Err()
	if err != nil {
		return errors.Wrapf(err, "Failed to update user %v", user.username)
	}

	return nil
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
	items := UserPages{}
	err := xml.Unmarshal([]byte(user.Pagemonitor), &items)
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
	items := UserOPML{}
	err := xml.Unmarshal([]byte(user.Opml), &items)
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
