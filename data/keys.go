package data

import (
	"encoding/base64"
	"strings"

	"github.com/pkg/errors"
)

const separator = "/"

func encodePart(part string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(part))
}

func decodePart(part string) (string, error) {
	res, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// LastSeenKeyPrefix is the key prefix for LastSeen entries.
const LastSeenKeyPrefix = "lastseen" + separator

// CreateLastSeenKey creates a LastSeen key for itemKey.
func CreateLastSeenKey(itemKey []byte) []byte {
	return append([]byte(LastSeenKeyPrefix), itemKey...)
}

// FetchStatusKeyPrefix is the key prefix for FetchStatus entries.
const FetchStatusKeyPrefix = "fetchstatus" + separator

// CreateFetchStatusKey creates a FetchStatus key for itemKey.
func CreateFetchStatusKey(itemKey []byte) []byte {
	return append([]byte(FetchStatusKeyPrefix), itemKey...)
}

// UserKeyPrefix is the key prefix for User entries.
const UserKeyPrefix = "user" + separator

// CreateUserKey creates a key for user.
func CreateUserKey(username string) []byte {
	return []byte(UserKeyPrefix + encodePart(username))
}

// CreateKey creates a key for user.
func (user User) CreateKey() []byte {
	return CreateUserKey(user.username)
}

// DecodeUserKey decodes the username from a user key.
func DecodeUserKey(key []byte) (string, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, UserKeyPrefix) {
		return "", errors.Errorf("Not a user key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 2 {
		return "", errors.Errorf("Invalid format of user key: %v", keyString)
	}
	username, err := decodePart(parts[1])
	if err != nil {
		return "", errors.Errorf("Failed to decode username: %v because of %v", keyString, err)
	}
	return username, nil
}

// PagemonitorKeyPrefix is the key prefix for Pagemonitor.
const PagemonitorKeyPrefix = "pagemonitor" + separator

// CreateKey creates a key for a Pagemonitor entry.
func (pm UserPagemonitor) CreateKey() []byte {
	keyURL := encodePart(pm.URL)
	keyMatch := encodePart(pm.Match)
	keyReplace := encodePart(pm.Replace)
	return []byte(PagemonitorKeyPrefix + keyURL + separator + keyMatch + separator + keyReplace)
}

// DecodePagemonitorKey decodes the Pagemonitor configuration from a Pagemonitor key.
func DecodePagemonitorKey(key []byte) (UserPagemonitor, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, PagemonitorKeyPrefix) {
		return UserPagemonitor{}, errors.Errorf("Not a Pagemonitor key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 4 {
		return UserPagemonitor{}, errors.Errorf("Invalid format of Pagemonitor key: %v", keyString)
	}
	res := UserPagemonitor{}
	var err error
	res.URL, err = decodePart(parts[1])
	if err != nil {
		return UserPagemonitor{}, errors.Errorf("Failed to decode URL of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Match, err = decodePart(parts[2])
	if err != nil {
		return UserPagemonitor{}, errors.Errorf("Failed to decode Match of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Replace, err = decodePart(parts[3])
	if err != nil {
		return UserPagemonitor{}, errors.Errorf("Failed to decode Replace of Pagemonitor key: %v because of %v", keyString, err)
	}
	return res, nil
}

// FeedKeyPrefix is the key prefix for a UserFeed.
const FeedKeyPrefix = "feed" + separator

// CreateKey creates a key for a Pagemonitor entry.
func (feed UserFeed) CreateKey() []byte {
	keyURL := encodePart(feed.URL)
	return []byte(FeedKeyPrefix + keyURL)
}

// FeeditemKeyPrefix is the key prefix for Feeditem.
const FeeditemKeyPrefix = "feeditem" + separator

// CreateKey creates a key for a Feeditem entry.
func (key FeeditemKey) CreateKey() []byte {
	keyURL := encodePart(key.FeedURL)
	keyGUID := encodePart(key.GUID)
	return []byte(FeeditemKeyPrefix + keyURL + separator + keyGUID)
}

// DecodeFeeditemKey decodes the Feeditem configuration from a Feeditem key.
func DecodeFeeditemKey(key []byte) (FeeditemKey, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, FeeditemKeyPrefix) {
		return FeeditemKey{}, errors.Errorf("Not a Feeditem key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 3 {
		return FeeditemKey{}, errors.Errorf("Invalid format of Feeditem key: %v", keyString)
	}
	res := FeeditemKey{}
	var err error
	res.FeedURL, err = decodePart(parts[1])
	if err != nil {
		return FeeditemKey{}, errors.Errorf("Failed to decode Feed URL of Feeditem key: %v because of %v", keyString, err)
	}
	res.GUID, err = decodePart(parts[2])
	if err != nil {
		return FeeditemKey{}, errors.Errorf("Failed to decode GUID of Feeditem key: %v because of %v", keyString, err)
	}
	return res, nil
}

// ReadStatusPrefix is the key prefix for an item read status.
const ReadStatusPrefix = "feed" + separator

// CreateReadStatusPrefix creates a read status key prefix for user.
func (user User) CreateReadStatusPrefix() string {
	return ReadStatusPrefix + encodePart(user.username) + separator
}

// CreateReadStatusKey creates a read status key for an item key.
func (user User) CreateReadStatusKey(itemKey []byte) []byte {
	return append([]byte(user.CreateReadStatusPrefix()), itemKey...)
}

// DecodeReadStatusKey decodes the item key from the read status key.
func DecodeReadStatusKey(key []byte) ([]byte, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, ReadStatusPrefix) {
		return nil, errors.Errorf("Not a read status key: %v", keyString)
	}
	parts := strings.SplitN(keyString, separator, 3)
	if len(parts) != 3 {
		return nil, errors.Errorf("Invalid format of read status key: %v", keyString)
	}
	return []byte(parts[2]), nil
}

// ServerConfigKeyPrefix is the key prefix for a ServerConfig item.
const ServerConfigKeyPrefix = "serverconfig" + separator

// CreateServerConfigKey creates a key for a ServerConfig item.
func CreateServerConfigKey(varName string) []byte {
	return []byte(ServerConfigKeyPrefix + encodePart(varName))
}

// DecodeServerConfigKey decodes the name of a ServerConfig key.
func DecodeServerConfigKey(key []byte) (string, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, ServerConfigKeyPrefix) {
		return "", errors.Errorf("Not a config item key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 2 {
		return "", errors.Errorf("Invalid format of config item key: %v", keyString)
	}
	value, err := decodePart(parts[1])
	if err != nil {
		return "", errors.Errorf("Failed to decode config item value: %v because of %v", keyString, err)
	}
	return value, nil
}
