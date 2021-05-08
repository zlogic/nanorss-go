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
const lastSeenKeyPrefix = "lastseen"

// createLastSeenKey creates a LastSeen key for itemKey.
func createLastSeenKey(itemKey []byte) []byte {
	return append([]byte(lastSeenKeyPrefix+separator), itemKey...)
}

// fetchStatusKeyPrefix is the key prefix for FetchStatus entries.
const fetchStatusKeyPrefix = "fetchstatus"

// createFetchStatusKey creates a FetchStatus key for itemKey.
func createFetchStatusKey(itemKey []byte) []byte {
	return append([]byte(fetchStatusKeyPrefix+separator), itemKey...)
}

// userKeyPrefix is the key prefix for User entries.
const userKeyPrefix = "user"

// createUserKey creates a key for user.
func createUserKey(username string) []byte {
	return []byte(userKeyPrefix + separator + encodePart(username))
}

// createKey creates a key for user.
func (user *User) createKey() []byte {
	return createUserKey(user.username)
}

// pagemonitorKeyPrefix is the key prefix for Pagemonitor.
const pagemonitorKeyPrefix = "pagemonitor"

// CreateKey creates a key for a Pagemonitor entry.
func (pm *UserPagemonitor) CreateKey() []byte {
	keyURL := encodePart(pm.URL)
	keyMatch := encodePart(pm.Match)
	keyReplace := encodePart(pm.Replace)
	return []byte(pagemonitorKeyPrefix + separator + keyURL + separator + keyMatch + separator + keyReplace)
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
const feedKeyPrefix = "feed"

// feedContentsSuffix is the key suffix used to access contents of a Feeditem.
const feedContentsSuffix = separator + "contents"

// CreateKey creates a key for a UserFeed entry.
func (feed *UserFeed) CreateKey() []byte {
	keyURL := encodePart(feed.URL)
	return []byte(feedKeyPrefix + separator + keyURL)
}

// createItemsIndexKey creates an index key for a Feeditem entries in a Feed.
// This should generate the same key as FeeditemKey.createIndexKey.
func (feed *UserFeed) createItemsIndexKey() []byte {
	keyURL := encodePart(feed.URL)
	return []byte(feedKeyPrefix + separator + keyURL + separator)
}

// CreateKey creates a key for a Feeditem entry.
func (key *FeeditemKey) CreateKey() []byte {
	keyURL := encodePart(key.FeedURL)
	keyGUID := encodePart(key.GUID)
	return []byte(feedKeyPrefix + separator + keyURL + separator + keyGUID)
}

// createContentsKey creates a key for a Feeditem entry.
func (key *FeeditemKey) createContentsKey() []byte {
	feedItemKey := key.CreateKey()
	return append(feedItemKey, []byte(feedContentsSuffix)...)
}

// createIndexKey creates an index key for a Feeditem entry.
// This should generate the same key as UserFeed.createItemsIndexKey.
func (key *FeeditemKey) createIndexKey() []byte {
	keyURL := encodePart(key.FeedURL)
	return []byte(feedKeyPrefix + separator + keyURL + separator)
}

// DecodeFeeditemKey decodes the Feeditem configuration from a Feeditem key.
func DecodeFeeditemKey(key []byte) (*FeeditemKey, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, feedKeyPrefix) {
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
const readStatusPrefix = "readstatus"

// createReadStatusPrefix creates a read status key prefix for user.
func (user *User) createReadStatusPrefix() []byte {
	return []byte(readStatusPrefix + separator + encodePart(user.username))
}

// serverConfigKeyPrefix is the key prefix for a ServerConfig item.
const serverConfigKeyPrefix = "serverconfig"

// createServerConfigKey creates a key for a ServerConfig item.
func createServerConfigKey(varName string) []byte {
	return []byte(serverConfigKeyPrefix + separator + encodePart(varName))
}

// IsFeeditemKey returns if key is referencing a Feeditem.
func IsFeeditemKey(key []byte) bool {
	return strings.HasPrefix(string(key), feedKeyPrefix)
}

// IsPagemonitorKey returns if key is referencing a Pagemonitor entry.
func IsPagemonitorKey(key []byte) bool {
	return strings.HasPrefix(string(key), pagemonitorKeyPrefix)
}
