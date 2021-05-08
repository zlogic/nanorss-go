package data

import (
	"github.com/akrylysov/pogreb"
	"github.com/akrylysov/pogreb/fs"
)

var dbService *DBService

func resetDb() (err error) {
	if dbService != nil {
		it := dbService.db.Items()
		for {
			k, _, err := it.Next()
			if err == pogreb.ErrIterationDone {
				break
			} else if err != nil {
				return err
			}
			err = dbService.db.Delete(k)
			if err != nil {
				return err
			}
		}
		return
	}
	opts := pogreb.Options{FileSystem: fs.Mem}

	dbService, err = Open(opts)
	return
}

func getAllUsers() ([]*User, error) {
	usernames, err := dbService.GetUsers()
	if err != nil {
		return nil, err
	}
	users := make([]*User, 0, len(usernames))
	for _, username := range usernames {
		user, err := dbService.GetUser(username)
		if err != nil {
			return nil, err
		}
		if user == nil {
			continue
		}

		users = append(users, user)
	}
	return users, nil
}

func getFeedItems(user *User) ([]*Feeditem, error) {
	feedItems, err := dbService.GetFeeditems(user)
	if err != nil {
		return nil, err
	}

	for _, feedItem := range feedItems {
		fullItem, err := dbService.GetFeeditem(feedItem.Key)
		if err != nil {
			return nil, err
		}
		*feedItem = *fullItem
	}
	return feedItems, nil
}
