package data

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const separator = ":"

func encodePart(part string) string {
	return url.QueryEscape(part)
}

func decodePart(part string) (string, error) {
	return url.QueryUnescape(part)
}

// LastSeenKeyPrefix is the key prefix for LastSeen entries.
const LastSeenKeyPrefix = "lastseen" + separator

// CreateLastSeenKey creates a LastSeen key for itemKey.
func CreateLastSeenKey(itemKey string) string {
	return LastSeenKeyPrefix + itemKey
}

// FetchStatusKeyPrefix is the key prefix for FetchStatus entries.
const FetchStatusKeyPrefix = "fetchstatus" + separator

// CreateFetchStatusKey creates a FetchStatus key for itemKey.
func CreateFetchStatusKey(itemKey string) string {
	return FetchStatusKeyPrefix + itemKey
}

// UserKeyPrefix is the key prefix for User entries.
const UserKeyPrefix = "user" + separator

// CreateUserKey creates a key for user.
func CreateUserKey(username string) string {
	return UserKeyPrefix + username
}

// CreateKey creates a key for user.
func (user *User) CreateKey() string {
	return CreateUserKey(user.username)
}

// DecodeUserKey decodes the username from a user key.
func DecodeUserKey(key string) (string, error) {
	if !strings.HasPrefix(key, UserKeyPrefix) {
		return "", errors.Errorf("Not a user key: %v", key)
	}
	parts := strings.SplitN(key, separator, 2)
	if len(parts) != 2 {
		return "", errors.Errorf("Invalid format of user key: %v", key)
	}
	username, err := decodePart(parts[1])
	if err != nil {
		return "", errors.Errorf("Failed to decode username: %v because of %v", key, err)
	}
	return username, nil
}

// PagemonitorKeyPrefix is the key prefix for Pagemonitor.
const PagemonitorKeyPrefix = "pagemonitor" + separator

// CreateKey creates a key for a Pagemonitor entry.
func (pm *UserPagemonitor) CreateKey() string {
	keyURL := encodePart(pm.URL)
	keyMatch := encodePart(pm.Match)
	keyReplace := encodePart(pm.Replace)
	return PagemonitorKeyPrefix + keyURL + separator + keyMatch + separator + keyReplace
}

// DecodePagemonitorKey decodes the Pagemonitor configuration from a Pagemonitor key.
func DecodePagemonitorKey(key string) (*UserPagemonitor, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, PagemonitorKeyPrefix) {
		return nil, errors.Errorf("Not a Pagemonitor key: %v", keyString)
	}
	parts := strings.SplitN(keyString, separator, 4)
	if len(parts) != 4 {
		return nil, errors.Errorf("Invalid format of Pagemonitor key: %v", keyString)
	}
	res := &UserPagemonitor{}
	var err error
	res.URL, err = decodePart(parts[1])
	if err != nil {
		return nil, errors.Errorf("Failed to decode URL of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Match, err = decodePart(parts[2])
	if err != nil {
		return nil, errors.Errorf("Failed to decode Match of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Replace, err = decodePart(parts[3])
	if err != nil {
		return nil, errors.Errorf("Failed to decode Replace of Pagemonitor key: %v because of %v", keyString, err)
	}
	return res, nil
}

// FeedKeyPrefix is the key prefix for a UserFeed.
const FeedKeyPrefix = "feed" + separator

// CreateKey creates a key for a Pagemonitor entry.
func (feed *UserFeed) CreateKey() string {
	keyURL := encodePart(feed.URL)
	return FeedKeyPrefix + keyURL
}

// FeeditemKeyPrefix is the key prefix for Feeditem.
const FeeditemKeyPrefix = "feeditem" + separator

// CreateKey creates a key for a Feeditem entry.
func (key *FeeditemKey) CreateKey() string {
	keyURL := encodePart(key.FeedURL)
	keyGUID := encodePart(key.GUID)
	return FeeditemKeyPrefix + keyURL + separator + keyGUID
}

// DecodeFeeditemKey decodes the Feeditem configuration from a Feeditem key.
func DecodeFeeditemKey(key string) (*FeeditemKey, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, FeeditemKeyPrefix) {
		return nil, errors.Errorf("Not a Feeditem key: %v", keyString)
	}
	parts := strings.SplitN(keyString, separator, 3)
	if len(parts) != 3 {
		return nil, errors.Errorf("Invalid format of Feeditem key: %v", keyString)
	}
	res := &FeeditemKey{}
	var err error
	res.FeedURL, err = decodePart(parts[1])
	if err != nil {
		return nil, errors.Errorf("Failed to decode Feed URL of Feeditem key: %v because of %v", keyString, err)
	}
	res.GUID, err = decodePart(parts[2])
	if err != nil {
		return nil, errors.Errorf("Failed to decode GUID of Feeditem key: %v because of %v", keyString, err)
	}
	return res, nil
}

// ReadStatusPrefix is the key prefix for user's item read status.
const ReadStatusPrefix = "readstatus"

// CreateReadStatusKey creates a read status key prefix for user.
func CreateReadStatusKey(username string) string {
	return ReadStatusPrefix + separator + username
}

// CreateReadStatusKey creates a read status key prefix for user.
func (user *User) CreateReadStatusKey() string {
	return CreateReadStatusKey(user.username)
}

// ServerConfigKey is the key prefix for server configuration items.
const ServerConfigKey = "serverconfig"
