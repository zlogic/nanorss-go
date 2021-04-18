package data

import (
	"bytes"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// This should be separate from regular data classes in case the structures change and we need to restore data from an older version

// backupUser is a backup-friendly version of User.
type backupUser struct {
	User
	Username  string
	ReadItems []string
}

// backupFeeditem is a backup-friendly version of Feeditem and its FeeditemKey.
type backupFeeditem struct {
	Feeditem
	FeeditemKey
}

// backupPagemonitor is a backup-friendly version of PagemonitorPage and its configuration UserPagemonitor.
type backupPagemonitor struct {
	PagemonitorPage
	UserPagemonitor
}

// backupData is the toplevel structure exported in a backup.
type backupData struct {
	Users        []*backupUser
	Feeds        []*backupFeeditem
	Pagemonitor  []*backupPagemonitor
	ServerConfig map[string]string
}

// Backup returns a serialized copy of all data.
func (service *DBService) Backup() (string, error) {
	data := backupData{}

	usernames, err := service.GetUsers()
	if err != nil {
		return "", fmt.Errorf("failed to get usernames: %w", err)
	}

	for _, username := range usernames {
		user, err := service.GetUser(username)
		if err != nil {
			return "", fmt.Errorf("failed to get user %v: %w", username, err)
		}
		backupUser := &backupUser{User: *user, Username: user.username}
		data.Users = append(data.Users, backupUser)

		// Extract feeds.
		feeds, err := service.GetFeeditems(user)
		if err != nil {
			return "", fmt.Errorf("failed to get feeds for user %v: %w", username, err)
		}
		for _, feedItem := range feeds {
			// Check if item already exists.
			exists := false
			for i := range data.Feeds {
				if bytes.Equal(data.Feeds[i].FeeditemKey.CreateKey(), feedItem.Key.CreateKey()) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			// Flatten/reformat data.
			backupFeeditem := &backupFeeditem{
				Feeditem:    *feedItem,
				FeeditemKey: *feedItem.Key,
			}
			backupFeeditem.Feeditem.Key = nil

			data.Feeds = append(data.Feeds, backupFeeditem)
		}

		// Extract pages.
		pages, err := service.GetPages(user)
		if err != nil {
			return "", fmt.Errorf("failed to get pages for user %v: %w", username, err)
		}
		for _, page := range pages {
			// Check if item already exists.
			exists := false
			for i := range data.Pagemonitor {
				if bytes.Equal(data.Pagemonitor[i].CreateKey(), page.Config.CreateKey()) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			// Flatten/reformat data.
			backupPagemonitor := &backupPagemonitor{
				PagemonitorPage: *page,
				UserPagemonitor: *page.Config,
			}
			backupPagemonitor.PagemonitorPage.Config = nil
			data.Pagemonitor = append(data.Pagemonitor, backupPagemonitor)
		}
	}

	serverConfig, err := service.GetAllConfigVariables()
	if err != nil {
		return "", fmt.Errorf("failed to backup server configuration: %w", err)
	}
	data.ServerConfig = serverConfig

	for _, user := range data.Users {
		readItems, err := service.GetReadItems(&user.User)
		if err != nil {
			return "", fmt.Errorf("failed to get read status for user: %w", err)
		}

		user.ReadItems = make([]string, 0, len(readItems))

		for _, itemKey := range readItems {
			user.ReadItems = append(user.ReadItems, string(itemKey))
		}
	}

	value, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal json: %w", err)
	}

	return string(value), nil
}

// Restore replaces database data with the provided serialized value.
func (service *DBService) Restore(value string) error {
	data := backupData{}
	failed := false
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	for _, user := range data.Users {
		user.username = user.Username
		if err := service.SaveUser(&user.User); err != nil {
			failed = true
			log.WithField("user", user).WithError(err).Printf("Error saving user")
		}
		for _, readStatus := range user.ReadItems {
			if err := service.SetReadStatus(&user.User, []byte(readStatus), true); err != nil {
				failed = true
				log.WithField("user", user).WithField("item", readStatus).WithError(err).Printf("Error saving read status")
			}
		}
	}
	convertFeeditems := func() []*Feeditem {
		convertedFeeditems := make([]*Feeditem, 0, len(data.Feeds))

		for _, feedItem := range data.Feeds {
			feedItem.Key = &feedItem.FeeditemKey
			convertedFeeditems = append(convertedFeeditems, &feedItem.Feeditem)
		}
		return convertedFeeditems
	}
	if err := service.SaveFeeditems(convertFeeditems()...); err != nil {
		failed = true
		log.WithError(err).Error("Error saving feed items")
	}
	for _, page := range data.Pagemonitor {
		page.Config = &page.UserPagemonitor
		if err := service.SavePage(&page.PagemonitorPage); err != nil {
			failed = true
			log.WithField("page", page).WithError(err).Error("Error saving page")
		}
	}
	for key, value := range data.ServerConfig {
		if err := service.SetConfigVariable(key, value); err != nil {
			failed = true
			log.WithField("key", key).WithField("value", value).WithError(err).Error("Error saving config variable")
		}
	}
	if failed {
		return fmt.Errorf("failed to restore at least one item")
	}
	return nil
}
