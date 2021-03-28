package data

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// separatator is used to separate parts of an item key.
const separator = "/"

// encodePart encodes a part of the key into a string so that it can be safely joined with separator.
func encodePart(part string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(part))
}

// decodePart decodes a part of the key into a string.
func decodePart(part string) (string, error) {
	res, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// lastSeenKeyPrefix is the key prefix for LastSeen entries.
const lastSeenKeyPrefix = "lastseen" + separator

// createLastSeenKey creates a LastSeen key for itemKey.
func createLastSeenKey(itemKey []byte) []byte {
	return append([]byte(lastSeenKeyPrefix), itemKey...)
}

// fetchStatusKeyPrefix is the key prefix for FetchStatus entries.
const fetchStatusKeyPrefix = "fetchstatus" + separator

// createFetchStatusKey creates a FetchStatus key for itemKey.
func createFetchStatusKey(itemKey []byte) []byte {
	return append([]byte(fetchStatusKeyPrefix), itemKey...)
}

// userKeyPrefix is the key prefix for User entries.
const userKeyPrefix = "user" + separator

// createUserKey creates a key for user.
func createUserKey(username string) []byte {
	return []byte(userKeyPrefix + encodePart(username))
}

// createKey creates a key for user.
func (user *User) createKey() []byte {
	return createUserKey(user.username)
}

// decodeUserKey decodes the username from a user key.
func decodeUserKey(key []byte) (*string, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, userKeyPrefix) {
		return nil, fmt.Errorf("not a user key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format of user key: %v", keyString)
	}
	username, err := decodePart(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode username %v: %w", keyString, err)
	}
	return &username, nil
}

// pagemonitorKeyPrefix is the key prefix for Pagemonitor.
const pagemonitorKeyPrefix = "pagemonitor" + separator

// CreateKey creates a key for a Pagemonitor entry.
func (pm *UserPagemonitor) CreateKey() []byte {
	keyURL := encodePart(pm.URL)
	keyMatch := encodePart(pm.Match)
	keyReplace := encodePart(pm.Replace)
	return []byte(pagemonitorKeyPrefix + keyURL + separator + keyMatch + separator + keyReplace)
}

// DecodePagemonitorKey decodes the Pagemonitor configuration from a Pagemonitor key.
func DecodePagemonitorKey(key []byte) (*UserPagemonitor, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, pagemonitorKeyPrefix) {
		return nil, fmt.Errorf("not a Pagemonitor key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid format of Pagemonitor key: %v", keyString)
	}
	res := &UserPagemonitor{}
	var err error
	res.URL, err = decodePart(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode URL of Pagemonitor key %v: %w", keyString, err)
	}
	res.Match, err = decodePart(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode Match of Pagemonitor key %v: %w", keyString, err)
	}
	res.Replace, err = decodePart(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode Replace of Pagemonitor key %v: %w", keyString, err)
	}
	return res, nil
}

// feedKeyPrefix is the key prefix for a UserFeed.
const feedKeyPrefix = "feed" + separator

// CreateKey creates a key for a Pagemonitor entry.
func (feed *UserFeed) CreateKey() []byte {
	keyURL := encodePart(feed.URL)
	return []byte(feedKeyPrefix + keyURL)
}

// feeditemKeyPrefix is the key prefix for Feeditem.
const feeditemKeyPrefix = "feeditem" + separator

// CreateKey creates a key for a Feeditem entry.
func (key *FeeditemKey) CreateKey() []byte {
	keyURL := encodePart(key.FeedURL)
	keyGUID := encodePart(key.GUID)
	return []byte(feeditemKeyPrefix + keyURL + separator + keyGUID)
}

// DecodeFeeditemKey decodes the Feeditem configuration from a Feeditem key.
func DecodeFeeditemKey(key []byte) (*FeeditemKey, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, feeditemKeyPrefix) {
		return nil, fmt.Errorf("not a Feeditem key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid format of Feeditem key: %v", keyString)
	}
	res := &FeeditemKey{}
	var err error
	res.FeedURL, err = decodePart(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode Feed URL of Feeditem key %v: %w", keyString, err)
	}
	res.GUID, err = decodePart(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode GUID of Feeditem key %v: %w", keyString, err)
	}
	return res, nil
}

// readStatusPrefix is the key prefix for an item read status.
const readStatusPrefix = "feed" + separator

// createReadStatusPrefix creates a read status key prefix for user.
func (user *User) createReadStatusPrefix() string {
	return readStatusPrefix + encodePart(user.username) + separator
}

// createReadStatusKey creates a read status key for an item key.
func (user *User) createReadStatusKey(itemKey []byte) []byte {
	return append([]byte(user.createReadStatusPrefix()), itemKey...)
}

// decodeReadStatusKey decodes the item key from the read status key.
func decodeReadStatusKey(key []byte) ([]byte, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, readStatusPrefix) {
		return nil, fmt.Errorf("not a read status key: %v", keyString)
	}
	parts := strings.SplitN(keyString, separator, 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid format of read status key: %v", keyString)
	}
	return []byte(parts[2]), nil
}

// serverConfigKeyPrefix is the key prefix for a ServerConfig item.
const serverConfigKeyPrefix = "serverconfig" + separator

// createServerConfigKey creates a key for a ServerConfig item.
func createServerConfigKey(varName string) []byte {
	return []byte(serverConfigKeyPrefix + encodePart(varName))
}

// decodeServerConfigKey decodes the name of a ServerConfig key.
func decodeServerConfigKey(key []byte) (string, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, serverConfigKeyPrefix) {
		return "", fmt.Errorf("not a config item key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid format of config item key: %v", keyString)
	}
	value, err := decodePart(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode config item value %v: %w", keyString, err)
	}
	return value, nil
}

// IsFeeditemKey returns if key is referencing a Feeditem.
func IsFeeditemKey(key []byte) bool {
	return strings.HasPrefix(string(key), feeditemKeyPrefix)
}

// IsPagemonitorKey returns if key is referencing a Pagemonitor entry.
func IsPagemonitorKey(key []byte) bool {
	return strings.HasPrefix(string(key), pagemonitorKeyPrefix)
}

// isFetchStatusKey returns if key is referencing a FetchStatus entry.
func isFetchStatusKey(key []byte) bool {
	return strings.HasPrefix(string(key), fetchStatusKeyPrefix)
}

// isUserKey returns if key is referencing a User entry.
func isUserKey(key []byte) bool {
	return strings.HasPrefix(string(key), userKeyPrefix)
}

// isReadStatusKey returns if key is referencing a read status entry.
func isReadStatusKey(key []byte) bool {
	return strings.HasPrefix(string(key), readStatusPrefix)
}

// isServerConfigKey returns if key is referencing a ServerConfig entry.
func isServerConfigKey(key []byte) bool {
	return strings.HasPrefix(string(key), serverConfigKeyPrefix)
}
