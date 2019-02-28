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

const LastSeenKeyPrefix = "lastseen" + separator

func CreateLastSeenKey(itemKey []byte) []byte {
	return append([]byte(LastSeenKeyPrefix), itemKey...)
}

func DecodeLastSeenKey(lastSeenKey []byte) []byte {
	return lastSeenKey[len(LastSeenKeyPrefix):]
}

const UserKeyPrefix = "user" + separator

func (s *UserService) CreateKey() []byte {
	return []byte(UserKeyPrefix + s.Username)
}

func DecodeUserKey(key []byte) (*string, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, UserKeyPrefix) {
		return nil, errors.Errorf("Not a user key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 2 {
		return nil, errors.Errorf("Invalid format of user key: %v", keyString)
	}
	return &parts[1], nil
}

const PagemonitorKeyPrefix = "pagemonitor" + separator

func (pm *UserPagemonitor) CreateKey() []byte {
	keyURL := encodePart(pm.URL)
	keyMatch := encodePart(pm.Match)
	keyReplace := encodePart(pm.Replace)
	return []byte(PagemonitorKeyPrefix + keyURL + separator + keyMatch + separator + keyReplace)
}

func DecodePagemonitorKey(key []byte) (*UserPagemonitor, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, PagemonitorKeyPrefix) {
		return nil, errors.Errorf("Not a Pagemonitor key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
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

const FeeditemKeyPrefix = "feeditem" + separator

func (key *FeeditemKey) CreateKey() []byte {
	keyURL := encodePart(key.FeedURL)
	keyGUID := encodePart(key.GUID)
	return []byte(FeeditemKeyPrefix + keyURL + separator + keyGUID)
}

func DecodeFeeditemKey(key []byte) (*FeeditemKey, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, FeeditemKeyPrefix) {
		return nil, errors.Errorf("Not a Feeditem key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
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

const ServerConfigKeyPrefix = "serverconfig" + separator

func CreateServerConfigKey(varName string) []byte {
	return []byte(ServerConfigKeyPrefix + encodePart(varName))
}

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
		return "", errors.Errorf("Failed to config item valye: %v because of %v", keyString, err)
	}
	return value, nil
}
