package data

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// This should be separate from regular data classes in case the structures change and we need to restore data from an older version

// BackupUser is a backup-friendly version of User.
type BackupUser struct {
	User
	Username  string
	ReadItems []string
}

// BackupFeeditem is a backup-friendly version of Feeditem and its FeeditemKey.
type BackupFeeditem struct {
	Feeditem
	FeeditemKey
}

// BackupPagemonitor is a backup-friendly version of PagemonitorPage and its configuration UserPagemonitor.
type BackupPagemonitor struct {
	PagemonitorPage
	UserPagemonitor
}

// BackupData is the toplevel structure exported in a backup.
type BackupData struct {
	Users        []*BackupUser
	Feeds        []*BackupFeeditem
	Pagemonitor  []*BackupPagemonitor
	ServerConfig map[string]string
}

// Backup returns a serialized copy of all data.
func (service *DBService) Backup() (string, error) {
	data := BackupData{}

	done := make(chan bool)
	userChan := make(chan *User)
	go func() {
		for user := range userChan {
			backupUser := &BackupUser{User: *user, Username: user.username}
			data.Users = append(data.Users, backupUser)
		}
		done <- true
	}()
	if err := service.ReadAllUsers(userChan); err != nil {
		return "", fmt.Errorf("Error backing up users because of %w", err)
	}
	<-done

	for _, user := range data.Users {
		readStatus, err := service.GetReadStatus(&user.User)
		if err != nil {
			return "", fmt.Errorf("Error backing up item read status for user because of %w", err)
		}
		user.ReadItems = make([]string, len(readStatus))
		for i, readItemKey := range readStatus {
			user.ReadItems[i] = string(readItemKey)
		}
	}

	feedChan := make(chan *Feeditem)
	go func() {
		for feedItem := range feedChan {
			// Flatten/reformat data
			backupFeeditem := &BackupFeeditem{
				Feeditem:    *feedItem,
				FeeditemKey: *feedItem.Key,
			}
			backupFeeditem.Feeditem.Key = nil
			data.Feeds = append(data.Feeds, backupFeeditem)
		}
		done <- true
	}()
	if err := service.ReadAllFeedItems(feedChan); err != nil {
		return "", fmt.Errorf("Error backing up feed items because of %w", err)
	}
	<-done

	pageChan := make(chan *PagemonitorPage)
	go func() {
		for page := range pageChan {
			// Flatten/reformat data
			backupPagemonitor := &BackupPagemonitor{
				PagemonitorPage: *page,
				UserPagemonitor: *page.Config,
			}
			backupPagemonitor.PagemonitorPage.Config = nil
			data.Pagemonitor = append(data.Pagemonitor, backupPagemonitor)
		}
		close(done)
	}()

	if err := service.ReadAllPages(pageChan); err != nil {
		return "", fmt.Errorf("Error backing up pagemonitor pages because of %w", err)
	}
	<-done

	serverConfig, err := service.GetAllConfigVariables()
	if err != nil {
		return "", fmt.Errorf("Error backing up server configuration because of %w", err)
	}
	data.ServerConfig = serverConfig

	value, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("Error marshaling json because of %w", err)
	}

	return string(value), nil
}

// Restore replaces database data with the provided serialized value.
func (service *DBService) Restore(value string) error {
	data := BackupData{}
	failed := false
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return fmt.Errorf("Error unmarshaling json because of %w", err)
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
		return fmt.Errorf("Failed to restore at least one item")
	}
	return nil
}
