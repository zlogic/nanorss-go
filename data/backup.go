package data

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// This should be separate from regular data classes in case the structures change and we need to restore data from an older version

// BackupUser is a backup-friendly version of User.
type BackupUser struct {
	User
	Username  string
	ReadItems []string
}

// BackupData is the toplevel structure exported in a backup.
type BackupData struct {
	Users        []BackupUser
	Feeds        []Feeditem
	Pagemonitor  []PagemonitorPage
	ServerConfig map[string]string
}

// Backup returns a serialized copy of all data.
func (service DBService) Backup() (string, error) {
	data := BackupData{}

	done := make(chan bool)
	userChan := make(chan User)
	go func() {
		for user := range userChan {
			backupUser := BackupUser{User: user, Username: user.username}
			data.Users = append(data.Users, backupUser)
		}
		done <- true
	}()
	if err := service.ReadAllUsers(userChan); err != nil {
		return "", errors.Wrap(err, "Error backing up users")
	}
	<-done

	for i, user := range data.Users {
		readStatus, err := service.GetReadStatus(user.User)
		if err != nil {
			return "", errors.Wrap(err, "Error backing up item read status for user")
		}
		readItems := make([]string, len(readStatus))
		for i, readItemKey := range readStatus {
			readItems[i] = string(readItemKey)
		}
		data.Users[i].ReadItems = readItems
	}

	feedChan := make(chan Feeditem)
	go func() {
		for feedItem := range feedChan {
			data.Feeds = append(data.Feeds, feedItem)
		}
		done <- true
	}()
	if err := service.ReadAllFeedItems(feedChan); err != nil {
		return "", errors.Wrap(err, "Error backing up feed items")
	}
	<-done

	pageChan := make(chan PagemonitorPage)
	go func() {
		for page := range pageChan {
			data.Pagemonitor = append(data.Pagemonitor, page)
		}
		close(done)
	}()

	if err := service.ReadAllPages(pageChan); err != nil {
		return "", errors.Wrap(err, "Error backing up pagemonitor pages")
	}
	<-done

	serverConfig, err := service.GetAllConfigVariables()
	if err != nil {
		return "", errors.Wrap(err, "Error backing up server configuration")
	}
	data.ServerConfig = serverConfig

	value, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "Error marshaling json")
	}

	return string(value), nil
}

// Restore replaces database data with the provided serialized value.
func (service DBService) Restore(value string) error {
	data := BackupData{}
	failed := false
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return errors.Wrap(err, "Error unmarshaling json")
	}

	for _, user := range data.Users {
		user.username = user.Username
		if err := service.SaveUser(&user.User); err != nil {
			failed = true
			log.WithField("user", user).WithError(err).Printf("Error saving user")
		}
		for _, readStatus := range user.ReadItems {
			if err := service.SetReadStatus(user.User, []byte(readStatus), true); err != nil {
				failed = true
				log.WithField("user", user).WithField("item", readStatus).WithError(err).Printf("Error saving read status")
			}
		}
	}
	if err := service.SaveFeeditems(data.Feeds...); err != nil {
		failed = true
		log.WithError(err).Error("Error saving feed items")
	}
	for _, page := range data.Pagemonitor {
		if err := service.SavePage(page); err != nil {
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
