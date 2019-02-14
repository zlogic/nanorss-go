package data

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const separator = "/"

const UserKeyPrefix = "user" + separator

func (user *User) CreateKey(id string) []byte {
	return []byte(UserKeyPrefix + id)
}

const PagemonitorKeyPrefix = "pagemonitor" + separator

func (pm *UserPagemonitor) CreateKey() []byte {
	keyURL := url.PathEscape(pm.URL)
	keyMatch := url.PathEscape(pm.Match)
	keyReplace := url.PathEscape(pm.Replace)
	keyFlags := url.PathEscape(pm.Flags)
	return []byte(PagemonitorKeyPrefix + keyURL + separator + keyMatch + separator + keyReplace + separator + keyFlags)
}

func DecodePagemonitorKey(key []byte) (*UserPagemonitor, error) {
	keyString := string(key)
	if !strings.HasPrefix(keyString, PagemonitorKeyPrefix) {
		return nil, errors.Errorf("Not a Pagemonitor key: %v", keyString)
	}
	parts := strings.Split(keyString, separator)
	if len(parts) != 5 {
		return nil, errors.Errorf("Invalid format of Pagemonitor key: %v", keyString)
	}
	res := &UserPagemonitor{}
	var err error
	res.URL, err = url.PathUnescape(parts[1])
	if err != nil {
		return nil, errors.Errorf("Failed to decode URL of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Match, err = url.PathUnescape(parts[2])
	if err != nil {
		return nil, errors.Errorf("Failed to decode Match of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Replace, err = url.PathUnescape(parts[3])
	if err != nil {
		return nil, errors.Errorf("Failed to decode Replace of Pagemonitor key: %v because of %v", keyString, err)
	}
	res.Flags, err = url.PathUnescape(parts[4])
	if err != nil {
		return nil, errors.Errorf("Failed to decode Flags of Pagemonitor key: %v because of %v", keyString, err)
	}
	return res, nil
}
